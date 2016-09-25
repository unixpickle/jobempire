package jobproto

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
)

const maxTestListenAttempts = 100

func TestingMasterSlave() (MasterConn, SlaveConn, error) {
	for i := 0; i < maxTestListenAttempts; i++ {
		port := rand.Intn(10000) + 1024
		server, err := net.Listen("tcp", ":"+strconv.Itoa(port))
		if err != nil {
			continue
		}
		accepted := make(chan net.Conn, 1)
		go func() {
			c, _ := server.Accept()
			accepted <- c
		}()
		slaveConn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(port))
		if err != nil {
			server.Close()
			return nil, nil, err
		}
		masterConn := <-accepted
		server.Close()
		if masterConn == nil {
			return nil, nil, errors.New("failed to accept connection")
		}

		master, err := NewMasterConnNet(masterConn)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create master: %s", err)
		}
		slave, err := NewSlaveConnNet(slaveConn)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create slave: %s", err)
		}
		return master, slave, nil
	}
	return nil, nil, errors.New("could not find port to listen on")
}
