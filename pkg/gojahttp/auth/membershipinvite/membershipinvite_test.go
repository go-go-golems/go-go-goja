package membershipinvite_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/membershipinvite"
)

type acceptingStore struct{}

func (acceptingStore) Begin(context.Context, string, time.Time) (membershipinvite.Pending, error) {
	return membershipinvite.Pending{}, nil
}

func (acceptingStore) Accept(context.Context, string, string, time.Time) (membershipinvite.Result, error) {
	return membershipinvite.Result{}, nil
}

func (acceptingStore) AcceptPending(context.Context, string, string, time.Time) (membershipinvite.Result, error) {
	return membershipinvite.Result{CapabilityID: "cap-1", UserID: "user-1", TenantID: "o1", Role: "viewer"}, nil
}

func TestSuccessfulAcceptanceAuditIsTenantQueryable(t *testing.T) {
	t.Parallel()

	sink := &audit.MemorySink{}
	service := membershipinvite.Service{Acceptor: acceptingStore{}, Audit: sink}
	if _, err := service.AcceptPending(context.Background(), "pending", "user-1"); err != nil {
		t.Fatalf("AcceptPending: %v", err)
	}
	records := sink.Snapshot()
	if len(records) != 1 {
		t.Fatalf("audit record count = %d, want 1", len(records))
	}
	if records[0].Event != "org.invite.accepted" || records[0].TenantID != "o1" || records[0].ResourceID != "o1" {
		t.Fatalf("acceptance audit = %#v", records[0])
	}
}
