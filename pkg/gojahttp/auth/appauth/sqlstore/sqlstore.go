// Package sqlstore provides a database/sql-backed appauth store.
package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
)

// Dialect selects SQL placeholder and schema syntax.
type Dialect string

const (
	DialectSQLite   Dialect = "sqlite"
	DialectPostgres Dialect = "postgres"
)

// Config controls Store construction.
type Config struct {
	DB      *sql.DB
	Dialect Dialect
}

// Store persists app-owned users, memberships, tenants, and resources in SQL.
type Store struct {
	db      *sql.DB
	dialect Dialect
}

// New creates a SQL-backed appauth store.
func New(cfg Config) (*Store, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("appauth/sqlstore: db is required")
	}
	if cfg.Dialect == "" {
		cfg.Dialect = DialectPostgres
	}
	switch cfg.Dialect {
	case DialectSQLite, DialectPostgres:
	default:
		return nil, fmt.Errorf("appauth/sqlstore: unsupported dialect %q", cfg.Dialect)
	}
	return &Store{db: cfg.DB, dialect: cfg.Dialect}, nil
}

// Schema returns the DDL for the configured dialect.
func (s *Store) Schema() string {
	if s.dialect == DialectSQLite {
		return SQLiteSchema
	}
	return PostgresSchema
}

// ApplySchema executes the configured schema. It is intended for tests,
// examples, and simple migrations; production hosts can run the same DDL with
// their migration tool of choice.
func (s *Store) ApplySchema(ctx context.Context) error {
	for _, stmt := range splitSQLStatements(s.Schema()) {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply appauth schema: %w", err)
		}
	}
	return nil
}

// AddUser inserts or replaces a user fixture. It is primarily for tests,
// examples, and simple bootstrap migrations.
func (s *Store) AddUser(ctx context.Context, user appauth.User) error {
	_, err := s.db.ExecContext(ctx, s.upsertUserQuery(),
		user.ID,
		nullString(user.KeycloakSub),
		user.Email,
		user.DisplayName,
		user.EmailVerified,
		nullTime(user.DisabledAt),
	)
	if err != nil {
		return fmt.Errorf("add appauth user: %w", err)
	}
	return nil
}

// AddTenant inserts or replaces a tenant fixture.
func (s *Store) AddTenant(ctx context.Context, tenant appauth.Tenant) error {
	_, err := s.db.ExecContext(ctx, s.upsertTenantQuery(), tenant.ID, nullString(tenant.Slug), tenant.Name, nullTime(tenant.DisabledAt))
	if err != nil {
		return fmt.Errorf("add appauth tenant: %w", err)
	}
	return nil
}

// AddMembership inserts or replaces a membership fixture.
func (s *Store) AddMembership(ctx context.Context, membership appauth.Membership) error {
	_, err := s.db.ExecContext(ctx, s.upsertMembershipQuery(), membership.UserID, membership.TenantID, membership.Role, nullTime(membership.RevokedAt))
	if err != nil {
		return fmt.Errorf("add appauth membership: %w", err)
	}
	return nil
}

// AddResource inserts or replaces a resource fixture.
func (s *Store) AddResource(ctx context.Context, resource appauth.Resource) error {
	claims, err := marshalClaims(resource.Claims)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.upsertResourceQuery(), resource.Type, resource.ID, resource.Name, resource.TenantID, resource.OwnerID, string(claims))
	if err != nil {
		return fmt.Errorf("add appauth resource: %w", err)
	}
	return nil
}

func (s *Store) ByID(ctx context.Context, id string) (*appauth.User, error) {
	user, err := scanUser(s.db.QueryRowContext(ctx, s.userByIDQuery(), id))
	if err != nil {
		return nil, err
	}
	if user.DisabledAt != nil {
		return nil, gojahttp.ErrNotFound
	}
	return user, nil
}

func (s *Store) ByExternalIdentity(ctx context.Context, issuer, subject string) (*appauth.User, error) {
	query := `SELECT ` + userColumns + ` FROM auth_app_users u JOIN auth_app_external_identities e ON e.user_id = u.id WHERE e.issuer = ` + s.placeholder(1) + ` AND e.subject = ` + s.placeholder(2)
	user, err := scanUser(s.db.QueryRowContext(ctx, query, issuer, subject))
	if err != nil {
		return nil, err
	}
	if user.DisabledAt != nil {
		return nil, gojahttp.ErrNotFound
	}
	return user, nil
}

func (s *Store) ByKeycloakSub(ctx context.Context, sub string) (*appauth.User, error) {
	user, err := scanUser(s.db.QueryRowContext(ctx, s.userBySubQuery(), sub))
	if err != nil {
		return nil, err
	}
	if user.DisabledAt != nil {
		return nil, gojahttp.ErrNotFound
	}
	return user, nil
}

func (s *Store) UpsertFromOIDC(ctx context.Context, sub, email string, emailVerified bool) (*appauth.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin oidc user upsert: %w", err)
	}
	defer rollback(tx)

	candidate := appauth.User{ID: "user:" + sub, KeycloakSub: sub, Email: email, EmailVerified: emailVerified}
	if _, err := tx.ExecContext(ctx, s.upsertOIDCUserQuery(), candidate.ID, candidate.KeycloakSub, candidate.Email, candidate.DisplayName, candidate.EmailVerified, nullTime(candidate.DisabledAt)); err != nil {
		return nil, fmt.Errorf("upsert oidc user: %w", err)
	}
	user, err := scanUser(tx.QueryRowContext(ctx, s.userBySubQuery(), sub))
	if err != nil {
		return nil, err
	}
	if user.DisabledAt != nil {
		return nil, gojahttp.ErrNotFound
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit oidc user upsert: %w", err)
	}
	return user, nil
}

func (s *Store) MembershipsForUser(ctx context.Context, userID string) ([]appauth.Membership, error) {
	rows, err := s.db.QueryContext(ctx, s.membershipsForUserQuery(), userID)
	if err != nil {
		return nil, fmt.Errorf("query memberships: %w", err)
	}
	defer closeRows(rows)
	out := []appauth.Membership{}
	for rows.Next() {
		membership, err := scanMembership(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, membership)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memberships: %w", err)
	}
	return out, nil
}

func (s *Store) IsMember(ctx context.Context, userID, tenantID string) (bool, error) {
	return s.exists(ctx, s.isMemberQuery(), userID, tenantID)
}

func (s *Store) HasRole(ctx context.Context, userID, tenantID string, roles ...string) (bool, error) {
	for _, role := range roles {
		ok, err := s.exists(ctx, s.hasRoleQuery(), userID, tenantID, role)
		if err != nil || ok {
			return ok, err
		}
	}
	return false, nil
}

func (s *Store) GetResource(ctx context.Context, typ, id string) (*appauth.Resource, error) {
	return scanResource(s.db.QueryRowContext(ctx, s.resourceByIDQuery(), typ, id))
}

func (s *Store) exists(ctx context.Context, query string, args ...any) (bool, error) {
	var value int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&value); err != nil {
		return false, fmt.Errorf("query appauth existence: %w", err)
	}
	return value > 0, nil
}

func scanUser(row scanner) (*appauth.User, error) {
	var user appauth.User
	var keycloakSub sql.NullString
	var disabledAt sql.NullTime
	if err := row.Scan(&user.ID, &keycloakSub, &user.Email, &user.DisplayName, &user.EmailVerified, &disabledAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, gojahttp.ErrNotFound
		}
		return nil, fmt.Errorf("scan appauth user: %w", err)
	}
	user.KeycloakSub = keycloakSub.String
	if disabledAt.Valid {
		user.DisabledAt = &disabledAt.Time
	}
	return &user, nil
}

func scanMembership(row scanner) (appauth.Membership, error) {
	var membership appauth.Membership
	var revokedAt sql.NullTime
	if err := row.Scan(&membership.UserID, &membership.TenantID, &membership.Role, &revokedAt); err != nil {
		return appauth.Membership{}, fmt.Errorf("scan appauth membership: %w", err)
	}
	if revokedAt.Valid {
		membership.RevokedAt = &revokedAt.Time
	}
	return membership, nil
}

func scanResource(row scanner) (*appauth.Resource, error) {
	var resource appauth.Resource
	var claimsJSON string
	if err := row.Scan(&resource.Type, &resource.ID, &resource.Name, &resource.TenantID, &resource.OwnerID, &claimsJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, gojahttp.ErrNotFound
		}
		return nil, fmt.Errorf("scan appauth resource: %w", err)
	}
	if err := json.Unmarshal([]byte(claimsJSON), &resource.Claims); err != nil {
		return nil, fmt.Errorf("decode appauth resource claims: %w", err)
	}
	if resource.Claims == nil {
		resource.Claims = map[string]any{}
	}
	return &resource, nil
}

const (
	userColumns       = `id, keycloak_sub, email, display_name, email_verified, disabled_at`
	membershipColumns = `user_id, tenant_id, role, revoked_at`
	resourceColumns   = `type, id, name, tenant_id, owner_id, claims_json`
)

const (
	upsertUserSQLite       = `INSERT INTO auth_app_users (` + userColumns + `) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET keycloak_sub = excluded.keycloak_sub, email = excluded.email, display_name = excluded.display_name, email_verified = excluded.email_verified, disabled_at = excluded.disabled_at`
	upsertUserPostgres     = `INSERT INTO auth_app_users (` + userColumns + `) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT(id) DO UPDATE SET keycloak_sub = excluded.keycloak_sub, email = excluded.email, display_name = excluded.display_name, email_verified = excluded.email_verified, disabled_at = excluded.disabled_at`
	upsertOIDCUserSQLite   = `INSERT INTO auth_app_users (` + userColumns + `) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(keycloak_sub) DO UPDATE SET email = excluded.email, email_verified = excluded.email_verified WHERE auth_app_users.disabled_at IS NULL`
	upsertOIDCUserPostgres = `INSERT INTO auth_app_users (` + userColumns + `) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT(keycloak_sub) DO UPDATE SET email = excluded.email, email_verified = excluded.email_verified WHERE auth_app_users.disabled_at IS NULL`
	userByIDSQLite         = `SELECT ` + userColumns + ` FROM auth_app_users WHERE id = ?`
	userByIDPostgres       = `SELECT ` + userColumns + ` FROM auth_app_users WHERE id = $1`
	userBySubSQLite        = `SELECT ` + userColumns + ` FROM auth_app_users WHERE keycloak_sub = ?`
	userBySubPostgres      = `SELECT ` + userColumns + ` FROM auth_app_users WHERE keycloak_sub = $1`
)

const (
	upsertTenantSQLite     = `INSERT INTO auth_app_tenants (id, slug, name, disabled_at) VALUES (?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET slug = excluded.slug, name = excluded.name, disabled_at = excluded.disabled_at`
	upsertTenantPostgres   = `INSERT INTO auth_app_tenants (id, slug, name, disabled_at) VALUES ($1, $2, $3, $4) ON CONFLICT(id) DO UPDATE SET slug = excluded.slug, name = excluded.name, disabled_at = excluded.disabled_at`
	upsertMemberSQLite     = `INSERT INTO auth_app_memberships (` + membershipColumns + `) VALUES (?, ?, ?, ?) ON CONFLICT(user_id, tenant_id, role) DO UPDATE SET revoked_at = excluded.revoked_at`
	upsertMemberPostgres   = `INSERT INTO auth_app_memberships (` + membershipColumns + `) VALUES ($1, $2, $3, $4) ON CONFLICT(user_id, tenant_id, role) DO UPDATE SET revoked_at = excluded.revoked_at`
	membershipsUserSQLite  = `SELECT ` + membershipColumns + ` FROM auth_app_memberships WHERE user_id = ? AND revoked_at IS NULL ORDER BY tenant_id, role`
	membershipsUserPG      = `SELECT ` + membershipColumns + ` FROM auth_app_memberships WHERE user_id = $1 AND revoked_at IS NULL ORDER BY tenant_id, role`
	isMemberSQLite         = `SELECT COUNT(1) FROM auth_app_memberships WHERE user_id = ? AND tenant_id = ? AND revoked_at IS NULL`
	isMemberPostgres       = `SELECT COUNT(1) FROM auth_app_memberships WHERE user_id = $1 AND tenant_id = $2 AND revoked_at IS NULL`
	hasRoleSQLite          = `SELECT COUNT(1) FROM auth_app_memberships WHERE user_id = ? AND tenant_id = ? AND role = ? AND revoked_at IS NULL`
	hasRolePostgres        = `SELECT COUNT(1) FROM auth_app_memberships WHERE user_id = $1 AND tenant_id = $2 AND role = $3 AND revoked_at IS NULL`
	upsertResourceSQLite   = `INSERT INTO auth_app_resources (` + resourceColumns + `) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(type, id) DO UPDATE SET name = excluded.name, tenant_id = excluded.tenant_id, owner_id = excluded.owner_id, claims_json = excluded.claims_json`
	upsertResourcePostgres = `INSERT INTO auth_app_resources (` + resourceColumns + `) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT(type, id) DO UPDATE SET name = excluded.name, tenant_id = excluded.tenant_id, owner_id = excluded.owner_id, claims_json = excluded.claims_json`
	resourceByIDSQLite     = `SELECT ` + resourceColumns + ` FROM auth_app_resources WHERE type = ? AND id = ?`
	resourceByIDPostgres   = `SELECT ` + resourceColumns + ` FROM auth_app_resources WHERE type = $1 AND id = $2`
)

func (s *Store) placeholder(index int) string {
	if s.dialect == DialectPostgres {
		return "$" + strconv.Itoa(index)
	}
	return "?"
}

func (s *Store) upsertUserQuery() string {
	if s.dialect == DialectPostgres {
		return upsertUserPostgres
	}
	return upsertUserSQLite
}

func (s *Store) upsertOIDCUserQuery() string {
	if s.dialect == DialectPostgres {
		return upsertOIDCUserPostgres
	}
	return upsertOIDCUserSQLite
}

func (s *Store) userByIDQuery() string {
	if s.dialect == DialectPostgres {
		return userByIDPostgres
	}
	return userByIDSQLite
}

func (s *Store) userBySubQuery() string {
	if s.dialect == DialectPostgres {
		return userBySubPostgres
	}
	return userBySubSQLite
}

func (s *Store) upsertTenantQuery() string {
	if s.dialect == DialectPostgres {
		return upsertTenantPostgres
	}
	return upsertTenantSQLite
}

func (s *Store) upsertMembershipQuery() string {
	if s.dialect == DialectPostgres {
		return upsertMemberPostgres
	}
	return upsertMemberSQLite
}

func (s *Store) membershipsForUserQuery() string {
	if s.dialect == DialectPostgres {
		return membershipsUserPG
	}
	return membershipsUserSQLite
}

func (s *Store) isMemberQuery() string {
	if s.dialect == DialectPostgres {
		return isMemberPostgres
	}
	return isMemberSQLite
}

func (s *Store) hasRoleQuery() string {
	if s.dialect == DialectPostgres {
		return hasRolePostgres
	}
	return hasRoleSQLite
}

func (s *Store) upsertResourceQuery() string {
	if s.dialect == DialectPostgres {
		return upsertResourcePostgres
	}
	return upsertResourceSQLite
}

func (s *Store) resourceByIDQuery() string {
	if s.dialect == DialectPostgres {
		return resourceByIDPostgres
	}
	return resourceByIDSQLite
}

type scanner interface {
	Scan(dest ...any) error
}

func marshalClaims(claims map[string]any) ([]byte, error) {
	if claims == nil {
		claims = map[string]any{}
	}
	data, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("marshal appauth resource claims: %w", err)
	}
	return data, nil
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func nullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func rollback(tx *sql.Tx) { _ = tx.Rollback() }

func closeRows(rows *sql.Rows) { _ = rows.Close() }

func splitSQLStatements(schema string) []string {
	pieces := strings.Split(schema, ";")
	out := make([]string, 0, len(pieces))
	for _, piece := range pieces {
		stmt := strings.TrimSpace(piece)
		if stmt != "" {
			out = append(out, stmt)
		}
	}
	return out
}
