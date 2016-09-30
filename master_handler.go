package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"math/rand"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/unixpickle/jobempire/jobadmin"
)

type MasterHandler struct {
	Scheduler *jobadmin.Scheduler
	Auth      *MasterAuth
	Templates *template.Template

	JobsLock sync.Mutex
	JobsPath string
}

func (m *MasterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		if m.Auth.IsAuth(r) {
			http.Redirect(w, r, "/jobs", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}
		return
	case "/login":
		m.ServeLoginPage(w, r)
		return
	}

	if !m.Auth.IsAuth(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	switch cleanPath {
	case "/jobs":
		m.ServeJobsPage(w, r)
	case "/slaves":
		m.ServeSlavesPage(w, r)
	case "/addjob":
		m.ServeAddJobPage(w, r)
	case "/editjob":
		m.ServeEditJobPage(w, r)
	case "/savejob":
		m.ServeSaveJob(w, r)
	default:
		m.serveNotFound(w, r)
	}
}

func (m *MasterHandler) ServeLoginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		m.serveTemplate(w, "login", nil)
	} else if m.Auth.CheckPass(r.FormValue("password")) {
		m.Auth.Auth(w, r)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/login?badpass=1", http.StatusSeeOther)
	}
}

func (m *MasterHandler) ServeJobsPage(w http.ResponseWriter, r *http.Request) {
	m.serveTemplate(w, "jobs", m.Scheduler)
}

func (m *MasterHandler) ServeSlavesPage(w http.ResponseWriter, r *http.Request) {
	m.serveTemplate(w, "slaves", m.Scheduler)
}

func (m *MasterHandler) ServeAddJobPage(w http.ResponseWriter, r *http.Request) {
	job := &jobadmin.Job{
		MaxInstances: 1,
	}
	m.serveTemplate(w, "jobEdit", job)
}

func (m *MasterHandler) ServeEditJobPage(w http.ResponseWriter, r *http.Request) {
	jobID := r.FormValue("id")
	jobs, err := m.Scheduler.Jobs()
	if err != nil {
		m.serveError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, j := range jobs {
		if j.ID == jobID {
			m.serveTemplate(w, "jobEdit", j)
			return
		}
	}
	m.serveError(w, "job ID not found: "+jobID, http.StatusBadRequest)
}

func (m *MasterHandler) ServeSaveJob(w http.ResponseWriter, r *http.Request) {
	jobData := []byte(r.FormValue("job"))
	var job jobadmin.Job
	if err := json.Unmarshal(jobData, &job); err != nil {
		m.serveError(w, "bad JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	var err error
	if job.ID == "" {
		err = m.addJob(&job)
	} else {
		err = m.modifyJob(&job)
	}
	if err != nil {
		m.serveError(w, err.Error(), http.StatusInternalServerError)
	} else {
		http.Redirect(w, r, "/jobs", http.StatusSeeOther)
	}
}

func (m *MasterHandler) serveAsset(w http.ResponseWriter, r *http.Request, cleanPath string) {
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

func (m *MasterHandler) serveTemplate(w http.ResponseWriter, name string, obj interface{}) {
	var buf bytes.Buffer
	if err := m.Templates.ExecuteTemplate(&buf, name, obj); err != nil {
		m.serveError(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
		io.Copy(w, &buf)
	}
}

func (m *MasterHandler) serveNotFound(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func (m *MasterHandler) serveError(w http.ResponseWriter, msg string, code int) {
	http.Error(w, msg, code)
}

func (m *MasterHandler) addJob(job *jobadmin.Job) error {
	m.JobsLock.Lock()
	defer m.JobsLock.Unlock()

	jobs, err := m.Scheduler.Jobs()
	if err != nil {
		return err
	}

	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return err
	}
	idString := hex.EncodeToString(idBytes)

	job.ID = idString
	jobs = append([]*jobadmin.Job{job}, jobs...)
	if err := m.Scheduler.SetJobs(jobs); err != nil {
		return err
	}

	return m.saveJobs(jobs)
}

func (m *MasterHandler) modifyJob(job *jobadmin.Job) error {
	m.JobsLock.Lock()
	defer m.JobsLock.Unlock()

	jobs, err := m.Scheduler.Jobs()
	if err != nil {
		return err
	}

	newJobs := make([]*jobadmin.Job, len(jobs))
	copy(newJobs, jobs)

	var found bool
	for i, x := range newJobs {
		if x.ID == job.ID {
			newJobs[i] = job
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("job ID not found: %s", job.ID)
	}

	if err := m.Scheduler.SetJobs(newJobs); err != nil {
		return err
	}

	return m.saveJobs(newJobs)
}

func (m *MasterHandler) saveJobs(jobs []*jobadmin.Job) error {
	encoded, err := json.Marshal(jobs)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(m.JobsPath, encoded, 0755)
}