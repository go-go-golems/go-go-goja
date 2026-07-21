// Package sqlstore provides atomic SQL-backed application invite acceptance.
package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/membershipinvite"
)

type Dialect string

const (
	DialectSQLite   Dialect = "sqlite"
	DialectPostgres Dialect = "postgres"
)

type Config struct {
	DB           *sql.DB
	Dialect      Dialect
	AllowedRoles []string
}

type Store struct {
	db           *sql.DB
	dialect      Dialect
	allowedRoles map[string]struct{}
}

var _ membershipinvite.Acceptor = (*Store)(nil)

func New(cfg Config) (*Store, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("membershipinvite/sqlstore: db is required")
	}
	if cfg.Dialect == "" {
		cfg.Dialect = DialectPostgres
	}
	if cfg.Dialect != DialectSQLite && cfg.Dialect != DialectPostgres {
		return nil, fmt.Errorf("membershipinvite/sqlstore: unsupported dialect %q", cfg.Dialect)
	}
	roles := cfg.AllowedRoles
	if len(roles) == 0 {
		roles = []string{"viewer", "member", "admin"}
	}
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		if role = strings.TrimSpace(role); role != "" {
			allowed[role] = struct{}{}
		}
	}
	return &Store{db: cfg.DB, dialect: cfg.Dialect, allowedRoles: allowed}, nil
}

type capabilityRow struct {
	id, purpose, resourceType, resourceID string
	claims                                map[string]string
	expiresAt                             time.Time
	singleUse                             bool
	usedAt, revokedAt                     sql.NullTime
}

func (s *Store) Accept(ctx context.Context, token, actorUserID string, now time.Time) (membershipinvite.Result, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return membershipinvite.Result{}, fmt.Errorf("begin membership invite acceptance: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	record, err := s.loadCapability(ctx, tx, capability.HashToken(token))
	if err != nil {
		return membershipinvite.Result{}, err
	}
	if err := validateCapability(record, now); err != nil {
		return membershipinvite.Result{}, err
	}
	var email string
	var verified bool
	var disabledAt sql.NullTime
	if err := tx.QueryRowContext(ctx, s.userQuery(), actorUserID).Scan(&email, &verified, &disabledAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return membershipinvite.Result{}, membershipinvite.ErrUnauthenticated
		}
		return membershipinvite.Result{}, fmt.Errorf("load membership invite actor: %w", err)
	}
	if disabledAt.Valid {
		return membershipinvite.Result{}, membershipinvite.ErrUnauthenticated
	}
	expectedEmail := strings.TrimSpace(record.claims["email"])
	if expectedEmail != "" {
		if !verified {
			return membershipinvite.Result{}, membershipinvite.ErrEmailUnverified
		}
		if !strings.EqualFold(strings.TrimSpace(email), expectedEmail) {
			return membershipinvite.Result{}, membershipinvite.ErrEmailMismatch
		}
	}
	role := strings.TrimSpace(record.claims["role"])
	if _, ok := s.allowedRoles[role]; !ok {
		return membershipinvite.Result{}, membershipinvite.ErrRoleNotAllowed
	}
	var tenantDisabled sql.NullTime
	if err := tx.QueryRowContext(ctx, s.tenantQuery(), record.resourceID).Scan(&tenantDisabled); err != nil || tenantDisabled.Valid {
		if errors.Is(err, sql.ErrNoRows) || tenantDisabled.Valid {
			return membershipinvite.Result{}, fmt.Errorf("membership invite tenant is unavailable")
		}
		return membershipinvite.Result{}, fmt.Errorf("load membership invite tenant: %w", err)
	}
	if _, err := tx.ExecContext(ctx, s.membershipQuery(), actorUserID, record.resourceID, role); err != nil {
		return membershipinvite.Result{}, fmt.Errorf("create invited membership: %w", err)
	}
	if record.singleUse {
		result, err := tx.ExecContext(ctx, s.consumeQuery(), now, record.id)
		if err != nil {
			return membershipinvite.Result{}, fmt.Errorf("consume membership invite: %w", err)
		}
		if n, err := result.RowsAffected(); err != nil || n != 1 {
			return membershipinvite.Result{}, capability.ErrUsed
		}
	}
	if err := tx.Commit(); err != nil {
		return membershipinvite.Result{}, fmt.Errorf("commit membership invite acceptance: %w", err)
	}
	return membershipinvite.Result{CapabilityID: record.id, UserID: actorUserID, TenantID: record.resourceID, Role: role}, nil
}

func (s *Store) loadCapability(ctx context.Context, tx *sql.Tx, hash []byte) (capabilityRow, error) {
	var record capabilityRow
	var claimsJSON string
	err := tx.QueryRowContext(ctx, s.capabilityQuery(), hash).Scan(&record.id, &record.purpose, &record.resourceType, &record.resourceID, &claimsJSON, &record.expiresAt, &record.singleUse, &record.usedAt, &record.revokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return record, capability.ErrNotFound
	}
	if err != nil {
		return record, fmt.Errorf("load membership invite: %w", err)
	}
	if err := json.Unmarshal([]byte(claimsJSON), &record.claims); err != nil {
		return record, fmt.Errorf("decode membership invite claims: %w", err)
	}
	return record, nil
}

func validateCapability(record capabilityRow, now time.Time) error {
	if record.purpose != capability.PurposeOrgInviteAccept || record.resourceType != "org" {
		return capability.ErrWrongPurpose
	}
	if record.revokedAt.Valid {
		return capability.ErrRevoked
	}
	if !record.expiresAt.IsZero() && now.After(record.expiresAt) {
		return capability.ErrExpired
	}
	if record.singleUse && record.usedAt.Valid {
		return capability.ErrUsed
	}
	return nil
}

func (s *Store) capabilityQuery() string {
	q := `SELECT id, purpose, resource_type, resource_id, claims_json, expires_at, single_use, used_at, revoked_at FROM auth_capabilities WHERE token_hash = `
	if s.dialect == DialectPostgres {
		return q + `$1 FOR UPDATE`
	}
	return q + `?`
}
func (s *Store) userQuery() string {
	if s.dialect == DialectPostgres {
		return `SELECT email, email_verified, disabled_at FROM auth_app_users WHERE id = $1`
	}
	return `SELECT email, email_verified, disabled_at FROM auth_app_users WHERE id = ?`
}
func (s *Store) tenantQuery() string {
	if s.dialect == DialectPostgres {
		return `SELECT disabled_at FROM auth_app_tenants WHERE id = $1`
	}
	return `SELECT disabled_at FROM auth_app_tenants WHERE id = ?`
}
func (s *Store) membershipQuery() string {
	if s.dialect == DialectPostgres {
		return `INSERT INTO auth_app_memberships (user_id, tenant_id, role, revoked_at) VALUES ($1, $2, $3, NULL) ON CONFLICT(user_id, tenant_id, role) DO UPDATE SET revoked_at = NULL`
	}
	return `INSERT INTO auth_app_memberships (user_id, tenant_id, role, revoked_at) VALUES (?, ?, ?, NULL) ON CONFLICT(user_id, tenant_id, role) DO UPDATE SET revoked_at = NULL`
}
func (s *Store) consumeQuery() string {
	if s.dialect == DialectPostgres {
		return `UPDATE auth_capabilities SET used_at = $1 WHERE id = $2 AND used_at IS NULL`
	}
	return `UPDATE auth_capabilities SET used_at = ? WHERE id = ? AND used_at IS NULL`
}
