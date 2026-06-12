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
)

// RoutePlan is the Go-owned security contract compiled by the Express fluent
// route builder at registration time.
type RoutePlan struct {
	Name      string
	Method    string
	Pattern   string
	Security  SecuritySpec
	Resources []ResourceSpec
	Action    string
}

// SecuritySpec describes who may enter a planned route.
type SecuritySpec struct {
	Mode           SecurityMode
	Required       bool
	MFAFreshWithin time.Duration
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

// Actor is the minimal host-owned authenticated principal exposed to planned
// route handlers.
type Actor struct {
	ID        string         `json:"id"`
	Kind      string         `json:"kind"`
	TenantIDs []string       `json:"tenantIds,omitempty"`
	Claims    map[string]any `json:"claims,omitempty"`
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

type AuthOptions struct {
	Authenticator Authenticator
	Resources     ResourceResolver
	Authorizer    Authorizer
}

type Authenticator interface {
	Authenticate(ctx context.Context, req *http.Request, session *SessionDTO, spec SecuritySpec) (*Actor, error)
}

type ResourceResolver interface {
	ResolveResource(ctx context.Context, req ResourceRequest) (*ResourceRef, error)
}

type Authorizer interface {
	Authorize(ctx context.Context, req AuthorizationRequest) (AuthorizationDecision, error)
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

type AuthorizationDecision struct {
	Allowed bool
	Reason  string
}

func ValidateRoutePlan(plan RoutePlan) (RoutePlan, error) {
	plan.Method = strings.ToUpper(strings.TrimSpace(plan.Method))
	plan.Pattern = cleanPath(plan.Pattern)
	plan.Name = strings.TrimSpace(plan.Name)
	plan.Action = strings.TrimSpace(plan.Action)

	if plan.Method == "" {
		return RoutePlan{}, fmt.Errorf("planned route method is required")
	}
	if plan.Pattern == "" {
		return RoutePlan{}, fmt.Errorf("planned route pattern is required")
	}

	switch plan.Security.Mode {
	case SecurityModePublic:
		plan.Security.Required = false
	case SecurityModeUser:
		plan.Security.Required = true
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
