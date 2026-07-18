package hostauth

import (
	"context"
	"fmt"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/programauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/tinyidpauth"
)

type oauthVerifierSet map[string]*tinyidpauth.Verifier

func (s oauthVerifierSet) AuthenticateOAuthBearer(ctx context.Context, raw string, requirement gojahttp.OAuthRequirement) (gojahttp.AuthResult, error) {
	verifier := s[requirement.Issuer]
	if verifier == nil {
		return gojahttp.AuthResult{}, fmt.Errorf("%w: no oauth verifier profile", gojahttp.ErrUnauthenticated)
	}
	return verifier.AuthenticateOAuthBearer(ctx, raw, requirement)
}

var _ programauth.OAuthBearerAuthenticator = oauthVerifierSet{}

func buildOAuthVerifierSet(ctx context.Context, cfg []ResolvedOAuthResourceConfig, resolver tinyidpauth.IdentityResolver) (programauth.OAuthBearerAuthenticator, error) {
	if len(cfg) == 0 {
		return nil, nil
	}
	out := oauthVerifierSet{}
	for _, profile := range cfg {
		verifier, err := tinyidpauth.New(ctx, tinyidpauth.Config{Issuer: profile.IssuerURL, ClientID: profile.ClientID, ClientSecret: profile.ClientSecret, Resolver: resolver})
		if err != nil {
			return nil, fmt.Errorf("oauth profile %s: %w", profile.IssuerURL, err)
		}
		out[profile.IssuerURL] = verifier
	}
	return out, nil
}
