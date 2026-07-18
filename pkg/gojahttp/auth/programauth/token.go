package programauth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

var (
	ErrAPITokenNotFound = errors.New("programauth api token not found")
	ErrAPITokenExpired  = errors.New("programauth api token expired")
	ErrAPITokenRevoked  = errors.New("programauth api token revoked")
)

const defaultAPITokenPrefix = "ggpat"

// TokenHasher hashes raw bearer tokens before storage and lookup.
type TokenHasher interface {
	HashAPIToken(raw string) ([]byte, error)
}

type SHA256TokenHasher struct{}

func (SHA256TokenHasher) HashAPIToken(raw string) ([]byte, error) {
	sum := sha256.Sum256([]byte(raw))
	return sum[:], nil
}

type HMACTokenHasher struct{ Pepper []byte }

func (h HMACTokenHasher) HashAPIToken(raw string) ([]byte, error) {
	if len(h.Pepper) == 0 {
		return nil, fmt.Errorf("api token pepper is required")
	}
	mac := hmac.New(sha256.New, h.Pepper)
	_, _ = mac.Write([]byte(raw))
	return mac.Sum(nil), nil
}

// APIToken is the redacted persistent API-token record. TokenHash is retained
// for store implementations but must not be returned by JavaScript/list APIs.
type APIToken struct {
	ID            string
	Name          string
	AgentID       string
	SubjectUserID string
	TokenHash     []byte
	TokenPrefix   string
	CreatedBy     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ExpiresAt     *time.Time
	LastUsedAt    *time.Time
	RevokedAt     *time.Time
	Grants        gojahttp.GrantSet
}

func (t APIToken) Revoked() bool { return t.RevokedAt != nil }

func (t APIToken) Expired(now time.Time) bool {
	return t.ExpiresAt != nil && !now.Before(*t.ExpiresAt)
}

func (t APIToken) CredentialHint() string {
	if t.TokenPrefix == "" {
		return ""
	}
	return defaultAPITokenPrefix + "_" + t.TokenPrefix
}

// APITokenView is a list/detail-safe API-token projection with no raw value or
// hash.
type APITokenView struct {
	ID             string
	Name           string
	AgentID        string
	SubjectUserID  string
	TokenPrefix    string
	CredentialHint string
	CreatedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ExpiresAt      *time.Time
	LastUsedAt     *time.Time
	RevokedAt      *time.Time
	Scopes         []string
}

type APITokenIssueSpec struct {
	ID            string
	Name          string
	AgentID       string
	SubjectUserID string
	CreatedBy     string
	ExpiresAt     *time.Time
	Grants        gojahttp.GrantSet
}

type IssuedAPIToken struct {
	Token APITokenView
	Value string
}

type APITokenQuery struct {
	AgentID        string
	SubjectUserID  string
	IncludeRevoked bool
}

type APITokenStore interface {
	CreateAPIToken(ctx context.Context, token APIToken) (APIToken, error)
	GetAPITokenByID(ctx context.Context, id string) (APIToken, error)
	FindAPITokenByPrefix(ctx context.Context, prefix string) ([]APIToken, error)
	ListAPITokens(ctx context.Context, query APITokenQuery) ([]APIToken, error)
	RevokeAPIToken(ctx context.Context, id string, revokedAt time.Time) (APIToken, error)
	TouchAPIToken(ctx context.Context, id string, usedAt time.Time) error
}

type APITokenService struct {
	Store  APITokenStore
	Agents AgentService
	Hasher TokenHasher
	Now    func() time.Time
	NewID  func() (string, error)
	Random func(n int) ([]byte, error)
}

func (s APITokenService) IssueAPIToken(ctx context.Context, spec APITokenIssueSpec) (IssuedAPIToken, error) {
	if s.Store == nil {
		return IssuedAPIToken{}, fmt.Errorf("programauth api token store is required")
	}
	token, err := s.normalizeIssueSpec(ctx, spec)
	if err != nil {
		return IssuedAPIToken{}, err
	}
	value, prefix, err := s.newRawAPIToken()
	if err != nil {
		return IssuedAPIToken{}, err
	}
	token.TokenPrefix = prefix
	token.TokenHash, err = s.hasher().HashAPIToken(value)
	if err != nil {
		return IssuedAPIToken{}, err
	}
	created, err := s.Store.CreateAPIToken(ctx, token)
	if err != nil {
		return IssuedAPIToken{}, err
	}
	return IssuedAPIToken{Token: APITokenToView(created), Value: value}, nil
}

func (s APITokenService) ListAPITokens(ctx context.Context, query APITokenQuery) ([]APITokenView, error) {
	if s.Store == nil {
		return nil, fmt.Errorf("programauth api token store is required")
	}
	query.AgentID = strings.TrimSpace(query.AgentID)
	query.SubjectUserID = strings.TrimSpace(query.SubjectUserID)
	tokens, err := s.Store.ListAPITokens(ctx, query)
	if err != nil {
		return nil, err
	}
	out := make([]APITokenView, len(tokens))
	for i, token := range tokens {
		out[i] = APITokenToView(token)
	}
	return out, nil
}

func (s APITokenService) RevokeAPIToken(ctx context.Context, id string) (APITokenView, error) {
	if s.Store == nil {
		return APITokenView{}, fmt.Errorf("programauth api token store is required")
	}
	token, err := s.Store.RevokeAPIToken(ctx, strings.TrimSpace(id), s.now())
	if err != nil {
		return APITokenView{}, err
	}
	return APITokenToView(token), nil
}

func (s APITokenService) AuthenticateBearer(ctx context.Context, raw string, _ gojahttp.SecuritySpec) (gojahttp.AuthResult, error) {
	if s.Store == nil {
		return gojahttp.AuthResult{}, fmt.Errorf("programauth api token store is required")
	}
	prefix, err := PrefixFromAPIToken(raw)
	if err != nil {
		return gojahttp.AuthResult{}, fmt.Errorf("%w: invalid bearer token", gojahttp.ErrUnauthenticated)
	}
	hash, err := s.hasher().HashAPIToken(raw)
	if err != nil {
		return gojahttp.AuthResult{}, err
	}
	candidates, err := s.Store.FindAPITokenByPrefix(ctx, prefix)
	if err != nil {
		return gojahttp.AuthResult{}, err
	}
	now := s.now()
	for _, token := range candidates {
		if subtle.ConstantTimeCompare(token.TokenHash, hash) != 1 {
			continue
		}
		if token.Revoked() {
			return gojahttp.AuthResult{}, fmt.Errorf("%w: %v", gojahttp.ErrUnauthenticated, ErrAPITokenRevoked)
		}
		if token.Expired(now) {
			return gojahttp.AuthResult{}, fmt.Errorf("%w: %v", gojahttp.ErrUnauthenticated, ErrAPITokenExpired)
		}
		agent, err := s.Agents.GetAgent(ctx, token.AgentID)
		if err != nil {
			return gojahttp.AuthResult{}, fmt.Errorf("%w: %v", gojahttp.ErrUnauthenticated, err)
		}
		_ = s.Store.TouchAPIToken(ctx, token.ID, now)
		grants := token.Grants.Clone()
		return gojahttp.AuthResult{
			Actor:          agent.Actor(),
			Method:         gojahttp.AuthMethodAPIToken,
			PrincipalKind:  gojahttp.PrincipalKindAgent,
			PrincipalID:    agent.ID,
			CredentialID:   token.ID,
			CredentialHint: token.CredentialHint(),
			Grants:         grants,
			Scopes:         grants.ScopeStrings(),
			CSRFRequired:   false,
		}, nil
	}
	return gojahttp.AuthResult{}, fmt.Errorf("%w: invalid bearer token", gojahttp.ErrUnauthenticated)
}

func (s APITokenService) normalizeIssueSpec(ctx context.Context, spec APITokenIssueSpec) (APIToken, error) {
	now := s.now()
	token := APIToken{
		ID:            strings.TrimSpace(spec.ID),
		Name:          strings.TrimSpace(spec.Name),
		AgentID:       strings.TrimSpace(spec.AgentID),
		SubjectUserID: strings.TrimSpace(spec.SubjectUserID),
		CreatedBy:     strings.TrimSpace(spec.CreatedBy),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if token.Name == "" {
		return APIToken{}, fmt.Errorf("api token name is required")
	}
	if token.AgentID == "" {
		return APIToken{}, fmt.Errorf("api token agent id is required")
	}
	if _, err := s.Agents.GetAgent(ctx, token.AgentID); err != nil {
		return APIToken{}, err
	}
	if spec.ExpiresAt != nil {
		expiresAt := spec.ExpiresAt.UTC()
		if !expiresAt.After(now) {
			return APIToken{}, fmt.Errorf("api token expiration must be in the future")
		}
		token.ExpiresAt = &expiresAt
	}
	policy, err := spec.Grants.Normalize()
	if err != nil {
		return APIToken{}, err
	}
	token.Grants = policy
	if token.ID == "" {
		token.ID, err = s.newID()
		if err != nil {
			return APIToken{}, err
		}
	}
	return token, nil
}

func (s APITokenService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s APITokenService) newID() (string, error) {
	if s.NewID != nil {
		return s.NewID()
	}
	buf, err := s.random(12)
	if err != nil {
		return "", err
	}
	return "tok_" + hex.EncodeToString(buf), nil
}

func (s APITokenService) newRawAPIToken() (string, string, error) {
	prefixBytes, err := s.random(4)
	if err != nil {
		return "", "", err
	}
	secretBytes, err := s.random(32)
	if err != nil {
		return "", "", err
	}
	prefix := hex.EncodeToString(prefixBytes)
	value := defaultAPITokenPrefix + "_" + prefix + "_" + hex.EncodeToString(secretBytes)
	return value, prefix, nil
}

func (s APITokenService) random(n int) ([]byte, error) {
	if s.Random != nil {
		return s.Random(n)
	}
	buf := make([]byte, n)
	_, err := rand.Read(buf)
	return buf, err
}

func (s APITokenService) hasher() TokenHasher {
	if s.Hasher != nil {
		return s.Hasher
	}
	return SHA256TokenHasher{}
}

func APITokenToView(token APIToken) APITokenView {
	return APITokenView{
		ID:             token.ID,
		Name:           token.Name,
		AgentID:        token.AgentID,
		SubjectUserID:  token.SubjectUserID,
		TokenPrefix:    token.TokenPrefix,
		CredentialHint: token.CredentialHint(),
		CreatedBy:      token.CreatedBy,
		CreatedAt:      token.CreatedAt,
		UpdatedAt:      token.UpdatedAt,
		ExpiresAt:      cloneTimePtr(token.ExpiresAt),
		LastUsedAt:     cloneTimePtr(token.LastUsedAt),
		RevokedAt:      cloneTimePtr(token.RevokedAt),
		Scopes:         token.Grants.ScopeStrings(),
	}
}

func PrefixFromAPIToken(raw string) (string, error) {
	parts := strings.Split(strings.TrimSpace(raw), "_")
	if len(parts) != 3 || parts[0] != defaultAPITokenPrefix || parts[1] == "" || parts[2] == "" {
		return "", fmt.Errorf("invalid api token format")
	}
	if _, err := hex.DecodeString(parts[1]); err != nil {
		return "", fmt.Errorf("invalid api token prefix")
	}
	if _, err := hex.DecodeString(parts[2]); err != nil {
		return "", fmt.Errorf("invalid api token secret")
	}
	return parts[1], nil
}

// BearerFromHeader extracts an Authorization-header bearer token and rejects
// alternate transports for planned route authentication.
func BearerFromHeader(r *http.Request) (string, bool, error) {
	if r == nil {
		return "", false, nil
	}
	if r.URL != nil && r.URL.Query().Has("access_token") {
		return "", false, fmt.Errorf("%w: access_token query parameter is not accepted", gojahttp.ErrUnauthenticated)
	}
	values := r.Header.Values("Authorization")
	if len(values) == 0 {
		return "", false, nil
	}
	if len(values) != 1 {
		return "", false, fmt.Errorf("%w: duplicate authorization header", gojahttp.ErrUnauthenticated)
	}
	fields := strings.Fields(values[0])
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") || strings.TrimSpace(fields[1]) == "" {
		return "", false, fmt.Errorf("%w: malformed bearer token", gojahttp.ErrUnauthenticated)
	}
	return fields[1], true, nil
}

func cloneAPIToken(token APIToken) APIToken {
	out := token
	out.TokenHash = append([]byte(nil), token.TokenHash...)
	out.Grants = token.Grants.Clone()
	out.ExpiresAt = cloneTimePtr(token.ExpiresAt)
	out.LastUsedAt = cloneTimePtr(token.LastUsedAt)
	out.RevokedAt = cloneTimePtr(token.RevokedAt)
	return out
}

func cloneTimePtr(in *time.Time) *time.Time {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
