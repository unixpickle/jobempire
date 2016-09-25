package jobproto

import (
	"net"
	"sync"

	"github.com/unixpickle/gobplexer"
)

type slaveConn struct {
	listener gobplexer.Listener
}

// NewSlaveConnNet creates a SlaveConn from a net.Conn.
func NewSlaveConnNet(n net.Conn) (SlaveConn, error) {
	rootCon := gobplexer.MultiplexListener(gobplexer.NewConnectionConn(n))
	c, err := gobplexer.KeepaliveListener(rootCon, pingInterval, pingMaxDelay)
	if err != nil {
		return nil, err
	}
	return &slaveConn{listener: gobplexer.MultiplexListener(c)}, nil
}

func (s *slaveConn) NextJob() (SlaveJob, error) {
	c, err := s.listener.Accept()
	if err != nil {
		s.listener.Close()
		return nil, err
	}
	return &slaveJob{listener: gobplexer.MultiplexListener(c)}, nil
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
			listener := gobplexer.MultiplexListener(conn)

			statusConn, err := listener.Accept()
			if err != nil {
				return
			}

			dataConn, err := listener.Accept()
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
			runErr := task.RunSlave(rootDir, dataConn)
			statusConn.Send(runErr)
		}()
	}
}
