package gojahttp

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// RateLimitKeyKind describes one normalized key component used to identify a
// limiter bucket. Route plans store key parts as data instead of asking
// JavaScript handlers to concatenate limiter keys themselves.
type RateLimitKeyKind string

const (
	RateLimitKeyIP          RateLimitKeyKind = "ip"
	RateLimitKeyRoute       RateLimitKeyKind = "route"
	RateLimitKeyActor       RateLimitKeyKind = "actor"
	RateLimitKeyParam       RateLimitKeyKind = "param"
	RateLimitKeyTenantParam RateLimitKeyKind = "tenantParam"
	RateLimitKeyHeader      RateLimitKeyKind = "header"
	RateLimitKeyBodyField   RateLimitKeyKind = "bodyField"
	RateLimitKeyResource    RateLimitKeyKind = "resource"
)

// RateLimitStage identifies whether a limiter key can be computed before
// authentication or needs authenticated/resource context.
type RateLimitStage string

const (
	RateLimitStagePreAuth  RateLimitStage = "pre-auth"
	RateLimitStagePostAuth RateLimitStage = "post-auth"
)

// RateLimitKeyPart is one Go-owned bucket-key component.
type RateLimitKeyPart struct {
	Kind RateLimitKeyKind
	Key  string
}

// RateLimitSpec is the route-plan declaration for request budgets. A route may
// have multiple policies, for example an IP pre-auth policy and an actor/tenant
// post-auth policy.
type RateLimitSpec struct {
	Policy   string
	Limit    int
	Window   time.Duration
	Burst    int
	KeyParts []RateLimitKeyPart
	FailOpen bool
}

// RateLimitRequest is passed to host-provided limiter implementations. Key is a
// normalized, non-empty bucket key built by the enforcer from KeyParts.
type RateLimitRequest struct {
	HTTPRequest *http.Request
	Request     *RequestDTO
	Plan        RoutePlan
	Spec        RateLimitSpec
	Stage       RateLimitStage
	Key         string
	KeyParts    map[string]string
	Actor       *Actor
	Resource    *ResourceRef
	Resources   map[string]*ResourceRef
}

// RateLimitDecision is returned by a limiter backend.
type RateLimitDecision struct {
	Allowed    bool
	RetryAfter time.Duration
	Reason     string
	Limit      int
	Remaining  int
	ResetAt    time.Time
}

// RateLimiter checks and consumes one route budget bucket.
type RateLimiter interface {
	CheckRateLimit(ctx context.Context, req RateLimitRequest) (RateLimitDecision, error)
}

// RateLimitError is returned when a route budget has been exhausted.
type RateLimitError struct {
	Policy     string
	RetryAfter time.Duration
	Reason     string
}

func (e *RateLimitError) Error() string {
	if e == nil {
		return "rate limit exceeded"
	}
	if e.Policy == "" {
		return "rate limit exceeded"
	}
	if e.Reason != "" {
		return fmt.Sprintf("rate limit %q exceeded: %s", e.Policy, e.Reason)
	}
	return fmt.Sprintf("rate limit %q exceeded", e.Policy)
}

func (e *RateLimitError) Is(target error) bool { return target == ErrRateLimited }

// MemoryRateLimiter is a small fixed-window limiter for tests, examples, and
// local generated hosts. Production hosts can provide a distributed limiter via
// AuthOptions.RateLimiter without changing route declarations.
type MemoryRateLimiter struct {
	mu      sync.Mutex
	now     func() time.Time
	buckets map[string]memoryRateBucket
}

type memoryRateBucket struct {
	windowStart time.Time
	count       int
}

// NewMemoryRateLimiter returns an in-memory fixed-window limiter.
func NewMemoryRateLimiter() *MemoryRateLimiter {
	return &MemoryRateLimiter{buckets: map[string]memoryRateBucket{}}
}

// SetNow overrides the clock used by the memory limiter. It is intended for
// tests.
func (l *MemoryRateLimiter) SetNow(now func() time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.now = now
}

func (l *MemoryRateLimiter) CheckRateLimit(_ context.Context, req RateLimitRequest) (RateLimitDecision, error) {
	if l == nil {
		return RateLimitDecision{}, fmt.Errorf("memory rate limiter is nil")
	}
	limit := effectiveRateLimit(req.Spec)
	if limit <= 0 || req.Spec.Window <= 0 {
		return RateLimitDecision{Allowed: true}, nil
	}
	key := strings.TrimSpace(req.Key)
	if key == "" {
		key = "missing"
	}
	bucketKey := req.Spec.Policy + "|" + key
	now := l.currentTime()

	l.mu.Lock()
	defer l.mu.Unlock()
	bucket := l.buckets[bucketKey]
	if bucket.windowStart.IsZero() || now.Sub(bucket.windowStart) >= req.Spec.Window || now.Before(bucket.windowStart) {
		bucket = memoryRateBucket{windowStart: now}
	}
	resetAt := bucket.windowStart.Add(req.Spec.Window)
	if bucket.count >= limit {
		return RateLimitDecision{Allowed: false, RetryAfter: resetAt.Sub(now), Reason: "budget exhausted", Limit: limit, Remaining: 0, ResetAt: resetAt}, nil
	}
	bucket.count++
	l.buckets[bucketKey] = bucket
	return RateLimitDecision{Allowed: true, Limit: limit, Remaining: limit - bucket.count, ResetAt: resetAt}, nil
}

func (l *MemoryRateLimiter) currentTime() time.Time {
	if l.now != nil {
		return l.now()
	}
	return time.Now()
}

func effectiveRateLimit(spec RateLimitSpec) int {
	if spec.Limit <= 0 {
		return 0
	}
	if spec.Burst > 0 {
		return spec.Limit + spec.Burst
	}
	return spec.Limit
}

func normalizeRateLimitSpec(plan RoutePlan, spec RateLimitSpec) (RateLimitSpec, error) {
	spec.Policy = strings.TrimSpace(spec.Policy)
	if spec.Policy == "" {
		spec.Policy = strings.ToLower(strings.TrimSpace(plan.Method + " " + plan.Pattern))
	}
	if spec.Limit <= 0 {
		return RateLimitSpec{}, fmt.Errorf("rate limit %q requires a positive limit", spec.Policy)
	}
	if spec.Window <= 0 {
		return RateLimitSpec{}, fmt.Errorf("rate limit %q requires a positive window", spec.Policy)
	}
	if spec.Burst < 0 {
		return RateLimitSpec{}, fmt.Errorf("rate limit %q burst must be non-negative", spec.Policy)
	}
	if len(spec.KeyParts) == 0 {
		spec.KeyParts = []RateLimitKeyPart{{Kind: RateLimitKeyRoute}, {Kind: RateLimitKeyIP}}
	}
	pathParams := pathParamSet(plan.Pattern)
	for i := range spec.KeyParts {
		part := &spec.KeyParts[i]
		part.Key = strings.TrimSpace(part.Key)
		switch part.Kind {
		case RateLimitKeyIP, RateLimitKeyRoute, RateLimitKeyActor:
			part.Key = ""
		case RateLimitKeyParam, RateLimitKeyTenantParam:
			if part.Key == "" {
				return RateLimitSpec{}, fmt.Errorf("rate limit %q key part %q requires a parameter name", spec.Policy, part.Kind)
			}
			if _, ok := pathParams[part.Key]; !ok {
				return RateLimitSpec{}, fmt.Errorf("rate limit %q references unknown route parameter %q", spec.Policy, part.Key)
			}
		case RateLimitKeyHeader, RateLimitKeyBodyField, RateLimitKeyResource:
			if part.Key == "" {
				return RateLimitSpec{}, fmt.Errorf("rate limit %q key part %q requires a key", spec.Policy, part.Kind)
			}
			if part.Kind == RateLimitKeyHeader {
				part.Key = http.CanonicalHeaderKey(part.Key)
			}
		default:
			return RateLimitSpec{}, fmt.Errorf("rate limit %q has unsupported key part %q", spec.Policy, part.Kind)
		}
	}
	return spec, nil
}

func rateLimitStage(spec RateLimitSpec) RateLimitStage {
	for _, part := range spec.KeyParts {
		switch part.Kind {
		case RateLimitKeyActor, RateLimitKeyResource:
			return RateLimitStagePostAuth
		case RateLimitKeyIP, RateLimitKeyRoute, RateLimitKeyParam, RateLimitKeyTenantParam, RateLimitKeyHeader, RateLimitKeyBodyField:
			continue
		}
	}
	return RateLimitStagePreAuth
}

func (e *Enforcer) checkRateLimits(ctx context.Context, httpReq *http.Request, req *RequestDTO, plan *RoutePlan, sec *SecureContext, stage RateLimitStage) error {
	if plan == nil || len(plan.RateLimits) == 0 {
		return nil
	}
	if e.auth.RateLimiter == nil {
		return fmt.Errorf("planned route %s %s declares rate limits but no rate limiter is configured", plan.Method, plan.Pattern)
	}
	for _, spec := range plan.RateLimits {
		if rateLimitStage(spec) != stage {
			continue
		}
		key, keyParts := buildRateLimitKey(httpReq, req, plan, sec, spec)
		var actor *Actor
		var resource *ResourceRef
		resources := map[string]*ResourceRef{}
		if sec != nil {
			actor = sec.Actor
			resource = sec.Resource
			resources = sec.Resources
		}
		decision, err := e.auth.RateLimiter.CheckRateLimit(ctx, RateLimitRequest{HTTPRequest: httpReq, Request: req, Plan: *plan, Spec: spec, Stage: stage, Key: key, KeyParts: keyParts, Actor: actor, Resource: resource, Resources: resources})
		if err != nil {
			e.observeRateLimit(ctx, spec.Policy, "error")
			if spec.FailOpen {
				continue
			}
			return err
		}
		if decision.Allowed {
			e.observeRateLimit(ctx, spec.Policy, "allowed")
		} else {
			e.observeRateLimit(ctx, spec.Policy, "denied")
		}
		if !decision.Allowed {
			return &RateLimitError{Policy: spec.Policy, RetryAfter: decision.RetryAfter, Reason: decision.Reason}
		}
	}
	return nil
}

func (e *Enforcer) observeRateLimit(ctx context.Context, policy, outcome string) {
	if e.auth.SecurityEvents != nil {
		e.auth.SecurityEvents.ObserveSecurityEvent(ctx, SecurityEvent{Name: "auth.rate_limit", Outcome: outcome, Reason: policy})
	}
	if e.auth.Audit != nil {
		_ = e.auth.Audit.RecordAudit(ctx, AuditEvent{Event: "auth.rate_limit", Outcome: outcome, Reason: policy, Method: "INTERNAL", Pattern: "rate-limit"})
	}
}

func buildRateLimitKey(httpReq *http.Request, req *RequestDTO, plan *RoutePlan, sec *SecureContext, spec RateLimitSpec) (string, map[string]string) {
	parts := map[string]string{}
	for _, part := range spec.KeyParts {
		name := rateLimitPartName(part)
		parts[name] = rateLimitPartValue(httpReq, req, plan, sec, part)
	}
	keys := make([]string, 0, len(parts))
	for key := range parts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	segments := make([]string, 0, len(keys))
	for _, key := range keys {
		segments = append(segments, key+"="+parts[key])
	}
	return strings.Join(segments, "|"), parts
}

func rateLimitPartName(part RateLimitKeyPart) string {
	if part.Key == "" {
		return string(part.Kind)
	}
	return string(part.Kind) + ":" + part.Key
}

func rateLimitPartValue(httpReq *http.Request, req *RequestDTO, plan *RoutePlan, sec *SecureContext, part RateLimitKeyPart) string {
	switch part.Kind {
	case RateLimitKeyIP:
		return requestIP(httpReq, req)
	case RateLimitKeyRoute:
		if plan == nil {
			return "unknown"
		}
		return plan.Method + " " + plan.Pattern
	case RateLimitKeyActor:
		if sec != nil && sec.Actor != nil && sec.Actor.ID != "" {
			return sec.Actor.Kind + ":" + sec.Actor.ID
		}
		return "anonymous"
	case RateLimitKeyParam, RateLimitKeyTenantParam:
		if req != nil {
			if value := strings.TrimSpace(req.Params[part.Key]); value != "" {
				return value
			}
		}
		return "missing"
	case RateLimitKeyHeader:
		if httpReq != nil {
			if value := strings.TrimSpace(httpReq.Header.Get(part.Key)); value != "" {
				return value
			}
		}
		return "missing"
	case RateLimitKeyBodyField:
		if req != nil {
			if body, ok := req.Body.(map[string]any); ok {
				if value, ok := body[part.Key]; ok {
					return strings.TrimSpace(fmt.Sprint(value))
				}
			}
		}
		return "missing"
	case RateLimitKeyResource:
		if sec != nil && sec.Resources != nil {
			if resource := sec.Resources[part.Key]; resource != nil {
				return resource.Type + ":" + resource.ID
			}
		}
		return "missing"
	default:
		return "unsupported"
	}
}

func requestIP(httpReq *http.Request, req *RequestDTO) string {
	if req != nil && strings.TrimSpace(req.IP) != "" {
		return strings.TrimSpace(req.IP)
	}
	if httpReq == nil {
		return "unknown"
	}
	return RequestClientIP(httpReq)
}
