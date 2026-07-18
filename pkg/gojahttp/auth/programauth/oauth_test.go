package programauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

type oauthAuthenticatorFunc func(context.Context, string, gojahttp.OAuthRequirement) (gojahttp.AuthResult, error)

func (f oauthAuthenticatorFunc) AuthenticateOAuthBearer(c context.Context, raw string, r gojahttp.OAuthRequirement) (gojahttp.AuthResult, error) {
	return f(c, raw, r)
}

func TestCompositeAuthenticatorUsesOAuthOnlyForOAuthRoute(t *testing.T) {
	called := false
	a := CompositeAuthenticator{OAuthTokens: oauthAuthenticatorFunc(func(_ context.Context, raw string, requirement gojahttp.OAuthRequirement) (gojahttp.AuthResult, error) {
		called = raw == "external-token" && requirement.Issuer == "https://idp.example"
		return gojahttp.AuthResult{Actor: &gojahttp.Actor{ID: "u1", Kind: "user"}, OAuth: &gojahttp.OAuthAuthContext{Issuer: requirement.Issuer, Subject: "s1", Resources: []string{requirement.Resource}, Scopes: requirement.Scopes}}, nil
	})}
	spec := gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser, Required: true, AuthRequirements: []gojahttp.AuthRequirement{{Method: gojahttp.AuthMethodAccessToken, OAuth: &gojahttp.OAuthRequirement{Issuer: "https://idp.example", Resource: "inbox", Scopes: []string{"inbox.read"}}}}}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer external-token")
	result, err := a.AuthenticateResult(context.Background(), req, nil, spec)
	if err != nil || !called || result.Method != gojahttp.AuthMethodAccessToken || result.OAuth == nil {
		t.Fatalf("result=%#v err=%v called=%v", result, err, called)
	}
}

func TestCompositeAuthenticatorRejectsSessionForOAuthRoute(t *testing.T) {
	a := CompositeAuthenticator{}
	spec := gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser, Required: true, AuthRequirements: []gojahttp.AuthRequirement{{Method: gojahttp.AuthMethodAccessToken, OAuth: &gojahttp.OAuthRequirement{Issuer: "https://idp.example", Resource: "inbox", Scopes: []string{"inbox.read"}}}}}
	_, err := a.AuthenticateResult(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil), nil, spec)
	if err == nil {
		t.Fatal("expected oauth route without bearer to be rejected")
	}
}
