package gojahttp

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

const defaultSessionCookieName = "go_go_goja_session"

var sessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{22,128}$`)

// SessionOptions configures the lightweight cookie-backed session identity used
// by go-go-goja. The session stores only an opaque ID in a cookie; application
// state remains in the application's database.
type SessionOptions struct {
	Disabled   bool
	CookieName string
	Path       string
	MaxAge     time.Duration
	Secure     bool
	SameSite   http.SameSite
}

type SessionDTO struct {
	ID         string
	IsNew      bool
	CookieName string
}

func (s *SessionDTO) Map() map[string]any {
	if s == nil {
		return nil
	}
	return map[string]any{
		"id":         s.ID,
		"isNew":      s.IsNew,
		"cookieName": s.CookieName,
	}
}

type SessionManager struct{ opts SessionOptions }

func NewSessionManager(opts SessionOptions) *SessionManager {
	if opts.CookieName == "" {
		opts.CookieName = defaultSessionCookieName
	}
	if opts.Path == "" {
		opts.Path = "/"
	}
	if opts.MaxAge == 0 {
		opts.MaxAge = 365 * 24 * time.Hour
	}
	if opts.SameSite == 0 {
		opts.SameSite = http.SameSiteLaxMode
	}
	return &SessionManager{opts: opts}
}

func (m *SessionManager) Session(w http.ResponseWriter, r *http.Request) (*SessionDTO, error) {
	if m == nil || m.opts.Disabled {
		return nil, nil
	}
	if cookie, err := r.Cookie(m.opts.CookieName); err == nil && validSessionID(cookie.Value) {
		return &SessionDTO{ID: cookie.Value, CookieName: m.opts.CookieName}, nil
	}
	id, err := newSessionID()
	if err != nil {
		return nil, err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     m.opts.CookieName,
		Value:    id,
		Path:     m.opts.Path,
		MaxAge:   int(m.opts.MaxAge.Seconds()),
		HttpOnly: true,
		Secure:   m.opts.Secure,
		SameSite: m.opts.SameSite,
	})
	return &SessionDTO{ID: id, IsNew: true, CookieName: m.opts.CookieName}, nil
}

func newSessionID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func validSessionID(id string) bool {
	return sessionIDPattern.MatchString(id)
}
