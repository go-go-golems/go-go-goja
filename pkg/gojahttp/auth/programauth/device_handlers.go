package programauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
)

// DeviceEndpointPolicy is the host-owned boundary for application device grants.
type DeviceEndpointPolicy struct {
	AllowedActions  map[string]struct{}
	MaxActions      int
	VerificationURI string
}

type DeviceHandlersConfig struct {
	Service        DeviceService
	SessionManager *sessionauth.Manager
	Audit          gojahttp.AuditSink
	SecurityEvents gojahttp.SecurityEventObserver
	RateLimiter    gojahttp.RateLimiter
	Policy         DeviceEndpointPolicy
	Users          appauth.UserStore
}

type DeviceHandlers struct {
	service        DeviceService
	sessionManager *sessionauth.Manager
	audit          gojahttp.AuditSink
	securityEvents gojahttp.SecurityEventObserver
	policy         DeviceEndpointPolicy
	rateLimiter    gojahttp.RateLimiter
	users          appauth.UserStore
}

func NewDeviceHandlers(cfg DeviceHandlersConfig) (*DeviceHandlers, error) {
	if cfg.Service.Store == nil {
		return nil, fmt.Errorf("device handlers require device service store")
	}
	if cfg.Service.OAuthTokens.AccessTokens == nil || cfg.Service.OAuthTokens.RefreshTokens == nil {
		return nil, fmt.Errorf("device handlers require oauth token service")
	}
	return &DeviceHandlers{service: cfg.Service, sessionManager: cfg.SessionManager, audit: cfg.Audit, securityEvents: cfg.SecurityEvents, policy: cfg.Policy, rateLimiter: cfg.RateLimiter, users: cfg.Users}, nil
}

func (h *DeviceHandlers) StartHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			h.observe(r, "programauth.device.start", "rejected", "method")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !h.allowRequest(w, r, "auth.device.start", 10) {
			return
		}
		var body deviceStartRequest
		if err := decodeJSONRequest(r, &body); err != nil {
			h.observe(r, "programauth.device.start", "rejected", "invalid_request")
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error(), 0)
			return
		}
		grants, err := h.grantsFromActions(body.Actions, body.TenantID)
		if err != nil {
			h.observe(r, "programauth.device.start", "rejected", "invalid_scope")
			writeOAuthError(w, http.StatusBadRequest, "invalid_scope", err.Error(), 0)
			return
		}
		started, err := h.service.StartDeviceAuthorization(r.Context(), DeviceStartSpec{ClientName: firstNonEmpty(body.ClientName, body.ClientNameSnake), TenantID: body.TenantID, Grants: grants, VerificationURI: h.policy.VerificationURI})
		if err != nil {
			h.observe(r, "programauth.device.start", "failed", "persistence")
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error(), 0)
			return
		}
		writeJSONResponse(w, http.StatusOK, map[string]any{
			"device_code":               started.DeviceCode,
			"user_code":                 started.UserCode,
			"verification_uri":          started.Device.VerificationURI,
			"verification_uri_complete": started.Device.VerificationURIComplete,
			"expires_in":                int(started.Device.ExpiresAt.Sub(started.Device.CreatedAt).Seconds()),
			"interval":                  started.Device.PollIntervalSeconds,
		})
		h.observe(r, "programauth.device.start", "issued", "")
	})
}

func (h *DeviceHandlers) TokenHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			h.observe(r, "programauth.device.poll", "rejected", "method")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !h.allowRequest(w, r, "auth.device.poll", 60) {
			return
		}
		deviceCode, grantType, err := readDeviceTokenRequest(r)
		if err != nil {
			h.observe(r, "programauth.device.poll", "rejected", "invalid_request")
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error(), 0)
			return
		}
		if grantType != "" && grantType != "urn:ietf:params:oauth:grant-type:device_code" {
			h.observe(r, "programauth.device.poll", "rejected", "grant_type")
			writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "unsupported grant_type", 0)
			return
		}
		issued, err := h.service.PollDeviceAuthorization(r.Context(), deviceCode)
		if err != nil {
			h.observe(r, "programauth.device.poll", "rejected", devicePollReason(err))
			handleDevicePollError(w, err)
			return
		}
		writeIssuedTokenPair(w, issued)
		h.observe(r, "programauth.device.poll", "issued", "")
	})
}

// RefreshHandler exchanges a programauth refresh token for a rotated token
// pair. It is application-owned: tiny-idp authenticates the browser user who
// approves the device, while this endpoint manages credentials for this host.
func (h *DeviceHandlers) RefreshHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			h.observe(r, "programauth.refresh", "rejected", "method")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !h.allowRequest(w, r, "auth.device.refresh", 30) {
			return
		}
		refreshToken, grantType, err := readRefreshTokenRequest(r)
		if err != nil {
			h.observe(r, "programauth.refresh", "rejected", "invalid_request")
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error(), 0)
			return
		}
		if grantType != "" && grantType != "refresh_token" {
			h.observe(r, "programauth.refresh", "rejected", "grant_type")
			writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "unsupported grant_type", 0)
			return
		}
		issued, err := h.service.OAuthTokens.RefreshTokenPair(r.Context(), refreshToken, 0, 0)
		if err != nil {
			h.observe(r, "programauth.refresh", "rejected", "invalid_grant")
			writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "invalid refresh token", 0)
			return
		}
		writeIssuedTokenPair(w, issued)
		h.observe(r, "programauth.refresh", "rotated", "")
	})
}

// RevokeHandler revokes the refresh-token family identified by the supplied
// credential. As in OAuth revocation, an invalid or already-revoked token is
// acknowledged without revealing whether it existed.
func (h *DeviceHandlers) RevokeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			h.observe(r, "programauth.refresh_revoke", "rejected", "method")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !h.allowRequest(w, r, "auth.device.revoke", 30) {
			return
		}
		refreshToken, _, err := readRefreshTokenRequest(r)
		if err != nil {
			h.observe(r, "programauth.refresh_revoke", "rejected", "invalid_request")
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error(), 0)
			return
		}
		if err := h.service.OAuthTokens.RevokeRefreshToken(r.Context(), refreshToken); err != nil && !errors.Is(err, gojahttp.ErrUnauthenticated) {
			h.observe(r, "programauth.refresh_revoke", "failed", "persistence")
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "refresh token revocation failed", 0)
			return
		}
		writeJSONResponse(w, http.StatusOK, map[string]any{"ok": true})
		h.observe(r, "programauth.refresh_revoke", "accepted", "")
	})
}

func (h *DeviceHandlers) ApproveHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			h.observe(r, "programauth.device.approve", "rejected", "method")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !h.allowRequest(w, r, "auth.device.approval", 30) {
			return
		}
		if h.sessionManager == nil {
			h.observe(r, "programauth.device.approve", "rejected", "session_unavailable")
			writeOAuthError(w, http.StatusUnauthorized, "unauthorized", "session manager is not configured", 0)
			return
		}
		session, err := h.sessionManager.SessionFromRequest(r.Context(), r)
		if err != nil {
			h.observe(r, "programauth.device.approve", "rejected", "unauthenticated")
			writeOAuthError(w, http.StatusUnauthorized, "unauthorized", "unauthenticated", 0)
			return
		}
		if err := h.sessionManager.VerifyCSRF(r.Context(), gojahttp.CSRFRequest{HTTPRequest: r, Actor: &gojahttp.Actor{ID: session.UserID}}); err != nil {
			h.observe(r, "programauth.device.approve", "rejected", "csrf")
			writeOAuthError(w, http.StatusForbidden, "access_denied", "missing or invalid CSRF token", 0)
			return
		}
		var body deviceApproveRequest
		if err := decodeJSONRequest(r, &body); err != nil {
			h.observe(r, "programauth.device.approve", "rejected", "invalid_request")
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error(), 0)
			return
		}
		grants, err := grantsFromActions(body.Actions, body.TenantID)
		if err != nil {
			h.observe(r, "programauth.device.approve", "rejected", "invalid_scope")
			writeOAuthError(w, http.StatusBadRequest, "invalid_scope", err.Error(), 0)
			return
		}
		approved, err := h.service.ApproveDeviceAuthorization(r.Context(), DeviceApprovalSpec{UserCode: firstNonEmpty(body.UserCode, body.UserCodeSnake), SubjectUserID: session.UserID, TenantID: body.TenantID, AgentName: body.AgentName, AgentKind: AgentKindDevice, Grants: grants})
		if err != nil {
			h.observe(r, "programauth.device.approve", "rejected", "approval")
			writeOAuthError(w, statusForDeviceApprovalError(err), "invalid_request", err.Error(), 0)
			return
		}
		writeJSONResponse(w, http.StatusOK, map[string]any{"ok": true, "device": approved})
		h.observe(r, "programauth.device.approve", "accepted", "")
	})
}

// RequestHandler returns a redacted pending request to an authenticated browser
// user. POST keeps user codes out of query-string logs and browser history.
func (h *DeviceHandlers) RequestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !h.allowRequest(w, r, "auth.device.approval", 30) {
			return
		}
		if !h.requireSessionCSRF(w, r, "programauth.device.inspect") {
			return
		}
		var body deviceUserCodeRequest
		if err := decodeJSONRequest(r, &body); err != nil {
			h.observe(r, "programauth.device.inspect", "rejected", "invalid_request")
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error(), 0)
			return
		}
		pending, err := h.service.InspectDeviceAuthorization(r.Context(), firstNonEmpty(body.UserCode, body.UserCodeSnake))
		if err != nil {
			h.observe(r, "programauth.device.inspect", "rejected", "request")
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid device request", 0)
			return
		}
		writeJSONResponse(w, http.StatusOK, map[string]any{"clientName": pending.ClientName, "requestedActions": pending.RequestedActions, "expiresAt": pending.ExpiresAt, "status": pending.Status})
		h.observe(r, "programauth.device.inspect", "accepted", "")
	})
}

// DenyHandler terminally denies a pending device request for the authenticated
// browser user. Polling the corresponding device code subsequently returns
// access_denied.
func (h *DeviceHandlers) DenyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !h.allowRequest(w, r, "auth.device.approval", 30) {
			return
		}
		if !h.requireSessionCSRF(w, r, "programauth.device.deny") {
			return
		}
		var body deviceUserCodeRequest
		if err := decodeJSONRequest(r, &body); err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error(), 0)
			return
		}
		if _, err := h.service.DenyDeviceAuthorization(r.Context(), firstNonEmpty(body.UserCode, body.UserCodeSnake)); err != nil {
			h.observe(r, "programauth.device.deny", "rejected", "request")
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", "invalid device request", 0)
			return
		}
		writeJSONResponse(w, http.StatusOK, map[string]any{"ok": true})
		h.observe(r, "programauth.device.deny", "accepted", "")
	})
}

func (h *DeviceHandlers) requireSessionCSRF(w http.ResponseWriter, r *http.Request, event string) bool {
	if h.sessionManager == nil {
		h.observe(r, event, "rejected", "session_unavailable")
		writeOAuthError(w, http.StatusUnauthorized, "unauthorized", "session manager is not configured", 0)
		return false
	}
	session, err := h.sessionManager.SessionFromRequest(r.Context(), r)
	if err != nil {
		h.observe(r, event, "rejected", "unauthenticated")
		writeOAuthError(w, http.StatusUnauthorized, "unauthorized", "unauthenticated", 0)
		return false
	}
	if err := h.sessionManager.VerifyCSRF(r.Context(), gojahttp.CSRFRequest{HTTPRequest: r, Actor: &gojahttp.Actor{ID: session.UserID}}); err != nil {
		h.observe(r, event, "rejected", "csrf")
		writeOAuthError(w, http.StatusForbidden, "access_denied", "missing or invalid CSRF token", 0)
		return false
	}
	return true
}

func (h *DeviceHandlers) allowRequest(w http.ResponseWriter, r *http.Request, policy string, limit int) bool {
	if h.rateLimiter == nil {
		return true
	}
	decision, err := h.rateLimiter.CheckRateLimit(r.Context(), gojahttp.RateLimitRequest{HTTPRequest: r, Spec: gojahttp.RateLimitSpec{Policy: policy, Limit: limit, Window: time.Minute}, Key: gojahttp.RequestClientIP(r)})
	if err != nil || decision.Allowed {
		return true
	}
	if decision.RetryAfter > 0 {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(decision.RetryAfter.Seconds())+1))
	}
	h.observe(r, "programauth.device.rate_limit", "rejected", policy)
	writeOAuthError(w, http.StatusTooManyRequests, "rate_limited", "too many requests", 0)
	return false
}

func (h *DeviceHandlers) grantsFromActions(actions []string, tenantID string) (gojahttp.GrantSet, error) {
	grants, err := grantsFromActions(actions, tenantID)
	if err != nil {
		return gojahttp.GrantSet{}, err
	}
	if len(grants.Grants) == 0 {
		return gojahttp.GrantSet{}, fmt.Errorf("at least one action is required")
	}
	if h.policy.MaxActions > 0 && len(grants.Grants) > h.policy.MaxActions {
		return gojahttp.GrantSet{}, fmt.Errorf("too many requested actions")
	}
	for _, grant := range grants.Grants {
		if len(h.policy.AllowedActions) != 0 {
			if _, ok := h.policy.AllowedActions[grant.Action]; !ok {
				return gojahttp.GrantSet{}, fmt.Errorf("action %q is not allowed", grant.Action)
			}
		}
	}
	return grants, nil
}

// ListAgentsHandler returns only the authenticated local user's agents.
func (h *DeviceHandlers) ListAgentsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if h.sessionManager == nil {
			writeOAuthError(w, http.StatusUnauthorized, "unauthorized", "session manager is not configured", 0)
			return
		}
		session, err := h.sessionManager.SessionFromRequest(r.Context(), r)
		if err != nil {
			writeOAuthError(w, http.StatusUnauthorized, "unauthorized", "unauthenticated", 0)
			return
		}
		agents, err := h.service.Agents.ListOwnedAgents(r.Context(), session.UserID)
		if err != nil {
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "agent listing failed", 0)
			return
		}
		writeJSONResponse(w, http.StatusOK, map[string]any{"agents": agents})
	})
}

// DisableUserHandler disables the current local user and immediately blocks future lookups.
func (h *DeviceHandlers) DisableUserHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !h.requireSessionCSRF(w, r, "programauth.user.disable") {
			return
		}
		if h.users == nil {
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "user store is not configured", 0)
			return
		}
		session, err := h.sessionManager.SessionFromRequest(r.Context(), r)
		if err != nil {
			writeOAuthError(w, http.StatusUnauthorized, "unauthorized", "unauthenticated", 0)
			return
		}
		if err := h.users.DisableUser(r.Context(), session.UserID, time.Now()); err != nil {
			writeOAuthError(w, http.StatusNotFound, "not_found", "user not found", 0)
			return
		}
		writeJSONResponse(w, http.StatusOK, map[string]any{"ok": true})
		h.observe(r, "programauth.user.disable", "accepted", "")
	})
}

// DisableAgentHandler immediately disables an agent owned by the authenticated
// local user. The service enforces ownership independently of this route.
func (h *DeviceHandlers) DisableAgentHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !h.requireSessionCSRF(w, r, "programauth.agent.disable") {
			return
		}
		var body struct {
			AgentID string `json:"agentId"`
		}
		if err := decodeJSONRequest(r, &body); err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error(), 0)
			return
		}
		session, err := h.sessionManager.SessionFromRequest(r.Context(), r)
		if err != nil {
			writeOAuthError(w, http.StatusUnauthorized, "unauthorized", "unauthenticated", 0)
			return
		}
		if _, err := h.service.Agents.DisableOwnedAgent(r.Context(), session.UserID, body.AgentID); err != nil {
			writeOAuthError(w, http.StatusNotFound, "not_found", "agent not found", 0)
			return
		}
		writeJSONResponse(w, http.StatusOK, map[string]any{"ok": true})
		h.observe(r, "programauth.agent.disable", "accepted", "")
	})
}

// ListRefreshFamiliesHandler exposes redacted refresh credential metadata only to its owner.
func (h *DeviceHandlers) ListRefreshFamiliesHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if h.sessionManager == nil {
			writeOAuthError(w, http.StatusUnauthorized, "unauthorized", "session manager is not configured", 0)
			return
		}
		session, err := h.sessionManager.SessionFromRequest(r.Context(), r)
		if err != nil {
			writeOAuthError(w, http.StatusUnauthorized, "unauthorized", "unauthenticated", 0)
			return
		}
		tokens, err := h.service.OAuthTokens.ListOwnedRefreshTokens(r.Context(), session.UserID)
		if err != nil {
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "refresh token listing failed", 0)
			return
		}
		writeJSONResponse(w, http.StatusOK, map[string]any{"refreshTokens": tokens})
	})
}

// RevokeRefreshFamilyHandler revokes one session-owned refresh-token family.
func (h *DeviceHandlers) RevokeRefreshFamilyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !h.requireSessionCSRF(w, r, "programauth.refresh.revoke") {
			return
		}
		var body struct {
			FamilyID string `json:"familyId"`
		}
		if err := decodeJSONRequest(r, &body); err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error(), 0)
			return
		}
		session, err := h.sessionManager.SessionFromRequest(r.Context(), r)
		if err != nil {
			writeOAuthError(w, http.StatusUnauthorized, "unauthorized", "unauthenticated", 0)
			return
		}
		if err := h.service.OAuthTokens.RevokeOwnedRefreshTokenFamily(r.Context(), session.UserID, body.FamilyID); err != nil {
			writeOAuthError(w, http.StatusNotFound, "not_found", "refresh family not found", 0)
			return
		}
		writeJSONResponse(w, http.StatusOK, map[string]any{"ok": true})
		h.observe(r, "programauth.refresh.revoke", "accepted", "")
	})
}

func (h *DeviceHandlers) observe(r *http.Request, event, outcome, reason string) {
	if h.securityEvents != nil {
		h.securityEvents.ObserveSecurityEvent(r.Context(), gojahttp.SecurityEvent{Name: event, Outcome: outcome, Reason: reason})
	}
	if h.audit != nil {
		_ = h.audit.RecordAudit(r.Context(), gojahttp.AuditEvent{Event: event, Outcome: outcome, Reason: reason, Method: "INTERNAL", Pattern: "device-auth", HTTPRequest: r})
	}
}

func devicePollReason(err error) string {
	var deviceErr DeviceError
	if errors.As(err, &deviceErr) {
		switch {
		case errors.Is(deviceErr, ErrDeviceAuthorizationPending):
			return "pending"
		case errors.Is(deviceErr, ErrDeviceSlowDown):
			return "slow_down"
		}
	}
	switch {
	case errors.Is(err, ErrDeviceExpired):
		return "expired"
	case errors.Is(err, ErrDeviceDenied):
		return "denied"
	case errors.Is(err, ErrDeviceConsumed):
		return "consumed"
	case errors.Is(err, gojahttp.ErrUnauthenticated):
		return "invalid_code"
	default:
		return "failed"
	}
}

type deviceStartRequest struct {
	ClientName      string   `json:"clientName"`
	ClientNameSnake string   `json:"client_name"`
	TenantID        string   `json:"tenantId"`
	Actions         []string `json:"actions"`
}

type deviceUserCodeRequest struct {
	UserCode      string `json:"userCode"`
	UserCodeSnake string `json:"user_code"`
}

type deviceApproveRequest struct {
	UserCode      string   `json:"userCode"`
	UserCodeSnake string   `json:"user_code"`
	TenantID      string   `json:"tenantId"`
	AgentName     string   `json:"agentName"`
	Actions       []string `json:"actions"`
}

type deviceTokenJSONRequest struct {
	GrantType       string `json:"grant_type"`
	DeviceCode      string `json:"deviceCode"`
	DeviceCodeSnake string `json:"device_code"`
}

type refreshTokenJSONRequest struct {
	GrantType         string `json:"grant_type"`
	RefreshToken      string `json:"refreshToken"`
	RefreshTokenSnake string `json:"refresh_token"`
	Token             string `json:"token"`
}

func readDeviceTokenRequest(r *http.Request) (string, string, error) {
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			return "", "", err
		}
		return strings.TrimSpace(r.Form.Get("device_code")), strings.TrimSpace(r.Form.Get("grant_type")), nil
	}
	var body deviceTokenJSONRequest
	if err := decodeJSONRequest(r, &body); err != nil {
		return "", "", err
	}
	return firstNonEmpty(body.DeviceCode, body.DeviceCodeSnake), strings.TrimSpace(body.GrantType), nil
}

func readRefreshTokenRequest(r *http.Request) (string, string, error) {
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			return "", "", err
		}
		return strings.TrimSpace(firstNonEmpty(r.Form.Get("refresh_token"), r.Form.Get("refreshToken"), r.Form.Get("token"))), strings.TrimSpace(r.Form.Get("grant_type")), nil
	}
	var body refreshTokenJSONRequest
	if err := decodeJSONRequest(r, &body); err != nil {
		return "", "", err
	}
	return firstNonEmpty(body.RefreshToken, body.RefreshTokenSnake, body.Token), strings.TrimSpace(body.GrantType), nil
}

func writeIssuedTokenPair(w http.ResponseWriter, issued IssuedOAuthTokenPair) {
	writeJSONResponse(w, http.StatusOK, map[string]any{
		"access_token":  issued.AccessValue,
		"refresh_token": issued.RefreshValue,
		"token_type":    "Bearer",
		"expires_in":    int(issued.AccessToken.ExpiresAt.Sub(issued.AccessToken.CreatedAt).Seconds()),
		"scope":         strings.Join(issued.AccessToken.Scopes, " "),
	})
}

func handleDevicePollError(w http.ResponseWriter, err error) {
	var deviceErr DeviceError
	if errors.As(err, &deviceErr) {
		switch {
		case errors.Is(deviceErr, ErrDeviceAuthorizationPending):
			writeOAuthError(w, http.StatusBadRequest, "authorization_pending", "authorization is pending", int(deviceErr.Interval/time.Second))
		case errors.Is(deviceErr, ErrDeviceSlowDown):
			writeOAuthError(w, http.StatusBadRequest, "slow_down", "poll interval increased", int(deviceErr.Interval/time.Second))
		default:
			writeOAuthError(w, http.StatusBadRequest, "invalid_grant", err.Error(), 0)
		}
		return
	}
	switch {
	case errors.Is(err, ErrDeviceExpired):
		writeOAuthError(w, http.StatusBadRequest, "expired_token", "device code expired", 0)
	case errors.Is(err, ErrDeviceDenied):
		writeOAuthError(w, http.StatusBadRequest, "access_denied", "device authorization denied", 0)
	case errors.Is(err, ErrDeviceConsumed):
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "device code already consumed", 0)
	case errors.Is(err, gojahttp.ErrUnauthenticated):
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "invalid device code", 0)
	default:
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "device token request failed", 0)
	}
}

func statusForDeviceApprovalError(err error) int {
	switch {
	case errors.Is(err, ErrDeviceExpired), errors.Is(err, ErrDeviceDenied), errors.Is(err, ErrDeviceConsumed):
		return http.StatusBadRequest
	default:
		return http.StatusBadRequest
	}
}

func grantsFromActions(actions []string, tenantID string) (gojahttp.GrantSet, error) {
	grants := make([]gojahttp.Grant, 0, len(actions))
	for _, action := range actions {
		action = strings.TrimSpace(action)
		if action == "" {
			continue
		}
		grants = append(grants, gojahttp.Grant{Action: action, TenantID: strings.TrimSpace(tenantID)})
	}
	return gojahttp.NewGrantSet(grants...)
}

func decodeJSONRequest(r *http.Request, out any) error {
	if r.Body == nil {
		return fmt.Errorf("request body is required")
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	return nil
}

func writeOAuthError(w http.ResponseWriter, status int, code, description string, interval int) {
	payload := map[string]any{"error": code, "error_description": description}
	if interval > 0 {
		payload["interval"] = interval
	}
	writeJSONResponse(w, status, payload)
}

func writeJSONResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
