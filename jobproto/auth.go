package jobproto

import (
	"bytes"
	"crypto/sha512"
	"errors"
	"io"
	"math/rand"
	"net"
	"time"
)

const (
	authTimeout       = time.Second * 30
	authChallengeSize = 32
)

var (
	ErrBadAuth = errors.New("bad authentication credentials")
)

// NewMasterConnAuth creates an authenticated Master.
// It returns ErrBadAuth if the other end does not know the
// correct password.
// If the handshake fails for any reason, c is closed.
func NewMasterConnAuth(c net.Conn, password string) (Master, error) {
	if err := sendChallenge(0, c, password); err != nil {
		return nil, err
	}
	if err := handleChallenge(1, c, password); err != nil {
		return nil, err
	}
	return NewMasterConn(c)
}

// NewSlaveConnAuth creates an authenticated Slave.
// It returns ErrBadAuth if the other end has a different
// password than we do.
// If the handshake fails for any reason, c is closed.
func NewSlaveConnAuth(c net.Conn, password string) (Slave, error) {
	if err := handleChallenge(0, c, password); err != nil {
		return nil, err
	}
	if err := sendChallenge(1, c, password); err != nil {
		return nil, err
	}
	return NewSlaveConn(c)
}

func sendChallenge(seq int, c net.Conn, password string) error {
	c.SetDeadline(time.Now().Add(authTimeout))
	challenge := make([]byte, authChallengeSize)
	if _, err := rand.Read(challenge); err != nil {
		return err
	}
	if _, err := c.Write(challenge); err != nil {
		c.Close()
		return err
	}
	expected := challengeResponse(seq, challenge, password)
	actual := make([]byte, sha512.Size)
	if _, err := io.ReadFull(c, actual); err != nil {
		c.Close()
		return err
	}

	if bytes.Equal(actual, expected) {
		c.SetDeadline(time.Time{})
		if _, err := c.Write([]byte{1}); err != nil {
			return err
		}
		return nil
	}

	c.Write([]byte{0})
	c.Close()
	return ErrBadAuth
}

func handleChallenge(seq int, c net.Conn, password string) error {
	c.SetDeadline(time.Now().Add(authTimeout))
	challenge := make([]byte, authChallengeSize)
	if _, err := io.ReadFull(c, challenge); err != nil {
		return err
	}
	response := challengeResponse(seq, challenge, password)
	if _, err := c.Write(response); err != nil {
		return err
	}

	status := make([]byte, 1)
	if _, err := io.ReadFull(c, status); err != nil {
		return err
	}
	c.SetDeadline(time.Time{})

	if status[0] == 1 {
		return nil
	} else {
		c.Close()
		return ErrBadAuth
	}
}

func challengeResponse(seq int, challenge []byte, password string) []byte {
	// Note that seq must be separated from the challenge
	// (in this case by the password).
	// Otherwise, a malicious server could potentially send
	// a challenge from one of its connectors to a
	// different connector, but with the seq tacked on.
	preamble := append([]byte{byte(seq)}, []byte(password)...)
	payload := append(preamble, challenge...)
	res := sha512.Sum512(payload)
	return res[:]
}
