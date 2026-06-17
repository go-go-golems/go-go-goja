package capability

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
)

func TestIssueRedeemSingleUseAndAudit(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 12, 18, 30, 0, 0, time.UTC)
	sink := &audit.MemorySink{Normalizer: audit.Normalizer{Now: func() time.Time { return now }}}
	service := Service{Store: NewMemoryStore(), Audit: sink, Now: func() time.Time { return now }}
	issued, err := service.Issue(ctx, IssueSpec{Purpose: "org.invite.accept", ResourceType: "org", ResourceID: "o1", Claims: map[string]string{"email": "new@example.test", "role": "viewer"}, TTL: time.Hour, SingleUse: true, CreatedBy: "u1"})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if issued.Token == "" || len(issued.Capability.TokenHash) != 0 {
		t.Fatalf("expected raw token once and redacted returned capability: %#v", issued)
	}
	stored, err := service.Store.ByID(ctx, issued.Capability.ID)
	if err != nil {
		t.Fatalf("stored by id: %v", err)
	}
	if len(stored.TokenHash) == 0 || stored.Claims["role"] != "viewer" {
		t.Fatalf("stored capability missing token hash/claims: %#v", stored)
	}
	validated, err := service.Validate(ctx, "org.invite.accept", issued.Token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if validated.ID != issued.Capability.ID || len(validated.TokenHash) != 0 || validated.UsedAt != nil {
		t.Fatalf("unexpected validated capability: %#v", validated)
	}
	redeemed, err := service.Consume(ctx, "org.invite.accept", issued.Token)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if redeemed.ID != issued.Capability.ID || len(redeemed.TokenHash) != 0 || redeemed.UsedAt == nil {
		t.Fatalf("unexpected consumed capability: %#v", redeemed)
	}
	_, err = service.Redeem(ctx, "org.invite.accept", issued.Token)
	if !errors.Is(err, ErrUsed) {
		t.Fatalf("second redeem err=%v", err)
	}
	auditJSON, err := json.Marshal(sink.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(auditJSON), issued.Token) {
		t.Fatalf("audit leaked raw token: %s", string(auditJSON))
	}
	if !strings.Contains(string(auditJSON), "capability.issued") || !strings.Contains(string(auditJSON), "capability.validated") || !strings.Contains(string(auditJSON), "capability.consumed") {
		t.Fatalf("expected issue/validate/consume audit events: %s", string(auditJSON))
	}
}

func TestRedeemFailures(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 12, 18, 30, 0, 0, time.UTC)
	service := Service{Store: NewMemoryStore(), Now: func() time.Time { return now }}
	issued, err := service.Issue(ctx, IssueSpec{Purpose: "email.verify", SubjectID: "u1", TTL: time.Minute, SingleUse: true})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	_, err = service.Redeem(ctx, "password.reset", issued.Token)
	if !errors.Is(err, ErrWrongPurpose) {
		t.Fatalf("wrong purpose err=%v", err)
	}
	_, err = service.Redeem(ctx, "email.verify", "not-the-token")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("not found err=%v", err)
	}
	now = now.Add(2 * time.Minute)
	_, err = service.Redeem(ctx, "email.verify", issued.Token)
	if !errors.Is(err, ErrExpired) {
		t.Fatalf("expired err=%v", err)
	}
}

func TestRevoke(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 12, 18, 30, 0, 0, time.UTC)
	sink := &audit.MemorySink{}
	service := Service{Store: NewMemoryStore(), Audit: sink, Now: func() time.Time { return now }}
	issued, err := service.Issue(ctx, IssueSpec{Purpose: "api.token", SubjectID: "u1", TTL: time.Hour})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if err := service.Revoke(ctx, issued.Capability.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	_, err = service.Redeem(ctx, "api.token", issued.Token)
	if !errors.Is(err, ErrRevoked) {
		t.Fatalf("revoked err=%v", err)
	}
	if len(sink.Snapshot()) < 2 {
		t.Fatalf("expected audit events")
	}
}

func TestOrgInviteFlow(t *testing.T) {
	ctx := context.Background()
	service := Service{Store: NewMemoryStore()}
	issued, err := service.IssueOrgInvite(ctx, OrgInviteSpec{OrgID: "o1", Email: "new@example.test", Role: "viewer", TTL: time.Hour, CreatedBy: "u1"})
	if err != nil {
		t.Fatalf("issue org invite: %v", err)
	}
	accepted, err := service.AcceptOrgInvite(ctx, issued.Token)
	if err != nil {
		t.Fatalf("accept invite: %v", err)
	}
	if accepted.OrgID != "o1" || accepted.Email != "new@example.test" || accepted.Role != "viewer" {
		t.Fatalf("unexpected accepted invite: %#v", accepted)
	}
	_, err = service.AcceptOrgInvite(ctx, issued.Token)
	if !errors.Is(err, ErrUsed) {
		t.Fatalf("expected invite token single-use, got %v", err)
	}
}

func TestIssueValidation(t *testing.T) {
	service := Service{Store: NewMemoryStore()}
	if _, err := service.Issue(context.Background(), IssueSpec{SubjectID: "u1", TTL: time.Hour}); err == nil {
		t.Fatalf("expected missing purpose error")
	}
	if _, err := service.Issue(context.Background(), IssueSpec{Purpose: "p", TTL: time.Hour}); err == nil {
		t.Fatalf("expected missing subject/resource error")
	}
	if _, err := service.Issue(context.Background(), IssueSpec{Purpose: "p", SubjectID: "u1"}); err == nil {
		t.Fatalf("expected missing expiry error")
	}
}
