package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"

	"github.com/unixpickle/jobempire/jobproto"
)

func SlaveMain(host string, port int, password string) {
	conn, err := net.Dial("tcp", host+":"+strconv.Itoa(port))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to connect:", err)
		os.Exit(1)
	}
	slave, err := jobproto.NewSlaveConnAuth(conn, password)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to authenticate:", err)
		os.Exit(1)
	}
	defer slave.Close()

	for {
		job, err := slave.NextJob()
		if err != nil {
			break
		}
		go func() {
			rootDir, err := ioutil.TempDir("", "job")
			if err != nil {
				slave.Close()
				return
			}
			defer os.Remove(rootDir)
			job.RunTasks(rootDir)
		}()
	}
}
