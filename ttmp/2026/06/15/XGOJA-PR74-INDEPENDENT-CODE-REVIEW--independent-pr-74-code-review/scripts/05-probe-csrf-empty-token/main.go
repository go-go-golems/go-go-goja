package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
)

func main() {
	store := sessionauth.NewMemoryStore()
	manager, err := sessionauth.New(sessionauth.Config{Store: store, AllowInsecureHTTP: true})
	if err != nil {
		panic(err)
	}
	now := time.Now()
	if err := store.Create(context.Background(), sessionauth.Session{
		ID:                "AAAAAAAAAAAAAAAAAAAAAA",
		UserID:            "u1",
		CSRFToken:         "",
		CreatedAt:         now,
		IdleExpiresAt:     now.Add(time.Hour),
		AbsoluteExpiresAt: now.Add(time.Hour),
	}); err != nil {
		panic(err)
	}
	req := httptest.NewRequest("PATCH", "/mutate", nil)
	manager.SetCookie(httptest.NewRecorder(), "unused")
	cookie := &httpCookie{Name: "goja_app_session", Value: "AAAAAAAAAAAAAAAAAAAAAA"}
	req.AddCookie(cookie.Cookie())
	err = manager.VerifyCSRF(context.Background(), gojahttp.CSRFRequest{HTTPRequest: req, Actor: &gojahttp.Actor{ID: "u1"}})
	if err == nil {
		fmt.Println("VerifyCSRF accepted missing header when stored csrf token is empty")
	} else {
		fmt.Printf("VerifyCSRF rejected request: %v\n", err)
	}
}

type httpCookie struct{ Name, Value string }

func (c *httpCookie) Cookie() *http.Cookie { return &http.Cookie{Name: c.Name, Value: c.Value} }
