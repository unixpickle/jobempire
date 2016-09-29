package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
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

	m := &Master{
		Scheduler: jobadmin.NewScheduler(),
		AdminPass: adminPass,
		Templates: parseTemplates(),
		JobsPath:  jobFile,

		Cookies: sessions.NewCookieStore(securecookie.GenerateRandomKey(16),
			securecookie.GenerateRandomKey(16)),
	}
	m.Scheduler.SetJobs(jobs)

	go http.Serve(adminListener, m)
	go func() {
		for {
			conn, err := slaveListener.Accept()
			if err != nil {
				return
			}
			if m.Scheduler.Terminated() {
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
				m.Scheduler.AddMaster(jobadmin.RunLiveMaster(master), false)
			}()
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	fmt.Println("\nShutting down...")

	m.Scheduler.Terminate()
	m.Scheduler.Wait(nil)
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

type Master struct {
	Scheduler *jobadmin.Scheduler
	AdminPass string
	Templates *template.Template

	Cookies *sessions.CookieStore

	JobsLock sync.Mutex
	JobsPath string
}

func (m *Master) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cleanPath := path.Clean(r.URL.Path)
	if strings.HasPrefix(cleanPath, "/assets/") {
		m.serveAsset(w, r, cleanPath)
		return
	}

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	switch cleanPath {
	case "/":
		if m.IsAuth(r) {
			http.Redirect(w, r, "/jobs", http.StatusTemporaryRedirect)
		} else {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		}
		return
	case "/login":
		m.ServeLoginPage(w, r)
		return
	}

	if !m.IsAuth(r) {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	switch cleanPath {
	case "/jobs":
		m.ServeJobsPage(w, r)
	default:
		m.serveNotFound(w, r)
	}
}

func (m *Master) serveAsset(w http.ResponseWriter, r *http.Request, cleanPath string) {
	if asset, err := Asset(cleanPath[1:]); err != nil {
		m.serveNotFound(w, r)
	} else {
		mimeType := mime.TypeByExtension(path.Ext(cleanPath))
		if mimeType == "" {
			mimeType = "text/plain"
		}
		w.Header().Set("Content-Type", mimeType)
		w.Header().Set("Content-Length", strconv.Itoa(len(asset)))
		w.Write(asset)
	}
}

func (m *Master) serveNotFound(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func (m *Master) saveJobs(jobs []*jobadmin.Job) error {
	encoded, err := json.Marshal(jobs)
	if err != nil {
		return err
	}
	m.JobsLock.Lock()
	defer m.JobsLock.Unlock()
	return ioutil.WriteFile(m.JobsPath, encoded, 0755)
}
