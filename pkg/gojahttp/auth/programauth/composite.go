package programauth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

type BearerAuthenticator interface {
	AuthenticateBearer(ctx context.Context, raw string, spec gojahttp.SecuritySpec) (gojahttp.AuthResult, error)
}

// CompositeAuthenticator selects Authorization-header bearer credentials first
// and falls back to the configured session authenticator when no bearer token is
// present. It intentionally does not accept query/body bearer tokens.
type CompositeAuthenticator struct {
	Session      gojahttp.Authenticator
	APITokens    BearerAuthenticator
	AccessTokens BearerAuthenticator
	// OAuthTokens is consulted exclusively for routes that declare an OAuth requirement.
	OAuthTokens OAuthBearerAuthenticator
}

func (a CompositeAuthenticator) Authenticate(ctx context.Context, req *http.Request, session *gojahttp.SessionDTO, spec gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
	result, err := a.AuthenticateResult(ctx, req, session, spec)
	if err != nil {
		return nil, err
	}
	return result.Actor, nil
}

func (a CompositeAuthenticator) AuthenticateResult(ctx context.Context, req *http.Request, session *gojahttp.SessionDTO, spec gojahttp.SecuritySpec) (gojahttp.AuthResult, error) {
	raw, ok, err := BearerFromHeader(req)
	if err != nil {
		return gojahttp.AuthResult{}, err
	}
	if requirement, oauthRoute := oauthRequirement(spec); oauthRoute {
		if !ok {
			return gojahttp.AuthResult{}, gojahttp.ErrUnauthenticated
		}
		return authenticateOAuthBearer(ctx, a.OAuthTokens, raw, *requirement)
	}
	if ok {
		return a.authenticateBearer(ctx, raw, spec)
	}
	if a.Session == nil {
		return gojahttp.AuthResult{}, gojahttp.ErrUnauthenticated
	}
	if resultAuthenticator, ok := a.Session.(gojahttp.ResultAuthenticator); ok {
		return resultAuthenticator.AuthenticateResult(ctx, req, session, spec)
	}
	actor, err := a.Session.Authenticate(ctx, req, session, spec)
	if err != nil {
		return gojahttp.AuthResult{}, err
	}
	if actor == nil {
		return gojahttp.AuthResult{}, gojahttp.ErrUnauthenticated
	}
	return gojahttp.AuthResult{Actor: actor, Method: gojahttp.AuthMethodSession, PrincipalKind: gojahttp.PrincipalKindUser, PrincipalID: actor.ID, CSRFRequired: true}, nil
}

func (a CompositeAuthenticator) authenticateBearer(ctx context.Context, raw string, spec gojahttp.SecuritySpec) (gojahttp.AuthResult, error) {
	var lastErr error
	configured := false
	authenticators := []BearerAuthenticator{a.APITokens, a.AccessTokens}
	if strings.HasPrefix(strings.TrimSpace(raw), defaultAccessTokenPrefix+"_") {
		authenticators = []BearerAuthenticator{a.AccessTokens, a.APITokens}
	}
	for _, authenticator := range authenticators {
		if authenticator == nil {
			continue
		}
		configured = true
		result, err := authenticator.AuthenticateBearer(ctx, raw, spec)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	if !configured {
		return gojahttp.AuthResult{}, fmt.Errorf("%w: bearer authenticator is not configured", gojahttp.ErrUnauthenticated)
	}
	return gojahttp.AuthResult{}, lastErr
}
