package gojahttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// SecurityMode describes the route-level security envelope that must run before
// a planned JavaScript handler is invoked.
type SecurityMode string

const (
	SecurityModePublic SecurityMode = "public"
	SecurityModeUser   SecurityMode = "user"
)

// AuthMethod identifies the credential family that authenticated a request.
// Values are deliberately protocol-level and non-secret so they can be safely
// exposed in planned-route context and audit metadata.
type AuthMethod string

const (
	AuthMethodNone        AuthMethod = "none"
	AuthMethodSession     AuthMethod = "session"
	AuthMethodAPIToken    AuthMethod = "apiToken"
	AuthMethodAccessToken AuthMethod = "accessToken"
)

// PrincipalKind identifies the durable principal represented by an auth
// result. Credentials prove possession; principals carry ownership and policy.
type PrincipalKind string

const (
	PrincipalKindUser    PrincipalKind = "user"
	PrincipalKindAgent   PrincipalKind = "agent"
	PrincipalKindService PrincipalKind = "service"
)

// ValueSourceKind identifies where a route-plan value should be read from.
type ValueSourceKind string

const (
	ValueSourceParam   ValueSourceKind = "param"
	ValueSourceQuery   ValueSourceKind = "query"
	ValueSourceBody    ValueSourceKind = "body"
	ValueSourceLiteral ValueSourceKind = "literal"
)

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("not found")
	ErrCSRF            = errors.New("csrf invalid")
	ErrRateLimited     = errors.New("rate limit exceeded")
)

// RoutePlan is the Go-owned security contract compiled by the Express fluent
// route builder at registration time.
type RoutePlan struct {
	Name       string
	Method     string
	Pattern    string
	Security   SecuritySpec
	Resources  []ResourceSpec
	Action     string
	CSRF       CSRFSpec
	Audit      AuditSpec
	RateLimits []RateLimitSpec
}

// AuthRequirement constrains which authenticated principal families may enter
// a planned route. Empty fields are wildcards, so {PrincipalKind: "agent"}
// accepts any current or future credential family that authenticates as an
// agent, while {Method: "session", PrincipalKind: "user"} is browser-session
// user auth only.
type AuthRequirement struct {
	Method        AuthMethod
	PrincipalKind PrincipalKind
}

// SecuritySpec describes who may enter a planned route.
type SecuritySpec struct {
	Mode             SecurityMode
	Required         bool
	MFAFreshWithin   time.Duration
	AuthRequirements []AuthRequirement
}

// ValueSource describes a typed value extraction from the HTTP adapter layer.
// Resource resolvers receive the resolved value, not raw req.params maps.
type ValueSource struct {
	Kind  ValueSourceKind
	Key   string
	Value string
}

// ResourceSpec describes which resource a route touches and how its identity is
// extracted from the request adapter layer.
type ResourceSpec struct {
	Name      string
	Type      string
	ID        ValueSource
	Tenant    *ValueSource
	MustExist bool
}

// CSRFSpec describes whether a planned route requires host-owned CSRF
// verification before the JavaScript handler runs.
type CSRFSpec struct {
	Required bool
}

// AuditSpec describes the host-owned audit event emitted for a planned route.
type AuditSpec struct {
	Event string
}

// Actor is the minimal host-owned authenticated principal exposed to planned
// route handlers.
type Actor struct {
	ID        string         `json:"id"`
	Kind      string         `json:"kind"`
	TenantIDs []string       `json:"tenantIds,omitempty"`
	Claims    map[string]any `json:"claims,omitempty"`
}

type actorContextKey struct{}

// ContextWithActor returns a child context carrying the authenticated actor for
// trusted Go and native-module code invoked by a planned route. The actor is
// only installed after host-side enforcement succeeds.
func ContextWithActor(ctx context.Context, actor *Actor) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if actor == nil {
		return ctx
	}
	return context.WithValue(ctx, actorContextKey{}, actor)
}

// ActorFromContext returns the authenticated planned-route actor, if present.
func ActorFromContext(ctx context.Context) (*Actor, bool) {
	if ctx == nil {
		return nil, false
	}
	actor, ok := ctx.Value(actorContextKey{}).(*Actor)
	return actor, ok && actor != nil
}

// ResourceRef is the minimal host-owned resource handle exposed to planned
// route handlers after resolution and authorization.
type ResourceRef struct {
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	ID       string         `json:"id"`
	TenantID string         `json:"tenantId,omitempty"`
	Claims   map[string]any `json:"claims,omitempty"`
}

// AuthResult is the non-secret outcome of authenticating a planned-route
// request. Raw bearer tokens, token hashes, refresh-token identifiers, device
// codes, and other credentials must never be stored here.
type AuthResult struct {
	Actor          *Actor
	Method         AuthMethod
	PrincipalKind  PrincipalKind
	PrincipalID    string
	CredentialID   string
	CredentialHint string
	Grants         GrantSet
	Scopes         []string
	CSRFRequired   bool
}

type AuthOptions struct {
	Authenticator Authenticator
	Resources     ResourceResolver
	Authorizer    Authorizer
	CSRF          CSRFProtector
	Audit         AuditSink
	RateLimiter   RateLimiter
}

type Authenticator interface {
	Authenticate(ctx context.Context, req *http.Request, session *SessionDTO, spec SecuritySpec) (*Actor, error)
}

// ResultAuthenticator is the richer authentication interface used by
// programmatic auth implementations. Existing Authenticator implementations
// remain supported and are adapted to AuthResult as session user auth.
type ResultAuthenticator interface {
	AuthenticateResult(ctx context.Context, req *http.Request, session *SessionDTO, spec SecuritySpec) (AuthResult, error)
}

type ResourceResolver interface {
	ResolveResource(ctx context.Context, req ResourceRequest) (*ResourceRef, error)
}

type Authorizer interface {
	Authorize(ctx context.Context, req AuthorizationRequest) (AuthorizationDecision, error)
}

type CSRFProtector interface {
	VerifyCSRF(ctx context.Context, req CSRFRequest) error
}

type AuditSink interface {
	RecordAudit(ctx context.Context, event AuditEvent) error
}

type ResourceRequest struct {
	HTTPRequest *http.Request
	Request     *RequestDTO
	Actor       *Actor
	Spec        ResourceSpec
	ID          string
	TenantID    string
}

type AuthorizationRequest struct {
	HTTPRequest *http.Request
	Request     *RequestDTO
	Actor       *Actor
	Action      string
	Resource    *ResourceRef
	Resources   map[string]*ResourceRef
}

type CSRFRequest struct {
	HTTPRequest *http.Request
	Request     *RequestDTO
	Session     *SessionDTO
	Actor       *Actor
	Plan        RoutePlan
}

type AuditEvent struct {
	HTTPRequest *http.Request           `json:"-"`
	Request     *RequestDTO             `json:"-"`
	Event       string                  `json:"event"`
	Outcome     string                  `json:"outcome"`
	Reason      string                  `json:"reason,omitempty"`
	StatusCode  int                     `json:"statusCode,omitempty"`
	RouteName   string                  `json:"routeName,omitempty"`
	Method      string                  `json:"method"`
	Pattern     string                  `json:"pattern"`
	Action      string                  `json:"action,omitempty"`
	Actor       *Actor                  `json:"actor,omitempty"`
	Resource    *ResourceRef            `json:"resource,omitempty"`
	Resources   map[string]*ResourceRef `json:"resources,omitempty"`
	Attributes  map[string]any          `json:"attributes,omitempty"`
}

type AuthorizationDecision struct {
	Allowed bool
	Reason  string
}

func ValidateRoutePlan(plan RoutePlan) (RoutePlan, error) {
	plan.Method = strings.ToUpper(strings.TrimSpace(plan.Method))
	plan.Pattern = cleanPath(plan.Pattern)
	plan.Name = strings.TrimSpace(plan.Name)
	plan.Action = strings.TrimSpace(plan.Action)
	plan.Audit.Event = strings.TrimSpace(plan.Audit.Event)

	if plan.Method == "" {
		return RoutePlan{}, fmt.Errorf("planned route method is required")
	}
	if plan.Pattern == "" {
		return RoutePlan{}, fmt.Errorf("planned route pattern is required")
	}

	for i := range plan.RateLimits {
		limit, err := normalizeRateLimitSpec(plan, plan.RateLimits[i])
		if err != nil {
			return RoutePlan{}, err
		}
		plan.RateLimits[i] = limit
	}

	switch plan.Security.Mode {
	case SecurityModePublic:
		plan.Security.Required = false
		if len(plan.Security.AuthRequirements) > 0 {
			return RoutePlan{}, fmt.Errorf("planned public route %s %s cannot declare auth requirements", plan.Method, plan.Pattern)
		}
	case SecurityModeUser:
		plan.Security.Required = true
		authRequirements, err := normalizeAuthRequirements(plan.Security.AuthRequirements)
		if err != nil {
			return RoutePlan{}, fmt.Errorf("planned route %s %s auth requirements: %w", plan.Method, plan.Pattern, err)
		}
		plan.Security.AuthRequirements = authRequirements
		if plan.Action == "" {
			return RoutePlan{}, fmt.Errorf("planned user route %s %s requires .allow(action)", plan.Method, plan.Pattern)
		}
	default:
		return RoutePlan{}, fmt.Errorf("planned route %s %s must declare .public() or .auth(...) before .handle(...)", plan.Method, plan.Pattern)
	}

	pathParams := pathParamSet(plan.Pattern)
	for i := range plan.Resources {
		resource := &plan.Resources[i]
		resource.Name = strings.TrimSpace(resource.Name)
		resource.Type = strings.TrimSpace(resource.Type)
		if resource.Type == "" {
			return RoutePlan{}, fmt.Errorf("resource %d on %s %s requires a type", i+1, plan.Method, plan.Pattern)
		}
		if resource.Name == "" {
			resource.Name = resource.Type
		}
		if err := validateValueSource(resource.ID, pathParams, fmt.Sprintf("resource %q id", resource.Name)); err != nil {
			return RoutePlan{}, err
		}
		if resource.Tenant != nil {
			if err := validateValueSource(*resource.Tenant, pathParams, fmt.Sprintf("resource %q tenant", resource.Name)); err != nil {
				return RoutePlan{}, err
			}
		}
	}
	return plan, nil
}

func normalizeAuthRequirements(in []AuthRequirement) ([]AuthRequirement, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make([]AuthRequirement, 0, len(in))
	seen := map[AuthRequirement]struct{}{}
	for _, requirement := range in {
		requirement.Method = AuthMethod(strings.TrimSpace(string(requirement.Method)))
		requirement.PrincipalKind = PrincipalKind(strings.TrimSpace(string(requirement.PrincipalKind)))
		if requirement.Method == "" && requirement.PrincipalKind == "" {
			return nil, fmt.Errorf("empty auth requirement")
		}
		if err := validateAuthMethod(requirement.Method); err != nil {
			return nil, err
		}
		if err := validatePrincipalKind(requirement.PrincipalKind); err != nil {
			return nil, err
		}
		if _, ok := seen[requirement]; ok {
			continue
		}
		seen[requirement] = struct{}{}
		out = append(out, requirement)
	}
	return out, nil
}

func validateAuthMethod(method AuthMethod) error {
	switch method {
	case "", AuthMethodNone, AuthMethodSession, AuthMethodAPIToken, AuthMethodAccessToken:
		return nil
	default:
		return fmt.Errorf("unsupported auth method %q", method)
	}
}

func validatePrincipalKind(kind PrincipalKind) error {
	switch kind {
	case "", PrincipalKindUser, PrincipalKindAgent, PrincipalKindService:
		return nil
	default:
		return fmt.Errorf("unsupported principal kind %q", kind)
	}
}

func validateValueSource(source ValueSource, pathParams map[string]struct{}, label string) error {
	source.Key = strings.TrimSpace(source.Key)
	source.Value = strings.TrimSpace(source.Value)
	switch source.Kind {
	case ValueSourceParam:
		if source.Key == "" {
			return fmt.Errorf("%s requires a route parameter name", label)
		}
		if _, ok := pathParams[source.Key]; !ok {
			return fmt.Errorf("%s references missing route parameter %q", label, source.Key)
		}
	case ValueSourceQuery, ValueSourceBody:
		if source.Key == "" {
			return fmt.Errorf("%s requires a %s key", label, source.Kind)
		}
	case ValueSourceLiteral:
		if source.Value == "" {
			return fmt.Errorf("%s requires a literal value", label)
		}
	default:
		return fmt.Errorf("%s has unsupported value source %q", label, source.Kind)
	}
	return nil
}

func pathParamSet(pattern string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, part := range splitPath(pattern) {
		if strings.HasPrefix(part, ":") {
			name := strings.TrimPrefix(part, ":")
			if name != "" {
				out[name] = struct{}{}
			}
		}
	}
	return out
}

func isUnsafeMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	default:
		return true
	}
}
