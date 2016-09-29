package main

import "net/http"

func (m *Master) ServeLoginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		m.serveTemplate(w, "login", nil)
	} else if m.CheckPass(r.FormValue("password")) {
		m.Auth(w, r)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/login?badpass=1", http.StatusSeeOther)
	}
}

func (m *Master) ServeJobsPage(w http.ResponseWriter, r *http.Request) {
	m.serveTemplate(w, "jobs", m.Scheduler)
}

func (m *Master) serveTemplate(w http.ResponseWriter, name string, obj interface{}) {
	w.Header().Set("Content-Type", "text/html")
	if err := m.Templates.ExecuteTemplate(w, name, obj); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
