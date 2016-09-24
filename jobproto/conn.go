package jobproto

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

// An objectConn facilitates Go object transfer.
//
// All methods on an objectConn are safe to call
// concurrently.
type objectConn interface {
	// Send sends an object over the connection.
	Send(obj interface{}) error

	// Receive receives an object.
	Receive() (interface{}, error)

	// Close closes the connection.
	Close() error
}

// A gobConn uses gob to implement an objectConn over a
// network connection.
type gobConn struct {
	conn net.Conn

	readLock sync.Mutex
	dec      *gob.Decoder

	writeLock sync.Mutex
	enc       *gob.Encoder
}

func newGobConn(c net.Conn) *gobConn {
	return &gobConn{
		conn: c,
		dec:  gob.NewDecoder(c),
		enc:  gob.NewEncoder(c),
	}
}

func (g *gobConn) Send(obj interface{}) error {
	g.writeLock.Lock()
	defer g.writeLock.Unlock()
	return g.enc.Encode(obj)
}

func (g *gobConn) Receive() (interface{}, error) {
	g.readLock.Lock()
	defer g.readLock.Unlock()
	var obj interface{}
	if err := g.dec.Decode(&obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func (g *gobConn) Close() error {
	return g.conn.Close()
}

type multiplexMsg struct {
	ID       int64
	New      bool
	Close    bool
	CloseAck bool
	Payload  interface{}
}

type connMsg struct {
	Payload interface{}
	EOF     bool
}

// A connMultiplexer multiplexes an objectConn by giving
// IDs to different virtual connections.
type connMultiplexer struct {
	conn objectConn

	lock    sync.RWMutex
	curID   int64
	chanMap map[int64]chan connMsg

	idChan chan int64

	termLock      sync.Mutex
	terminated    bool
	terminateChan chan struct{}

	errLock sync.Mutex
	err     error
}

func newConnMultiplexer(conn objectConn, accepts bool) *connMultiplexer {
	c := &connMultiplexer{
		conn:          conn,
		chanMap:       map[int64]chan connMsg{},
		terminateChan: make(chan struct{}),
	}
	if accepts {
		c.idChan = make(chan int64)
	}
	go c.receiveLoop()
	return c
}

func (c *connMultiplexer) Accept() (int64, error) {
	select {
	case id := <-c.idChan:
		return id, nil
	case <-c.terminateChan:
		return 0, c.firstError()
	}
}

func (c *connMultiplexer) Establish() (int64, error) {
	c.lock.Lock()
	id := c.curID
	c.curID++
	c.chanMap[id] = make(chan connMsg, 1)
	c.lock.Unlock()

	select {
	case <-c.terminateChan:
		return 0, c.firstError()
	default:
	}

	err := c.conn.Send(multiplexMsg{
		ID:  id,
		New: true,
	})
	return id, err
}

func (c *connMultiplexer) CloseID(id int64) error {
	return c.conn.Send(multiplexMsg{
		ID:    id,
		Close: true,
	})
}

func (c *connMultiplexer) Send(id int64, obj interface{}) error {
	return c.conn.Send(multiplexMsg{
		ID:      id,
		Payload: obj,
	})
}

func (c *connMultiplexer) Receive(id int64) (interface{}, error) {
	c.lock.RLock()
	ch := c.chanMap[id]
	c.lock.RUnlock()

	if ch == nil {
		return nil, io.EOF
	}

	select {
	case res, ok := <-ch:
		if !ok {
			return nil, c.firstError()
		}
		if res.EOF {
			return nil, io.EOF
		}
		return res.Payload, nil
	case <-c.terminateChan:
		return nil, c.firstError()
	}
}

func (c *connMultiplexer) Close() error {
	c.gotError(errors.New("multiplexer closed"))
	c.termLock.Lock()
	if c.terminated {
		c.termLock.Unlock()
		return c.firstError()
	}
	c.terminated = true
	close(c.terminateChan)
	c.termLock.Unlock()
	return c.conn.Close()
}

func (c *connMultiplexer) receiveLoop() {
	defer func() {
		c.Close()
		c.lock.Lock()
		for _, ch := range c.chanMap {
			close(ch)
		}
		c.chanMap = map[int64]chan connMsg{}
		c.lock.Unlock()
	}()
	for {
		msg, err := c.conn.Receive()
		if err != nil {
			c.gotError(err)
			return
		}
		msgVal, ok := msg.(multiplexMsg)
		if !ok {
			c.gotError(fmt.Errorf("unexpected multiplexer message type: %T", msg))
			return
		}
		switch true {
		case msgVal.New:
			if c.idChan == nil {
				c.gotError(errors.New("multiplexer cannot accept connections"))
				return
			}
			c.lock.Lock()
			id := c.curID
			c.curID++
			c.chanMap[id] = make(chan connMsg, 1)
			c.lock.Unlock()
			select {
			case c.idChan <- id:
			case <-c.terminateChan:
				return
			}
		case msgVal.Close:
			c.conn.Send(multiplexMsg{
				ID:       msgVal.ID,
				CloseAck: true,
			})
			fallthrough
		case msgVal.CloseAck:
			c.lock.Lock()
			ch := c.chanMap[msgVal.ID]
			delete(c.chanMap, msgVal.ID)
			c.lock.Unlock()
			if ch != nil {
				select {
				case ch <- connMsg{EOF: true}:
					close(ch)
				case <-c.terminateChan:
					return
				}
			}
		default:
			c.lock.RLock()
			ch := c.chanMap[msgVal.ID]
			c.lock.RUnlock()
			if ch != nil {
				select {
				case ch <- connMsg{Payload: msgVal.Payload}:
				case <-c.terminateChan:
					return
				}
			}
		}
	}
}

func (c *connMultiplexer) gotError(e error) {
	c.errLock.Lock()
	defer c.errLock.Unlock()
	if c.err == nil {
		c.err = e
	}
}

func (c *connMultiplexer) firstError() error {
	c.errLock.Lock()
	defer c.errLock.Unlock()
	return c.err
}

type subConn struct {
	multiplexer *connMultiplexer
	id          int64
}

func (s *subConn) Send(obj interface{}) error {
	return s.multiplexer.Send(s.id, obj)
}

func (s *subConn) Receive() (interface{}, error) {
	return s.multiplexer.Receive(s.id)
}

func (s *subConn) Close() error {
	return s.multiplexer.CloseID(s.id)
}
