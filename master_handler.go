package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"io/ioutil"
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
			http.Redirect(w, r, "/jobs", http.StatusTemporaryRedirect)
		} else {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		}
		return
	case "/login":
		m.ServeLoginPage(w, r)
		return
	}

	if !m.Auth.IsAuth(r) {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	switch cleanPath {
	case "/jobs":
		m.ServeJobsPage(w, r)
	case "/slaves":
		m.ServeSlavesPage(w, r)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
		io.Copy(w, &buf)
	}
}

func (m *MasterHandler) serveNotFound(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func (m *MasterHandler) saveJobs(jobs []*jobadmin.Job) error {
	encoded, err := json.Marshal(jobs)
	if err != nil {
		return err
	}
	m.JobsLock.Lock()
	defer m.JobsLock.Unlock()
	return ioutil.WriteFile(m.JobsPath, encoded, 0755)
}
