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
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
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

func (s *Store) ApplySchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, s.schema())
	if err != nil {
		return fmt.Errorf("apply membership invite schema: %w", err)
	}
	return nil
}

func (s *Store) Begin(ctx context.Context, token string, now time.Time) (membershipinvite.Pending, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return membershipinvite.Pending{}, fmt.Errorf("begin pending membership invite: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	record, err := s.loadCapability(ctx, tx, capability.HashToken(token))
	if err != nil {
		return membershipinvite.Pending{}, err
	}
	if err := validateCapability(record, now); err != nil {
		return membershipinvite.Pending{}, err
	}
	role := strings.TrimSpace(record.claims["role"])
	if _, ok := s.allowedRoles[role]; !ok {
		return membershipinvite.Pending{}, membershipinvite.ErrRoleNotAllowed
	}
	handle, err := sessionauth.RandomToken()
	if err != nil {
		return membershipinvite.Pending{}, fmt.Errorf("create pending membership invite handle: %w", err)
	}
	expiresAt := now.Add(15 * time.Minute)
	if record.expiresAt.Before(expiresAt) {
		expiresAt = record.expiresAt
	}
	if _, err := tx.ExecContext(ctx, s.insertPendingQuery(), capability.HashToken(handle), record.id, expiresAt, now); err != nil {
		return membershipinvite.Pending{}, fmt.Errorf("store pending membership invite: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return membershipinvite.Pending{}, fmt.Errorf("commit pending membership invite: %w", err)
	}
	return membershipinvite.Pending{Handle: handle, CapabilityID: record.id, TenantID: record.resourceID, Email: record.claims["email"], Role: role, ExpiresAt: expiresAt}, nil
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
	return s.acceptRecord(ctx, tx, record, actorUserID, now, nil)
}

func (s *Store) AcceptPending(ctx context.Context, handle, actorUserID string, now time.Time) (membershipinvite.Result, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return membershipinvite.Result{}, fmt.Errorf("begin pending membership invite acceptance: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var capabilityID string
	var expiresAt time.Time
	var usedAt sql.NullTime
	if err := tx.QueryRowContext(ctx, s.pendingQuery(), capability.HashToken(handle)).Scan(&capabilityID, &expiresAt, &usedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return membershipinvite.Result{}, capability.ErrNotFound
		}
		return membershipinvite.Result{}, fmt.Errorf("load pending membership invite: %w", err)
	}
	if usedAt.Valid {
		return membershipinvite.Result{}, capability.ErrUsed
	}
	if now.After(expiresAt) {
		return membershipinvite.Result{}, capability.ErrExpired
	}
	record, err := s.loadCapabilityByID(ctx, tx, capabilityID)
	if err != nil {
		return membershipinvite.Result{}, err
	}
	if err := validateCapability(record, now); err != nil {
		return membershipinvite.Result{}, err
	}
	return s.acceptRecord(ctx, tx, record, actorUserID, now, capability.HashToken(handle))
}

func (s *Store) acceptRecord(ctx context.Context, tx *sql.Tx, record capabilityRow, actorUserID string, now time.Time, pendingHash []byte) (membershipinvite.Result, error) {
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
	if len(pendingHash) > 0 {
		result, err := tx.ExecContext(ctx, s.consumePendingQuery(), now, pendingHash)
		if err != nil {
			return membershipinvite.Result{}, fmt.Errorf("consume pending membership invite: %w", err)
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

func (s *Store) loadCapabilityByID(ctx context.Context, tx *sql.Tx, id string) (capabilityRow, error) {
	var record capabilityRow
	var claimsJSON string
	err := tx.QueryRowContext(ctx, s.capabilityByIDQuery(), id).Scan(&record.id, &record.purpose, &record.resourceType, &record.resourceID, &claimsJSON, &record.expiresAt, &record.singleUse, &record.usedAt, &record.revokedAt)
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
func (s *Store) capabilityByIDQuery() string {
	q := `SELECT id, purpose, resource_type, resource_id, claims_json, expires_at, single_use, used_at, revoked_at FROM auth_capabilities WHERE id = `
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
func (s *Store) insertPendingQuery() string {
	if s.dialect == DialectPostgres {
		return `INSERT INTO auth_pending_membership_invites (handle_hash, capability_id, expires_at, created_at) VALUES ($1, $2, $3, $4)`
	}
	return `INSERT INTO auth_pending_membership_invites (handle_hash, capability_id, expires_at, created_at) VALUES (?, ?, ?, ?)`
}
func (s *Store) pendingQuery() string {
	q := `SELECT capability_id, expires_at, used_at FROM auth_pending_membership_invites WHERE handle_hash = `
	if s.dialect == DialectPostgres {
		return q + `$1 FOR UPDATE`
	}
	return q + `?`
}
func (s *Store) consumePendingQuery() string {
	if s.dialect == DialectPostgres {
		return `UPDATE auth_pending_membership_invites SET used_at = $1 WHERE handle_hash = $2 AND used_at IS NULL`
	}
	return `UPDATE auth_pending_membership_invites SET used_at = ? WHERE handle_hash = ? AND used_at IS NULL`
}
func (s *Store) schema() string {
	if s.dialect == DialectPostgres {
		return `CREATE TABLE IF NOT EXISTS auth_pending_membership_invites (handle_hash BYTEA PRIMARY KEY, capability_id TEXT NOT NULL REFERENCES auth_capabilities(id), expires_at TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL, used_at TIMESTAMPTZ NULL); CREATE INDEX IF NOT EXISTS idx_auth_pending_membership_invites_expiry ON auth_pending_membership_invites(expires_at);`
	}
	return `CREATE TABLE IF NOT EXISTS auth_pending_membership_invites (handle_hash BLOB PRIMARY KEY, capability_id TEXT NOT NULL REFERENCES auth_capabilities(id), expires_at TIMESTAMP NOT NULL, created_at TIMESTAMP NOT NULL, used_at TIMESTAMP NULL); CREATE INDEX IF NOT EXISTS idx_auth_pending_membership_invites_expiry ON auth_pending_membership_invites(expires_at);`
}
