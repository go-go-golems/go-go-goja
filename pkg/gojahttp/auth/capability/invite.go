package capability

import (
	"context"
	"time"
)

const PurposeOrgInviteAccept = "org.invite.accept"

// OrgInviteSpec describes a concrete organization invite capability flow.
type OrgInviteSpec struct {
	OrgID     string
	Email     string
	Role      string
	TTL       time.Duration
	ExpiresAt time.Time
	CreatedBy string
}

// AcceptedOrgInvite is the redeemed invite payload.
type AcceptedOrgInvite struct {
	CapabilityID string
	OrgID        string
	Email        string
	Role         string
}

// IssueOrgInvite issues a single-use organization invite capability.
func (s Service) IssueOrgInvite(ctx context.Context, spec OrgInviteSpec) (IssueResult, error) {
	return s.Issue(ctx, IssueSpec{
		Purpose:      PurposeOrgInviteAccept,
		ResourceType: "org",
		ResourceID:   spec.OrgID,
		Claims:       map[string]string{"email": spec.Email, "role": spec.Role},
		TTL:          spec.TTL,
		ExpiresAt:    spec.ExpiresAt,
		SingleUse:    true,
		CreatedBy:    spec.CreatedBy,
	})
}

// AcceptOrgInvite redeems an organization invite capability and returns the
// application payload needed to create a membership.
func (s Service) AcceptOrgInvite(ctx context.Context, token string) (AcceptedOrgInvite, error) {
	capability, err := s.Redeem(ctx, PurposeOrgInviteAccept, token)
	if err != nil {
		return AcceptedOrgInvite{}, err
	}
	return AcceptedOrgInvite{CapabilityID: capability.ID, OrgID: capability.ResourceID, Email: capability.Claims["email"], Role: capability.Claims["role"]}, nil
}
