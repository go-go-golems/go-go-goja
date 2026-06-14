// Package devauth provides a small in-memory authentication kit for examples,
// smoke tests, and local development. It is intentionally not a production
// identity system; it implements the gojahttp auth interfaces so demos exercise
// the same planned-route enforcement path as real hosts.
package devauth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

const (
	DefaultCookieName = "go_go_goja_dev_auth"
	DefaultUsername   = "demo@example.test"
	DefaultPassword   = "demo-password"
	DefaultUserID     = "u1"
	DefaultTenantID   = "o1"
	DefaultProjectID  = "p1"
)

// Config controls the in-memory development auth kit.
type Config struct {
	CookieName   string
	SecureCookie bool
	CookieMaxAge time.Duration
	Seed         Seed
	Logger       *stdlog.Logger
}

// Seed describes the users and resources available to the dev auth kit.
type Seed struct {
	Users    []User
	Projects []Project
}

// User is a simple in-memory development user.
type User struct {
	ID            string
	Username      string
	Password      string
	Email         string
	EmailVerified bool
	TenantIDs     []string
	RolesByTenant map[string][]string
}

// Project is a simple in-memory development resource.
type Project struct {
	ID       string
	TenantID string
	Name     string
}

// Session is a simple in-memory development session.
type Session struct {
	ID        string
	UserID    string
	CSRFToken string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Kit implements gojahttp auth interfaces and exposes development login/logout
// handlers.
type Kit struct {
	mu           sync.Mutex
	cookieName   string
	secureCookie bool
	cookieMaxAge time.Duration
	logger       *stdlog.Logger

	usersByUsername map[string]User
	usersByID       map[string]User
	projects        map[string]Project
	sessions        map[string]Session
	audits          []gojahttp.AuditEvent
}

// DefaultSeed returns a tiny but complete demo world.
func DefaultSeed() Seed {
	return Seed{
		Users: []User{{
			ID:            DefaultUserID,
			Username:      DefaultUsername,
			Password:      DefaultPassword,
			Email:         DefaultUsername,
			EmailVerified: true,
			TenantIDs:     []string{DefaultTenantID},
			RolesByTenant: map[string][]string{DefaultTenantID: {"editor"}},
		}},
		Projects: []Project{{ID: DefaultProjectID, TenantID: DefaultTenantID, Name: "Demo Project"}},
	}
}

// New creates an in-memory development auth kit.
func New(cfg Config) *Kit {
	if cfg.CookieName == "" {
		cfg.CookieName = DefaultCookieName
	}
	if cfg.CookieMaxAge == 0 {
		cfg.CookieMaxAge = 8 * time.Hour
	}
	if len(cfg.Seed.Users) == 0 && len(cfg.Seed.Projects) == 0 {
		cfg.Seed = DefaultSeed()
	}
	if cfg.Logger == nil {
		cfg.Logger = stdlog.New(os.Stderr, "devauth: ", stdlog.LstdFlags)
	}
	k := &Kit{
		cookieName:      cfg.CookieName,
		secureCookie:    cfg.SecureCookie,
		cookieMaxAge:    cfg.CookieMaxAge,
		logger:          cfg.Logger,
		usersByUsername: map[string]User{},
		usersByID:       map[string]User{},
		projects:        map[string]Project{},
		sessions:        map[string]Session{},
	}
	for _, user := range cfg.Seed.Users {
		if user.Email == "" {
			user.Email = user.Username
		}
		k.usersByUsername[strings.ToLower(user.Username)] = user
		k.usersByID[user.ID] = user
	}
	for _, project := range cfg.Seed.Projects {
		k.projects[project.ID] = project
	}
	return k
}

// AuthOptions returns the gojahttp interface implementations for this kit.
func (k *Kit) AuthOptions() gojahttp.AuthOptions {
	return gojahttp.AuthOptions{Authenticator: k, Resources: k, Authorizer: k, CSRF: k, Audit: k}
}

// LoginHandler returns a handler for POST /auth/dev/login.
func (k *Kit) LoginHandler() http.Handler { return http.HandlerFunc(k.handleLogin) }

// LogoutHandler returns a handler for POST /auth/dev/logout.
func (k *Kit) LogoutHandler() http.Handler { return http.HandlerFunc(k.handleLogout) }

// SessionHandler returns a handler for GET /auth/dev/session.
func (k *Kit) SessionHandler() http.Handler { return http.HandlerFunc(k.handleSession) }

// AuditCount returns the number of audit events captured by the kit.
func (k *Kit) AuditCount() int {
	k.mu.Lock()
	defer k.mu.Unlock()
	return len(k.audits)
}

// AuditEvents returns a snapshot of captured audit events.
func (k *Kit) AuditEvents() []gojahttp.AuditEvent {
	k.mu.Lock()
	defer k.mu.Unlock()
	out := make([]gojahttp.AuditEvent, len(k.audits))
	copy(out, k.audits)
	return out
}

func (k *Kit) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid login request", http.StatusBadRequest)
		return
	}
	user, ok := k.findUserByCredentials(input.Username, input.Password)
	if !ok {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	session, err := k.createSession(user.ID, time.Now())
	if err != nil {
		http.Error(w, "create session", http.StatusInternalServerError)
		return
	}
	k.setSessionCookie(w, session.ID, int(k.cookieMaxAge.Seconds()))
	writeJSON(w, http.StatusOK, map[string]any{"user": userPayload(user), "csrfToken": session.CSRFToken})
}

func (k *Kit) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, ok := k.sessionFromRequest(r)
	if !ok {
		k.clearSessionCookie(w)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !constantTimeEqual(r.Header.Get("X-CSRF-Token"), session.CSRFToken) {
		http.Error(w, "missing or invalid X-CSRF-Token", http.StatusForbidden)
		return
	}
	k.mu.Lock()
	delete(k.sessions, session.ID)
	k.mu.Unlock()
	k.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (k *Kit) handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, ok := k.sessionFromRequest(r)
	if !ok {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	user, ok := k.userByID(session.UserID)
	if !ok {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": userPayload(user), "csrfToken": session.CSRFToken})
}

// Authenticate implements gojahttp.Authenticator.
func (k *Kit) Authenticate(_ context.Context, req *http.Request, _ *gojahttp.SessionDTO, _ gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
	session, ok := k.sessionFromRequest(req)
	if !ok {
		return nil, gojahttp.ErrUnauthenticated
	}
	user, ok := k.userByID(session.UserID)
	if !ok {
		return nil, gojahttp.ErrUnauthenticated
	}
	return actorForUser(user), nil
}

// VerifyCSRF implements gojahttp.CSRFProtector.
func (k *Kit) VerifyCSRF(_ context.Context, req gojahttp.CSRFRequest) error {
	session, ok := k.sessionFromRequest(req.HTTPRequest)
	if !ok {
		return gojahttp.ErrUnauthenticated
	}
	if req.Actor != nil && req.Actor.ID != session.UserID {
		return errors.New("session actor mismatch")
	}
	if !constantTimeEqual(req.HTTPRequest.Header.Get("X-CSRF-Token"), session.CSRFToken) {
		return errors.New("missing or invalid X-CSRF-Token")
	}
	return nil
}

// ResolveResource implements gojahttp.ResourceResolver.
func (k *Kit) ResolveResource(_ context.Context, req gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error) {
	switch req.Spec.Type {
	case "project":
		project, ok := k.projectByID(req.ID)
		if !ok {
			return nil, gojahttp.ErrNotFound
		}
		if req.TenantID != "" && project.TenantID != req.TenantID {
			return nil, gojahttp.ErrNotFound
		}
		return &gojahttp.ResourceRef{Name: req.Spec.Name, Type: req.Spec.Type, ID: project.ID, TenantID: project.TenantID, Claims: map[string]any{"name": project.Name}}, nil
	default:
		return nil, gojahttp.ErrNotFound
	}
}

// Authorize implements gojahttp.Authorizer.
func (k *Kit) Authorize(_ context.Context, req gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
	if req.Actor == nil {
		return deny("missing actor"), nil
	}
	switch req.Action {
	case "user.self.read":
		return allow(), nil
	case "project.read":
		if req.Resource == nil {
			return deny("missing project"), nil
		}
		if actorHasTenant(req.Actor, req.Resource.TenantID) {
			return allow(), nil
		}
		return deny("tenant access denied"), nil
	case "project.update":
		if req.Resource == nil {
			return deny("missing project"), nil
		}
		if k.actorHasAnyTenantRole(req.Actor.ID, req.Resource.TenantID, "admin", "editor") {
			return allow(), nil
		}
		return deny("project update denied"), nil
	default:
		return deny("unknown action"), nil
	}
}

// RecordAudit implements gojahttp.AuditSink.
func (k *Kit) RecordAudit(_ context.Context, event gojahttp.AuditEvent) error {
	k.mu.Lock()
	k.audits = append(k.audits, event)
	k.mu.Unlock()
	if k.logger != nil {
		k.logger.Printf("audit event=%s outcome=%s actor=%s action=%s status=%d reason=%s", event.Event, event.Outcome, actorID(event.Actor), event.Action, event.StatusCode, event.Reason)
	}
	return nil
}

func (k *Kit) findUserByCredentials(username, password string) (User, bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	user, ok := k.usersByUsername[strings.ToLower(strings.TrimSpace(username))]
	if !ok || !constantTimeEqual(password, user.Password) {
		return User{}, false
	}
	return user, true
}

func (k *Kit) createSession(userID string, now time.Time) (Session, error) {
	id, err := randomToken()
	if err != nil {
		return Session{}, err
	}
	csrf, err := randomToken()
	if err != nil {
		return Session{}, err
	}
	session := Session{ID: id, UserID: userID, CSRFToken: csrf, CreatedAt: now, ExpiresAt: now.Add(k.cookieMaxAge)}
	k.mu.Lock()
	k.sessions[id] = session
	k.mu.Unlock()
	return session, nil
}

func (k *Kit) sessionFromRequest(r *http.Request) (Session, bool) {
	cookie, err := r.Cookie(k.cookieName)
	if err != nil || cookie.Value == "" {
		return Session{}, false
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	session, ok := k.sessions[cookie.Value]
	if !ok || time.Now().After(session.ExpiresAt) {
		return Session{}, false
	}
	return session, true
}

func (k *Kit) userByID(id string) (User, bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	user, ok := k.usersByID[id]
	return user, ok
}

func (k *Kit) projectByID(id string) (Project, bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	project, ok := k.projects[id]
	return project, ok
}

func (k *Kit) actorHasAnyTenantRole(userID, tenantID string, roles ...string) bool {
	user, ok := k.userByID(userID)
	if !ok {
		return false
	}
	userRoles := user.RolesByTenant[tenantID]
	for _, role := range roles {
		for _, userRole := range userRoles {
			if userRole == role {
				return true
			}
		}
	}
	return false
}

func (k *Kit) setSessionCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{ // #nosec G104 G124 -- development-only cookie helper; Secure is configurable for localhost demos.
		Name:     k.cookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   k.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (k *Kit) clearSessionCookie(w http.ResponseWriter) {
	k.setSessionCookie(w, "", -1)
}

func actorForUser(user User) *gojahttp.Actor {
	return &gojahttp.Actor{ID: user.ID, Kind: "user", TenantIDs: append([]string(nil), user.TenantIDs...), Claims: map[string]any{"email": user.Email, "emailVerified": user.EmailVerified}}
}

func userPayload(user User) map[string]any {
	return map[string]any{"id": user.ID, "username": user.Username, "email": user.Email, "emailVerified": user.EmailVerified, "tenantIds": user.TenantIDs}
}

func actorHasTenant(actor *gojahttp.Actor, tenantID string) bool {
	for _, actorTenantID := range actor.TenantIDs {
		if actorTenantID == tenantID {
			return true
		}
	}
	return false
}

func allow() gojahttp.AuthorizationDecision { return gojahttp.AuthorizationDecision{Allowed: true} }

func deny(reason string) gojahttp.AuthorizationDecision {
	return gojahttp.AuthorizationDecision{Allowed: false, Reason: reason}
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func actorID(actor *gojahttp.Actor) string {
	if actor == nil {
		return ""
	}
	return actor.ID
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		// At this point the status and headers are already sent. Logging is the only
		// useful action for this development helper.
		stdlog.Printf("devauth: write json: %v", err)
	}
}
