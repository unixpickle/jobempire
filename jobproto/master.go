package jobproto

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/unixpickle/gobplexer"
)

const (
	pingInterval = time.Second * 30
	pingMaxDelay = time.Minute
)

// A Master provides control over the master side of a
// master-slave connection.
type Master interface {
	// StartJob creates a new job over the connection.
	// Multiple jobs may be running simultaneously.
	StartJob() (MasterJob, error)

	// Terminate terminates the connection immediately.
	// If any jobs were running, the slave and master
	// will be left to handle the cleanup.
	// All created jobs and tasks will fail with an error
	// when they try to communicate with the remote end.
	Terminate()
}

// A MasterJob provides control over a job.
type MasterJob interface {
	// Finish terminates the job.
	// If no tasks were running, it performs a graceful
	// shutdown of the job.
	// If tasks were running, their connections are closed
	// and they must handle the failure.
	Finish() error

	// Run runs a task in the context of the job.
	// It blocks until the task has completed on both the
	// master and the slave.
	// It returns an error if the task fails on either end,
	// or if the job is finished early.
	// Multiple tasks may be run on a job simultaneously.
	Run(t Task) error
}

type masterConn struct {
	connector gobplexer.Connector
}

// NewMasterConn creates a Master from a net.Conn.
// If the handshake fails, c is closed.
func NewMasterConn(c net.Conn) (Master, error) {
	return newMasterConn(gobplexer.NewConnectionConn(c))
}

func newMasterConn(rawCon gobplexer.Connection) (Master, error) {
	rootCon := gobplexer.MultiplexConnector(rawCon)
	c, err := gobplexer.KeepaliveConnector(rootCon, pingInterval, pingMaxDelay)
	if err != nil {
		rawCon.Close()
		return nil, err
	}
	return &masterConn{connector: gobplexer.MultiplexConnector(c)}, nil
}

func (m *masterConn) StartJob() (MasterJob, error) {
	c, err := m.connector.Connect()
	if err != nil {
		return nil, err
	}
	return &masterJob{connector: gobplexer.MultiplexConnector(c)}, nil
}

func (m *masterConn) Terminate() {
	m.connector.Close()
}

type masterJob struct {
	connector gobplexer.Connector
}

func (m *masterJob) Finish() error {
	return m.connector.Close()
}

func (m *masterJob) Run(t Task) error {
	taskConn, err := m.connector.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect for task: %s", err)
	}
	defer taskConn.Close()

	connector := gobplexer.MultiplexConnector(taskConn)
	statusConn, err := connector.Connect()
	if err != nil {
		return fmt.Errorf("failed to establish status channel: %s", err)
	}

	dataConn, err := connector.Connect()
	if err != nil {
		return fmt.Errorf("failed to establish data channel: %s", err)
	}

	if err := dataConn.Send(t); err != nil {
		return fmt.Errorf("failed to send task: %s", err)
	}
	runErr := t.RunMaster(dataConn)
	dataConn.Close()

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
		return fmt.Errorf("unexpected status type: %T", value)
	}
}
