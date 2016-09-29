package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/unixpickle/jobempire/jobadmin"
	"github.com/unixpickle/jobempire/jobproto"
)

func MasterMain(slavePort, adminPort int, slavePass, adminPass string, jobFile string) {
	jobs, err := readJobs(jobFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to read jobs:", err)
		os.Exit(1)
	}

	slaveListener, err := net.Listen("tcp", ":"+strconv.Itoa(slavePort))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to listen for slaves:", err)
		os.Exit(1)
	}

	adminListener, err := net.Listen("tcp", ":"+strconv.Itoa(adminPort))
	if err != nil {
		slaveListener.Close()
		fmt.Fprintln(os.Stderr, "Failed to listen for admins:", err)
		os.Exit(1)
	}

	log.Println("Listening on ports", slavePort, "and", adminPort)

	defer slaveListener.Close()
	defer adminListener.Close()

	handler := &MasterHandler{
		Scheduler: jobadmin.NewScheduler(),
		Auth:      NewMasterAuth(adminPass),
		Templates: parseTemplates(),
		JobsPath:  jobFile,
	}
	handler.Scheduler.SetJobs(jobs)

	go http.Serve(adminListener, handler)
	go func() {
		for {
			conn, err := slaveListener.Accept()
			if err != nil {
				return
			}
			if handler.Scheduler.Terminated() {
				conn.Close()
				return
			}
			go func() {
				master, err := jobproto.NewMasterConnAuth(conn, slavePass)
				if err != nil {
					log.Println("Slave", conn.RemoteAddr(), "failed to authenticate.")
					return
				}
				log.Println("Slave", conn.RemoteAddr(), "successfully joined.")
				handler.Scheduler.AddMaster(jobadmin.RunLiveMaster(master), false)
			}()
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	fmt.Println("\nShutting down...")

	handler.Scheduler.Terminate()
	handler.Scheduler.Wait(nil)
}

func readJobs(file string) ([]*jobadmin.Job, error) {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, nil
	}
	contents, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var jobs []*jobadmin.Job
	if err := json.Unmarshal(contents, &jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func parseTemplates() *template.Template {
	files := []string{"assets/header.html", "assets/jobs.html", "assets/login.html"}
	var body bytes.Buffer
	for _, f := range files {
		data, err := Asset(f)
		if err != nil {
			panic(err)
		}
		body.Write(data)
	}
	return template.Must(template.New("master").Parse(body.String()))
}
