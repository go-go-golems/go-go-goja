package adminbootstrap_test

import (
	"context"
	"database/sql"
	stderrors "errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth/adminbootstrap"
	appsql "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth/sqlstore"
	auditsql "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit/sqlstore"
)

func TestBootstrapAdminCreatesAndReconcilesDesiredState(t *testing.T) {
	db, reconciler := fixture(t)
	request := validRequest()
	first, err := reconciler.BootstrapAdmin(context.Background(), request)
	if err != nil {
		t.Fatalf("first bootstrap: %v", err)
	}
	wantID, _ := appauth.OIDCUserID(request.Issuer, request.Subject)
	if first.UserID != wantID || first.OrganizationID != "o1" || first.Role != "admin" {
		t.Fatalf("unexpected result: %#v", first)
	}
	if _, err := db.Exec(`UPDATE auth_app_users SET disabled_at = CURRENT_TIMESTAMP; UPDATE auth_app_memberships SET revoked_at = CURRENT_TIMESTAMP`); err != nil {
		t.Fatalf("disable fixture: %v", err)
	}
	request.Email = "updated@example.test"
	request.DisplayName = "Updated Operator"
	request.OrganizationName = "Updated Organization"
	if _, err := reconciler.BootstrapAdmin(context.Background(), request); err != nil {
		t.Fatalf("repeat bootstrap: %v", err)
	}

	assertCount(t, db, `SELECT COUNT(*) FROM auth_app_users WHERE id = ? AND email = ? AND display_name = ? AND email_verified = 1 AND disabled_at IS NULL`, 1, wantID, request.Email, request.DisplayName)
	assertCount(t, db, `SELECT COUNT(*) FROM auth_app_external_identities WHERE issuer = ? AND subject = ? AND user_id = ?`, 1, request.Issuer, request.Subject, wantID)
	assertCount(t, db, `SELECT COUNT(*) FROM auth_app_tenants WHERE id = 'o1' AND slug = 'local-demo' AND name = 'Updated Organization' AND disabled_at IS NULL`, 1)
	assertCount(t, db, `SELECT COUNT(*) FROM auth_app_resources WHERE type = 'org' AND id = 'o1' AND tenant_id = 'o1' AND name = 'Updated Organization'`, 1)
	assertCount(t, db, `SELECT COUNT(*) FROM auth_app_memberships WHERE user_id = ? AND tenant_id = 'o1' AND role = 'admin' AND revoked_at IS NULL`, 1, wantID)
	assertCount(t, db, `SELECT COUNT(*) FROM auth_audit_records WHERE event = 'operator.bootstrap_admin' AND outcome = 'success' AND actor_id = 'deployment-operator' AND tenant_id = 'o1'`, 2)
}

func TestBootstrapAdminRejectsConflictsWithoutPartialMutation(t *testing.T) {
	db, reconciler := fixture(t)
	request := validRequest()
	wantID, _ := appauth.OIDCUserID(request.Issuer, request.Subject)
	if _, err := db.Exec(`INSERT INTO auth_app_users (id, email) VALUES ('other', 'other@example.test'); INSERT INTO auth_app_external_identities (issuer, subject, user_id) VALUES (?, ?, 'other')`, request.Issuer, request.Subject); err != nil {
		t.Fatalf("seed conflict: %v", err)
	}
	_, err := reconciler.BootstrapAdmin(context.Background(), request)
	if !stderrors.Is(err, adminbootstrap.ErrConflict) {
		t.Fatalf("bootstrap error = %v, want conflict", err)
	}
	assertCount(t, db, `SELECT COUNT(*) FROM auth_app_users WHERE id = ?`, 0, wantID)
	assertCount(t, db, `SELECT COUNT(*) FROM auth_app_tenants`, 0)
	assertCount(t, db, `SELECT COUNT(*) FROM auth_audit_records`, 0)
}

func TestBootstrapAdminRejectsNormalizedIdentityOwnedByDifferentUser(t *testing.T) {
	db, reconciler := fixture(t)
	request := validRequest()
	if _, err := db.Exec(`INSERT INTO auth_app_users (id, oidc_issuer, oidc_subject, email) VALUES ('legacy-user', ?, ?, 'legacy@example.test')`, request.Issuer, request.Subject); err != nil {
		t.Fatalf("seed normalized identity: %v", err)
	}
	_, err := reconciler.BootstrapAdmin(context.Background(), request)
	if !stderrors.Is(err, adminbootstrap.ErrConflict) {
		t.Fatalf("bootstrap error = %v, want conflict", err)
	}
	assertCount(t, db, `SELECT COUNT(*) FROM auth_app_tenants`, 0)
}

func TestBootstrapAdminRejectsOrganizationOwnershipConflict(t *testing.T) {
	db, reconciler := fixture(t)
	if _, err := db.Exec(`INSERT INTO auth_app_resources (type, id, name, tenant_id) VALUES ('org', 'o1', 'Wrong', 'other')`); err != nil {
		t.Fatalf("seed resource: %v", err)
	}
	_, err := reconciler.BootstrapAdmin(context.Background(), validRequest())
	if !stderrors.Is(err, adminbootstrap.ErrConflict) {
		t.Fatalf("bootstrap error = %v, want conflict", err)
	}
	assertCount(t, db, `SELECT COUNT(*) FROM auth_app_memberships`, 0)
}

func TestBootstrapAdminValidatesBeforeMutation(t *testing.T) {
	db, reconciler := fixture(t)
	request := validRequest()
	request.Issuer = "http://idp.example.test"
	if _, err := reconciler.BootstrapAdmin(context.Background(), request); err == nil {
		t.Fatal("expected insecure issuer error")
	}
	assertCount(t, db, `SELECT COUNT(*) FROM auth_app_users`, 0)
}

func fixture(t *testing.T) (*sql.DB, *adminbootstrap.Reconciler) {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	appStore, _ := appsql.New(appsql.Config{DB: db, Dialect: appsql.DialectSQLite})
	if err := appStore.ApplySchema(context.Background()); err != nil {
		t.Fatalf("apply app schema: %v", err)
	}
	auditStore, _ := auditsql.New(auditsql.Config{DB: db, Dialect: auditsql.DialectSQLite})
	if err := auditStore.ApplySchema(context.Background()); err != nil {
		t.Fatalf("apply audit schema: %v", err)
	}
	reconciler, err := adminbootstrap.New(db, adminbootstrap.DialectSQLite, func() time.Time { return time.Date(2026, 7, 21, 20, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatalf("new reconciler: %v", err)
	}
	return db, reconciler
}

func validRequest() adminbootstrap.Request {
	return adminbootstrap.Request{
		Issuer: "https://idp.example.test", Subject: "operator-subject", Email: "operator@example.test",
		DisplayName: "Deployment Operator", OrganizationID: "o1", OrganizationSlug: "local-demo", OrganizationName: "Local Demo Organization",
	}
}

func assertCount(t *testing.T, db *sql.DB, query string, want int, args ...any) {
	t.Helper()
	var got int
	if err := db.QueryRow(query, args...).Scan(&got); err != nil {
		t.Fatalf("query count: %v", err)
	}
	if got != want {
		t.Fatalf("count = %d, want %d for %s", got, want, query)
	}
}
