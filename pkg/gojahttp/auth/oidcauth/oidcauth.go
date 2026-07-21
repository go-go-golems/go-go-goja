// Package oidcauth provides an opinionated OIDC browser-login adapter for
// gojahttp hosts. It keeps identity-provider tokens server-side and creates an
// opaque application session for planned route authentication.
package oidcauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
	"golang.org/x/oauth2"
)

// Config controls the OIDC login/callback/logout handlers.
type Config struct {
	IssuerURL      string
	ClientID       string
	ClientSecret   string
	RedirectURL    string
	Scopes         []string
	AfterLoginURL  string
	AfterLogoutURL string

	SessionManager   *sessionauth.Manager
	UserNormalizer   UserNormalizer
	TransactionStore TransactionStore
	Audit            gojahttp.AuditSink
	SecurityEvents   gojahttp.SecurityEventObserver
	// HTTPClient is used for OIDC discovery, token exchange, and remote key
	// retrieval. When nil, the standard context HTTP client is used.
	HTTPClient *http.Client
}

// OIDCClaims is the normalized identity material extracted from the verified ID
// token. Subject is the stable identity key; email is not treated as stable.
type OIDCClaims struct {
	Issuer            string
	Subject           string
	Email             string
	EmailVerified     bool
	Name              string
	PreferredUsername string
	Groups            []string
	Raw               map[string]any
}

// UserSession is the application session projection returned by a normalizer.
type UserSession struct {
	UserID        string
	Email         string
	EmailVerified bool
	TenantIDs     []string
	Claims        map[string]any
}

// UserNormalizer maps a verified OIDC subject into an application user/session.
type UserNormalizer interface {
	NormalizeOIDCUser(ctx context.Context, claims OIDCClaims) (UserSession, error)
}

// UserNormalizerFunc adapts a function into UserNormalizer.
type UserNormalizerFunc func(ctx context.Context, claims OIDCClaims) (UserSession, error)

func (f UserNormalizerFunc) NormalizeOIDCUser(ctx context.Context, claims OIDCClaims) (UserSession, error) {
	return f(ctx, claims)
}

// Transaction stores the state needed to verify an OIDC callback.
type Transaction struct {
	State        string
	Nonce        string
	PKCEVerifier string
	CreatedAt    time.Time
	RedirectURL  string
}

// ErrTransactionUnavailable is returned when a login transaction cannot be
// consumed. Callers deliberately receive the same public outcome for a missing,
// expired, or previously consumed state so callback behavior does not disclose
// which security check failed.
var ErrTransactionUnavailable = errors.New("oidc login transaction unavailable")

// TransactionStore persists short-lived login transactions keyed by state.
type TransactionStore interface {
	Put(ctx context.Context, tx Transaction) error
	Take(ctx context.Context, state string) (Transaction, error)
}

// TransactionCleanup removes expired persisted login transactions. It is kept
// separate from TransactionStore so lightweight stores need not expose a
// maintenance operation to request handlers.
type TransactionCleanup interface {
	Cleanup(ctx context.Context) (int64, error)
}

// Handlers owns OIDC login/callback/logout HTTP handlers.
type Handlers struct {
	oauth2Config   oauth2.Config
	verifier       *oidc.IDTokenVerifier
	sessionManager *sessionauth.Manager
	normalizer     UserNormalizer
	transactions   TransactionStore
	afterLoginURL  string
	afterLogoutURL string
	audit          gojahttp.AuditSink
	securityEvents gojahttp.SecurityEventObserver
	httpClient     *http.Client
}

// New discovers the OIDC provider and returns login/callback/logout handlers.
func New(ctx context.Context, cfg Config) (*Handlers, error) {
	if cfg.IssuerURL == "" {
		return nil, fmt.Errorf("oidcauth: issuer URL is required")
	}
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("oidcauth: client ID is required")
	}
	if cfg.RedirectURL == "" {
		return nil, fmt.Errorf("oidcauth: redirect URL is required")
	}
	if cfg.SessionManager == nil {
		return nil, fmt.Errorf("oidcauth: session manager is required")
	}
	if cfg.UserNormalizer == nil {
		return nil, fmt.Errorf("oidcauth: user normalizer is required")
	}
	if cfg.TransactionStore == nil {
		cfg.TransactionStore = NewMemoryTransactionStore(10 * time.Minute)
	}
	discoveryCtx := withHTTPClient(ctx, cfg.HTTPClient)
	provider, err := oidc.NewProvider(discoveryCtx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("oidcauth: discover provider: %w", err)
	}
	scopes := ensureOpenIDScope(cfg.Scopes)
	endpoint := provider.Endpoint()
	// A public PKCE client has no secret and must send client_id in the token
	// request body. Avoid auth-style probing because a strict provider may
	// consume the one-time code before oauth2 retries with another style.
	if cfg.ClientSecret == "" {
		endpoint.AuthStyle = oauth2.AuthStyleInParams
	}
	oauth2Config := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     endpoint,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
	}
	return &Handlers{
		oauth2Config:   oauth2Config,
		verifier:       provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		sessionManager: cfg.SessionManager,
		normalizer:     cfg.UserNormalizer,
		transactions:   cfg.TransactionStore,
		afterLoginURL:  defaultIfEmpty(cfg.AfterLoginURL, "/"),
		afterLogoutURL: defaultIfEmpty(cfg.AfterLogoutURL, "/"),
		audit:          cfg.Audit,
		securityEvents: cfg.SecurityEvents,
		httpClient:     cfg.HTTPClient,
	}, nil
}

func (h *Handlers) LoginHandler() http.Handler    { return http.HandlerFunc(h.handleLogin) }
func (h *Handlers) CallbackHandler() http.Handler { return http.HandlerFunc(h.handleCallback) }
func (h *Handlers) LogoutHandler() http.Handler   { return http.HandlerFunc(h.handleLogout) }

func (h *Handlers) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.observe(r.Context(), "oidc.login", "rejected", "method")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	state, err := sessionauth.RandomToken()
	if err != nil {
		h.observe(r.Context(), "oidc.login", "failed", "state_generation")
		http.Error(w, "create login state", http.StatusInternalServerError)
		return
	}
	nonce, err := sessionauth.RandomToken()
	if err != nil {
		h.observe(r.Context(), "oidc.login", "failed", "nonce_generation")
		http.Error(w, "create login nonce", http.StatusInternalServerError)
		return
	}
	verifier, err := sessionauth.RandomToken()
	if err != nil {
		h.observe(r.Context(), "oidc.login", "failed", "verifier_generation")
		http.Error(w, "create pkce verifier", http.StatusInternalServerError)
		return
	}
	redirectURL := strings.TrimSpace(r.URL.Query().Get("return_to"))
	if redirectURL == "" || !strings.HasPrefix(redirectURL, "/") || strings.HasPrefix(redirectURL, "//") {
		redirectURL = h.afterLoginURL
	}
	if err := h.transactions.Put(r.Context(), Transaction{State: state, Nonce: nonce, PKCEVerifier: verifier, CreatedAt: time.Now(), RedirectURL: redirectURL}); err != nil {
		h.observe(r.Context(), "oidc.login", "failed", "transaction_store")
		http.Error(w, "store login transaction", http.StatusInternalServerError)
		return
	}
	url := h.oauth2Config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), oauth2.SetAuthURLParam("nonce", nonce))
	h.observe(r.Context(), "oidc.login", "issued", "")
	http.Redirect(w, r, url, http.StatusFound)
}

func (h *Handlers) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.observe(r.Context(), "oidc.callback", "rejected", "method")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if errText := r.URL.Query().Get("error"); errText != "" {
		h.observe(r.Context(), "oidc.callback", "rejected", "provider_error")
		http.Error(w, "oidc error: "+errText, http.StatusUnauthorized)
		return
	}
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	if state == "" || code == "" {
		h.observe(r.Context(), "oidc.callback", "rejected", "missing_parameters")
		http.Error(w, "missing oidc callback state or code", http.StatusBadRequest)
		return
	}
	tx, err := h.transactions.Take(r.Context(), state)
	if err != nil {
		h.observe(r.Context(), "oidc.callback", "rejected", "transaction_unavailable")
		http.Error(w, "invalid oidc state", http.StatusUnauthorized)
		return
	}
	callbackCtx := withHTTPClient(r.Context(), h.httpClient)
	token, err := h.oauth2Config.Exchange(callbackCtx, code, oauth2.VerifierOption(tx.PKCEVerifier))
	if err != nil {
		h.observe(r.Context(), "oidc.callback", "rejected", "token_exchange")
		http.Error(w, "oidc token exchange failed", http.StatusUnauthorized)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		h.observe(r.Context(), "oidc.callback", "rejected", "missing_id_token")
		http.Error(w, "oidc response missing id_token", http.StatusUnauthorized)
		return
	}
	idToken, err := h.verifier.Verify(callbackCtx, rawIDToken)
	if err != nil {
		h.observe(r.Context(), "oidc.callback", "rejected", "id_token_verification")
		http.Error(w, "oidc id_token verification failed", http.StatusUnauthorized)
		return
	}
	if idToken.Nonce != tx.Nonce {
		h.observe(r.Context(), "oidc.callback", "rejected", "nonce_mismatch")
		http.Error(w, "oidc nonce mismatch", http.StatusUnauthorized)
		return
	}
	claims, err := claimsFromIDToken(idToken)
	if err != nil {
		h.observe(r.Context(), "oidc.callback", "rejected", "claims")
		http.Error(w, "oidc claims invalid", http.StatusUnauthorized)
		return
	}
	userSession, err := h.normalizer.NormalizeOIDCUser(r.Context(), claims)
	if err != nil {
		h.observe(r.Context(), "oidc.callback", "rejected", "user_normalization")
		http.Error(w, "user normalization failed", http.StatusUnauthorized)
		return
	}
	if userSession.UserID == "" {
		h.observe(r.Context(), "oidc.callback", "rejected", "empty_user")
		http.Error(w, "user normalization returned empty user id", http.StatusUnauthorized)
		return
	}
	session, err := h.sessionManager.NewSession(r.Context(), userSession.UserID,
		sessionauth.WithEmail(userSession.Email, userSession.EmailVerified),
		sessionauth.WithTenantIDs(userSession.TenantIDs...),
		sessionauth.WithClaims(userSession.Claims),
	)
	if err != nil {
		h.observe(r.Context(), "oidc.callback", "failed", "session_creation")
		http.Error(w, "create app session", http.StatusInternalServerError)
		return
	}
	h.sessionManager.SetCookie(w, session.ID)
	h.observe(r.Context(), "oidc.callback", "accepted", "")
	http.Redirect(w, r, tx.RedirectURL, http.StatusFound)
}

func (h *Handlers) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.observe(r.Context(), "oidc.logout", "rejected", "method")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := h.sessionManager.VerifyCSRF(r.Context(), gojahttp.CSRFRequest{HTTPRequest: r}); err != nil {
		h.observe(r.Context(), "oidc.logout", "rejected", "csrf")
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := h.sessionManager.RevokeRequestSession(r.Context(), r); err != nil {
		h.observe(r.Context(), "oidc.logout", "failed", "session_revoke")
		http.Error(w, "revoke application session", http.StatusInternalServerError)
		return
	}
	h.observe(r.Context(), "oidc.logout", "accepted", "")
	h.sessionManager.ClearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func withHTTPClient(ctx context.Context, client *http.Client) context.Context {
	if client == nil {
		return ctx
	}
	return oidc.ClientContext(ctx, client)
}

func (h *Handlers) observe(ctx context.Context, event, outcome, reason string) {
	if h.securityEvents != nil {
		h.securityEvents.ObserveSecurityEvent(ctx, gojahttp.SecurityEvent{Name: event, Outcome: outcome, Reason: reason})
	}
	if h.audit != nil {
		_ = h.audit.RecordAudit(ctx, gojahttp.AuditEvent{Event: event, Outcome: outcome, Reason: reason, Method: "INTERNAL", Pattern: "oidc"})
	}
}

func claimsFromIDToken(idToken *oidc.IDToken) (OIDCClaims, error) {
	var raw map[string]any
	if err := idToken.Claims(&raw); err != nil {
		return OIDCClaims{}, err
	}
	claims := OIDCClaims{Issuer: idToken.Issuer, Subject: idToken.Subject, Raw: raw}
	if email, _ := raw["email"].(string); email != "" {
		claims.Email = email
	}
	if verified, _ := raw["email_verified"].(bool); verified {
		claims.EmailVerified = true
	}
	claims.Name, _ = raw["name"].(string)
	claims.PreferredUsername, _ = raw["preferred_username"].(string)
	claims.Groups = stringSliceClaim(raw["groups"])
	if claims.Subject == "" {
		return OIDCClaims{}, fmt.Errorf("subject is required")
	}
	return claims, nil
}

func stringSliceClaim(value any) []string {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if s, ok := value.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func ensureOpenIDScope(scopes []string) []string {
	out := append([]string(nil), scopes...)
	for _, scope := range out {
		if scope == oidc.ScopeOpenID {
			return out
		}
	}
	return append([]string{oidc.ScopeOpenID}, out...)
}

func defaultIfEmpty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// MemoryTransactionStore is a single-process TransactionStore for tests and
// simple deployments. Production multi-instance hosts should use shared storage.
type MemoryTransactionStore struct {
	mu   sync.Mutex
	ttl  time.Duration
	data map[string]Transaction
}

func NewMemoryTransactionStore(ttl time.Duration) *MemoryTransactionStore {
	if ttl == 0 {
		ttl = 10 * time.Minute
	}
	return &MemoryTransactionStore{ttl: ttl, data: map[string]Transaction{}}
}

func (s *MemoryTransactionStore) Put(_ context.Context, tx Transaction) error {
	if tx.State == "" || tx.Nonce == "" || tx.PKCEVerifier == "" {
		return fmt.Errorf("transaction state, nonce, and PKCE verifier are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[tx.State] = tx
	return nil
}

func (s *MemoryTransactionStore) Take(_ context.Context, state string) (Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, ok := s.data[state]
	if !ok {
		return Transaction{}, fmt.Errorf("%w", ErrTransactionUnavailable)
	}
	delete(s.data, state)
	if !tx.CreatedAt.IsZero() && time.Since(tx.CreatedAt) > s.ttl {
		return Transaction{}, fmt.Errorf("%w", ErrTransactionUnavailable)
	}
	return tx, nil
}

// Cleanup removes expired transactions from the in-memory store. It mirrors
// the SQL store's maintenance contract for tests and simple deployments.
func (s *MemoryTransactionStore) Cleanup(_ context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	var removed int64
	for state, tx := range s.data {
		if !tx.CreatedAt.IsZero() && !now.Before(tx.CreatedAt.Add(s.ttl)) {
			delete(s.data, state)
			removed++
		}
	}
	return removed, nil
}

// DebugTransactions writes the current transaction count for tests/debugging.
func (s *MemoryTransactionStore) DebugTransactions(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	count := len(s.data)
	s.mu.Unlock()
	_ = json.NewEncoder(w).Encode(map[string]int{"transactions": count})
}
