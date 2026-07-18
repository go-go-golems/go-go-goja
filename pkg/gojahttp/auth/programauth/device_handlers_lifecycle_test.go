package programauth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/programauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
)

func TestDeviceHandlersInspectAndDeny(t *testing.T) {
	now := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	service := newDeviceTestService(func() time.Time { return now })
	manager, err := sessionauth.New(sessionauth.Config{Store: sessionauth.NewMemoryStore(), AllowInsecureHTTP: true, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("sessionauth.New: %v", err)
	}
	handlers, err := programauth.NewDeviceHandlers(programauth.DeviceHandlersConfig{Service: service, SessionManager: manager})
	if err != nil {
		t.Fatalf("NewDeviceHandlers: %v", err)
	}

	started := httptest.NewRecorder()
	handlers.StartHandler().ServeHTTP(started, jsonRequest(t, "/auth/device/start", map[string]any{"clientName": "cli", "actions": []string{"report.read"}}))
	var response map[string]any
	decodeRecorderJSON(t, started, &response)
	userCode := response["user_code"].(string)
	deviceCode := response["device_code"].(string)

	session, err := manager.NewSession(context.Background(), "u1")
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	requestWithSession := func(path string) *http.Request {
		req := jsonRequest(t, path, map[string]any{"user_code": userCode})
		cookieWriter := httptest.NewRecorder()
		manager.SetCookie(cookieWriter, session.ID)
		for _, cookie := range cookieWriter.Result().Cookies() {
			req.AddCookie(cookie)
		}
		req.Header.Set(sessionauth.CSRFHeaderName, session.CSRFToken)
		return req
	}

	inspect := httptest.NewRecorder()
	handlers.RequestHandler().ServeHTTP(inspect, requestWithSession("/auth/device/request"))
	if inspect.Code != http.StatusOK {
		t.Fatalf("inspect=%d %s", inspect.Code, inspect.Body.String())
	}
	if body := inspect.Body.String(); strings.Contains(body, deviceCode) || strings.Contains(body, userCode) {
		t.Fatalf("inspection leaked code: %s", body)
	}

	deny := httptest.NewRecorder()
	handlers.DenyHandler().ServeHTTP(deny, requestWithSession("/auth/device/deny"))
	if deny.Code != http.StatusOK {
		t.Fatalf("deny=%d %s", deny.Code, deny.Body.String())
	}

	poll := httptest.NewRecorder()
	handlers.TokenHandler().ServeHTTP(poll, jsonRequest(t, "/auth/device/token", map[string]any{"device_code": deviceCode}))
	if poll.Code != http.StatusBadRequest || !strings.Contains(poll.Body.String(), "access_denied") {
		t.Fatalf("poll=%d %s", poll.Code, poll.Body.String())
	}
}
