package jobproto

import (
	"fmt"
	"net"
	"time"

	"github.com/unixpickle/gobplexer"
)

const (
	pingInterval = time.Second * 30
	pingMaxDelay = time.Minute
)

type masterConn struct {
	connector gobplexer.Connector
}

// NewMasterConnNet creates a MasterConn from a net.Conn.
func NewMasterConnNet(n net.Conn) (MasterConn, error) {
	rootCon := gobplexer.MultiplexConnector(gobplexer.NewConnectionConn(n))
	c, err := gobplexer.KeepaliveConnector(rootCon, pingInterval, pingMaxDelay)
	if err != nil {
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
	} else if errVal, ok := value.(error); ok {
		return errVal
	} else {
		return fmt.Errorf("unexpected status type: %T", value)
	}
}
