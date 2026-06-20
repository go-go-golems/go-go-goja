package programauth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

type BearerAuthenticator interface {
	AuthenticateBearer(ctx context.Context, raw string, spec gojahttp.SecuritySpec) (gojahttp.AuthResult, error)
}

// CompositeAuthenticator selects Authorization-header bearer credentials first
// and falls back to the configured session authenticator when no bearer token is
// present. It intentionally does not accept query/body bearer tokens.
type CompositeAuthenticator struct {
	Session   gojahttp.Authenticator
	APITokens BearerAuthenticator
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
	if ok {
		if a.APITokens == nil {
			return gojahttp.AuthResult{}, fmt.Errorf("%w: api token authenticator is not configured", gojahttp.ErrUnauthenticated)
		}
		return a.APITokens.AuthenticateBearer(ctx, raw, spec)
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
