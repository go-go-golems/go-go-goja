package hostauth

import (
	"context"
	"fmt"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
)

// externalIdentityResolver adapts the application user store to OAuth verifier
// identity resolution. It deliberately resolves only existing enabled users.
type externalIdentityResolver struct {
	users       appauth.UserStore
	memberships appauth.MembershipStore
}

func (r externalIdentityResolver) ByExternalIdentity(ctx context.Context, issuer, subject string) (*gojahttp.Actor, error) {
	if r.users == nil {
		return nil, fmt.Errorf("external identity user store is required")
	}
	user, err := r.users.ByExternalIdentity(ctx, issuer, subject)
	if err != nil {
		return nil, err
	}
	actor := &gojahttp.Actor{ID: user.ID, Kind: "user"}
	if r.memberships != nil {
		memberships, err := r.memberships.MembershipsForUser(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		for _, m := range memberships {
			actor.TenantIDs = append(actor.TenantIDs, m.TenantID)
		}
	}
	return actor, nil
}
