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
	ErrDeviceNotFound             = errors.New("programauth device authorization not found")
	ErrDeviceExpired              = errors.New("programauth device authorization expired")
	ErrDeviceDenied               = errors.New("programauth device authorization denied")
	ErrDeviceConsumed             = errors.New("programauth device authorization consumed")
	ErrDeviceAuthorizationPending = errors.New("programauth device authorization pending")
	ErrDeviceSlowDown             = errors.New("programauth device authorization slow down")
)

const (
	defaultDeviceCodePrefix = "ggdc"
	defaultDeviceExpiry     = 10 * time.Minute
	defaultDeviceInterval   = 5 * time.Second
)

type DeviceError struct {
	Err      error
	Interval time.Duration
}

func (e DeviceError) Error() string {
	if e.Err == nil {
		return "programauth device error"
	}
	return e.Err.Error()
}

func (e DeviceError) Unwrap() error { return e.Err }

type DeviceAuthorization struct {
	ID                      string
	ClientName              string
	DeviceCodeHash          []byte
	DeviceCodePrefix        string
	UserCodeHash            []byte
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	CreatedAt               time.Time
	UpdatedAt               time.Time
	ExpiresAt               time.Time
	PollInterval            time.Duration
	LastPolledAt            *time.Time
	ApprovedAt              *time.Time
	DeniedAt                *time.Time
	ConsumedAt              *time.Time
	AgentID                 string
	SubjectUserID           string
	TenantID                string
	Grants                  gojahttp.GrantSet
}

func (d DeviceAuthorization) Expired(now time.Time) bool { return !now.Before(d.ExpiresAt) }
func (d DeviceAuthorization) Approved() bool             { return d.ApprovedAt != nil }
func (d DeviceAuthorization) Denied() bool               { return d.DeniedAt != nil }
func (d DeviceAuthorization) Consumed() bool             { return d.ConsumedAt != nil }

type DeviceAuthorizationView struct {
	ID                      string
	ClientName              string
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	CreatedAt               time.Time
	UpdatedAt               time.Time
	ExpiresAt               time.Time
	PollIntervalSeconds     int
	ApprovedAt              *time.Time
	DeniedAt                *time.Time
	ConsumedAt              *time.Time
	AgentID                 string
	SubjectUserID           string
	TenantID                string
	Scopes                  []string
}

type DeviceStartSpec struct {
	ID              string
	ClientName      string
	TenantID        string
	Grants          gojahttp.GrantSet
	ExpiresIn       time.Duration
	PollInterval    time.Duration
	VerificationURI string
}

type StartedDeviceAuthorization struct {
	Device     DeviceAuthorizationView
	DeviceCode string
	UserCode   string
}

type DeviceApprovalSpec struct {
	UserCode      string
	SubjectUserID string
	TenantID      string
	AgentName     string
	AgentKind     AgentKind
	Grants        gojahttp.GrantSet
}

type DeviceAuthorizationStore interface {
	CreateDeviceAuthorization(ctx context.Context, device DeviceAuthorization) (DeviceAuthorization, error)
	FindDeviceAuthorizationByDeviceCodePrefix(ctx context.Context, prefix string) ([]DeviceAuthorization, error)
	GetDeviceAuthorizationByUserCodeHash(ctx context.Context, hash []byte) (DeviceAuthorization, error)
	RecordDevicePoll(ctx context.Context, id string, polledAt time.Time, interval time.Duration) (DeviceAuthorization, error)
	ApproveDeviceAuthorization(ctx context.Context, id string, approved DeviceAuthorization, approvedAt time.Time) (DeviceAuthorization, error)
	DenyDeviceAuthorization(ctx context.Context, id string, deniedAt time.Time) (DeviceAuthorization, error)
	ConsumeDeviceAuthorization(ctx context.Context, id string, consumedAt time.Time) (DeviceAuthorization, error)
}

type DeviceService struct {
	Store           DeviceAuthorizationStore
	Agents          AgentService
	OAuthTokens     OAuthTokenService
	Hasher          TokenHasher
	Now             func() time.Time
	NewID           func(prefix string) (string, error)
	Random          func(n int) ([]byte, error)
	VerificationURI string
}

func (s DeviceService) StartDeviceAuthorization(ctx context.Context, spec DeviceStartSpec) (StartedDeviceAuthorization, error) {
	if s.Store == nil {
		return StartedDeviceAuthorization{}, fmt.Errorf("programauth device authorization store is required")
	}
	now := s.now()
	clientName := strings.TrimSpace(spec.ClientName)
	if clientName == "" {
		return StartedDeviceAuthorization{}, fmt.Errorf("device client name is required")
	}
	policy, err := spec.Grants.Normalize()
	if err != nil {
		return StartedDeviceAuthorization{}, err
	}
	id := strings.TrimSpace(spec.ID)
	if id == "" {
		id, err = s.newID("dev")
		if err != nil {
			return StartedDeviceAuthorization{}, err
		}
	}
	expiresIn := spec.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = defaultDeviceExpiry
	}
	interval := spec.PollInterval
	if interval <= 0 {
		interval = defaultDeviceInterval
	}
	verificationURI := strings.TrimSpace(spec.VerificationURI)
	if verificationURI == "" {
		verificationURI = strings.TrimSpace(s.VerificationURI)
	}
	deviceCode, devicePrefix, err := s.newOpaqueDeviceCode()
	if err != nil {
		return StartedDeviceAuthorization{}, err
	}
	userCode, err := s.newUserCode()
	if err != nil {
		return StartedDeviceAuthorization{}, err
	}
	deviceHash, err := s.hasher().HashAPIToken(deviceCode)
	if err != nil {
		return StartedDeviceAuthorization{}, err
	}
	userHash, err := s.hasher().HashAPIToken(normalizeUserCode(userCode))
	if err != nil {
		return StartedDeviceAuthorization{}, err
	}
	device := DeviceAuthorization{ID: id, ClientName: clientName, DeviceCodeHash: deviceHash, DeviceCodePrefix: devicePrefix, UserCodeHash: userHash, UserCode: userCode, VerificationURI: verificationURI, VerificationURIComplete: verificationURIComplete(verificationURI, userCode), CreatedAt: now, UpdatedAt: now, ExpiresAt: now.Add(expiresIn), PollInterval: interval, TenantID: strings.TrimSpace(spec.TenantID), Grants: policy}
	created, err := s.Store.CreateDeviceAuthorization(ctx, device)
	if err != nil {
		return StartedDeviceAuthorization{}, err
	}
	return StartedDeviceAuthorization{Device: DeviceAuthorizationToView(created), DeviceCode: deviceCode, UserCode: userCode}, nil
}

func (s DeviceService) ApproveDeviceAuthorization(ctx context.Context, spec DeviceApprovalSpec) (DeviceAuthorizationView, error) {
	if s.Store == nil {
		return DeviceAuthorizationView{}, fmt.Errorf("programauth device authorization store is required")
	}
	if s.Agents.Store == nil {
		return DeviceAuthorizationView{}, fmt.Errorf("programauth agent service is required")
	}
	now := s.now()
	device, err := s.lookupDeviceByUserCode(ctx, spec.UserCode)
	if err != nil {
		return DeviceAuthorizationView{}, err
	}
	if err := validateDeviceForApproval(device, now); err != nil {
		return DeviceAuthorizationView{}, err
	}
	grants := device.Grants.Clone()
	if len(spec.Grants.Grants) > 0 {
		grants, err = device.Grants.Intersect(spec.Grants)
		if err != nil {
			return DeviceAuthorizationView{}, err
		}
		if len(grants.Grants) == 0 {
			return DeviceAuthorizationView{}, fmt.Errorf("device approval grants do not intersect requested grants")
		}
	}
	grants, err = grants.Normalize()
	if err != nil {
		return DeviceAuthorizationView{}, err
	}
	tenantID := strings.TrimSpace(spec.TenantID)
	if tenantID == "" {
		tenantID = device.TenantID
	}
	agentName := strings.TrimSpace(spec.AgentName)
	if agentName == "" {
		agentName = device.ClientName
	}
	agentKind := spec.AgentKind
	if agentKind == "" {
		agentKind = AgentKindDevice
	}
	agent, err := s.Agents.CreateAgent(ctx, AgentCreateSpec{Name: agentName, Kind: agentKind, OwnerUserID: strings.TrimSpace(spec.SubjectUserID), TenantID: tenantID, CreatedBy: strings.TrimSpace(spec.SubjectUserID), Policy: grants})
	if err != nil {
		return DeviceAuthorizationView{}, err
	}
	device.AgentID = agent.ID
	device.SubjectUserID = strings.TrimSpace(spec.SubjectUserID)
	device.TenantID = tenantID
	device.Grants = grants
	approved, err := s.Store.ApproveDeviceAuthorization(ctx, device.ID, device, now)
	if err != nil {
		return DeviceAuthorizationView{}, err
	}
	return DeviceAuthorizationToView(approved), nil
}

func (s DeviceService) PollDeviceAuthorization(ctx context.Context, rawDeviceCode string) (IssuedOAuthTokenPair, error) {
	if s.Store == nil {
		return IssuedOAuthTokenPair{}, fmt.Errorf("programauth device authorization store is required")
	}
	if s.OAuthTokens.AccessTokens == nil || s.OAuthTokens.RefreshTokens == nil {
		return IssuedOAuthTokenPair{}, fmt.Errorf("programauth oauth token service is required")
	}
	now := s.now()
	device, err := s.lookupDeviceByDeviceCode(ctx, rawDeviceCode)
	if err != nil {
		return IssuedOAuthTokenPair{}, err
	}
	if device.Expired(now) {
		return IssuedOAuthTokenPair{}, fmt.Errorf("%w: %w", gojahttp.ErrUnauthenticated, ErrDeviceExpired)
	}
	if device.Denied() {
		return IssuedOAuthTokenPair{}, fmt.Errorf("%w: %w", gojahttp.ErrUnauthenticated, ErrDeviceDenied)
	}
	if device.Consumed() {
		return IssuedOAuthTokenPair{}, fmt.Errorf("%w: %w", gojahttp.ErrUnauthenticated, ErrDeviceConsumed)
	}
	if device.LastPolledAt != nil && now.Before(device.LastPolledAt.Add(device.PollInterval)) {
		nextInterval := device.PollInterval + 5*time.Second
		_, _ = s.Store.RecordDevicePoll(ctx, device.ID, now, nextInterval)
		return IssuedOAuthTokenPair{}, DeviceError{Err: ErrDeviceSlowDown, Interval: nextInterval}
	}
	device, err = s.Store.RecordDevicePoll(ctx, device.ID, now, device.PollInterval)
	if err != nil {
		return IssuedOAuthTokenPair{}, err
	}
	if !device.Approved() {
		return IssuedOAuthTokenPair{}, DeviceError{Err: ErrDeviceAuthorizationPending, Interval: device.PollInterval}
	}
	if strings.TrimSpace(device.AgentID) == "" {
		return IssuedOAuthTokenPair{}, fmt.Errorf("approved device authorization has no agent")
	}
	consumed, err := s.Store.ConsumeDeviceAuthorization(ctx, device.ID, now)
	if err != nil {
		return IssuedOAuthTokenPair{}, err
	}
	return s.OAuthTokens.IssueTokenPair(ctx, OAuthTokenIssueSpec{AgentID: consumed.AgentID, SubjectUserID: consumed.SubjectUserID, Grants: consumed.Grants.Clone()})
}

func (s DeviceService) DenyDeviceAuthorization(ctx context.Context, userCode string) (DeviceAuthorizationView, error) {
	if s.Store == nil {
		return DeviceAuthorizationView{}, fmt.Errorf("programauth device authorization store is required")
	}
	device, err := s.lookupDeviceByUserCode(ctx, userCode)
	if err != nil {
		return DeviceAuthorizationView{}, err
	}
	denied, err := s.Store.DenyDeviceAuthorization(ctx, device.ID, s.now())
	if err != nil {
		return DeviceAuthorizationView{}, err
	}
	return DeviceAuthorizationToView(denied), nil
}

func (s DeviceService) lookupDeviceByDeviceCode(ctx context.Context, raw string) (DeviceAuthorization, error) {
	prefix, err := PrefixFromDeviceCode(raw)
	if err != nil {
		return DeviceAuthorization{}, fmt.Errorf("%w: invalid device code", gojahttp.ErrUnauthenticated)
	}
	hash, err := s.hasher().HashAPIToken(strings.TrimSpace(raw))
	if err != nil {
		return DeviceAuthorization{}, err
	}
	candidates, err := s.Store.FindDeviceAuthorizationByDeviceCodePrefix(ctx, prefix)
	if err != nil {
		return DeviceAuthorization{}, err
	}
	for _, device := range candidates {
		if subtle.ConstantTimeCompare(device.DeviceCodeHash, hash) == 1 {
			return device, nil
		}
	}
	return DeviceAuthorization{}, fmt.Errorf("%w: invalid device code", gojahttp.ErrUnauthenticated)
}

func (s DeviceService) lookupDeviceByUserCode(ctx context.Context, raw string) (DeviceAuthorization, error) {
	code := normalizeUserCode(raw)
	if code == "" {
		return DeviceAuthorization{}, fmt.Errorf("device user code is required")
	}
	hash, err := s.hasher().HashAPIToken(code)
	if err != nil {
		return DeviceAuthorization{}, err
	}
	return s.Store.GetDeviceAuthorizationByUserCodeHash(ctx, hash)
}

func validateDeviceForApproval(device DeviceAuthorization, now time.Time) error {
	if device.Expired(now) {
		return ErrDeviceExpired
	}
	if device.Denied() {
		return ErrDeviceDenied
	}
	if device.Consumed() {
		return ErrDeviceConsumed
	}
	if device.Approved() {
		return fmt.Errorf("device authorization already approved")
	}
	return nil
}

func (s DeviceService) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s DeviceService) newID(prefix string) (string, error) {
	if s.NewID != nil {
		return s.NewID(prefix)
	}
	buf, err := s.random(12)
	if err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(buf), nil
}

func (s DeviceService) newOpaqueDeviceCode() (string, string, error) {
	prefixBytes, err := s.random(4)
	if err != nil {
		return "", "", err
	}
	secretBytes, err := s.random(32)
	if err != nil {
		return "", "", err
	}
	prefix := hex.EncodeToString(prefixBytes)
	return defaultDeviceCodePrefix + "_" + prefix + "_" + hex.EncodeToString(secretBytes), prefix, nil
}

func (s DeviceService) newUserCode() (string, error) {
	buf, err := s.random(6)
	if err != nil {
		return "", err
	}
	raw := strings.ToUpper(hex.EncodeToString(buf))
	return raw[0:4] + "-" + raw[4:8] + "-" + raw[8:12], nil
}

func (s DeviceService) random(n int) ([]byte, error) {
	if s.Random != nil {
		return s.Random(n)
	}
	buf := make([]byte, n)
	_, err := rand.Read(buf)
	return buf, err
}

func (s DeviceService) hasher() TokenHasher {
	if s.Hasher != nil {
		return s.Hasher
	}
	return SHA256TokenHasher{}
}

func DeviceAuthorizationToView(device DeviceAuthorization) DeviceAuthorizationView {
	return DeviceAuthorizationView{ID: device.ID, ClientName: device.ClientName, UserCode: device.UserCode, VerificationURI: device.VerificationURI, VerificationURIComplete: device.VerificationURIComplete, CreatedAt: device.CreatedAt, UpdatedAt: device.UpdatedAt, ExpiresAt: device.ExpiresAt, PollIntervalSeconds: int(device.PollInterval / time.Second), ApprovedAt: cloneTimePtr(device.ApprovedAt), DeniedAt: cloneTimePtr(device.DeniedAt), ConsumedAt: cloneTimePtr(device.ConsumedAt), AgentID: device.AgentID, SubjectUserID: device.SubjectUserID, TenantID: device.TenantID, Scopes: device.Grants.ScopeStrings()}
}

func PrefixFromDeviceCode(raw string) (string, error) {
	return prefixFromOpaqueToken(raw, defaultDeviceCodePrefix)
}

func normalizeUserCode(raw string) string {
	replacer := strings.NewReplacer("-", "", " ", "")
	return strings.ToUpper(replacer.Replace(strings.TrimSpace(raw)))
}

func verificationURIComplete(uri, userCode string) string {
	if strings.TrimSpace(uri) == "" {
		return ""
	}
	separator := "?"
	if strings.Contains(uri, "?") {
		separator = "&"
	}
	return uri + separator + "user_code=" + userCode
}

func cloneDeviceAuthorization(device DeviceAuthorization) DeviceAuthorization {
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
