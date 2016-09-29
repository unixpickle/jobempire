package main

import "net/http"

// IsAuth returns whether or not the request is from an
// authenticated source.
func (m *Master) IsAuth(r *http.Request) bool {
	s, _ := m.Cookies.Get(r, "sessid")
	val, _ := s.Values["authenticated"].(bool)
	return val
}

// CheckPass returns whether the admin password is right.
func (m *Master) CheckPass(password string) bool {
	return m.AdminPass == password
}

// Auth authenticates the remote HTTP client.
func (m *Master) Auth(w http.ResponseWriter, r *http.Request) {
	s, _ := m.Cookies.Get(r, "sessid")
	s.Values["authenticated"] = true
	s.Save(r, w)
}
