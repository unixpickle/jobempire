package jobproto

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/unixpickle/gobplexer"
)

const (
	pingInterval = time.Second * 30
	pingMaxDelay = time.Minute
)

// A LogEntry stores one line of logged output from either
// end of a Task.
type LogEntry struct {
	FromMaster bool
	Message    string
}

// A Master provides control over the master side of a
// master-slave connection.
type Master interface {
	// SlaveInfo returns various information about the
	// remote slave.
	SlaveInfo() SlaveInfo

	// StartJob creates a new job on the slave.
	// Multiple jobs may be running simultaneously.
	StartJob() (MasterJob, error)

	// Wait waits for the remote end to disconnect or
	// for the Master to be closed.
	Wait()

	// Close terminates the connection.
	// It should be called to cleanup any connections or
	// resources being consumed by the Master.
	//
	// If any jobs were running, the slave and master will
	// be left to handle the cleanup.
	// All created jobs and tasks will fail with an error
	// when they try to communicate with the remote end.
	//
	// Close may be called multiple times, but any time
	// after the first will have no effect.
	Close() error
}

// A MasterJob provides control over a job.
type MasterJob interface {
	// Close terminates the job.
	// It should be called to cleanup a job once it has
	// failed or completed properly.
	//
	// If no tasks were running, it performs a graceful
	// shutdown of the job.
	// If tasks were running, their connections are closed
	// and they must handle the failure.
	//
	// Close may be called multiple times, but any time
	// after the first will have no effect.
	Close() error

	// Run runs a task in the context of the job.
	// It blocks until the task has completed on both ends.
	// It returns an error if the task fails on either end,
	// or if the job is closed.
	//
	// The log channel should be read from continually in
	// order to prevent the task from blocking indefinitely.
	// Alternatively, the log channel may be nil to ignore
	// logged output.
	//
	// Multiple tasks may be run on a job simultaneously.
	Run(t Task, log chan<- LogEntry) error
}

type masterConn struct {
	connector gobplexer.Connector
	doneChan  <-chan struct{}
	info      SlaveInfo
}

// NewMasterConn creates a Master from a net.Conn.
// If the handshake fails, c is closed.
func NewMasterConn(c net.Conn) (m Master, e error) {
	defer func() {
		if e != nil {
			c.Close()
		}
	}()

	gobCon := gobplexer.NetConnection(c)
	rootConnector := gobplexer.MultiplexConnector(gobCon)
	keptAlive, err := gobplexer.KeepaliveConnector(rootConnector,
		pingInterval, pingMaxDelay)
	if err != nil {
		return nil, err
	}

	connector := gobplexer.MultiplexConnector(keptAlive)
	if conn, err := connector.Connect(); err != nil {
		return nil, fmt.Errorf("connect for info: %s", err)
	} else if infoObj, err := conn.Receive(); err != nil {
		return nil, fmt.Errorf("read info: %s", err)
	} else if info, ok := infoObj.(SlaveInfo); !ok {
		return nil, fmt.Errorf("invalid info type: %T", infoObj)
	} else {
		doneChan := make(chan struct{})
		go func() {
			// The other end leaves this sub-connection open so we
			// can poll from it to see when the connection dies.
			conn.Receive()
			c.Close()
			close(doneChan)
		}()
		return &masterConn{
			connector: connector,
			doneChan:  doneChan,
			info:      info,
		}, nil
	}
}

func (m *masterConn) SlaveInfo() SlaveInfo {
	return m.info
}

func (m *masterConn) StartJob() (MasterJob, error) {
	c, err := m.connector.Connect()
	if err != nil {
		return nil, err
	}
	return &masterJob{connector: gobplexer.MultiplexConnector(c)}, nil
}

func (m *masterConn) Wait() {
	<-m.doneChan
}

func (m *masterConn) Close() error {
	return m.connector.Close()
}

type masterJob struct {
	connector gobplexer.Connector
}

func (m *masterJob) Close() error {
	return m.connector.Close()
}

func (m *masterJob) Run(t Task, log chan<- LogEntry) error {
	taskConn, err := m.connector.Connect()
	if err != nil {
		return fmt.Errorf("connect task: %s", err)
	}
	defer taskConn.Close()

	connector := gobplexer.MultiplexConnector(taskConn)
	statusConn, err := connector.Connect()
	if err != nil {
		return fmt.Errorf("establish status channel: %s", err)
	}

	dataConn, err := connector.Connect()
	if err != nil {
		return fmt.Errorf("establish data channel: %s", err)
	}

	logConn, err := connector.Connect()
	if err != nil {
		return fmt.Errorf("establish log channel: %s", err)
	}

	var logWg sync.WaitGroup
	logWg.Add(1)
	go func() {
		defer logWg.Done()
		for {
			msg, err := logConn.Receive()
			if err != nil {
				return
			}
			msgStr, ok := msg.(string)
			if ok && log != nil {
				log <- LogEntry{Message: msgStr}
			}
		}
	}()

	if err := dataConn.Send(t); err != nil {
		return fmt.Errorf("send task: %s", err)
	}
	runErr := t.RunMaster(masterTaskConn{dataConn, log})
	dataConn.Close()
	logWg.Wait()

	remoteStatus := readStatusObj(statusConn)
	if runErr != nil {
		return runErr
	} else if remoteStatus != nil {
		return fmt.Errorf("external error: %s", remoteStatus)
	} else {
		return nil
	}
}

// readStatusObj reads the first error/nil value from the
// connection.
func readStatusObj(c gobplexer.Connection) error {
	value, err := c.Receive()
	if err != nil {
		return err
	}

	// Allow the other end to fully disconnect.
	c.Send(nil)

	if value == nil {
		return nil
	} else if errVal, ok := value.(string); ok {
		return errors.New(errVal)
	} else {
		return fmt.Errorf("invalid status type: %T", value)
	}
}

type masterTaskConn struct {
	gobplexer.Connection
	log chan<- LogEntry
}

func (m masterTaskConn) Log(message string) {
	if m.log != nil {
		m.log <- LogEntry{FromMaster: true, Message: message}
	}
}
