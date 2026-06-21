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
