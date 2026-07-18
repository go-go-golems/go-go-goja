// Package tinyidpauth verifies opaque tiny-idp access tokens through RFC 7662.
package tinyidpauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

type IdentityResolver interface {
	ByExternalIdentity(context.Context, string, string) (*gojahttp.Actor, error)
}
type Config struct {
	Issuer, ClientID, ClientSecret string
	HTTPClient                     *http.Client
	Timeout                        time.Duration
	Resolver                       IdentityResolver
	SecurityEvents                 gojahttp.SecurityEventObserver
}
type Verifier struct {
	issuer, endpoint, clientID, clientSecret string
	client                                   *http.Client
	timeout                                  time.Duration
	resolver                                 IdentityResolver
	securityEvents                           gojahttp.SecurityEventObserver
}
type discovery struct {
	Issuer                string   `json:"issuer"`
	IntrospectionEndpoint string   `json:"introspection_endpoint"`
	Methods               []string `json:"introspection_endpoint_auth_methods_supported"`
}
type response struct {
	Active    bool   `json:"active"`
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	ClientID  string `json:"client_id"`
	TokenType string `json:"token_type"`
	Scope     string `json:"scope"`
	Audience  any    `json:"aud"`
	ExpiresAt int64  `json:"exp"`
}

func New(ctx context.Context, cfg Config) (*Verifier, error) {
	cfg.Issuer = strings.TrimRight(strings.TrimSpace(cfg.Issuer), "/")
	if cfg.Issuer == "" || cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.Resolver == nil {
		return nil, fmt.Errorf("tinyidp verifier requires issuer, client id, client secret, and identity resolver")
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, cfg.Issuer+"/.well-known/openid-configuration", nil)
	if err != nil {
		return nil, err
	}
	r, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tinyidp discovery: %w", err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tinyidp discovery status %d", r.StatusCode)
	}
	var d discovery
	if err = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&d); err != nil {
		return nil, fmt.Errorf("tinyidp discovery decode: %w", err)
	}
	if strings.TrimRight(d.Issuer, "/") != cfg.Issuer {
		return nil, fmt.Errorf("tinyidp discovery issuer mismatch")
	}
	ep, err := url.Parse(d.IntrospectionEndpoint)
	if err != nil || ep.Scheme == "" || ep.Host == "" {
		return nil, fmt.Errorf("tinyidp discovery has invalid introspection endpoint")
	}
	issuerURL, _ := url.Parse(cfg.Issuer)
	if ep.Scheme != issuerURL.Scheme || ep.Host != issuerURL.Host {
		return nil, fmt.Errorf("tinyidp introspection endpoint must share issuer origin")
	}
	basic := false
	for _, m := range d.Methods {
		if m == "client_secret_basic" {
			basic = true
		}
	}
	if !basic {
		return nil, fmt.Errorf("tinyidp discovery does not support client_secret_basic")
	}
	return &Verifier{issuer: cfg.Issuer, endpoint: d.IntrospectionEndpoint, clientID: cfg.ClientID, clientSecret: cfg.ClientSecret, client: cfg.HTTPClient, timeout: cfg.Timeout, resolver: cfg.Resolver, securityEvents: cfg.SecurityEvents}, nil
}
func (v *Verifier) observe(ctx context.Context, outcome, reason string) {
	if v != nil && v.securityEvents != nil {
		v.securityEvents.ObserveSecurityEvent(ctx, gojahttp.SecurityEvent{Name: "oauth.introspection", Outcome: outcome, Reason: reason})
	}
}

func (v *Verifier) AuthenticateOAuthBearer(ctx context.Context, raw string, need gojahttp.OAuthRequirement) (gojahttp.AuthResult, error) {
	if v == nil || need.Issuer != v.issuer {
		return gojahttp.AuthResult{}, gojahttp.ErrUnauthenticated
	}
	cctx, cancel := context.WithTimeout(ctx, v.timeout)
	defer cancel()
	form := url.Values{"token": {raw}}.Encode()
	req, err := http.NewRequestWithContext(cctx, http.MethodPost, v.endpoint, strings.NewReader(form))
	if err != nil {
		return gojahttp.AuthResult{}, fmt.Errorf("oauth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(v.clientID, v.clientSecret)
	res, err := v.client.Do(req)
	if err != nil {
		v.observe(ctx, "unavailable", "transport")
		return gojahttp.AuthResult{}, fmt.Errorf("%w: introspection request", gojahttp.ErrAuthUnavailable)
	}
	defer res.Body.Close()
	if res.StatusCode >= 500 {
		v.observe(ctx, "unavailable", "provider_status")
		return gojahttp.AuthResult{}, fmt.Errorf("%w: introspection status", gojahttp.ErrAuthUnavailable)
	}
	if res.StatusCode != http.StatusOK {
		v.observe(ctx, "inactive", "provider_rejected")
		return gojahttp.AuthResult{}, gojahttp.ErrUnauthenticated
	}
	var out response
	if err = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&out); err != nil {
		v.observe(ctx, "unavailable", "malformed_response")
		return gojahttp.AuthResult{}, fmt.Errorf("%w: introspection response", gojahttp.ErrAuthUnavailable)
	}
	scopes := strings.Fields(out.Scope)
	resources := audiences(out.Audience)
	if !out.Active || out.Issuer != v.issuer || !strings.EqualFold(out.TokenType, "Bearer") || out.Subject == "" || time.Unix(out.ExpiresAt, 0).Before(time.Now()) || !contains(resources, need.Resource) || !all(scopes, need.Scopes) {
		v.observe(ctx, "inactive", "assertion_failed")
		return gojahttp.AuthResult{}, gojahttp.ErrUnauthenticated
	}
	actor, err := v.resolver.ByExternalIdentity(ctx, v.issuer, out.Subject)
	if err != nil || actor == nil {
		v.observe(ctx, "rejected", "identity_unmapped")
		return gojahttp.AuthResult{}, gojahttp.ErrUnauthenticated
	}
	v.observe(ctx, "accepted", "")
	return gojahttp.AuthResult{Actor: actor, Method: gojahttp.AuthMethodAccessToken, PrincipalKind: gojahttp.PrincipalKindUser, PrincipalID: actor.ID, OAuth: &gojahttp.OAuthAuthContext{Issuer: v.issuer, Subject: out.Subject, ClientID: out.ClientID, Resources: resources, Scopes: scopes, ExpiresAt: time.Unix(out.ExpiresAt, 0), TokenType: out.TokenType}}, nil
}
func audiences(a any) []string {
	switch x := a.(type) {
	case string:
		return []string{x}
	case []any:
		out := []string{}
		for _, v := range x {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
func contains(v []string, s string) bool {
	for _, x := range v {
		if x == s {
			return true
		}
	}
	return false
}
func all(v, w []string) bool {
	for _, s := range w {
		if !contains(v, s) {
			return false
		}
	}
	return true
}
