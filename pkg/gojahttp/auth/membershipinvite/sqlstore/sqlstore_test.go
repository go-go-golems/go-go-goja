package sqlstore_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
	appauthsql "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth/sqlstore"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability"
	capabilitysql "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability/sqlstore"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/membershipinvite"
	invitesql "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/membershipinvite/sqlstore"
)

func TestAcceptAtomicallyBindsVerifiedActorAndConsumesInvite(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	db, apps, capabilities, acceptor := fixture(t)
	seed(t, apps, appauth.User{ID: "u1", Email: "User@Example.com", EmailVerified: true})
	issued := issue(t, capabilities, now, "user@example.com", "member")

	result, err := acceptor.Accept(ctx, issued.Token, "u1", now)
	if err != nil {
		t.Fatalf("Accept: %v", err)
	}
	if result.UserID != "u1" || result.TenantID != "o1" || result.Role != "member" {
		t.Fatalf("result = %#v", result)
	}
	assertMembershipCount(t, db, 1)
	if _, err := acceptor.Accept(ctx, issued.Token, "u1", now); !errors.Is(err, capability.ErrUsed) {
		t.Fatalf("replay error = %v, want ErrUsed", err)
	}
	assertMembershipCount(t, db, 1)
}

func TestAcceptRejectsIdentityAndRoleViolationsWithoutMutation(t *testing.T) {
	cases := []struct {
		name, email, inviteEmail, role string
		verified                       bool
		want                           error
	}{
		{name: "unverified", email: "u@example.com", inviteEmail: "u@example.com", role: "member", want: membershipinvite.ErrEmailUnverified},
		{name: "different email", email: "other@example.com", inviteEmail: "u@example.com", role: "member", verified: true, want: membershipinvite.ErrEmailMismatch},
		{name: "missing identity binding", email: "u@example.com", inviteEmail: "", role: "member", verified: true, want: membershipinvite.ErrIdentityBinding},
		{name: "unrecognized role", email: "u@example.com", inviteEmail: "u@example.com", role: "owner", verified: true, want: membershipinvite.ErrRoleNotAllowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
			db, apps, capabilities, acceptor := fixture(t)
			seed(t, apps, appauth.User{ID: "u1", Email: tc.email, EmailVerified: tc.verified})
			issued := issue(t, capabilities, now, tc.inviteEmail, tc.role)
			_, err := acceptor.Accept(ctx, issued.Token, "u1", now)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
			assertMembershipCount(t, db, 0)
			var used int
			if err := db.QueryRow(`SELECT COUNT(*) FROM auth_capabilities WHERE used_at IS NOT NULL`).Scan(&used); err != nil {
				t.Fatal(err)
			}
			if used != 0 {
				t.Fatalf("used invite count = %d, want 0", used)
			}
		})
	}
}

func TestAcceptSupportsApplicationSubjectBinding(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	db, apps, capabilities, acceptor := fixture(t)
	seed(t, apps, appauth.User{ID: "u1", Email: "unverified@example.com"})
	capabilities.Now = func() time.Time { return now }
	issued, err := capabilities.Issue(ctx, capability.IssueSpec{
		Purpose: capability.PurposeOrgInviteAccept, SubjectID: "u1",
		ResourceType: "org", ResourceID: "o1", Claims: map[string]string{"role": "member"},
		TTL: time.Hour, SingleUse: true, CreatedBy: "admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acceptor.Accept(ctx, issued.Token, "u1", now); err != nil {
		t.Fatalf("Accept subject-bound invitation: %v", err)
	}
	assertMembershipCount(t, db, 1)
}

func TestAcceptRejectsDifferentApplicationSubject(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	db, apps, capabilities, acceptor := fixture(t)
	seed(t, apps, appauth.User{ID: "u2", Email: "unverified@example.com"})
	capabilities.Now = func() time.Time { return now }
	issued, err := capabilities.Issue(ctx, capability.IssueSpec{
		Purpose: capability.PurposeOrgInviteAccept, SubjectID: "u1",
		ResourceType: "org", ResourceID: "o1", Claims: map[string]string{"role": "member"},
		TTL: time.Hour, SingleUse: true, CreatedBy: "admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acceptor.Accept(ctx, issued.Token, "u2", now); !errors.Is(err, membershipinvite.ErrSubjectMismatch) {
		t.Fatalf("error = %v, want ErrSubjectMismatch", err)
	}
	assertMembershipCount(t, db, 0)
}

func TestAcceptRollsBackMembershipWhenCapabilityUseFails(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	db, apps, capabilities, acceptor := fixture(t)
	seed(t, apps, appauth.User{ID: "u1", Email: "u@example.com", EmailVerified: true})
	issued := issue(t, capabilities, now, "u@example.com", "member")
	if _, err := db.Exec(`CREATE TRIGGER reject_capability_use BEFORE UPDATE OF used_at ON auth_capabilities BEGIN SELECT RAISE(ABORT, 'injected failure'); END`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}
	if _, err := acceptor.Accept(ctx, issued.Token, "u1", now); err == nil {
		t.Fatal("Accept succeeded despite injected consume failure")
	}
	assertMembershipCount(t, db, 0)
}

func TestConcurrentAcceptanceHasOneWinner(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	_, apps, capabilities, acceptor := fixture(t)
	seed(t, apps, appauth.User{ID: "u1", Email: "u@example.com", EmailVerified: true})
	issued := issue(t, capabilities, now, "u@example.com", "member")
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := acceptor.Accept(ctx, issued.Token, "u1", now)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	successes := 0
	for err := range errs {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful accepts = %d, want 1", successes)
	}
}

func TestPendingHandleCarriesInviteWithoutPersistingRawToken(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	db, apps, capabilities, acceptor := fixture(t)
	seed(t, apps, appauth.User{ID: "u1", Email: "u@example.com", EmailVerified: true})
	issued := issue(t, capabilities, now, "u@example.com", "viewer")
	pending, err := acceptor.Begin(ctx, issued.Token, now)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if pending.Handle == "" || pending.Handle == issued.Token || pending.TenantID != "o1" {
		t.Fatalf("pending = %#v", pending)
	}
	var rawTokenRows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_pending_membership_invites WHERE CAST(handle_hash AS TEXT) = ?`, issued.Token).Scan(&rawTokenRows); err != nil {
		t.Fatal(err)
	}
	if rawTokenRows != 0 {
		t.Fatal("raw invitation token was persisted")
	}
	result, err := acceptor.AcceptPending(ctx, pending.Handle, "u1", now)
	if err != nil {
		t.Fatalf("AcceptPending: %v", err)
	}
	if result.Role != "viewer" {
		t.Fatalf("result = %#v", result)
	}
	if _, err := acceptor.AcceptPending(ctx, pending.Handle, "u1", now); !errors.Is(err, capability.ErrUsed) {
		t.Fatalf("pending replay error = %v, want ErrUsed", err)
	}
}

func TestPendingAcceptanceFailureRemainsRetryable(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	_, apps, capabilities, acceptor := fixture(t)
	seed(t, apps, appauth.User{ID: "u1", Email: "wrong@example.com", EmailVerified: true})
	issued := issue(t, capabilities, now, "right@example.com", "member")
	pending, err := acceptor.Begin(ctx, issued.Token, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acceptor.AcceptPending(ctx, pending.Handle, "u1", now); !errors.Is(err, membershipinvite.ErrEmailMismatch) {
		t.Fatalf("first accept: %v", err)
	}
	if err := apps.AddUser(ctx, appauth.User{ID: "u1", Email: "right@example.com", EmailVerified: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := acceptor.AcceptPending(ctx, pending.Handle, "u1", now); err != nil {
		t.Fatalf("retry after correction: %v", err)
	}
}

func fixture(t *testing.T) (*sql.DB, *appauthsql.Store, capability.Service, *invitesql.Store) {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "auth.db") + "?_busy_timeout=5000&_journal_mode=WAL"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(4)
	t.Cleanup(func() { _ = db.Close() })
	apps, err := appauthsql.New(appauthsql.Config{DB: db, Dialect: appauthsql.DialectSQLite})
	if err != nil {
		t.Fatal(err)
	}
	if err := apps.ApplySchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	caps, err := capabilitysql.New(capabilitysql.Config{DB: db, Dialect: capabilitysql.DialectSQLite})
	if err != nil {
		t.Fatal(err)
	}
	if err := caps.ApplySchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	acceptor, err := invitesql.New(invitesql.Config{DB: db, Dialect: invitesql.DialectSQLite})
	if err != nil {
		t.Fatal(err)
	}
	if err := acceptor.ApplySchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	return db, apps, capability.Service{Store: caps}, acceptor
}

func seed(t *testing.T, apps *appauthsql.Store, user appauth.User) {
	t.Helper()
	ctx := context.Background()
	if err := apps.AddUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	if err := apps.AddTenant(ctx, appauth.Tenant{ID: "o1", Slug: "example", Name: "Example"}); err != nil {
		t.Fatal(err)
	}
}

func issue(t *testing.T, service capability.Service, now time.Time, email, role string) capability.IssueResult {
	t.Helper()
	service.Now = func() time.Time { return now }
	issued, err := service.IssueOrgInvite(context.Background(), capability.OrgInviteSpec{OrgID: "o1", Email: email, Role: role, TTL: time.Hour, CreatedBy: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	return issued
}

func assertMembershipCount(t *testing.T, db *sql.DB, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_app_memberships`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("membership count = %d, want %d", got, want)
	}
}
