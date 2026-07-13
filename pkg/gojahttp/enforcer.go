package gojahttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// EnforcerOptions configures the reusable planned-auth enforcement pipeline.
type EnforcerOptions struct {
	Dev      bool
	Sessions SessionOptions
	Auth     AuthOptions
}

// Enforcer owns the host-side planned-auth pipeline independently of any
// particular router. Routers and adapters build a RequestDTO, then ask Enforcer
// to authenticate, verify CSRF, resolve resources, authorize, audit, and invoke
// Go planned handlers.
type Enforcer struct {
	dev      bool
	sessions *SessionManager
	auth     AuthOptions
}

// NewEnforcer returns a reusable planned-auth enforcer for Go hosts,
// middleware, generated code, and custom router adapters.
func NewEnforcer(opts EnforcerOptions) *Enforcer {
	return &Enforcer{dev: opts.Dev, sessions: NewSessionManager(opts.Sessions), auth: opts.Auth}
}

// SetAuthOptions replaces the host-owned planned-auth services used by future
// Enforce calls.
func (e *Enforcer) SetAuthOptions(auth AuthOptions) {
	if e == nil {
		return
	}
	e.auth = auth
}

// Session loads or creates the request session using the enforcer's session
// manager.
func (e *Enforcer) Session(w http.ResponseWriter, r *http.Request) (*SessionDTO, error) {
	if e == nil {
		return nil, fmt.Errorf("gojahttp enforcer is nil")
	}
	return e.sessions.Session(w, r)
}

// Request builds the planned-route request DTO for router-extracted params.
func (e *Enforcer) Request(w http.ResponseWriter, r *http.Request, params map[string]string) (*RequestDTO, error) {
	session, err := e.Session(w, r)
	if err != nil {
		return nil, err
	}
	return NewRequestDTO(r, params, session)
}

// Enforce runs planned-route auth checks and returns the secure context that may
// be passed to a planned handler. The returned status is non-zero when err is
// non-nil and is suitable for an HTTP response.
func (e *Enforcer) Enforce(ctx context.Context, httpReq *http.Request, req *RequestDTO, plan *RoutePlan) (*SecureContext, int, error) {
	if e == nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("gojahttp enforcer is nil")
	}
	if plan == nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("planned route is missing route plan")
	}
	validatedPlan, err := ValidateRoutePlan(*plan)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	plan = &validatedPlan
	sec := &SecureContext{Plan: *plan, Request: req, Auth: AuthResult{Method: AuthMethodNone}, Params: cloneStringMap(req.Params), Body: req.Body, Resources: map[string]*ResourceRef{}}
	if err := e.checkRateLimits(ctx, httpReq, req, plan, sec, RateLimitStagePreAuth); err != nil {
		return sec, statusForAuthError(err), err
	}
	var actor *Actor
	switch plan.Security.Mode {
	case SecurityModePublic:
		// No actor required.
	case SecurityModeUser:
		if e.auth.Authenticator == nil {
			return sec, http.StatusInternalServerError, fmt.Errorf("planned route %s %s requires authenticator", plan.Method, plan.Pattern)
		}
		auth, err := authenticateResult(ctx, e.auth.Authenticator, httpReq, req.Session, plan.Security)
		if err != nil {
			return sec, statusForAuthError(err), err
		}
		auth = normalizeAuthResult(auth)
		if auth.Actor == nil {
			return sec, http.StatusUnauthorized, ErrUnauthenticated
		}
		sec.Auth = auth
		if err := checkAuthRequirements(plan.Security, auth); err != nil {
			return sec, http.StatusForbidden, err
		}
		actor = auth.Actor
		sec.Actor = actor
	default:
		return sec, http.StatusInternalServerError, fmt.Errorf("unsupported planned route security mode %q", plan.Security.Mode)
	}

	if plan.CSRF.Required && isUnsafeMethod(httpReq.Method) && shouldVerifyCSRF(plan.Security.Mode, sec.Auth) {
		if e.auth.CSRF == nil {
			return sec, http.StatusInternalServerError, fmt.Errorf("planned route %s %s requires csrf protector", plan.Method, plan.Pattern)
		}
		if err := e.auth.CSRF.VerifyCSRF(ctx, CSRFRequest{HTTPRequest: httpReq, Request: req, Session: req.Session, Actor: actor, Plan: *plan}); err != nil {
			return sec, statusForAuthError(fmt.Errorf("%w: %v", ErrCSRF, err)), err
		}
	}

	if len(plan.Resources) > 0 {
		if e.auth.Resources == nil {
			return sec, http.StatusInternalServerError, fmt.Errorf("planned route %s %s requires resource resolver", plan.Method, plan.Pattern)
		}
		for _, spec := range plan.Resources {
			id, err := resolveValueSource(req, spec.ID)
			if err != nil {
				return sec, http.StatusBadRequest, err
			}
			tenantID := ""
			if spec.Tenant != nil {
				tenantID, err = resolveValueSource(req, *spec.Tenant)
				if err != nil {
					return sec, http.StatusBadRequest, err
				}
			}
			resource, err := e.auth.Resources.ResolveResource(ctx, ResourceRequest{HTTPRequest: httpReq, Request: req, Actor: actor, Spec: spec, ID: id, TenantID: tenantID})
			if err != nil {
				return sec, statusForAuthError(err), err
			}
			if resource == nil {
				return sec, http.StatusNotFound, ErrNotFound
			}
			if resource.Name == "" {
				resource.Name = spec.Name
			}
			if resource.Type == "" {
				resource.Type = spec.Type
			}
			sec.Resources[spec.Name] = resource
		}
	}

	sec.Resource = firstPlannedResource(plan, sec.Resources)
	if plan.Action != "" && len(sec.Auth.Grants.Grants) > 0 && !sec.Auth.Grants.Allows(plan.Action, sec.Resource) {
		return sec, http.StatusForbidden, fmt.Errorf("%w: insufficient grant for %s", ErrForbidden, plan.Action)
	}

	if plan.Security.Mode != SecurityModePublic && plan.Action != "" {
		if e.auth.Authorizer == nil {
			return sec, http.StatusInternalServerError, fmt.Errorf("planned route %s %s requires authorizer", plan.Method, plan.Pattern)
		}
		resource := firstPlannedResource(plan, sec.Resources)
		sec.Resource = resource
		decision, err := e.auth.Authorizer.Authorize(ctx, AuthorizationRequest{HTTPRequest: httpReq, Request: req, Actor: actor, Action: plan.Action, Resource: resource, Resources: sec.Resources})
		if err != nil {
			return sec, statusForAuthError(err), err
		}
		if !decision.Allowed {
			if decision.Reason != "" {
				return sec, http.StatusForbidden, fmt.Errorf("%w: %s", ErrForbidden, decision.Reason)
			}
			return sec, http.StatusForbidden, ErrForbidden
		}
	}
	sec.Resource = firstPlannedResource(plan, sec.Resources)
	// Post-auth limits are charged only after grant and authorizer checks pass.
	// A caller denied by policy must not exhaust a shared resource bucket.
	if err := e.checkRateLimits(ctx, httpReq, req, plan, sec, RateLimitStagePostAuth); err != nil {
		return sec, statusForAuthError(err), err
	}
	return sec, 0, nil
}

func (e *Enforcer) servePlannedHTTP(w http.ResponseWriter, r *http.Request, plan *RoutePlan, req *RequestDTO, handler PlannedHTTPHandler, loggingWriter *accessLogResponseWriter) {
	sec, status, err := e.Enforce(r.Context(), r, req, plan)
	if err != nil {
		e.recordAudit(r.Context(), r, req, plan, sec, "denied", status, err)
		e.writePlannedHTTPError(w, loggingWriter, status, err)
		return
	}
	e.recordAudit(r.Context(), r, req, plan, sec, "allowed", 0, nil)
	if handler == nil {
		err := fmt.Errorf("planned HTTP route %s %s has nil handler", plan.Method, plan.Pattern)
		e.recordAudit(r.Context(), r, req, plan, sec, "failed", http.StatusInternalServerError, err)
		e.writePlannedHTTPError(w, loggingWriter, http.StatusInternalServerError, err)
		return
	}
	if err := handler(r.Context(), sec, w, r); err != nil {
		e.recordAudit(r.Context(), r, req, plan, sec, "failed", http.StatusInternalServerError, err)
		e.writePlannedHTTPError(w, loggingWriter, http.StatusInternalServerError, err)
		return
	}
	status = 0
	if loggingWriter != nil {
		status = loggingWriter.status
	}
	e.recordAudit(r.Context(), r, req, plan, sec, "completed", status, nil)
}

func authenticateResult(ctx context.Context, authenticator Authenticator, req *http.Request, session *SessionDTO, spec SecuritySpec) (AuthResult, error) {
	if resultAuthenticator, ok := authenticator.(ResultAuthenticator); ok {
		return resultAuthenticator.AuthenticateResult(ctx, req, session, spec)
	}
	actor, err := authenticator.Authenticate(ctx, req, session, spec)
	if err != nil {
		return AuthResult{}, err
	}
	if actor == nil {
		return AuthResult{}, ErrUnauthenticated
	}
	return AuthResult{Actor: actor, Method: AuthMethodSession, PrincipalKind: PrincipalKindUser, PrincipalID: actor.ID, CSRFRequired: true}, nil
}

func normalizeAuthResult(auth AuthResult) AuthResult {
	if auth.Method == "" {
		auth.Method = AuthMethodSession
	}
	if auth.Actor != nil {
		if auth.PrincipalID == "" {
			auth.PrincipalID = auth.Actor.ID
		}
		if auth.PrincipalKind == "" {
			auth.PrincipalKind = PrincipalKind(auth.Actor.Kind)
		}
	}
	if auth.PrincipalKind == "" && auth.Method == AuthMethodSession {
		auth.PrincipalKind = PrincipalKindUser
	}
	if len(auth.Grants.Grants) > 0 {
		if normalized, err := auth.Grants.Normalize(); err == nil {
			auth.Grants = normalized
		}
	}
	if auth.Scopes != nil {
		auth.Scopes = append([]string(nil), auth.Scopes...)
	} else if len(auth.Grants.Grants) > 0 {
		auth.Scopes = auth.Grants.ScopeStrings()
	}
	return auth
}

func checkAuthRequirements(spec SecuritySpec, auth AuthResult) error {
	if len(spec.AuthRequirements) == 0 {
		return nil
	}
	for _, requirement := range spec.AuthRequirements {
		methodMatches := requirement.Method == "" || requirement.Method == auth.Method
		kindMatches := requirement.PrincipalKind == "" || requirement.PrincipalKind == auth.PrincipalKind
		if methodMatches && kindMatches {
			return nil
		}
	}
	return fmt.Errorf("%w: authenticated principal does not satisfy route auth requirements", ErrForbidden)
}

func shouldVerifyCSRF(mode SecurityMode, auth AuthResult) bool {
	if mode == SecurityModePublic {
		return true
	}
	return auth.CSRFRequired
}

func (e *Enforcer) writePlannedHTTPError(w http.ResponseWriter, loggingWriter *accessLogResponseWriter, status int, err error) {
	if loggingWriter != nil && loggingWriter.wroteHeader {
		return
	}
	if status == 0 {
		status = http.StatusInternalServerError
	}
	if rateErr := (*RateLimitError)(nil); errors.As(err, &rateErr) && rateErr.RetryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(rateErr.RetryAfter.Seconds()+0.999)))
	}
	message := http.StatusText(status)
	if e.dev && err != nil && status >= 500 {
		message = err.Error()
	}
	http.Error(w, message, status)
}

func (e *Enforcer) recordAudit(ctx context.Context, httpReq *http.Request, req *RequestDTO, plan *RoutePlan, sec *SecureContext, outcome string, status int, err error) {
	if e == nil || e.auth.Audit == nil || plan == nil || plan.Audit.Event == "" {
		return
	}
	var actor *Actor
	resources := map[string]*ResourceRef{}
	if sec != nil {
		actor = sec.Actor
		resources = copyResourceRefs(sec.Resources)
	}
	resource := firstPlannedResource(plan, resources)
	reason := ""
	if err != nil {
		reason = err.Error()
	}
	attributes := map[string]any(nil)
	if sec != nil {
		attributes = authAuditAttributes(sec.Auth)
	}
	if rateErr := (*RateLimitError)(nil); errors.As(err, &rateErr) {
		if attributes == nil {
			attributes = map[string]any{}
		}
		attributes["rateLimitPolicy"] = rateErr.Policy
		if rateErr.RetryAfter > 0 {
			attributes["retryAfterSeconds"] = int(rateErr.RetryAfter.Seconds() + 0.999)
		}
	}
	_ = e.auth.Audit.RecordAudit(ctx, AuditEvent{
		HTTPRequest: httpReq,
		Request:     req,
		Event:       plan.Audit.Event,
		Outcome:     outcome,
		Reason:      reason,
		StatusCode:  status,
		RouteName:   plan.Name,
		Method:      plan.Method,
		Pattern:     plan.Pattern,
		Action:      plan.Action,
		Actor:       actor,
		Resource:    resource,
		Resources:   resources,
		Attributes:  attributes,
	})
}
