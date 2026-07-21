// Package sessionauth provides reusable session-cookie authentication and CSRF
// helpers for gojahttp planned routes.
package sessionauth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

const (
	SecureCookieName   = "__Host-app"
	InsecureCookieName = "goja_app_session"
	CSRFHeaderName     = "X-CSRF-Token"
)

var sessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{22,128}$`)

var (
	ErrMissingCookie = errors.New("missing session cookie")
	ErrInvalidCookie = errors.New("invalid session cookie")
	ErrExpired       = errors.New("session expired")
	ErrRevoked       = errors.New("session revoked")
	ErrMFARequired   = errors.New("fresh mfa required")
	ErrNoActor       = errors.New("session actor missing")
)

// Session is the reusable server-side session shape.
type Session struct {
	ID                string
	UserID            string
	OIDCSubject       string
	Email             string
	EmailVerified     bool
	TenantIDs         []string
	CSRFToken         string
	MFAAt             *time.Time
	CreatedAt         time.Time
	LastSeenAt        time.Time
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
	RevokedAt         *time.Time
	Claims            map[string]any
}

// Store persists server-side sessions.
type Store interface {
	Create(ctx context.Context, session Session) error
	Get(ctx context.Context, id string) (*Session, error)
	Touch(ctx context.Context, id string, now time.Time, idleExpiresAt time.Time) error
	Rotate(ctx context.Context, oldID string, next Session) error
	Revoke(ctx context.Context, id string) error
}

// ActorLoader projects an application session into the gojahttp actor shape.
type ActorLoader interface {
	ActorForSession(ctx context.Context, session *Session) (*gojahttp.Actor, error)
}

// ActorLoaderFunc adapts a function into ActorLoader.
type ActorLoaderFunc func(ctx context.Context, session *Session) (*gojahttp.Actor, error)

func (f ActorLoaderFunc) ActorForSession(ctx context.Context, session *Session) (*gojahttp.Actor, error) {
	return f(ctx, session)
}

// Config controls Manager behavior.
type Config struct {
	Store             Store
	ActorLoader       ActorLoader
	CookieName        string
	Path              string
	SameSite          http.SameSite
	IdleTimeout       time.Duration
	AbsoluteTimeout   time.Duration
	AllowInsecureHTTP bool
	Now               func() time.Time
}

// Manager implements gojahttp.Authenticator and gojahttp.CSRFProtector from a
// server-side session store.
type Manager struct {
	store             Store
	actorLoader       ActorLoader
	cookieName        string
	path              string
	sameSite          http.SameSite
	idleTimeout       time.Duration
	absoluteTimeout   time.Duration
	allowInsecureHTTP bool
	now               func() time.Time
}

// New creates a session manager with secure-by-default cookie settings. For
// localhost HTTP demos, set AllowInsecureHTTP.
func New(cfg Config) (*Manager, error) {
	if cfg.Store == nil {
		return nil, fmt.Errorf("sessionauth: store is required")
	}
	if cfg.ActorLoader == nil {
		cfg.ActorLoader = ActorLoaderFunc(DefaultActorForSession)
	}
	if cfg.CookieName == "" {
		if cfg.AllowInsecureHTTP {
			cfg.CookieName = InsecureCookieName
		} else {
			cfg.CookieName = SecureCookieName
		}
	}
	if cfg.Path == "" {
		cfg.Path = "/"
	}
	if cfg.SameSite == 0 {
		cfg.SameSite = http.SameSiteLaxMode
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 30 * time.Minute
	}
	if cfg.AbsoluteTimeout == 0 {
		cfg.AbsoluteTimeout = 12 * time.Hour
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Manager{store: cfg.Store, actorLoader: cfg.ActorLoader, cookieName: cfg.CookieName, path: cfg.Path, sameSite: cfg.SameSite, idleTimeout: cfg.IdleTimeout, absoluteTimeout: cfg.AbsoluteTimeout, allowInsecureHTTP: cfg.AllowInsecureHTTP, now: cfg.Now}, nil
}

// AuthOptions returns the gojahttp auth fields implemented by this manager.
func (m *Manager) AuthOptions() gojahttp.AuthOptions {
	return gojahttp.AuthOptions{Authenticator: m, CSRF: m}
}

// NewSession creates, stores, and returns a new server-side session.
func (m *Manager) NewSession(ctx context.Context, userID string, opts ...SessionOption) (*Session, error) {
	now := m.now()
	id, err := RandomToken()
	if err != nil {
		return nil, err
	}
	csrf, err := RandomToken()
	if err != nil {
		return nil, err
	}
	session := Session{ID: id, UserID: userID, CSRFToken: csrf, CreatedAt: now, LastSeenAt: now, IdleExpiresAt: now.Add(m.idleTimeout), AbsoluteExpiresAt: now.Add(m.absoluteTimeout)}
	for _, opt := range opts {
		opt(&session)
	}
	if err := m.store.Create(ctx, session); err != nil {
		return nil, err
	}
	return &session, nil
}

// SetCookie writes the manager's session cookie.
func (m *Manager) SetCookie(w http.ResponseWriter, sessionID string) {
	m.setCookie(w, sessionID, int(m.absoluteTimeout.Seconds()))
}

// ClearCookie clears the manager's session cookie.
func (m *Manager) ClearCookie(w http.ResponseWriter) { m.setCookie(w, "", -1) }

// RevokeRequestSession revokes the session referenced by the request cookie.
func (m *Manager) RevokeRequestSession(ctx context.Context, r *http.Request) error {
	id, err := m.sessionIDFromRequest(r)
	if err != nil {
		return authError(err)
	}
	return m.store.Revoke(ctx, id)
}

// SessionFromRequest loads and validates a session from the request cookie.
func (m *Manager) SessionFromRequest(ctx context.Context, r *http.Request) (*Session, error) {
	id, err := m.sessionIDFromRequest(r)
	if err != nil {
		return nil, err
	}
	session, err := m.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrInvalidCookie
	}
	if err := validateSession(session, m.now()); err != nil {
		return nil, err
	}
	return session, nil
}

// Authenticate implements gojahttp.Authenticator.
func (m *Manager) Authenticate(ctx context.Context, req *http.Request, _ *gojahttp.SessionDTO, spec gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
	session, err := m.SessionFromRequest(ctx, req)
	if err != nil {
		return nil, authError(err)
	}
	if err := validateMFAFreshness(session, spec, m.now()); err != nil {
		return nil, authError(err)
	}
	actor, err := m.actorLoader.ActorForSession(ctx, session)
	if err != nil {
		return nil, err
	}
	if actor == nil {
		return nil, ErrNoActor
	}
	now := m.now()
	if err := m.store.Touch(ctx, session.ID, now, now.Add(m.idleTimeout)); err != nil {
		return nil, err
	}
	return actor, nil
}

// VerifyCSRF implements gojahttp.CSRFProtector.
func (m *Manager) VerifyCSRF(ctx context.Context, req gojahttp.CSRFRequest) error {
	session, err := m.SessionFromRequest(ctx, req.HTTPRequest)
	if err != nil {
		return authError(err)
	}
	if req.Actor != nil && req.Actor.ID != session.UserID {
		return errors.New("session actor mismatch")
	}
	headerToken := strings.TrimSpace(req.HTTPRequest.Header.Get(CSRFHeaderName))
	storedToken := strings.TrimSpace(session.CSRFToken)
	if headerToken == "" || storedToken == "" || !constantTimeEqual(headerToken, storedToken) {
		return errors.New("missing or invalid X-CSRF-Token")
	}
	return nil
}

func (m *Manager) sessionIDFromRequest(r *http.Request) (string, error) {
	cookie, err := r.Cookie(m.cookieName)
	if err != nil || cookie.Value == "" {
		return "", ErrMissingCookie
	}
	if !sessionIDPattern.MatchString(cookie.Value) {
		return "", ErrInvalidCookie
	}
	return cookie.Value, nil
}

func (m *Manager) setCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{ // #nosec G104 G124 -- cookie write cannot report an error; Secure is enabled unless AllowInsecureHTTP is explicitly configured for localhost demos.
		Name:     m.cookieName,
		Value:    value,
		Path:     m.path,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   !m.allowInsecureHTTP,
		SameSite: m.sameSite,
	})
}

func validateSession(session *Session, now time.Time) error {
	if session.RevokedAt != nil {
		return ErrRevoked
	}
	if !session.IdleExpiresAt.IsZero() && now.After(session.IdleExpiresAt) {
		return ErrExpired
	}
	if !session.AbsoluteExpiresAt.IsZero() && now.After(session.AbsoluteExpiresAt) {
		return ErrExpired
	}
	return nil
}

func validateMFAFreshness(session *Session, spec gojahttp.SecuritySpec, now time.Time) error {
	if spec.MFAFreshWithin <= 0 {
		return nil
	}
	if session.MFAAt == nil {
		return ErrMFARequired
	}
	if now.Sub(*session.MFAAt) > spec.MFAFreshWithin {
		return ErrMFARequired
	}
	return nil
}

func authError(err error) error {
	switch {
	case errors.Is(err, ErrMissingCookie), errors.Is(err, ErrInvalidCookie), errors.Is(err, ErrExpired), errors.Is(err, ErrRevoked), errors.Is(err, ErrMFARequired):
		return gojahttp.ErrUnauthenticated
	default:
		return err
	}
}

// DefaultActorForSession projects the session into a basic user actor.
func DefaultActorForSession(_ context.Context, session *Session) (*gojahttp.Actor, error) {
	if session == nil || session.UserID == "" {
		return nil, ErrNoActor
	}
	claims := map[string]any{}
	for key, value := range session.Claims {
		claims[key] = value
	}
	if session.Email != "" {
		claims["email"] = session.Email
	}
	if session.EmailVerified {
		claims["emailVerified"] = true
	}
	return &gojahttp.Actor{ID: session.UserID, Kind: "user", TenantIDs: append([]string(nil), session.TenantIDs...), Claims: claims}, nil
}

// SessionOption mutates a new session before it is stored.
type SessionOption func(*Session)

func WithEmail(email string, verified bool) SessionOption {
	return func(session *Session) { session.Email, session.EmailVerified = email, verified }
}

func WithTenantIDs(tenantIDs ...string) SessionOption {
	return func(session *Session) { session.TenantIDs = append([]string(nil), tenantIDs...) }
}

func WithClaims(claims map[string]any) SessionOption {
	return func(session *Session) {
		session.Claims = map[string]any{}
		for key, value := range claims {
			session.Claims[key] = value
		}
	}
}

func WithMFAAt(mfaAt time.Time) SessionOption {
	return func(session *Session) { session.MFAAt = &mfaAt }
}

// RandomToken returns an unguessable URL-safe token.
func RandomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// MemoryStore is an in-memory Store for tests and local development.
type MemoryStore struct {
	mu       sync.Mutex
	sessions map[string]Session
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{sessions: map[string]Session{}} }

func (s *MemoryStore) Create(_ context.Context, session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if session.ID == "" {
		return fmt.Errorf("session id is required")
	}
	s.sessions[session.ID] = cloneSession(session)
	return nil
}

func (s *MemoryStore) Get(_ context.Context, id string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil, ErrInvalidCookie
	}
	clone := cloneSession(session)
	return &clone, nil
}

func (s *MemoryStore) Touch(_ context.Context, id string, now time.Time, idleExpiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return ErrInvalidCookie
	}
	session.LastSeenAt = now
	session.IdleExpiresAt = idleExpiresAt
	s.sessions[id] = session
	return nil
}

func (s *MemoryStore) Rotate(_ context.Context, oldID string, next Session) error {
	if next.ID == "" {
		return fmt.Errorf("next session id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[oldID]; !ok {
		return ErrInvalidCookie
	}
	delete(s.sessions, oldID)
	s.sessions[next.ID] = cloneSession(next)
	return nil
}

func (s *MemoryStore) Revoke(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil
	}
	now := time.Now()
	session.RevokedAt = &now
	s.sessions[id] = session
	return nil
}

func cloneSession(session Session) Session {
	out := session
	out.TenantIDs = append([]string(nil), session.TenantIDs...)
	if session.MFAAt != nil {
		mfaAt := *session.MFAAt
		out.MFAAt = &mfaAt
	}
	if session.RevokedAt != nil {
		revokedAt := *session.RevokedAt
		out.RevokedAt = &revokedAt
	}
	if session.Claims != nil {
		out.Claims = map[string]any{}
		for key, value := range session.Claims {
			out.Claims[key] = value
		}
	}
	return out
}
