// Package adminbootstrap reconciles the first application administrator in one
// SQL transaction. It is an offline operator primitive, not a request-path API.
package adminbootstrap

import (
	"context"
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"net/url"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
	"github.com/pkg/errors"
)

type Dialect string

const (
	DialectSQLite   Dialect = "sqlite"
	DialectPostgres Dialect = "postgres"
	AdminRole               = "admin"
)

var ErrConflict = stderrors.New("administrator bootstrap conflict")

type Request struct {
	Issuer           string
	Subject          string
	Email            string
	DisplayName      string
	OrganizationID   string
	OrganizationSlug string
	OrganizationName string
	OperatorID       string
}

type Result struct {
	UserID         string `json:"userId"`
	OrganizationID string `json:"organizationId"`
	Role           string `json:"role"`
}

type Reconciler struct {
	db      *sql.DB
	dialect Dialect
	now     func() time.Time
}

func New(db *sql.DB, dialect Dialect, now func() time.Time) (*Reconciler, error) {
	if db == nil {
		return nil, errors.New("administrator bootstrap database is required")
	}
	if dialect != DialectSQLite && dialect != DialectPostgres {
		return nil, errors.Errorf("unsupported administrator bootstrap dialect %q", dialect)
	}
	if now == nil {
		now = time.Now
	}
	return &Reconciler{db: db, dialect: dialect, now: now}, nil
}

func (r *Reconciler) BootstrapAdmin(ctx context.Context, request Request) (Result, error) {
	request, userID, err := normalizeRequest(request)
	if err != nil {
		return Result{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, errors.Wrap(err, "begin administrator bootstrap")
	}
	defer func() { _ = tx.Rollback() }()

	if err := r.checkIdentityConflicts(ctx, tx, request, userID); err != nil {
		return Result{}, err
	}
	if err := r.reconcile(ctx, tx, request, userID); err != nil {
		return Result{}, err
	}
	if err := r.insertAudit(ctx, tx, request, userID); err != nil {
		return Result{}, err
	}
	if err := tx.Commit(); err != nil {
		return Result{}, errors.Wrap(err, "commit administrator bootstrap")
	}
	return Result{UserID: userID, OrganizationID: request.OrganizationID, Role: AdminRole}, nil
}

func normalizeRequest(request Request) (Request, string, error) {
	request.Issuer = strings.TrimSpace(request.Issuer)
	request.Subject = strings.TrimSpace(request.Subject)
	request.Email = strings.TrimSpace(request.Email)
	request.DisplayName = strings.TrimSpace(request.DisplayName)
	request.OrganizationID = strings.TrimSpace(request.OrganizationID)
	request.OrganizationSlug = strings.TrimSpace(request.OrganizationSlug)
	request.OrganizationName = strings.TrimSpace(request.OrganizationName)
	request.OperatorID = strings.TrimSpace(request.OperatorID)
	if parsed, err := url.Parse(request.Issuer); err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.Fragment != "" || parsed.RawQuery != "" {
		return Request{}, "", errors.New("issuer must be an absolute HTTPS URL without query or fragment")
	}
	if request.Email == "" || !strings.Contains(request.Email, "@") {
		return Request{}, "", errors.New("email is required")
	}
	if request.OrganizationID == "" || request.OrganizationSlug == "" || request.OrganizationName == "" {
		return Request{}, "", errors.New("organization id, slug, and name are required")
	}
	if request.OperatorID == "" {
		request.OperatorID = "deployment-operator"
	}
	userID, err := appauth.OIDCUserID(request.Issuer, request.Subject)
	if err != nil {
		return Request{}, "", errors.Wrap(err, "validate OIDC identity")
	}
	return request, userID, nil
}

func (r *Reconciler) checkIdentityConflicts(ctx context.Context, tx *sql.Tx, request Request, userID string) error {
	var normalizedUserID string
	err := tx.QueryRowContext(ctx, r.query(
		`SELECT id FROM auth_app_users WHERE oidc_issuer = ? AND oidc_subject = ?`,
		`SELECT id FROM auth_app_users WHERE oidc_issuer = $1 AND oidc_subject = $2 FOR UPDATE`,
	), request.Issuer, request.Subject).Scan(&normalizedUserID)
	if err != nil && !stderrors.Is(err, sql.ErrNoRows) {
		return errors.Wrap(err, "load existing normalized identity")
	}
	if err == nil && normalizedUserID != userID {
		return errors.Wrapf(ErrConflict, "OIDC identity is already normalized as user %q", normalizedUserID)
	}

	var boundUserID string
	err = tx.QueryRowContext(ctx, r.query(
		`SELECT user_id FROM auth_app_external_identities WHERE issuer = ? AND subject = ?`,
		`SELECT user_id FROM auth_app_external_identities WHERE issuer = $1 AND subject = $2 FOR UPDATE`,
	), request.Issuer, request.Subject).Scan(&boundUserID)
	if err != nil && !stderrors.Is(err, sql.ErrNoRows) {
		return errors.Wrap(err, "load existing external identity")
	}
	if err == nil && boundUserID != userID {
		return errors.Wrapf(ErrConflict, "OIDC identity is already bound to user %q", boundUserID)
	}

	var issuer, subject sql.NullString
	err = tx.QueryRowContext(ctx, r.query(
		`SELECT oidc_issuer, oidc_subject FROM auth_app_users WHERE id = ?`,
		`SELECT oidc_issuer, oidc_subject FROM auth_app_users WHERE id = $1 FOR UPDATE`,
	), userID).Scan(&issuer, &subject)
	if err != nil && !stderrors.Is(err, sql.ErrNoRows) {
		return errors.Wrap(err, "load deterministic application user")
	}
	if err == nil && (issuer.String != request.Issuer || subject.String != request.Subject) {
		return errors.Wrapf(ErrConflict, "application user %q belongs to another OIDC identity", userID)
	}

	var tenantID string
	err = tx.QueryRowContext(ctx, r.query(
		`SELECT tenant_id FROM auth_app_resources WHERE type = 'org' AND id = ?`,
		`SELECT tenant_id FROM auth_app_resources WHERE type = 'org' AND id = $1 FOR UPDATE`,
	), request.OrganizationID).Scan(&tenantID)
	if err != nil && !stderrors.Is(err, sql.ErrNoRows) {
		return errors.Wrap(err, "load existing organization resource")
	}
	if err == nil && tenantID != request.OrganizationID {
		return errors.Wrapf(ErrConflict, "organization resource %q belongs to tenant %q", request.OrganizationID, tenantID)
	}
	return nil
}

func (r *Reconciler) reconcile(ctx context.Context, tx *sql.Tx, request Request, userID string) error {
	statements := []struct {
		sql  string
		args []any
	}{
		{r.query(
			`INSERT INTO auth_app_users (id, oidc_issuer, oidc_subject, email, display_name, email_verified, disabled_at) VALUES (?, ?, ?, ?, ?, 1, NULL) ON CONFLICT(id) DO UPDATE SET email = excluded.email, display_name = excluded.display_name, email_verified = 1, disabled_at = NULL`,
			`INSERT INTO auth_app_users (id, oidc_issuer, oidc_subject, email, display_name, email_verified, disabled_at) VALUES ($1, $2, $3, $4, $5, true, NULL) ON CONFLICT(id) DO UPDATE SET email = excluded.email, display_name = excluded.display_name, email_verified = true, disabled_at = NULL`,
		), []any{userID, request.Issuer, request.Subject, request.Email, request.DisplayName}},
		{r.query(
			`INSERT INTO auth_app_external_identities (issuer, subject, user_id) VALUES (?, ?, ?) ON CONFLICT(issuer, subject) DO NOTHING`,
			`INSERT INTO auth_app_external_identities (issuer, subject, user_id) VALUES ($1, $2, $3) ON CONFLICT(issuer, subject) DO NOTHING`,
		), []any{request.Issuer, request.Subject, userID}},
		{r.query(
			`INSERT INTO auth_app_tenants (id, slug, name, disabled_at) VALUES (?, ?, ?, NULL) ON CONFLICT(id) DO UPDATE SET slug = excluded.slug, name = excluded.name, disabled_at = NULL`,
			`INSERT INTO auth_app_tenants (id, slug, name, disabled_at) VALUES ($1, $2, $3, NULL) ON CONFLICT(id) DO UPDATE SET slug = excluded.slug, name = excluded.name, disabled_at = NULL`,
		), []any{request.OrganizationID, request.OrganizationSlug, request.OrganizationName}},
		{r.query(
			`INSERT INTO auth_app_resources (type, id, name, tenant_id, owner_id, claims_json) VALUES ('org', ?, ?, ?, '', '{}') ON CONFLICT(type, id) DO UPDATE SET name = excluded.name, tenant_id = excluded.tenant_id`,
			`INSERT INTO auth_app_resources (type, id, name, tenant_id, owner_id, claims_json) VALUES ('org', $1, $2, $3, '', '{}'::jsonb) ON CONFLICT(type, id) DO UPDATE SET name = excluded.name, tenant_id = excluded.tenant_id`,
		), []any{request.OrganizationID, request.OrganizationName, request.OrganizationID}},
		{r.query(
			`INSERT INTO auth_app_memberships (user_id, tenant_id, role, revoked_at) VALUES (?, ?, 'admin', NULL) ON CONFLICT(user_id, tenant_id, role) DO UPDATE SET revoked_at = NULL`,
			`INSERT INTO auth_app_memberships (user_id, tenant_id, role, revoked_at) VALUES ($1, $2, 'admin', NULL) ON CONFLICT(user_id, tenant_id, role) DO UPDATE SET revoked_at = NULL`,
		), []any{userID, request.OrganizationID}},
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement.sql, statement.args...); err != nil {
			return errors.Wrap(err, "reconcile administrator bootstrap state")
		}
	}
	return nil
}

func (r *Reconciler) insertAudit(ctx context.Context, tx *sql.Tx, request Request, userID string) error {
	attributes, err := json.Marshal(map[string]any{
		"issuer": request.Issuer, "subject": request.Subject, "userId": userID, "email": request.Email,
		"organizationSlug": request.OrganizationSlug, "role": AdminRole,
	})
	if err != nil {
		return errors.Wrap(err, "encode administrator bootstrap audit attributes")
	}
	query := r.query(
		`INSERT INTO auth_audit_records (event, outcome, reason, status_code, route_name, method, pattern, action, actor_id, actor_kind, tenant_id, resource_type, resource_id, request_id, ip_hash, user_agent, attributes_json, created_at) VALUES (?, ?, '', 0, '', '', '', ?, ?, ?, ?, ?, ?, '', '', '', ?, ?)`,
		`INSERT INTO auth_audit_records (event, outcome, reason, status_code, route_name, method, pattern, action, actor_id, actor_kind, tenant_id, resource_type, resource_id, request_id, ip_hash, user_agent, attributes_json, created_at) VALUES ($1, $2, '', 0, '', '', '', $3, $4, $5, $6, $7, $8, '', '', '', $9::jsonb, $10)`,
	)
	if _, err := tx.ExecContext(ctx, query, "operator.bootstrap_admin", "success", "bootstrap.admin", request.OperatorID, "operator", request.OrganizationID, "org", request.OrganizationID, string(attributes), r.now().UTC()); err != nil {
		return errors.Wrap(err, "insert administrator bootstrap audit record")
	}
	return nil
}

func (r *Reconciler) query(sqlite, postgres string) string {
	if r.dialect == DialectPostgres {
		return postgres
	}
	return sqlite
}
