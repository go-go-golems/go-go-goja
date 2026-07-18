package programauth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

var (
	ErrAccessTokenNotFound  = errors.New("programauth access token not found")
	ErrAccessTokenExpired   = errors.New("programauth access token expired")
	ErrAccessTokenRevoked   = errors.New("programauth access token revoked")
	ErrRefreshTokenNotFound = errors.New("programauth refresh token not found")
	ErrRefreshTokenExpired  = errors.New("programauth refresh token expired")
	ErrRefreshTokenRevoked  = errors.New("programauth refresh token revoked")
	ErrRefreshTokenUsed     = errors.New("programauth refresh token already used")
)

const (
	defaultAccessTokenPrefix  = "ggat"
	defaultRefreshTokenPrefix = "ggrt"
	defaultAccessTokenTTL     = 15 * time.Minute
	defaultRefreshTokenTTL    = 30 * 24 * time.Hour
)

// AccessToken is a short-lived bearer credential. It can authenticate planned
// routes. Raw values are never stored; only TokenHash and TokenPrefix are kept.
type AccessToken struct {
	ID            string
	AgentID       string
	SubjectUserID string
	FamilyID      string
	TokenHash     []byte
	TokenPrefix   string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ExpiresAt     time.Time
	LastUsedAt    *time.Time
	RevokedAt     *time.Time
	Grants        gojahttp.GrantSet
}

func (t AccessToken) Revoked() bool { return t.RevokedAt != nil }

func (t AccessToken) Expired(now time.Time) bool { return !now.Before(t.ExpiresAt) }

func (t AccessToken) CredentialHint() string {
	if t.TokenPrefix == "" {
		return ""
	}
	return defaultAccessTokenPrefix + "_" + t.TokenPrefix
}

// RefreshToken is a rotating credential that can issue replacement access and
// refresh tokens. It must not authenticate planned routes directly.
type RefreshToken struct {
	ID            string
	AgentID       string
	SubjectUserID string
	FamilyID      string
	Generation    int
	TokenHash     []byte
	TokenPrefix   string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ExpiresAt     time.Time
	UsedAt        *time.Time
	RevokedAt     *time.Time
	ReplacedByID  string
	Grants        gojahttp.GrantSet
}

func (t RefreshToken) Revoked() bool { return t.RevokedAt != nil }

func (t RefreshToken) Used() bool { return t.UsedAt != nil }

func (t RefreshToken) Expired(now time.Time) bool { return !now.Before(t.ExpiresAt) }

func (t RefreshToken) CredentialHint() string {
	if t.TokenPrefix == "" {
		return ""
	}
	return defaultRefreshTokenPrefix + "_" + t.TokenPrefix
}

type AccessTokenView struct {
	ID             string
	AgentID        string
	SubjectUserID  string
	FamilyID       string
	TokenPrefix    string
	CredentialHint string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ExpiresAt      time.Time
	LastUsedAt     *time.Time
	RevokedAt      *time.Time
	Scopes         []string
}

type RefreshTokenView struct {
	ID             string
	AgentID        string
	SubjectUserID  string
	FamilyID       string
	Generation     int
	TokenPrefix    string
	CredentialHint string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ExpiresAt      time.Time
	UsedAt         *time.Time
	RevokedAt      *time.Time
	ReplacedByID   string
	Scopes         []string
}

type OAuthTokenIssueSpec struct {
	AgentID       string
	SubjectUserID string
	FamilyID      string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	Grants        gojahttp.GrantSet
}

type IssuedOAuthTokenPair struct {
	AccessToken  AccessTokenView
	AccessValue  string
	RefreshToken RefreshTokenView
	RefreshValue string
}

type AccessTokenStore interface {
	CreateAccessToken(ctx context.Context, token AccessToken) (AccessToken, error)
	DeleteAccessToken(ctx context.Context, id string) error
	PurgeExpiredAccessTokens(ctx context.Context, before time.Time) (int, error)
	FindAccessTokenByPrefix(ctx context.Context, prefix string) ([]AccessToken, error)
	TouchAccessToken(ctx context.Context, id string, usedAt time.Time) error
}

type RefreshTokenQuery struct {
	SubjectUserID string
	FamilyID      string
}

type RefreshTokenStore interface {
	CreateRefreshToken(ctx context.Context, token RefreshToken) (RefreshToken, error)
	ListRefreshTokens(ctx context.Context, query RefreshTokenQuery) ([]RefreshToken, error)
	FindRefreshTokenByPrefix(ctx context.Context, prefix string) ([]RefreshToken, error)
	RotateRefreshToken(ctx context.Context, currentID string, next RefreshToken, usedAt time.Time) (RefreshToken, RefreshToken, error)
	RevokeRefreshTokenFamily(ctx context.Context, familyID string, revokedAt time.Time) error
	PurgeExpiredRefreshTokens(ctx context.Context, before time.Time) (int, error)
}

// OAuthTokenPairStore is an optional capability for stores which can persist
// both members of an OAuth token pair in one transaction. Hosts wire it only
// when the access- and refresh-token stores share the same transactional
// database. The separate store interfaces remain useful for lightweight and
// in-memory deployments.
type OAuthTokenPairStore interface {
	CreateOAuthTokenPair(ctx context.Context, access AccessToken, refresh RefreshToken) (AccessToken, RefreshToken, error)
	RotateOAuthTokenPair(ctx context.Context, currentRefreshID string, access AccessToken, nextRefresh RefreshToken, usedAt time.Time) (AccessToken, RefreshToken, error)
}

type OAuthTokenService struct {
	AccessTokens  AccessTokenStore
	RefreshTokens RefreshTokenStore
	PairStore     OAuthTokenPairStore
	Agents        AgentService
	Hasher        TokenHasher
	Now           func() time.Time
	NewID         func(prefix string) (string, error)
	Random        func(n int) ([]byte, error)
}

func (s OAuthTokenService) IssueTokenPair(ctx context.Context, spec OAuthTokenIssueSpec) (IssuedOAuthTokenPair, error) {
	if s.AccessTokens == nil {
		return IssuedOAuthTokenPair{}, fmt.Errorf("programauth access token store is required")
	}
	if s.RefreshTokens == nil {
		return IssuedOAuthTokenPair{}, fmt.Errorf("programauth refresh token store is required")
	}
	access, refresh, err := s.newPairRecords(ctx, spec, 1, "")
	if err != nil {
		return IssuedOAuthTokenPair{}, err
	}
	accessValue, accessPrefix, err := s.newRawToken(defaultAccessTokenPrefix)
	if err != nil {
		return IssuedOAuthTokenPair{}, err
	}
	refreshValue, refreshPrefix, err := s.newRawToken(defaultRefreshTokenPrefix)
	if err != nil {
		return IssuedOAuthTokenPair{}, err
	}
	access.TokenPrefix = accessPrefix
	access.TokenHash, err = s.hasher().HashAPIToken(accessValue)
	if err != nil {
		return IssuedOAuthTokenPair{}, err
	}
	refresh.TokenPrefix = refreshPrefix
	refresh.TokenHash, err = s.hasher().HashAPIToken(refreshValue)
	if err != nil {
		return IssuedOAuthTokenPair{}, err
	}
	var createdAccess AccessToken
	var createdRefresh RefreshToken
	if s.PairStore != nil {
		createdAccess, createdRefresh, err = s.PairStore.CreateOAuthTokenPair(ctx, access, refresh)
	} else {
		createdAccess, err = s.AccessTokens.CreateAccessToken(ctx, access)
		if err == nil {
			createdRefresh, err = s.RefreshTokens.CreateRefreshToken(ctx, refresh)
		}
		if err != nil {
			// A non-transactional pair store must not leave an unreturned access
			// token behind when refresh-token persistence fails.
			if createdAccess.ID != "" {
				_ = s.AccessTokens.DeleteAccessToken(ctx, createdAccess.ID)
			}
		}
	}
	if err != nil {
		return IssuedOAuthTokenPair{}, err
	}
	return IssuedOAuthTokenPair{AccessToken: AccessTokenToView(createdAccess), AccessValue: accessValue, RefreshToken: RefreshTokenToView(createdRefresh), RefreshValue: refreshValue}, nil
}

func (s OAuthTokenService) RefreshTokenPair(ctx context.Context, rawRefreshToken string, accessTTL time.Duration, refreshTTL time.Duration) (IssuedOAuthTokenPair, error) {
	if s.AccessTokens == nil {
		return IssuedOAuthTokenPair{}, fmt.Errorf("programauth access token store is required")
	}
	if s.RefreshTokens == nil {
		return IssuedOAuthTokenPair{}, fmt.Errorf("programauth refresh token store is required")
	}
	current, err := s.lookupRefreshToken(ctx, rawRefreshToken)
	if err != nil {
		return IssuedOAuthTokenPair{}, err
	}
	now := s.now()
	if current.Revoked() {
		return IssuedOAuthTokenPair{}, fmt.Errorf("%w: %v", gojahttp.ErrUnauthenticated, ErrRefreshTokenRevoked)
	}
	if current.Expired(now) {
		return IssuedOAuthTokenPair{}, fmt.Errorf("%w: %v", gojahttp.ErrUnauthenticated, ErrRefreshTokenExpired)
	}
	if current.Used() {
		_ = s.RefreshTokens.RevokeRefreshTokenFamily(ctx, current.FamilyID, now)
		return IssuedOAuthTokenPair{}, fmt.Errorf("%w: %v", gojahttp.ErrUnauthenticated, ErrRefreshTokenUsed)
	}
	spec := OAuthTokenIssueSpec{AgentID: current.AgentID, SubjectUserID: current.SubjectUserID, FamilyID: current.FamilyID, AccessTTL: accessTTL, RefreshTTL: refreshTTL, Grants: current.Grants.Clone()}
	access, nextRefresh, err := s.newPairRecords(ctx, spec, current.Generation+1, "")
	if err != nil {
		return IssuedOAuthTokenPair{}, err
	}
	accessValue, accessPrefix, err := s.newRawToken(defaultAccessTokenPrefix)
	if err != nil {
		return IssuedOAuthTokenPair{}, err
	}
	refreshValue, refreshPrefix, err := s.newRawToken(defaultRefreshTokenPrefix)
	if err != nil {
		return IssuedOAuthTokenPair{}, err
	}
	access.TokenPrefix = accessPrefix
	access.TokenHash, err = s.hasher().HashAPIToken(accessValue)
	if err != nil {
		return IssuedOAuthTokenPair{}, err
	}
	nextRefresh.TokenPrefix = refreshPrefix
	nextRefresh.TokenHash, err = s.hasher().HashAPIToken(refreshValue)
	if err != nil {
		return IssuedOAuthTokenPair{}, err
	}
	var createdAccess AccessToken
	var rotatedRefresh RefreshToken
	if s.PairStore != nil {
		createdAccess, rotatedRefresh, err = s.PairStore.RotateOAuthTokenPair(ctx, current.ID, access, nextRefresh, now)
	} else {
		// Persist the new access token before consuming the current refresh token.
		// If that insert fails, callers can retry with the original refresh token.
		createdAccess, err = s.AccessTokens.CreateAccessToken(ctx, access)
		if err == nil {
			_, rotatedRefresh, err = s.RefreshTokens.RotateRefreshToken(ctx, current.ID, nextRefresh, now)
		}
	}
	if err != nil {
		// A transactional PairStore has already rolled back. Separate stores use
		// best-effort compensation because no cross-store transaction exists.
		if s.PairStore == nil && createdAccess.ID != "" {
			if deleteErr := s.AccessTokens.DeleteAccessToken(ctx, createdAccess.ID); deleteErr != nil {
				return IssuedOAuthTokenPair{}, fmt.Errorf("rotate refresh token: %w (access-token cleanup failed: %v)", err, deleteErr)
			}
		}
		if errors.Is(err, ErrRefreshTokenUsed) {
			_ = s.RefreshTokens.RevokeRefreshTokenFamily(ctx, current.FamilyID, now)
			return IssuedOAuthTokenPair{}, fmt.Errorf("%w: %v", gojahttp.ErrUnauthenticated, ErrRefreshTokenUsed)
		}
		return IssuedOAuthTokenPair{}, err
	}
	return IssuedOAuthTokenPair{AccessToken: AccessTokenToView(createdAccess), AccessValue: accessValue, RefreshToken: RefreshTokenToView(rotatedRefresh), RefreshValue: refreshValue}, nil
}

// RevokeRefreshToken revokes every refresh credential in the family identified
// by rawRefreshToken. It deliberately does not claim immediate access-token
// revocation: already-issued access tokens remain bounded by their short TTL.
func (s OAuthTokenService) RevokeRefreshToken(ctx context.Context, rawRefreshToken string) error {
	if s.RefreshTokens == nil {
		return fmt.Errorf("programauth refresh token store is required")
	}
	current, err := s.lookupRefreshToken(ctx, rawRefreshToken)
	if err != nil {
		return err
	}
	return s.RefreshTokens.RevokeRefreshTokenFamily(ctx, current.FamilyID, s.now())
}

// ListOwnedRefreshTokens returns only redacted metadata for one local user.
// PurgeExpiredCredentials removes expired/revoked credential rows older than before.
func (s OAuthTokenService) PurgeExpiredCredentials(ctx context.Context, before time.Time) (int, error) {
	if s.AccessTokens == nil || s.RefreshTokens == nil {
		return 0, fmt.Errorf("programauth token stores are required")
	}
	a, err := s.AccessTokens.PurgeExpiredAccessTokens(ctx, before)
	if err != nil {
		return 0, err
	}
	r, err := s.RefreshTokens.PurgeExpiredRefreshTokens(ctx, before)
	return a + r, err
}

func (s OAuthTokenService) ListOwnedRefreshTokens(ctx context.Context, userID string) ([]RefreshTokenView, error) {
	if s.RefreshTokens == nil {
		return nil, fmt.Errorf("programauth refresh token store is required")
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("subject user id is required")
	}
	tokens, err := s.RefreshTokens.ListRefreshTokens(ctx, RefreshTokenQuery{SubjectUserID: userID})
	if err != nil {
		return nil, err
	}
	out := make([]RefreshTokenView, 0, len(tokens))
	for _, token := range tokens {
		out = append(out, RefreshTokenToView(token))
	}
	return out, nil
}

// RevokeOwnedRefreshTokenFamily revokes a family only after confirming ownership.
func (s OAuthTokenService) RevokeOwnedRefreshTokenFamily(ctx context.Context, userID, familyID string) error {
	if s.RefreshTokens == nil {
		return fmt.Errorf("programauth refresh token store is required")
	}
	tokens, err := s.RefreshTokens.ListRefreshTokens(ctx, RefreshTokenQuery{SubjectUserID: userID, FamilyID: familyID})
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return ErrRefreshTokenNotFound
	}
	return s.RefreshTokens.RevokeRefreshTokenFamily(ctx, familyID, s.now())
}

func (s OAuthTokenService) AuthenticateBearer(ctx context.Context, raw string, _ gojahttp.SecuritySpec) (gojahttp.AuthResult, error) {
	if s.AccessTokens == nil {
		return gojahttp.AuthResult{}, fmt.Errorf("programauth access token store is required")
	}
	prefix, err := PrefixFromAccessToken(raw)
	if err != nil {
		return gojahttp.AuthResult{}, fmt.Errorf("%w: invalid access token", gojahttp.ErrUnauthenticated)
	}
	hash, err := s.hasher().HashAPIToken(raw)
	if err != nil {
		return gojahttp.AuthResult{}, err
	}
	candidates, err := s.AccessTokens.FindAccessTokenByPrefix(ctx, prefix)
	if err != nil {
		return gojahttp.AuthResult{}, err
	}
	now := s.now()
	for _, token := range candidates {
		if subtle.ConstantTimeCompare(token.TokenHash, hash) != 1 {
			continue
		}
		if token.Revoked() {
			return gojahttp.AuthResult{}, fmt.Errorf("%w: %v", gojahttp.ErrUnauthenticated, ErrAccessTokenRevoked)
		}
		if token.Expired(now) {
			return gojahttp.AuthResult{}, fmt.Errorf("%w: %v", gojahttp.ErrUnauthenticated, ErrAccessTokenExpired)
		}
		agent, err := s.Agents.GetAgent(ctx, token.AgentID)
		if err != nil {
			return gojahttp.AuthResult{}, fmt.Errorf("%w: %v", gojahttp.ErrUnauthenticated, err)
		}
		_ = s.AccessTokens.TouchAccessToken(ctx, token.ID, now)
		grants := token.Grants.Clone()
		return gojahttp.AuthResult{Actor: agent.Actor(), Method: gojahttp.AuthMethodAccessToken, PrincipalKind: gojahttp.PrincipalKindAgent, PrincipalID: agent.ID, CredentialID: token.ID, CredentialHint: token.CredentialHint(), Grants: grants, Scopes: grants.ScopeStrings(), CSRFRequired: false}, nil
	}
	return gojahttp.AuthResult{}, fmt.Errorf("%w: invalid access token", gojahttp.ErrUnauthenticated)
}

func (s OAuthTokenService) lookupRefreshToken(ctx context.Context, raw string) (RefreshToken, error) {
	prefix, err := PrefixFromRefreshToken(raw)
	if err != nil {
		return RefreshToken{}, fmt.Errorf("%w: invalid refresh token", gojahttp.ErrUnauthenticated)
	}
	hash, err := s.hasher().HashAPIToken(raw)
	if err != nil {
		return RefreshToken{}, err
	}
	candidates, err := s.RefreshTokens.FindRefreshTokenByPrefix(ctx, prefix)
	if err != nil {
		return RefreshToken{}, err
	}
	for _, token := range candidates {
		if subtle.ConstantTimeCompare(token.TokenHash, hash) == 1 {
			return token, nil
		}
	}
	return RefreshToken{}, fmt.Errorf("%w: invalid refresh token", gojahttp.ErrUnauthenticated)
}

func (s OAuthTokenService) newPairRecords(ctx context.Context, spec OAuthTokenIssueSpec, generation int, accessID string) (AccessToken, RefreshToken, error) {
	now := s.now()
	agentID := strings.TrimSpace(spec.AgentID)
	if agentID == "" {
		return AccessToken{}, RefreshToken{}, fmt.Errorf("oauth token agent id is required")
	}
	if _, err := s.Agents.GetAgent(ctx, agentID); err != nil {
		return AccessToken{}, RefreshToken{}, err
	}
	policy, err := spec.Grants.Normalize()
	if err != nil {
		return AccessToken{}, RefreshToken{}, err
	}
	familyID := strings.TrimSpace(spec.FamilyID)
	if familyID == "" {
		familyID, err = s.newID("tfam")
		if err != nil {
			return AccessToken{}, RefreshToken{}, err
		}
	}
	accessTTL := spec.AccessTTL
	if accessTTL <= 0 {
		accessTTL = defaultAccessTokenTTL
	}
	refreshTTL := spec.RefreshTTL
	if refreshTTL <= 0 {
		refreshTTL = defaultRefreshTokenTTL
	}
	if accessID == "" {
		accessID, err = s.newID("at")
		if err != nil {
			return AccessToken{}, RefreshToken{}, err
		}
	}
	refreshID, err := s.newID("rt")
	if err != nil {
		return AccessToken{}, RefreshToken{}, err
	}
	access := AccessToken{ID: accessID, AgentID: agentID, SubjectUserID: strings.TrimSpace(spec.SubjectUserID), FamilyID: familyID, CreatedAt: now, UpdatedAt: now, ExpiresAt: now.Add(accessTTL), Grants: policy.Clone()}
	refresh := RefreshToken{ID: refreshID, AgentID: agentID, SubjectUserID: strings.TrimSpace(spec.SubjectUserID), FamilyID: familyID, Generation: generation, CreatedAt: now, UpdatedAt: now, ExpiresAt: now.Add(refreshTTL), Grants: policy.Clone()}
	return access, refresh, nil
}

func (s OAuthTokenService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s OAuthTokenService) newID(prefix string) (string, error) {
	if s.NewID != nil {
		return s.NewID(prefix)
	}
	buf, err := s.random(12)
	if err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(buf), nil
}

func (s OAuthTokenService) newRawToken(prefix string) (string, string, error) {
	prefixBytes, err := s.random(4)
	if err != nil {
		return "", "", err
	}
	secretBytes, err := s.random(32)
	if err != nil {
		return "", "", err
	}
	tokenPrefix := hex.EncodeToString(prefixBytes)
	return prefix + "_" + tokenPrefix + "_" + hex.EncodeToString(secretBytes), tokenPrefix, nil
}

func (s OAuthTokenService) random(n int) ([]byte, error) {
	if s.Random != nil {
		return s.Random(n)
	}
	buf := make([]byte, n)
	_, err := rand.Read(buf)
	return buf, err
}

func (s OAuthTokenService) hasher() TokenHasher {
	if s.Hasher != nil {
		return s.Hasher
	}
	return SHA256TokenHasher{}
}

func AccessTokenToView(token AccessToken) AccessTokenView {
	return AccessTokenView{ID: token.ID, AgentID: token.AgentID, SubjectUserID: token.SubjectUserID, FamilyID: token.FamilyID, TokenPrefix: token.TokenPrefix, CredentialHint: token.CredentialHint(), CreatedAt: token.CreatedAt, UpdatedAt: token.UpdatedAt, ExpiresAt: token.ExpiresAt, LastUsedAt: cloneTimePtr(token.LastUsedAt), RevokedAt: cloneTimePtr(token.RevokedAt), Scopes: token.Grants.ScopeStrings()}
}

func RefreshTokenToView(token RefreshToken) RefreshTokenView {
	return RefreshTokenView{ID: token.ID, AgentID: token.AgentID, SubjectUserID: token.SubjectUserID, FamilyID: token.FamilyID, Generation: token.Generation, TokenPrefix: token.TokenPrefix, CredentialHint: token.CredentialHint(), CreatedAt: token.CreatedAt, UpdatedAt: token.UpdatedAt, ExpiresAt: token.ExpiresAt, UsedAt: cloneTimePtr(token.UsedAt), RevokedAt: cloneTimePtr(token.RevokedAt), ReplacedByID: token.ReplacedByID, Scopes: token.Grants.ScopeStrings()}
}

func PrefixFromAccessToken(raw string) (string, error) {
	return prefixFromOpaqueToken(raw, defaultAccessTokenPrefix)
}

func PrefixFromRefreshToken(raw string) (string, error) {
	return prefixFromOpaqueToken(raw, defaultRefreshTokenPrefix)
}

func prefixFromOpaqueToken(raw string, tokenPrefix string) (string, error) {
	parts := strings.Split(strings.TrimSpace(raw), "_")
	if len(parts) != 3 || parts[0] != tokenPrefix || parts[1] == "" || parts[2] == "" {
		return "", fmt.Errorf("invalid token format")
	}
	if _, err := hex.DecodeString(parts[1]); err != nil {
		return "", fmt.Errorf("invalid token prefix")
	}
	if _, err := hex.DecodeString(parts[2]); err != nil {
		return "", fmt.Errorf("invalid token secret")
	}
	return parts[1], nil
}

func cloneAccessToken(token AccessToken) AccessToken {
	out := token
	out.TokenHash = append([]byte(nil), token.TokenHash...)
	out.Grants = token.Grants.Clone()
	out.LastUsedAt = cloneTimePtr(token.LastUsedAt)
	out.RevokedAt = cloneTimePtr(token.RevokedAt)
	return out
}

func cloneRefreshToken(token RefreshToken) RefreshToken {
	out := token
	out.TokenHash = append([]byte(nil), token.TokenHash...)
	out.Grants = token.Grants.Clone()
	out.UsedAt = cloneTimePtr(token.UsedAt)
	out.RevokedAt = cloneTimePtr(token.RevokedAt)
	return out
}
