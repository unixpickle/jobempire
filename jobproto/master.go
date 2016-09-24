package jobproto

import (
	"fmt"
	"net"

	"github.com/unixpickle/gobplexer"
)

type masterConn struct {
	connector gobplexer.Connector
}

// NewMasterConnNet creates a MasterConn from a net.Conn.
func NewMasterConnNet(c net.Conn) MasterConn {
	gobCon := gobplexer.NewConnectionConn(c)
	return &masterConn{connector: gobplexer.MultiplexConnector(gobCon)}
}

func (m *masterConn) StartJob() (MasterJob, error) {
	connection, err := m.connector.Connect()
	if err != nil {
		return nil, err
	}
	return newMasterJob(connection), nil
}

func (m *masterConn) Terminate() {
	m.connector.Close()
}

type masterJob struct {
	connector gobplexer.Connector
}

func newMasterJob(c gobplexer.Connection) *masterJob {
	return &masterJob{connector: gobplexer.MultiplexConnector(c)}
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

	statusChan := readStatusObj(statusConn)

	dataConn.Send(t)
	runErr := t.RunMaster(dataConn)
	if err := statusConn.Send(runErr); err != nil {
		return err
	}

	if runErr != nil {
		return runErr
	} else {
		return <-statusChan
	}
}

// readStatusObj reads the first error/nil value from the
// connection and spits it out through a channel.
// It continues to read from the connection until it fails,
// preventing unexpected packets on the connection from
// holding up an entire multiplexed connection.
func readStatusObj(c gobplexer.Connection) <-chan error {
	res := make(chan error, 1)
	go func() {
		value, err := c.Receive()
		if err != nil {
			res <- err
			close(res)
			return
		}
		if value == nil {
			res <- nil
		} else if errVal, ok := value.(error); ok {
			res <- errVal
		} else {
			res <- fmt.Errorf("unexpected status type: %T", value)
		}
		close(res)

		// We must continue to read from the Connection so
		// that it cannot gum up the whole multiplexed
		// connection.
		for {
			_, err = c.Receive()
			if err != nil {
				break
			}
		}
	}()
	return res
}
