package programauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
)

type DeviceHandlersConfig struct {
	Service        DeviceService
	SessionManager *sessionauth.Manager
}

type DeviceHandlers struct {
	service        DeviceService
	sessionManager *sessionauth.Manager
}

func NewDeviceHandlers(cfg DeviceHandlersConfig) (*DeviceHandlers, error) {
	if cfg.Service.Store == nil {
		return nil, fmt.Errorf("device handlers require device service store")
	}
	if cfg.Service.OAuthTokens.AccessTokens == nil || cfg.Service.OAuthTokens.RefreshTokens == nil {
		return nil, fmt.Errorf("device handlers require oauth token service")
	}
	return &DeviceHandlers{service: cfg.Service, sessionManager: cfg.SessionManager}, nil
}

func (h *DeviceHandlers) StartHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var body deviceStartRequest
		if err := decodeJSONRequest(r, &body); err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error(), 0)
			return
		}
		grants, err := grantsFromActions(body.Actions, body.TenantID)
		if err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_scope", err.Error(), 0)
			return
		}
		started, err := h.service.StartDeviceAuthorization(r.Context(), DeviceStartSpec{ClientName: firstNonEmpty(body.ClientName, body.ClientNameSnake), TenantID: body.TenantID, Grants: grants, VerificationURI: body.VerificationURI})
		if err != nil {
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
	})
}

func (h *DeviceHandlers) TokenHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		deviceCode, grantType, err := readDeviceTokenRequest(r)
		if err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error(), 0)
			return
		}
		if grantType != "" && grantType != "urn:ietf:params:oauth:grant-type:device_code" {
			writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "unsupported grant_type", 0)
			return
		}
		issued, err := h.service.PollDeviceAuthorization(r.Context(), deviceCode)
		if err != nil {
			handleDevicePollError(w, err)
			return
		}
		writeIssuedTokenPair(w, issued)
	})
}

// RefreshHandler exchanges a programauth refresh token for a rotated token
// pair. It is application-owned: tiny-idp authenticates the browser user who
// approves the device, while this endpoint manages credentials for this host.
func (h *DeviceHandlers) RefreshHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		refreshToken, grantType, err := readRefreshTokenRequest(r)
		if err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error(), 0)
			return
		}
		if grantType != "" && grantType != "refresh_token" {
			writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "unsupported grant_type", 0)
			return
		}
		issued, err := h.service.OAuthTokens.RefreshTokenPair(r.Context(), refreshToken, 0, 0)
		if err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "invalid refresh token", 0)
			return
		}
		writeIssuedTokenPair(w, issued)
	})
}

// RevokeHandler revokes the refresh-token family identified by the supplied
// credential. As in OAuth revocation, an invalid or already-revoked token is
// acknowledged without revealing whether it existed.
func (h *DeviceHandlers) RevokeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		refreshToken, _, err := readRefreshTokenRequest(r)
		if err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error(), 0)
			return
		}
		if err := h.service.OAuthTokens.RevokeRefreshToken(r.Context(), refreshToken); err != nil && !errors.Is(err, gojahttp.ErrUnauthenticated) {
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "refresh token revocation failed", 0)
			return
		}
		writeJSONResponse(w, http.StatusOK, map[string]any{"ok": true})
	})
}

func (h *DeviceHandlers) ApproveHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
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
		if err := h.sessionManager.VerifyCSRF(r.Context(), gojahttp.CSRFRequest{HTTPRequest: r, Actor: &gojahttp.Actor{ID: session.UserID}}); err != nil {
			writeOAuthError(w, http.StatusForbidden, "access_denied", "missing or invalid CSRF token", 0)
			return
		}
		var body deviceApproveRequest
		if err := decodeJSONRequest(r, &body); err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error(), 0)
			return
		}
		grants, err := grantsFromActions(body.Actions, body.TenantID)
		if err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_scope", err.Error(), 0)
			return
		}
		approved, err := h.service.ApproveDeviceAuthorization(r.Context(), DeviceApprovalSpec{UserCode: firstNonEmpty(body.UserCode, body.UserCodeSnake), SubjectUserID: session.UserID, TenantID: body.TenantID, AgentName: body.AgentName, AgentKind: AgentKindDevice, Grants: grants})
		if err != nil {
			writeOAuthError(w, statusForDeviceApprovalError(err), "invalid_request", err.Error(), 0)
			return
		}
		writeJSONResponse(w, http.StatusOK, map[string]any{"ok": true, "device": approved})
	})
}

type deviceStartRequest struct {
	ClientName      string   `json:"clientName"`
	ClientNameSnake string   `json:"client_name"`
	TenantID        string   `json:"tenantId"`
	Actions         []string `json:"actions"`
	VerificationURI string   `json:"verificationUri"`
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
