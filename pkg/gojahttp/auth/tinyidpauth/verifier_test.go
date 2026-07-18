package tinyidpauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

type resolverFunc func(context.Context, string, string) (*gojahttp.Actor, error)

func (f resolverFunc) ByExternalIdentity(c context.Context, i, s string) (*gojahttp.Actor, error) {
	return f(c, i, s)
}
func TestVerifierAcceptsOnlyVerifiedRedactedAssertions(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issuer":"` + server.URL + `","introspection_endpoint":"` + server.URL + `/introspect","introspection_endpoint_auth_methods_supported":["client_secret_basic"]}`))
		case "/introspect":
			user, pass, ok := r.BasicAuth()
			if !ok || user != "resource" || pass != "secret" {
				http.Error(w, "no", http.StatusUnauthorized)
				return
			}
			if err := r.ParseForm(); err != nil || r.Form.Get("token") != "raw-secret-token" {
				http.Error(w, "no", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"active":true,"iss":"` + server.URL + `","sub":"subject-1","client_id":"cli","token_type":"Bearer","scope":"inbox.read","aud":["inbox"],"exp":` + strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10) + `}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	v, err := New(context.Background(), Config{Issuer: server.URL, ClientID: "resource", ClientSecret: "secret", HTTPClient: server.Client(), Resolver: resolverFunc(func(context.Context, string, string) (*gojahttp.Actor, error) {
		return &gojahttp.Actor{ID: "u1", Kind: "user"}, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	result, err := v.AuthenticateOAuthBearer(context.Background(), "raw-secret-token", gojahttp.OAuthRequirement{Issuer: server.URL, Resource: "inbox", Scopes: []string{"inbox.read"}})
	if err != nil || result.OAuth == nil || result.OAuth.Subject != "subject-1" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if strings.Contains(fmt.Sprintf("%#v", result), "raw-secret-token") {
		t.Fatal("raw token leaked")
	}
}
func TestVerifierClassifiesProviderFailureUnavailable(t *testing.T) {
	v := &Verifier{issuer: "https://issuer", endpoint: "http://127.0.0.1:1", client: http.DefaultClient, timeout: time.Millisecond, resolver: resolverFunc(func(context.Context, string, string) (*gojahttp.Actor, error) { return nil, nil })}
	_, err := v.AuthenticateOAuthBearer(context.Background(), "token", gojahttp.OAuthRequirement{Issuer: "https://issuer"})
	if !errors.Is(err, gojahttp.ErrAuthUnavailable) {
		t.Fatalf("err=%v", err)
	}
}
