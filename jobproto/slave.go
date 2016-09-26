package jobproto

import (
	"encoding/gob"
	"fmt"
	"net"
	"runtime"
	"sync"

	"github.com/unixpickle/gobplexer"
)

func init() {
	gob.Register(SlaveInfo{})
}

// SlaveInfo stores global information about a slave.
type SlaveInfo struct {
	// NumCPU indicates the number of physical CPUs
	// on the slave.
	NumCPU int

	// MaxProcs indicates the value of GOMAXPROCS.
	MaxProcs int

	// OS indicates the value of GOOS.
	OS string

	// Arch indicates the value of GOARCH.
	Arch string
}

// CurrentSlaveInfo computes the SlaveInfo for the current
// Go process.
func CurrentSlaveInfo() SlaveInfo {
	return SlaveInfo{
		NumCPU:   runtime.NumCPU(),
		MaxProcs: runtime.GOMAXPROCS(0),
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
	}
}

// A Slave provides a stream of jobs from a remote Master.
type Slave interface {
	// NextJob receives the next job from the master.
	// This will return an error if the remote end has
	// terminated the connection.
	// After NextJob fails, the Close method should still
	// be called.
	NextJob() (SlaveJob, error)

	// Close terminates the connection.
	// Any pending jobs will fail when they try to talk to
	// the remote master.
	//
	// Close may be called multiple times, but any time
	// after the first will have no effect.
	Close() error
}

// A SlaveJob provides a stream of tasks from a Master.
type SlaveJob interface {
	// RunTasks runs the tasks from the Master.
	RunTasks(rootDir string)
}

type slaveConn struct {
	conn     net.Conn
	listener gobplexer.Listener
}

// NewSlaveConn creates a Slave from a net.Conn.
// If the handshake fails, c is closed.
func NewSlaveConn(c net.Conn) (s Slave, e error) {
	defer func() {
		if e != nil {
			c.Close()
		}
	}()

	gobCon := gobplexer.NetConnection(c)
	rootListener := gobplexer.MultiplexListener(gobCon)
	keptAlive, err := gobplexer.KeepaliveListener(rootListener,
		pingInterval, pingMaxDelay)
	if err != nil {
		return nil, err
	}
	listener := gobplexer.MultiplexListener(keptAlive)

	statusConn, err := listener.Accept()
	if err != nil {
		return nil, fmt.Errorf("accept info connection: %s", err)
	}
	if err := statusConn.Send(CurrentSlaveInfo()); err != nil {
		return nil, fmt.Errorf("send slave info: %s", err)
	}

	// Leave the statusConn open so that the remote end can
	// poll from it and tell when the connection has died.

	return &slaveConn{
		conn:     c,
		listener: listener,
	}, nil
}

func (s *slaveConn) NextJob() (SlaveJob, error) {
	c, err := s.listener.Accept()
	if err != nil {
		return nil, err
	}
	return &slaveJob{listener: gobplexer.MultiplexListener(c)}, nil
}

func (s *slaveConn) Close() error {
	return s.conn.Close()
}

type slaveJob struct {
	listener gobplexer.Listener
}

func (s *slaveJob) RunTasks(rootDir string) {
	var wg sync.WaitGroup
	defer func() {
		s.listener.Close()
		wg.Wait()
	}()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		wg.Add(1)
		go func() {
			defer conn.Close()
			defer wg.Done()
			s.runTask(rootDir, conn)
		}()
	}
}

func (s *slaveJob) runTask(rootDir string, conn gobplexer.Connection) {
	listener := gobplexer.MultiplexListener(conn)

	statusConn, err := listener.Accept()
	if err != nil {
		return
	}

	dataConn, err := listener.Accept()
	if err != nil {
		return
	}

	logConn, err := listener.Accept()
	if err != nil {
		return
	}

	taskObj, err := dataConn.Receive()
	if err != nil {
		return
	}

	task, ok := taskObj.(Task)
	if !ok {
		return
	}
	runErr := task.RunSlave(rootDir, slaveTaskConn{dataConn, logConn})
	logConn.Close()
	dataConn.Close()

	if runErr != nil {
		statusConn.Send(runErr.Error())
	} else {
		statusConn.Send(nil)
	}
	statusConn.Receive()
}

type slaveTaskConn struct {
	gobplexer.Connection
	logConn gobplexer.Connection
}

func (s slaveTaskConn) Log(message string) {
	s.logConn.Send(message)
}
