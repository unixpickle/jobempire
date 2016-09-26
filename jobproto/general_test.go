package jobproto

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"sync"
)

const maxTestListenAttempts = 100

func TestingMasterSlave() (Master, Slave, error) {
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

		var master Master
		var slave Slave
		var masterErr error
		var slaveErr error

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			master, masterErr = NewMasterConn(masterConn)
		}()
		go func() {
			defer wg.Done()
			slave, slaveErr = NewSlaveConn(slaveConn)
		}()
		wg.Wait()

		if masterErr != nil {
			return nil, nil, fmt.Errorf("master creation error: %s", masterErr)
		}
		if slaveErr != nil {
			return nil, nil, fmt.Errorf("slave creation error: %s", slaveErr)
		}
		return master, slave, nil
	}
	return nil, nil, errors.New("could not find port to listen on")
}
