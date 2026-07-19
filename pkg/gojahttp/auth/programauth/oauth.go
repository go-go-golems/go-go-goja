package programauth

import (
	"context"
	"fmt"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

// OAuthBearerAuthenticator validates bearer tokens issued by an external OAuth
// authorization server. It must return only verified, redacted assertions in
// AuthResult.OAuth; implementations must never retain the raw bearer token.
type OAuthBearerAuthenticator interface {
	AuthenticateOAuthBearer(ctx context.Context, raw string, requirement gojahttp.OAuthRequirement) (gojahttp.AuthResult, error)
}

func oauthRequirement(spec gojahttp.SecuritySpec) (*gojahttp.OAuthRequirement, bool) {
	for _, requirement := range spec.AuthRequirements {
		if requirement.OAuth != nil {
			return requirement.OAuth, true
		}
	}
	return nil, false
}

func authenticateOAuthBearer(ctx context.Context, authenticator OAuthBearerAuthenticator, raw string, requirement gojahttp.OAuthRequirement) (gojahttp.AuthResult, error) {
	if authenticator == nil {
		return gojahttp.AuthResult{}, fmt.Errorf("%w: oauth bearer authenticator is not configured", gojahttp.ErrUnauthenticated)
	}
	result, err := authenticator.AuthenticateOAuthBearer(ctx, raw, requirement)
	if err != nil {
		return gojahttp.AuthResult{}, err
	}
	if result.Actor == nil || result.OAuth == nil {
		return gojahttp.AuthResult{}, fmt.Errorf("%w: oauth bearer authenticator returned incomplete verified identity", gojahttp.ErrUnauthenticated)
	}
	result.Method = gojahttp.AuthMethodAccessToken
	result.CSRFRequired = false
	return result, nil
}
