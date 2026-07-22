// Package oidcauth provides an opinionated OIDC browser-login adapter for
// gojahttp hosts. It keeps identity-provider tokens server-side and creates an
// opaque application session for planned route authentication.
package oidcauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
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
	// CallbackErrorPage controls a bounded application-owned recovery page for
	// browser callback failures. OAuth query values never influence its copy,
	// links, or stylesheet URL.
	CallbackErrorPage CallbackErrorPage

	SessionManager   *sessionauth.Manager
	UserNormalizer   UserNormalizer
	TransactionStore TransactionStore
	Audit            gojahttp.AuditSink
	SecurityEvents   gojahttp.SecurityEventObserver
	// HTTPClient is used for OIDC discovery, token exchange, and remote key
	// retrieval. When nil, the standard context HTTP client is used.
	HTTPClient *http.Client
}

// CallbackErrorPage configures same-origin presentation for failed browser
// callbacks. Empty StylesheetPath intentionally leaves the safe page unstyled.
type CallbackErrorPage struct {
	StylesheetPath string
	RetryPath      string
	HomePath       string
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
	oauth2Config      oauth2.Config
	verifier          *oidc.IDTokenVerifier
	sessionManager    *sessionauth.Manager
	normalizer        UserNormalizer
	transactions      TransactionStore
	afterLoginURL     string
	afterLogoutURL    string
	audit             gojahttp.AuditSink
	securityEvents    gojahttp.SecurityEventObserver
	httpClient        *http.Client
	callbackErrorPage CallbackErrorPage
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
	callbackErrorPage, err := normalizeCallbackErrorPage(cfg.CallbackErrorPage)
	if err != nil {
		return nil, err
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
		oauth2Config:      oauth2Config,
		verifier:          provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		sessionManager:    cfg.SessionManager,
		normalizer:        cfg.UserNormalizer,
		transactions:      cfg.TransactionStore,
		afterLoginURL:     defaultIfEmpty(cfg.AfterLoginURL, "/"),
		afterLogoutURL:    defaultIfEmpty(cfg.AfterLogoutURL, "/"),
		audit:             cfg.Audit,
		securityEvents:    cfg.SecurityEvents,
		httpClient:        cfg.HTTPClient,
		callbackErrorPage: callbackErrorPage,
	}, nil
}

func (h *Handlers) LoginHandler() http.Handler { return http.HandlerFunc(h.handleLogin) }

// RegistrationHandler starts the same secure OIDC code flow while explicitly
// asking TinyIDP to present account creation. Other providers safely ignore the
// namespaced authorization parameter.
func (h *Handlers) RegistrationHandler() http.Handler { return http.HandlerFunc(h.handleRegistration) }
func (h *Handlers) CallbackHandler() http.Handler     { return http.HandlerFunc(h.handleCallback) }
func (h *Handlers) LogoutHandler() http.Handler       { return http.HandlerFunc(h.handleLogout) }

func (h *Handlers) handleLogin(w http.ResponseWriter, r *http.Request) {
	h.beginAuthorization(w, r, false)
}

func (h *Handlers) handleRegistration(w http.ResponseWriter, r *http.Request) {
	h.beginAuthorization(w, r, true)
}

func (h *Handlers) beginAuthorization(w http.ResponseWriter, r *http.Request, registration bool) {
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
	options := []oauth2.AuthCodeOption{oauth2.S256ChallengeOption(verifier), oauth2.SetAuthURLParam("nonce", nonce)}
	if registration {
		options = append(options, oauth2.SetAuthURLParam("tinyidp_signup", "1"))
	}
	url := h.oauth2Config.AuthCodeURL(state, options...)
	h.observe(r.Context(), "oidc.login", "issued", "")
	http.Redirect(w, r, url, http.StatusFound)
}

func (h *Handlers) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.observe(r.Context(), "oidc.callback", "rejected", "method")
		h.renderCallbackError(w, http.StatusMethodNotAllowed, callbackErrorRetry)
		return
	}
	if providerError := r.URL.Query().Get("error"); providerError != "" {
		h.observe(r.Context(), "oidc.callback", "rejected", "provider_error")
		h.renderCallbackError(w, http.StatusUnauthorized, callbackErrorFromProvider(providerError))
		return
	}
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	if state == "" || code == "" {
		h.observe(r.Context(), "oidc.callback", "rejected", "missing_parameters")
		h.renderCallbackError(w, http.StatusBadRequest, callbackErrorRetry)
		return
	}
	tx, err := h.transactions.Take(r.Context(), state)
	if err != nil {
		h.observe(r.Context(), "oidc.callback", "rejected", "transaction_unavailable")
		h.renderCallbackError(w, http.StatusUnauthorized, callbackErrorRestart)
		return
	}
	callbackCtx := withHTTPClient(r.Context(), h.httpClient)
	token, err := h.oauth2Config.Exchange(callbackCtx, code, oauth2.VerifierOption(tx.PKCEVerifier))
	if err != nil {
		h.observe(r.Context(), "oidc.callback", "rejected", "token_exchange")
		h.renderCallbackError(w, http.StatusUnauthorized, callbackErrorRestart)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		h.observe(r.Context(), "oidc.callback", "rejected", "missing_id_token")
		h.renderCallbackError(w, http.StatusUnauthorized, callbackErrorRestart)
		return
	}
	idToken, err := h.verifier.Verify(callbackCtx, rawIDToken)
	if err != nil {
		h.observe(r.Context(), "oidc.callback", "rejected", "id_token_verification")
		h.renderCallbackError(w, http.StatusUnauthorized, callbackErrorRestart)
		return
	}
	if idToken.Nonce != tx.Nonce {
		h.observe(r.Context(), "oidc.callback", "rejected", "nonce_mismatch")
		h.renderCallbackError(w, http.StatusUnauthorized, callbackErrorRestart)
		return
	}
	claims, err := claimsFromIDToken(idToken)
	if err != nil {
		h.observe(r.Context(), "oidc.callback", "rejected", "claims")
		h.renderCallbackError(w, http.StatusUnauthorized, callbackErrorRestart)
		return
	}
	userSession, err := h.normalizer.NormalizeOIDCUser(r.Context(), claims)
	if err != nil {
		h.observe(r.Context(), "oidc.callback", "rejected", "user_normalization")
		h.renderCallbackError(w, http.StatusUnauthorized, callbackErrorRetry)
		return
	}
	if userSession.UserID == "" {
		h.observe(r.Context(), "oidc.callback", "rejected", "empty_user")
		h.renderCallbackError(w, http.StatusUnauthorized, callbackErrorRetry)
		return
	}
	session, err := h.sessionManager.NewSession(r.Context(), userSession.UserID,
		sessionauth.WithEmail(userSession.Email, userSession.EmailVerified),
		sessionauth.WithTenantIDs(userSession.TenantIDs...),
		sessionauth.WithClaims(userSession.Claims),
	)
	if err != nil {
		h.observe(r.Context(), "oidc.callback", "failed", "session_creation")
		h.renderCallbackError(w, http.StatusInternalServerError, callbackErrorRetry)
		return
	}
	h.sessionManager.SetCookie(w, session.ID)
	h.observe(r.Context(), "oidc.callback", "accepted", "")
	http.Redirect(w, r, tx.RedirectURL, http.StatusFound)
}

type callbackErrorKind string

const (
	callbackErrorCanceled callbackErrorKind = "canceled"
	callbackErrorRestart  callbackErrorKind = "restart"
	callbackErrorRetry    callbackErrorKind = "retry"
)

func callbackErrorFromProvider(providerError string) callbackErrorKind {
	if providerError == "access_denied" {
		return callbackErrorCanceled
	}
	return callbackErrorRetry
}

func normalizeCallbackErrorPage(page CallbackErrorPage) (CallbackErrorPage, error) {
	var err error
	if page.StylesheetPath, err = localCallbackPath(page.StylesheetPath, true); err != nil {
		return CallbackErrorPage{}, fmt.Errorf("oidcauth: callback error stylesheet path: %w", err)
	}
	if page.RetryPath, err = localCallbackPath(page.RetryPath, true); err != nil {
		return CallbackErrorPage{}, fmt.Errorf("oidcauth: callback error retry path: %w", err)
	}
	if page.HomePath, err = localCallbackPath(page.HomePath, true); err != nil {
		return CallbackErrorPage{}, fmt.Errorf("oidcauth: callback error home path: %w", err)
	}
	if page.RetryPath == "" {
		page.RetryPath = "/auth/login"
	}
	if page.HomePath == "" {
		page.HomePath = "/"
	}
	return page, nil
}

func localCallbackPath(raw string, allowEmpty bool) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" && allowEmpty {
		return "", nil
	}
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.Contains(value, "\\") {
		return "", errors.New("must be an absolute same-origin path")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return "", errors.New("must be an absolute same-origin path")
	}
	return value, nil
}

func (h *Handlers) renderCallbackError(w http.ResponseWriter, status int, kind callbackErrorKind) {
	title, summary := callbackErrorCopy(kind)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.WriteHeader(status)
	_, _ = fmt.Fprint(w, "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title>", html.EscapeString(title), "</title>")
	if h.callbackErrorPage.StylesheetPath != "" {
		_, _ = fmt.Fprint(w, "<link rel=\"stylesheet\" href=\"", html.EscapeString(h.callbackErrorPage.StylesheetPath), "\">")
	}
	_, _ = fmt.Fprint(w, "</head><body><main><section class=\"card auth-callback-error\"><p class=\"badge\">SIGN-IN</p><h1>", html.EscapeString(title), "</h1><p>", html.EscapeString(summary), "</p><div class=\"actions\"><a class=\"button primary\" href=\"", html.EscapeString(h.callbackErrorPage.RetryPath), "\">Try signing in again</a><a class=\"button\" href=\"", html.EscapeString(h.callbackErrorPage.HomePath), "\">Return to the application</a></div></section></main></body></html>")
}

func callbackErrorCopy(kind callbackErrorKind) (string, string) {
	switch kind {
	case callbackErrorCanceled:
		return "Sign-in was canceled", "You chose not to approve access. You can try again whenever you are ready."
	case callbackErrorRestart:
		return "Sign-in needs to be restarted", "This sign-in link is no longer active. Return to the application and start again."
	case callbackErrorRetry:
		return "We couldn’t complete sign-in", "Try signing in again. If the problem continues, return to the application and begin a new sign-in."
	default:
		return "We couldn’t complete sign-in", "Try signing in again. If the problem continues, return to the application and begin a new sign-in."
	}
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
