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
