package programauth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/programauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
)

func TestDeviceHandlersStartApproveAndToken(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 21, 2, 0, 0, 0, time.UTC)
	current := now
	service := newDeviceTestService(func() time.Time { return current })
	manager, err := sessionauth.New(sessionauth.Config{Store: sessionauth.NewMemoryStore(), AllowInsecureHTTP: true, Now: func() time.Time { return current }})
	if err != nil {
		t.Fatalf("sessionauth.New: %v", err)
	}
	handlers, err := programauth.NewDeviceHandlers(programauth.DeviceHandlersConfig{Service: service, SessionManager: manager})
	if err != nil {
		t.Fatalf("NewDeviceHandlers: %v", err)
	}

	startRecorder := httptest.NewRecorder()
	startReq := jsonRequest(t, "/auth/device/start", map[string]any{"clientName": "goja-cli", "tenantId": "o1", "actions": []string{"report.read"}})
	handlers.StartHandler().ServeHTTP(startRecorder, startReq)
	if startRecorder.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", startRecorder.Code, startRecorder.Body.String())
	}
	var started map[string]any
	decodeRecorderJSON(t, startRecorder, &started)
	deviceCode, _ := started["device_code"].(string)
	userCode, _ := started["user_code"].(string)
	if deviceCode == "" || userCode == "" {
		t.Fatalf("start response = %#v", started)
	}

	pendingRecorder := httptest.NewRecorder()
	handlers.TokenHandler().ServeHTTP(pendingRecorder, jsonRequest(t, "/auth/device/token", map[string]any{"device_code": deviceCode, "grant_type": "urn:ietf:params:oauth:grant-type:device_code"}))
	if pendingRecorder.Code != http.StatusBadRequest || !bytes.Contains(pendingRecorder.Body.Bytes(), []byte("authorization_pending")) {
		t.Fatalf("pending status=%d body=%s", pendingRecorder.Code, pendingRecorder.Body.String())
	}

	session, err := manager.NewSession(ctx, "u1")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	approveRecorder := httptest.NewRecorder()
	approveReq := jsonRequest(t, "/auth/device/approve", map[string]any{"user_code": userCode})
	manager.SetCookie(approveRecorder, session.ID)
	for _, cookie := range approveRecorder.Result().Cookies() {
		approveReq.AddCookie(cookie)
	}
	approveReq.Header.Set(sessionauth.CSRFHeaderName, session.CSRFToken)
	approveRecorder = httptest.NewRecorder()
	handlers.ApproveHandler().ServeHTTP(approveRecorder, approveReq)
	if approveRecorder.Code != http.StatusOK {
		t.Fatalf("approve status=%d body=%s", approveRecorder.Code, approveRecorder.Body.String())
	}

	current = current.Add(10 * time.Second)
	tokenRecorder := httptest.NewRecorder()
	handlers.TokenHandler().ServeHTTP(tokenRecorder, formRequest("/auth/device/token", "grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Adevice_code&device_code="+deviceCode))
	if tokenRecorder.Code != http.StatusOK {
		t.Fatalf("token status=%d body=%s", tokenRecorder.Code, tokenRecorder.Body.String())
	}
	var token map[string]any
	decodeRecorderJSON(t, tokenRecorder, &token)
	if token["access_token"] == "" || token["refresh_token"] == "" || token["token_type"] != "Bearer" {
		t.Fatalf("token response = %#v", token)
	}
	refreshValue, _ := token["refresh_token"].(string)
	refreshRecorder := httptest.NewRecorder()
	handlers.RefreshHandler().ServeHTTP(refreshRecorder, formRequest("/auth/device/refresh", "grant_type=refresh_token&refresh_token="+refreshValue))
	if refreshRecorder.Code != http.StatusOK {
		t.Fatalf("refresh status=%d body=%s", refreshRecorder.Code, refreshRecorder.Body.String())
	}
	var refreshed map[string]any
	decodeRecorderJSON(t, refreshRecorder, &refreshed)
	rotatedRefresh, _ := refreshed["refresh_token"].(string)
	if refreshed["access_token"] == "" || rotatedRefresh == "" || rotatedRefresh == refreshValue {
		t.Fatalf("refresh response = %#v", refreshed)
	}

	revokeRecorder := httptest.NewRecorder()
	handlers.RevokeHandler().ServeHTTP(revokeRecorder, jsonRequest(t, "/auth/device/revoke", map[string]any{"refresh_token": rotatedRefresh}))
	if revokeRecorder.Code != http.StatusOK {
		t.Fatalf("revoke status=%d body=%s", revokeRecorder.Code, revokeRecorder.Body.String())
	}

	rejectedRefreshRecorder := httptest.NewRecorder()
	handlers.RefreshHandler().ServeHTTP(rejectedRefreshRecorder, jsonRequest(t, "/auth/device/refresh", map[string]any{"grant_type": "refresh_token", "refresh_token": rotatedRefresh}))
	if rejectedRefreshRecorder.Code != http.StatusBadRequest || !bytes.Contains(rejectedRefreshRecorder.Body.Bytes(), []byte("invalid_grant")) {
		t.Fatalf("revoked refresh status=%d body=%s", rejectedRefreshRecorder.Code, rejectedRefreshRecorder.Body.String())
	}
}

func TestDeviceTokenHandlerDoesNotRevealCodeEnumerationDetails(t *testing.T) {
	service := newDeviceTestService(time.Now)
	handlers, err := programauth.NewDeviceHandlers(programauth.DeviceHandlersConfig{Service: service})
	if err != nil {
		t.Fatalf("NewDeviceHandlers: %v", err)
	}
	for _, code := range []string{"not-a-device-code", "ggdc_deadbeef_deadbeef"} {
		recorder := httptest.NewRecorder()
		handlers.TokenHandler().ServeHTTP(recorder, jsonRequest(t, "/auth/device/token", map[string]any{"grant_type": "urn:ietf:params:oauth:grant-type:device_code", "device_code": code}))
		if recorder.Code != http.StatusBadRequest || !bytes.Contains(recorder.Body.Bytes(), []byte("invalid_grant")) {
			t.Fatalf("code %q status=%d body=%s", code, recorder.Code, recorder.Body.String())
		}
		if bytes.Contains(recorder.Body.Bytes(), []byte(code)) {
			t.Fatalf("response leaked supplied device code %q: %s", code, recorder.Body.String())
		}
	}
}

func jsonRequest(t *testing.T, path string, value any) *http.Request {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func formRequest(path, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func decodeRecorderJSON(t *testing.T, recorder *httptest.ResponseRecorder, out any) {
	t.Helper()
	if err := json.NewDecoder(recorder.Result().Body).Decode(out); err != nil {
		t.Fatalf("Decode JSON: %v body=%s", err, recorder.Body.String())
	}
}
