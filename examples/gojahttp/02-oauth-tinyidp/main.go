package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/programauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/tinyidpauth"
)

type identityResolver struct{}

func (identityResolver) ByExternalIdentity(_ context.Context, issuer, subject string) (*gojahttp.Actor, error) {
	if issuer == "" || subject != "alice" {
		return nil, gojahttp.ErrNotFound
	}
	return &gojahttp.Actor{ID: "user:alice", Kind: "user"}, nil
}

type policy struct{}

func (policy) ResolveResource(_ context.Context, req gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error) {
	return &gojahttp.ResourceRef{Name: req.Spec.Name, Type: req.Spec.Type, ID: req.ID}, nil
}
func (policy) Authorize(_ context.Context, req gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
	return gojahttp.AuthorizationDecision{Allowed: req.Actor != nil && req.Actor.ID == "user:alice" && req.Action == "inbox.read"}, nil
}

func newHandler(verifier programauth.OAuthBearerAuthenticator, issuer string) (http.Handler, error) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Auth: gojahttp.AuthOptions{Authenticator: programauth.CompositeAuthenticator{OAuthTokens: verifier}, Resources: policy{}, Authorizer: policy{}}})
	app := gojahttp.NewApp(host)
	if err := app.Get("/oauth/inbox").Auth(gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser, Required: true, AuthRequirements: []gojahttp.AuthRequirement{{Method: gojahttp.AuthMethodAccessToken, OAuth: &gojahttp.OAuthRequirement{Issuer: issuer, Resource: "inbox", Scopes: []string{"inbox.read"}}}}}).Audit("oauth.inbox").Allow("inbox.read").HandleJSON(func(_ context.Context, sec *gojahttp.SecureContext) (any, error) {
		return map[string]any{"actor": sec.Actor.ID, "oauthSubject": sec.Auth.OAuth.Subject}, nil
	}); err != nil {
		return nil, err
	}
	return host, nil
}

func runSmoke() error {
	var idp *httptest.Server
	idp = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": idp.URL, "introspection_endpoint": idp.URL + "/introspect", "introspection_endpoint_auth_methods_supported": []string{"client_secret_basic"}})
		case "/introspect":
			user, pass, ok := r.BasicAuth()
			if !ok || user != "inbox-resource" || pass != "secret" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			_ = r.ParseForm()
			active := r.Form.Get("token") == "tinyidp-access-token"
			_ = json.NewEncoder(w).Encode(map[string]any{"active": active, "iss": idp.URL, "sub": "alice", "client_id": "agent-cli", "token_type": "Bearer", "scope": "inbox.read", "aud": "inbox", "exp": time.Now().Add(time.Hour).Unix()})
		default:
			http.NotFound(w, r)
		}
	}))
	defer idp.Close()
	verifier, err := tinyidpauth.New(context.Background(), tinyidpauth.Config{Issuer: idp.URL, ClientID: "inbox-resource", ClientSecret: "secret", HTTPClient: idp.Client(), Resolver: identityResolver{}})
	if err != nil {
		return err
	}
	h, err := newHandler(verifier, idp.URL)
	if err != nil {
		return err
	}
	server := httptest.NewServer(h)
	defer server.Close()
	for _, tc := range []struct {
		auth, body string
		code       int
	}{{"", ``, 401}, {"Bearer tinyidp-access-token", `"oauthSubject":"alice"`, 200}, {"Bearer wrong", ``, 401}} {
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/oauth/inbox", nil)
		if tc.auth != "" {
			req.Header.Set("Authorization", tc.auth)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		bodyBytes, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		body := string(bodyBytes)
		if res.StatusCode != tc.code || tc.body != "" && !strings.Contains(body, tc.body) {
			return fmt.Errorf("status=%d body=%s", res.StatusCode, body)
		}
	}
	return nil
}
func main() {
	if err := runSmoke(); err != nil {
		panic(err)
	}
	fmt.Println("tinyidp oauth smoke ok")
}
