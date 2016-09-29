package main

import (
	"net/http"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

// A MasterAuth manages admin authentication.
type MasterAuth struct {
	adminPass string
	cookies   *sessions.CookieStore
}

// NewMasterAuth creates a MasterAuth with an admin
// password.
func NewMasterAuth(pass string) *MasterAuth {
	return &MasterAuth{
		adminPass: pass,
		cookies: sessions.NewCookieStore(securecookie.GenerateRandomKey(16),
			securecookie.GenerateRandomKey(16)),
	}
}

// IsAuth returns whether or not the request is from an
// authenticated source.
func (m *MasterAuth) IsAuth(r *http.Request) bool {
	s, _ := m.cookies.Get(r, "sessid")
	val, _ := s.Values["authenticated"].(bool)
	return val
}

// CheckPass returns whether the admin password is right.
func (m *MasterAuth) CheckPass(password string) bool {
	return m.adminPass == password
}

// Auth authenticates the remote HTTP client.
func (m *MasterAuth) Auth(w http.ResponseWriter, r *http.Request) {
	s, _ := m.cookies.Get(r, "sessid")
	s.Values["authenticated"] = true
	s.Save(r, w)
}
