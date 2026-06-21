// Package sqlstore provides database/sql-backed programauth stores.
package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/programauth"
)

type Dialect string

const (
	DialectSQLite   Dialect = "sqlite"
	DialectPostgres Dialect = "postgres"
)

type Config struct {
	DB      *sql.DB
	Dialect Dialect
}

type Store struct {
	db      *sql.DB
	dialect Dialect
}

func New(cfg Config) (*Store, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("programauth/sqlstore: db is required")
	}
	if cfg.Dialect == "" {
		cfg.Dialect = DialectPostgres
	}
	switch cfg.Dialect {
	case DialectSQLite, DialectPostgres:
	default:
		return nil, fmt.Errorf("programauth/sqlstore: unsupported dialect %q", cfg.Dialect)
	}
	return &Store{db: cfg.DB, dialect: cfg.Dialect}, nil
}

func (s *Store) Schema() string {
	if s.dialect == DialectSQLite {
		return SQLiteSchema
	}
	return PostgresSchema
}

func (s *Store) ApplySchema(ctx context.Context) error {
	for _, stmt := range splitSQLStatements(s.Schema()) {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply programauth schema: %w", err)
		}
	}
	return nil
}

func (s *Store) CreateAgent(ctx context.Context, agent programauth.Agent) (programauth.Agent, error) {
	agent = cloneAgent(agent)
	if agent.ID == "" {
		return programauth.Agent{}, fmt.Errorf("agent id is required")
	}
	policyJSON, err := marshalGrantSet(agent.Policy)
	if err != nil {
		return programauth.Agent{}, err
	}
	_, err = s.db.ExecContext(ctx, s.insertAgentQuery(), agent.ID, agent.Name, string(agent.Kind), agent.OwnerUserID, agent.TenantID, nullTime(agent.DisabledAt), agent.CreatedBy, agent.CreatedAt, agent.UpdatedAt, policyJSON)
	if err != nil {
		return programauth.Agent{}, fmt.Errorf("create programauth agent: %w", err)
	}
	return agent, nil
}

func (s *Store) GetAgent(ctx context.Context, id string) (programauth.Agent, error) {
	return scanAgent(s.db.QueryRowContext(ctx, s.agentByIDQuery(), strings.TrimSpace(id)))
}

func (s *Store) ListAgents(ctx context.Context, query programauth.AgentQuery) ([]programauth.Agent, error) {
	query.OwnerUserID = strings.TrimSpace(query.OwnerUserID)
	query.TenantID = strings.TrimSpace(query.TenantID)
	sqlQuery, args := s.listAgentsQuery(query)
	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("list programauth agents: %w", err)
	}
	defer closeRows(rows)
	out := []programauth.Agent{}
	for rows.Next() {
		agent, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, agent)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate programauth agents: %w", err)
	}
	return out, nil
}

func (s *Store) DisableAgent(ctx context.Context, id string, disabledAt time.Time) (programauth.Agent, error) {
	id = strings.TrimSpace(id)
	disabledAt = disabledAt.UTC()
	res, err := s.db.ExecContext(ctx, s.disableAgentQuery(), disabledAt, disabledAt, id)
	if err != nil {
		return programauth.Agent{}, fmt.Errorf("disable programauth agent: %w", err)
	}
	if err := requireAffected(res, programauth.ErrAgentNotFound); err != nil {
		return programauth.Agent{}, err
	}
	return s.GetAgent(ctx, id)
}

func (s *Store) CreateAPIToken(ctx context.Context, token programauth.APIToken) (programauth.APIToken, error) {
	token = cloneAPIToken(token)
	if token.ID == "" {
		return programauth.APIToken{}, fmt.Errorf("api token id is required")
	}
	if token.TokenPrefix == "" || len(token.TokenHash) == 0 {
		return programauth.APIToken{}, fmt.Errorf("api token hash and prefix are required")
	}
	grantsJSON, err := marshalGrantSet(token.Grants)
	if err != nil {
		return programauth.APIToken{}, err
	}
	_, err = s.db.ExecContext(ctx, s.insertAPITokenQuery(), token.ID, token.Name, token.AgentID, token.SubjectUserID, append([]byte(nil), token.TokenHash...), token.TokenPrefix, token.CreatedBy, token.CreatedAt, token.UpdatedAt, nullTime(token.ExpiresAt), nullTime(token.LastUsedAt), nullTime(token.RevokedAt), grantsJSON)
	if err != nil {
		return programauth.APIToken{}, fmt.Errorf("create programauth api token: %w", err)
	}
	return token, nil
}

func (s *Store) GetAPITokenByID(ctx context.Context, id string) (programauth.APIToken, error) {
	return scanAPIToken(s.db.QueryRowContext(ctx, s.apiTokenByIDQuery(), strings.TrimSpace(id)))
}

func (s *Store) FindAPITokenByPrefix(ctx context.Context, prefix string) ([]programauth.APIToken, error) {
	rows, err := s.db.QueryContext(ctx, s.apiTokensByPrefixQuery(), strings.TrimSpace(prefix))
	if err != nil {
		return nil, fmt.Errorf("find programauth api tokens by prefix: %w", err)
	}
	return scanAPITokenRows(rows)
}

func (s *Store) ListAPITokens(ctx context.Context, query programauth.APITokenQuery) ([]programauth.APIToken, error) {
	query.AgentID = strings.TrimSpace(query.AgentID)
	query.SubjectUserID = strings.TrimSpace(query.SubjectUserID)
	sqlQuery, args := s.listAPITokensQuery(query)
	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("list programauth api tokens: %w", err)
	}
	return scanAPITokenRows(rows)
}

func (s *Store) RevokeAPIToken(ctx context.Context, id string, revokedAt time.Time) (programauth.APIToken, error) {
	id = strings.TrimSpace(id)
	revokedAt = revokedAt.UTC()
	res, err := s.db.ExecContext(ctx, s.revokeAPITokenQuery(), revokedAt, revokedAt, id)
	if err != nil {
		return programauth.APIToken{}, fmt.Errorf("revoke programauth api token: %w", err)
	}
	if err := requireAffected(res, programauth.ErrAPITokenNotFound); err != nil {
		return programauth.APIToken{}, err
	}
	return s.GetAPITokenByID(ctx, id)
}

func (s *Store) TouchAPIToken(ctx context.Context, id string, usedAt time.Time) error {
	usedAt = usedAt.UTC()
	res, err := s.db.ExecContext(ctx, s.touchAPITokenQuery(), usedAt, usedAt, strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("touch programauth api token: %w", err)
	}
	return requireAffected(res, programauth.ErrAPITokenNotFound)
}

func (s *Store) CreateAccessToken(ctx context.Context, token programauth.AccessToken) (programauth.AccessToken, error) {
	token = cloneAccessToken(token)
	if token.ID == "" {
		return programauth.AccessToken{}, fmt.Errorf("access token id is required")
	}
	if token.TokenPrefix == "" || len(token.TokenHash) == 0 {
		return programauth.AccessToken{}, fmt.Errorf("access token hash and prefix are required")
	}
	grantsJSON, err := marshalGrantSet(token.Grants)
	if err != nil {
		return programauth.AccessToken{}, err
	}
	_, err = s.db.ExecContext(ctx, s.insertAccessTokenQuery(), token.ID, token.AgentID, token.SubjectUserID, token.FamilyID, append([]byte(nil), token.TokenHash...), token.TokenPrefix, token.CreatedAt, token.UpdatedAt, token.ExpiresAt, nullTime(token.LastUsedAt), nullTime(token.RevokedAt), grantsJSON)
	if err != nil {
		return programauth.AccessToken{}, fmt.Errorf("create programauth access token: %w", err)
	}
	return token, nil
}

func (s *Store) FindAccessTokenByPrefix(ctx context.Context, prefix string) ([]programauth.AccessToken, error) {
	rows, err := s.db.QueryContext(ctx, s.accessTokensByPrefixQuery(), strings.TrimSpace(prefix))
	if err != nil {
		return nil, fmt.Errorf("find programauth access tokens by prefix: %w", err)
	}
	return scanAccessTokenRows(rows)
}

func (s *Store) TouchAccessToken(ctx context.Context, id string, usedAt time.Time) error {
	usedAt = usedAt.UTC()
	res, err := s.db.ExecContext(ctx, s.touchAccessTokenQuery(), usedAt, usedAt, strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("touch programauth access token: %w", err)
	}
	return requireAffected(res, programauth.ErrAccessTokenNotFound)
}

func (s *Store) CreateRefreshToken(ctx context.Context, token programauth.RefreshToken) (programauth.RefreshToken, error) {
	token = cloneRefreshToken(token)
	if err := validateRefreshTokenForInsert(token); err != nil {
		return programauth.RefreshToken{}, err
	}
	if err := s.insertRefreshToken(ctx, s.db, token); err != nil {
		return programauth.RefreshToken{}, err
	}
	return token, nil
}

func (s *Store) FindRefreshTokenByPrefix(ctx context.Context, prefix string) ([]programauth.RefreshToken, error) {
	rows, err := s.db.QueryContext(ctx, s.refreshTokensByPrefixQuery(), strings.TrimSpace(prefix))
	if err != nil {
		return nil, fmt.Errorf("find programauth refresh tokens by prefix: %w", err)
	}
	return scanRefreshTokenRows(rows)
}

func (s *Store) RotateRefreshToken(ctx context.Context, currentID string, next programauth.RefreshToken, usedAt time.Time) (programauth.RefreshToken, programauth.RefreshToken, error) {
	currentID = strings.TrimSpace(currentID)
	next = cloneRefreshToken(next)
	if err := validateRefreshTokenForInsert(next); err != nil {
		return programauth.RefreshToken{}, programauth.RefreshToken{}, err
	}
	usedAt = usedAt.UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return programauth.RefreshToken{}, programauth.RefreshToken{}, fmt.Errorf("begin refresh token rotation: %w", err)
	}
	defer rollback(tx)
	current, err := scanRefreshToken(tx.QueryRowContext(ctx, s.refreshTokenByIDQuery(), currentID))
	if err != nil {
		return programauth.RefreshToken{}, programauth.RefreshToken{}, err
	}
	if current.Revoked() {
		return programauth.RefreshToken{}, programauth.RefreshToken{}, programauth.ErrRefreshTokenRevoked
	}
	if current.Used() {
		return programauth.RefreshToken{}, programauth.RefreshToken{}, programauth.ErrRefreshTokenUsed
	}
	next.FamilyID = current.FamilyID
	if err := s.insertRefreshToken(ctx, tx, next); err != nil {
		return programauth.RefreshToken{}, programauth.RefreshToken{}, err
	}
	res, err := tx.ExecContext(ctx, s.rotateRefreshTokenQuery(), usedAt, next.ID, usedAt, currentID)
	if err != nil {
		return programauth.RefreshToken{}, programauth.RefreshToken{}, fmt.Errorf("rotate programauth refresh token: %w", err)
	}
	if err := requireAffected(res, programauth.ErrRefreshTokenUsed); err != nil {
		return programauth.RefreshToken{}, programauth.RefreshToken{}, err
	}
	current.UsedAt = &usedAt
	current.ReplacedByID = next.ID
	current.UpdatedAt = usedAt
	if err := tx.Commit(); err != nil {
		return programauth.RefreshToken{}, programauth.RefreshToken{}, fmt.Errorf("commit refresh token rotation: %w", err)
	}
	return cloneRefreshToken(current), cloneRefreshToken(next), nil
}

func (s *Store) RevokeRefreshTokenFamily(ctx context.Context, familyID string, revokedAt time.Time) error {
	familyID = strings.TrimSpace(familyID)
	if familyID == "" {
		return fmt.Errorf("refresh token family id is required")
	}
	revokedAt = revokedAt.UTC()
	_, err := s.db.ExecContext(ctx, s.revokeRefreshTokenFamilyQuery(), revokedAt, revokedAt, familyID)
	if err != nil {
		return fmt.Errorf("revoke programauth refresh token family: %w", err)
	}
	return nil
}

func (s *Store) CreateDeviceAuthorization(ctx context.Context, device programauth.DeviceAuthorization) (programauth.DeviceAuthorization, error) {
	device = cloneDeviceAuthorization(device)
	if err := validateDeviceAuthorizationForInsert(device); err != nil {
		return programauth.DeviceAuthorization{}, err
	}
	grantsJSON, err := marshalGrantSet(device.Grants)
	if err != nil {
		return programauth.DeviceAuthorization{}, err
	}
	_, err = s.db.ExecContext(ctx, s.insertDeviceAuthorizationQuery(), device.ID, device.ClientName, append([]byte(nil), device.DeviceCodeHash...), device.DeviceCodePrefix, append([]byte(nil), device.UserCodeHash...), device.UserCode, device.VerificationURI, device.VerificationURIComplete, device.CreatedAt, device.UpdatedAt, device.ExpiresAt, int(device.PollInterval/time.Second), nullTime(device.LastPolledAt), nullTime(device.ApprovedAt), nullTime(device.DeniedAt), nullTime(device.ConsumedAt), device.AgentID, device.SubjectUserID, device.TenantID, grantsJSON)
	if err != nil {
		return programauth.DeviceAuthorization{}, fmt.Errorf("create programauth device authorization: %w", err)
	}
	return device, nil
}

func (s *Store) FindDeviceAuthorizationByDeviceCodePrefix(ctx context.Context, prefix string) ([]programauth.DeviceAuthorization, error) {
	rows, err := s.db.QueryContext(ctx, s.devicesByDeviceCodePrefixQuery(), strings.TrimSpace(prefix))
	if err != nil {
		return nil, fmt.Errorf("find programauth device authorizations by prefix: %w", err)
	}
	return scanDeviceAuthorizationRows(rows)
}

func (s *Store) GetDeviceAuthorizationByUserCodeHash(ctx context.Context, hash []byte) (programauth.DeviceAuthorization, error) {
	return scanDeviceAuthorization(s.db.QueryRowContext(ctx, s.deviceByUserCodeHashQuery(), append([]byte(nil), hash...)))
}

func (s *Store) RecordDevicePoll(ctx context.Context, id string, polledAt time.Time, interval time.Duration) (programauth.DeviceAuthorization, error) {
	id = strings.TrimSpace(id)
	polledAt = polledAt.UTC()
	res, err := s.db.ExecContext(ctx, s.recordDevicePollQuery(), polledAt, int(interval/time.Second), polledAt, id)
	if err != nil {
		return programauth.DeviceAuthorization{}, fmt.Errorf("record programauth device poll: %w", err)
	}
	if err := requireAffected(res, programauth.ErrDeviceNotFound); err != nil {
		return programauth.DeviceAuthorization{}, err
	}
	return s.getDeviceAuthorizationByID(ctx, id)
}

func (s *Store) ApproveDeviceAuthorization(ctx context.Context, id string, approved programauth.DeviceAuthorization, approvedAt time.Time) (programauth.DeviceAuthorization, error) {
	id = strings.TrimSpace(id)
	approved = cloneDeviceAuthorization(approved)
	approvedAt = approvedAt.UTC()
	grantsJSON, err := marshalGrantSet(approved.Grants)
	if err != nil {
		return programauth.DeviceAuthorization{}, err
	}
	res, err := s.db.ExecContext(ctx, s.approveDeviceAuthorizationQuery(), approvedAt, approvedAt, approved.AgentID, approved.SubjectUserID, approved.TenantID, grantsJSON, id)
	if err != nil {
		return programauth.DeviceAuthorization{}, fmt.Errorf("approve programauth device authorization: %w", err)
	}
	if err := requireAffected(res, programauth.ErrDeviceConsumed); err != nil {
		return programauth.DeviceAuthorization{}, s.deviceTransitionError(ctx, id, err)
	}
	return s.getDeviceAuthorizationByID(ctx, id)
}

func (s *Store) DenyDeviceAuthorization(ctx context.Context, id string, deniedAt time.Time) (programauth.DeviceAuthorization, error) {
	id = strings.TrimSpace(id)
	deniedAt = deniedAt.UTC()
	res, err := s.db.ExecContext(ctx, s.denyDeviceAuthorizationQuery(), deniedAt, deniedAt, id)
	if err != nil {
		return programauth.DeviceAuthorization{}, fmt.Errorf("deny programauth device authorization: %w", err)
	}
	if err := requireAffected(res, programauth.ErrDeviceConsumed); err != nil {
		return programauth.DeviceAuthorization{}, s.deviceTransitionError(ctx, id, err)
	}
	return s.getDeviceAuthorizationByID(ctx, id)
}

func (s *Store) ConsumeDeviceAuthorization(ctx context.Context, id string, consumedAt time.Time) (programauth.DeviceAuthorization, error) {
	id = strings.TrimSpace(id)
	consumedAt = consumedAt.UTC()
	res, err := s.db.ExecContext(ctx, s.consumeDeviceAuthorizationQuery(), consumedAt, consumedAt, id)
	if err != nil {
		return programauth.DeviceAuthorization{}, fmt.Errorf("consume programauth device authorization: %w", err)
	}
	if err := requireAffected(res, programauth.ErrDeviceConsumed); err != nil {
		return programauth.DeviceAuthorization{}, s.deviceTransitionError(ctx, id, err)
	}
	return s.getDeviceAuthorizationByID(ctx, id)
}

func (s *Store) getDeviceAuthorizationByID(ctx context.Context, id string) (programauth.DeviceAuthorization, error) {
	return scanDeviceAuthorization(s.db.QueryRowContext(ctx, s.deviceByIDQuery(), strings.TrimSpace(id)))
}

func (s *Store) deviceTransitionError(ctx context.Context, id string, fallback error) error {
	device, err := s.getDeviceAuthorizationByID(ctx, id)
	if err != nil {
		return err
	}
	switch {
	case device.Denied():
		return programauth.ErrDeviceDenied
	case device.Consumed():
		return programauth.ErrDeviceConsumed
	case device.Approved():
		return fmt.Errorf("device authorization already approved")
	default:
		return fallback
	}
}

func (s *Store) insertRefreshToken(ctx context.Context, exec sqlExecer, token programauth.RefreshToken) error {
	grantsJSON, err := marshalGrantSet(token.Grants)
	if err != nil {
		return err
	}
	_, err = exec.ExecContext(ctx, s.insertRefreshTokenQuery(), token.ID, token.AgentID, token.SubjectUserID, token.FamilyID, token.Generation, append([]byte(nil), token.TokenHash...), token.TokenPrefix, token.CreatedAt, token.UpdatedAt, token.ExpiresAt, nullTime(token.UsedAt), nullTime(token.RevokedAt), token.ReplacedByID, grantsJSON)
	if err != nil {
		return fmt.Errorf("create programauth refresh token: %w", err)
	}
	return nil
}

func scanAgent(row scanner) (programauth.Agent, error) {
	var agent programauth.Agent
	var kind string
	var disabledAt sql.NullTime
	var policyJSON string
	if err := row.Scan(&agent.ID, &agent.Name, &kind, &agent.OwnerUserID, &agent.TenantID, &disabledAt, &agent.CreatedBy, &agent.CreatedAt, &agent.UpdatedAt, &policyJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return programauth.Agent{}, programauth.ErrAgentNotFound
		}
		return programauth.Agent{}, fmt.Errorf("scan programauth agent: %w", err)
	}
	agent.Kind = programauth.AgentKind(kind)
	agent.DisabledAt = timePtr(disabledAt)
	policy, err := unmarshalGrantSet(policyJSON)
	if err != nil {
		return programauth.Agent{}, err
	}
	agent.Policy = policy
	return cloneAgent(agent), nil
}

func scanAPIToken(row scanner) (programauth.APIToken, error) {
	var token programauth.APIToken
	var expiresAt sql.NullTime
	var lastUsedAt sql.NullTime
	var revokedAt sql.NullTime
	var grantsJSON string
	if err := row.Scan(&token.ID, &token.Name, &token.AgentID, &token.SubjectUserID, &token.TokenHash, &token.TokenPrefix, &token.CreatedBy, &token.CreatedAt, &token.UpdatedAt, &expiresAt, &lastUsedAt, &revokedAt, &grantsJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return programauth.APIToken{}, programauth.ErrAPITokenNotFound
		}
		return programauth.APIToken{}, fmt.Errorf("scan programauth api token: %w", err)
	}
	token.ExpiresAt = timePtr(expiresAt)
	token.LastUsedAt = timePtr(lastUsedAt)
	token.RevokedAt = timePtr(revokedAt)
	grants, err := unmarshalGrantSet(grantsJSON)
	if err != nil {
		return programauth.APIToken{}, err
	}
	token.Grants = grants
	return cloneAPIToken(token), nil
}

func scanAPITokenRows(rows *sql.Rows) ([]programauth.APIToken, error) {
	defer closeRows(rows)
	out := []programauth.APIToken{}
	for rows.Next() {
		token, err := scanAPIToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate programauth api tokens: %w", err)
	}
	return out, nil
}

func scanAccessToken(row scanner) (programauth.AccessToken, error) {
	var token programauth.AccessToken
	var lastUsedAt sql.NullTime
	var revokedAt sql.NullTime
	var grantsJSON string
	if err := row.Scan(&token.ID, &token.AgentID, &token.SubjectUserID, &token.FamilyID, &token.TokenHash, &token.TokenPrefix, &token.CreatedAt, &token.UpdatedAt, &token.ExpiresAt, &lastUsedAt, &revokedAt, &grantsJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return programauth.AccessToken{}, programauth.ErrAccessTokenNotFound
		}
		return programauth.AccessToken{}, fmt.Errorf("scan programauth access token: %w", err)
	}
	token.LastUsedAt = timePtr(lastUsedAt)
	token.RevokedAt = timePtr(revokedAt)
	grants, err := unmarshalGrantSet(grantsJSON)
	if err != nil {
		return programauth.AccessToken{}, err
	}
	token.Grants = grants
	return cloneAccessToken(token), nil
}

func scanAccessTokenRows(rows *sql.Rows) ([]programauth.AccessToken, error) {
	defer closeRows(rows)
	out := []programauth.AccessToken{}
	for rows.Next() {
		token, err := scanAccessToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate programauth access tokens: %w", err)
	}
	return out, nil
}

func scanRefreshToken(row scanner) (programauth.RefreshToken, error) {
	var token programauth.RefreshToken
	var usedAt sql.NullTime
	var revokedAt sql.NullTime
	var grantsJSON string
	if err := row.Scan(&token.ID, &token.AgentID, &token.SubjectUserID, &token.FamilyID, &token.Generation, &token.TokenHash, &token.TokenPrefix, &token.CreatedAt, &token.UpdatedAt, &token.ExpiresAt, &usedAt, &revokedAt, &token.ReplacedByID, &grantsJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return programauth.RefreshToken{}, programauth.ErrRefreshTokenNotFound
		}
		return programauth.RefreshToken{}, fmt.Errorf("scan programauth refresh token: %w", err)
	}
	token.UsedAt = timePtr(usedAt)
	token.RevokedAt = timePtr(revokedAt)
	grants, err := unmarshalGrantSet(grantsJSON)
	if err != nil {
		return programauth.RefreshToken{}, err
	}
	token.Grants = grants
	return cloneRefreshToken(token), nil
}

func scanRefreshTokenRows(rows *sql.Rows) ([]programauth.RefreshToken, error) {
	defer closeRows(rows)
	out := []programauth.RefreshToken{}
	for rows.Next() {
		token, err := scanRefreshToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate programauth refresh tokens: %w", err)
	}
	return out, nil
}

func scanDeviceAuthorization(row scanner) (programauth.DeviceAuthorization, error) {
	var device programauth.DeviceAuthorization
	var pollIntervalSeconds int
	var lastPolledAt sql.NullTime
	var approvedAt sql.NullTime
	var deniedAt sql.NullTime
	var consumedAt sql.NullTime
	var grantsJSON string
	if err := row.Scan(&device.ID, &device.ClientName, &device.DeviceCodeHash, &device.DeviceCodePrefix, &device.UserCodeHash, &device.UserCode, &device.VerificationURI, &device.VerificationURIComplete, &device.CreatedAt, &device.UpdatedAt, &device.ExpiresAt, &pollIntervalSeconds, &lastPolledAt, &approvedAt, &deniedAt, &consumedAt, &device.AgentID, &device.SubjectUserID, &device.TenantID, &grantsJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return programauth.DeviceAuthorization{}, programauth.ErrDeviceNotFound
		}
		return programauth.DeviceAuthorization{}, fmt.Errorf("scan programauth device authorization: %w", err)
	}
	device.PollInterval = time.Duration(pollIntervalSeconds) * time.Second
	device.LastPolledAt = timePtr(lastPolledAt)
	device.ApprovedAt = timePtr(approvedAt)
	device.DeniedAt = timePtr(deniedAt)
	device.ConsumedAt = timePtr(consumedAt)
	grants, err := unmarshalGrantSet(grantsJSON)
	if err != nil {
		return programauth.DeviceAuthorization{}, err
	}
	device.Grants = grants
	return cloneDeviceAuthorization(device), nil
}

func scanDeviceAuthorizationRows(rows *sql.Rows) ([]programauth.DeviceAuthorization, error) {
	defer closeRows(rows)
	out := []programauth.DeviceAuthorization{}
	for rows.Next() {
		device, err := scanDeviceAuthorization(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, device)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate programauth device authorizations: %w", err)
	}
	return out, nil
}

func (s *Store) insertAgentQuery() string {
	return s.rebind(`INSERT INTO auth_program_agents (id, name, kind, owner_user_id, tenant_id, disabled_at, created_by, created_at, updated_at, policy_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
}

func (s *Store) agentByIDQuery() string {
	return s.rebind(`SELECT id, name, kind, owner_user_id, tenant_id, disabled_at, created_by, created_at, updated_at, policy_json FROM auth_program_agents WHERE id = ?`)
}

func (s *Store) disableAgentQuery() string {
	return s.rebind(`UPDATE auth_program_agents SET disabled_at = ?, updated_at = ? WHERE id = ?`)
}

func (s *Store) insertAPITokenQuery() string {
	return s.rebind(`INSERT INTO auth_program_api_tokens (id, name, agent_id, subject_user_id, token_hash, token_prefix, created_by, created_at, updated_at, expires_at, last_used_at, revoked_at, grants_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
}

func (s *Store) apiTokenByIDQuery() string {
	return s.rebind(`SELECT id, name, agent_id, subject_user_id, token_hash, token_prefix, created_by, created_at, updated_at, expires_at, last_used_at, revoked_at, grants_json FROM auth_program_api_tokens WHERE id = ?`)
}

func (s *Store) apiTokensByPrefixQuery() string {
	return s.rebind(`SELECT id, name, agent_id, subject_user_id, token_hash, token_prefix, created_by, created_at, updated_at, expires_at, last_used_at, revoked_at, grants_json FROM auth_program_api_tokens WHERE token_prefix = ? ORDER BY created_at ASC, id ASC`)
}

func (s *Store) revokeAPITokenQuery() string {
	return s.rebind(`UPDATE auth_program_api_tokens SET revoked_at = ?, updated_at = ? WHERE id = ?`)
}

func (s *Store) touchAPITokenQuery() string {
	return s.rebind(`UPDATE auth_program_api_tokens SET last_used_at = ?, updated_at = ? WHERE id = ?`)
}

func (s *Store) insertAccessTokenQuery() string {
	return s.rebind(`INSERT INTO auth_program_access_tokens (id, agent_id, subject_user_id, family_id, token_hash, token_prefix, created_at, updated_at, expires_at, last_used_at, revoked_at, grants_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
}

func (s *Store) accessTokensByPrefixQuery() string {
	return s.rebind(`SELECT id, agent_id, subject_user_id, family_id, token_hash, token_prefix, created_at, updated_at, expires_at, last_used_at, revoked_at, grants_json FROM auth_program_access_tokens WHERE token_prefix = ? ORDER BY created_at ASC, id ASC`)
}

func (s *Store) touchAccessTokenQuery() string {
	return s.rebind(`UPDATE auth_program_access_tokens SET last_used_at = ?, updated_at = ? WHERE id = ?`)
}

func (s *Store) insertRefreshTokenQuery() string {
	return s.rebind(`INSERT INTO auth_program_refresh_tokens (id, agent_id, subject_user_id, family_id, generation, token_hash, token_prefix, created_at, updated_at, expires_at, used_at, revoked_at, replaced_by_id, grants_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
}

func (s *Store) refreshTokenByIDQuery() string {
	return s.rebind(`SELECT id, agent_id, subject_user_id, family_id, generation, token_hash, token_prefix, created_at, updated_at, expires_at, used_at, revoked_at, replaced_by_id, grants_json FROM auth_program_refresh_tokens WHERE id = ?`)
}

func (s *Store) refreshTokensByPrefixQuery() string {
	return s.rebind(`SELECT id, agent_id, subject_user_id, family_id, generation, token_hash, token_prefix, created_at, updated_at, expires_at, used_at, revoked_at, replaced_by_id, grants_json FROM auth_program_refresh_tokens WHERE token_prefix = ? ORDER BY created_at ASC, id ASC`)
}

func (s *Store) rotateRefreshTokenQuery() string {
	return s.rebind(`UPDATE auth_program_refresh_tokens SET used_at = ?, replaced_by_id = ?, updated_at = ? WHERE id = ? AND used_at IS NULL AND revoked_at IS NULL`)
}

func (s *Store) revokeRefreshTokenFamilyQuery() string {
	return s.rebind(`UPDATE auth_program_refresh_tokens SET revoked_at = ?, updated_at = ? WHERE family_id = ? AND revoked_at IS NULL`)
}

func (s *Store) insertDeviceAuthorizationQuery() string {
	return s.rebind(`INSERT INTO auth_program_device_authorizations (id, client_name, device_code_hash, device_code_prefix, user_code_hash, user_code, verification_uri, verification_uri_complete, created_at, updated_at, expires_at, poll_interval_seconds, last_polled_at, approved_at, denied_at, consumed_at, agent_id, subject_user_id, tenant_id, grants_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
}

func (s *Store) deviceColumns() string {
	return `id, client_name, device_code_hash, device_code_prefix, user_code_hash, user_code, verification_uri, verification_uri_complete, created_at, updated_at, expires_at, poll_interval_seconds, last_polled_at, approved_at, denied_at, consumed_at, agent_id, subject_user_id, tenant_id, grants_json`
}

func (s *Store) deviceByIDQuery() string {
	return s.rebind(`SELECT ` + s.deviceColumns() + ` FROM auth_program_device_authorizations WHERE id = ?`)
}

func (s *Store) devicesByDeviceCodePrefixQuery() string {
	return s.rebind(`SELECT ` + s.deviceColumns() + ` FROM auth_program_device_authorizations WHERE device_code_prefix = ? ORDER BY created_at ASC, id ASC`)
}

func (s *Store) deviceByUserCodeHashQuery() string {
	return s.rebind(`SELECT ` + s.deviceColumns() + ` FROM auth_program_device_authorizations WHERE user_code_hash = ?`)
}

func (s *Store) recordDevicePollQuery() string {
	return s.rebind(`UPDATE auth_program_device_authorizations SET last_polled_at = ?, poll_interval_seconds = ?, updated_at = ? WHERE id = ?`)
}

func (s *Store) approveDeviceAuthorizationQuery() string {
	return s.rebind(`UPDATE auth_program_device_authorizations SET approved_at = ?, updated_at = ?, agent_id = ?, subject_user_id = ?, tenant_id = ?, grants_json = ? WHERE id = ? AND approved_at IS NULL AND denied_at IS NULL AND consumed_at IS NULL`)
}

func (s *Store) denyDeviceAuthorizationQuery() string {
	return s.rebind(`UPDATE auth_program_device_authorizations SET denied_at = ?, updated_at = ? WHERE id = ? AND approved_at IS NULL AND denied_at IS NULL AND consumed_at IS NULL`)
}

func (s *Store) consumeDeviceAuthorizationQuery() string {
	return s.rebind(`UPDATE auth_program_device_authorizations SET consumed_at = ?, updated_at = ? WHERE id = ? AND approved_at IS NOT NULL AND denied_at IS NULL AND consumed_at IS NULL`)
}

func (s *Store) listAgentsQuery(query programauth.AgentQuery) (string, []any) {
	var b strings.Builder
	b.WriteString(`SELECT id, name, kind, owner_user_id, tenant_id, disabled_at, created_by, created_at, updated_at, policy_json FROM auth_program_agents`)
	args := []any{}
	addWhere := func(column, value string) {
		if value == "" {
			return
		}
		if len(args) == 0 {
			b.WriteString(` WHERE `)
		} else {
			b.WriteString(` AND `)
		}
		args = append(args, value)
		b.WriteString(column)
		b.WriteString(` = `)
		b.WriteString(s.placeholder(len(args)))
	}
	addWhere("owner_user_id", query.OwnerUserID)
	addWhere("tenant_id", query.TenantID)
	if !query.IncludeDisabled {
		if len(args) == 0 {
			b.WriteString(` WHERE `)
		} else {
			b.WriteString(` AND `)
		}
		b.WriteString(`disabled_at IS NULL`)
	}
	b.WriteString(` ORDER BY created_at ASC, id ASC`)
	return b.String(), args
}

func (s *Store) listAPITokensQuery(query programauth.APITokenQuery) (string, []any) {
	var b strings.Builder
	b.WriteString(`SELECT id, name, agent_id, subject_user_id, token_hash, token_prefix, created_by, created_at, updated_at, expires_at, last_used_at, revoked_at, grants_json FROM auth_program_api_tokens`)
	args := []any{}
	addWhere := func(column, value string) {
		if value == "" {
			return
		}
		if len(args) == 0 {
			b.WriteString(` WHERE `)
		} else {
			b.WriteString(` AND `)
		}
		args = append(args, value)
		b.WriteString(column)
		b.WriteString(` = `)
		b.WriteString(s.placeholder(len(args)))
	}
	addWhere("agent_id", query.AgentID)
	addWhere("subject_user_id", query.SubjectUserID)
	if !query.IncludeRevoked {
		if len(args) == 0 {
			b.WriteString(` WHERE `)
		} else {
			b.WriteString(` AND `)
		}
		b.WriteString(`revoked_at IS NULL`)
	}
	b.WriteString(` ORDER BY created_at ASC, id ASC`)
	return b.String(), args
}

func (s *Store) rebind(query string) string {
	if s.dialect != DialectPostgres {
		return query
	}
	var b strings.Builder
	index := 1
	for _, r := range query {
		if r == '?' {
			_, _ = fmt.Fprintf(&b, "$%d", index)
			index++
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func (s *Store) placeholder(index int) string {
	if s.dialect == DialectPostgres {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

type scanner interface {
	Scan(dest ...any) error
}

type sqlExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func marshalGrantSet(grants gojahttp.GrantSet) (string, error) {
	normalized, err := grants.Normalize()
	if err != nil {
		return "", err
	}
	if normalized.Grants == nil {
		normalized.Grants = []gojahttp.Grant{}
	}
	data, err := json.Marshal(normalized.Grants)
	if err != nil {
		return "", fmt.Errorf("marshal programauth grants: %w", err)
	}
	return string(data), nil
}

func unmarshalGrantSet(raw string) (gojahttp.GrantSet, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "[]"
	}
	var grants []gojahttp.Grant
	if err := json.Unmarshal([]byte(raw), &grants); err != nil {
		return gojahttp.GrantSet{}, fmt.Errorf("unmarshal programauth grants: %w", err)
	}
	return gojahttp.NewGrantSet(grants...)
}

func nullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value.UTC(), Valid: true}
}

func timePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	out := value.Time.UTC()
	return &out
}

func cloneAgent(agent programauth.Agent) programauth.Agent {
	out := agent
	out.Policy = agent.Policy.Clone()
	out.DisabledAt = cloneTimePtr(agent.DisabledAt)
	return out
}

func cloneAPIToken(token programauth.APIToken) programauth.APIToken {
	out := token
	out.TokenHash = append([]byte(nil), token.TokenHash...)
	out.Grants = token.Grants.Clone()
	out.ExpiresAt = cloneTimePtr(token.ExpiresAt)
	out.LastUsedAt = cloneTimePtr(token.LastUsedAt)
	out.RevokedAt = cloneTimePtr(token.RevokedAt)
	return out
}

func cloneAccessToken(token programauth.AccessToken) programauth.AccessToken {
	out := token
	out.TokenHash = append([]byte(nil), token.TokenHash...)
	out.Grants = token.Grants.Clone()
	out.LastUsedAt = cloneTimePtr(token.LastUsedAt)
	out.RevokedAt = cloneTimePtr(token.RevokedAt)
	return out
}

func cloneRefreshToken(token programauth.RefreshToken) programauth.RefreshToken {
	out := token
	out.TokenHash = append([]byte(nil), token.TokenHash...)
	out.Grants = token.Grants.Clone()
	out.UsedAt = cloneTimePtr(token.UsedAt)
	out.RevokedAt = cloneTimePtr(token.RevokedAt)
	return out
}

func validateRefreshTokenForInsert(token programauth.RefreshToken) error {
	if token.ID == "" {
		return fmt.Errorf("refresh token id is required")
	}
	if token.FamilyID == "" {
		return fmt.Errorf("refresh token family id is required")
	}
	if token.TokenPrefix == "" || len(token.TokenHash) == 0 {
		return fmt.Errorf("refresh token hash and prefix are required")
	}
	return nil
}

func cloneDeviceAuthorization(device programauth.DeviceAuthorization) programauth.DeviceAuthorization {
	out := device
	out.DeviceCodeHash = append([]byte(nil), device.DeviceCodeHash...)
	out.UserCodeHash = append([]byte(nil), device.UserCodeHash...)
	out.LastPolledAt = cloneTimePtr(device.LastPolledAt)
	out.ApprovedAt = cloneTimePtr(device.ApprovedAt)
	out.DeniedAt = cloneTimePtr(device.DeniedAt)
	out.ConsumedAt = cloneTimePtr(device.ConsumedAt)
	out.Grants = device.Grants.Clone()
	return out
}

func validateDeviceAuthorizationForInsert(device programauth.DeviceAuthorization) error {
	if device.ID == "" {
		return fmt.Errorf("device authorization id is required")
	}
	if device.DeviceCodePrefix == "" || len(device.DeviceCodeHash) == 0 || len(device.UserCodeHash) == 0 {
		return fmt.Errorf("device authorization hashes and prefix are required")
	}
	return nil
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	out := value.UTC()
	return &out
}

func requireAffected(res sql.Result, missing error) error {
	count, err := res.RowsAffected()
	if err != nil {
		return nil
	}
	if count == 0 {
		return missing
	}
	return nil
}

func closeRows(rows *sql.Rows) { _ = rows.Close() }

func rollback(tx *sql.Tx) { _ = tx.Rollback() }

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

var _ programauth.AgentStore = (*Store)(nil)
var _ programauth.APITokenStore = (*Store)(nil)
var _ programauth.AccessTokenStore = (*Store)(nil)
var _ programauth.RefreshTokenStore = (*Store)(nil)
var _ programauth.DeviceAuthorizationStore = (*Store)(nil)
