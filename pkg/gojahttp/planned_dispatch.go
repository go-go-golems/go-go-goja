package gojahttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/dop251/goja"
)

type secureEnvelope struct {
	Plan      RoutePlan
	Request   *RequestDTO
	Actor     *Actor
	Resources map[string]*ResourceRef
	Body      any
}

func (h *Host) servePlannedRoute(w http.ResponseWriter, r *http.Request, route Route, req *RequestDTO) {
	res := NewResponse(w, h.renderer)
	envelope, status, err := h.buildSecureEnvelope(r.Context(), r, req, route.Plan)
	if err != nil {
		h.recordAudit(r.Context(), r, req, route.Plan, envelope, "denied", status, err)
		h.writePlannedError(w, res, status, err)
		return
	}
	h.recordAudit(r.Context(), r, req, route.Plan, envelope, "allowed", 0, nil)
	ret, err := h.owner.Call(r.Context(), "http-planned-handler", func(ctx context.Context, vm *goja.Runtime) (any, error) {
		result, err := route.Handler(goja.Undefined(), envelope.JSObject(vm), res.JSObject(vm))
		if err != nil {
			return nil, err
		}
		if promise, ok := result.Export().(*goja.Promise); ok {
			return promise, nil
		}
		return nil, h.finishHandlerResult(vm, res, result)
	})
	if err == nil {
		if promise, ok := ret.(*goja.Promise); ok {
			err = h.awaitAndFinishPromise(r.Context(), res, promise)
		}
	}
	if err != nil {
		h.recordAudit(r.Context(), r, req, route.Plan, envelope, "failed", http.StatusInternalServerError, err)
		if !res.Sent() {
			if h.dev {
				http.Error(w, fmt.Sprintf("JavaScript handler error: %v", err), http.StatusInternalServerError)
			} else {
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}
		return
	}
	h.recordAudit(r.Context(), r, req, route.Plan, envelope, "completed", res.Status(), nil)
}

func (h *Host) buildSecureEnvelope(ctx context.Context, httpReq *http.Request, req *RequestDTO, plan *RoutePlan) (*secureEnvelope, int, error) {
	if plan == nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("planned route is missing route plan")
	}
	envelope := &secureEnvelope{Plan: *plan, Request: req, Body: req.Body, Resources: map[string]*ResourceRef{}}
	var actor *Actor
	switch plan.Security.Mode {
	case SecurityModePublic:
		// No actor required.
	case SecurityModeUser:
		if h.auth.Authenticator == nil {
			return envelope, http.StatusInternalServerError, fmt.Errorf("planned route %s %s requires authenticator", plan.Method, plan.Pattern)
		}
		var err error
		actor, err = h.auth.Authenticator.Authenticate(ctx, httpReq, req.Session, plan.Security)
		if err != nil {
			return envelope, statusForAuthError(err), err
		}
		if actor == nil {
			return envelope, http.StatusUnauthorized, ErrUnauthenticated
		}
		envelope.Actor = actor
	default:
		return envelope, http.StatusInternalServerError, fmt.Errorf("unsupported planned route security mode %q", plan.Security.Mode)
	}

	if plan.CSRF.Required && isUnsafeMethod(plan.Method) {
		if h.auth.CSRF == nil {
			return envelope, http.StatusInternalServerError, fmt.Errorf("planned route %s %s requires csrf protector", plan.Method, plan.Pattern)
		}
		if err := h.auth.CSRF.VerifyCSRF(ctx, CSRFRequest{HTTPRequest: httpReq, Request: req, Session: req.Session, Actor: actor, Plan: *plan}); err != nil {
			return envelope, statusForAuthError(fmt.Errorf("%w: %v", ErrCSRF, err)), err
		}
	}

	if len(plan.Resources) > 0 {
		if h.auth.Resources == nil {
			return envelope, http.StatusInternalServerError, fmt.Errorf("planned route %s %s requires resource resolver", plan.Method, plan.Pattern)
		}
		for _, spec := range plan.Resources {
			id, err := resolveValueSource(req, spec.ID)
			if err != nil {
				return envelope, http.StatusBadRequest, err
			}
			tenantID := ""
			if spec.Tenant != nil {
				tenantID, err = resolveValueSource(req, *spec.Tenant)
				if err != nil {
					return envelope, http.StatusBadRequest, err
				}
			}
			resource, err := h.auth.Resources.ResolveResource(ctx, ResourceRequest{HTTPRequest: httpReq, Request: req, Actor: actor, Spec: spec, ID: id, TenantID: tenantID})
			if err != nil {
				return envelope, statusForAuthError(err), err
			}
			if resource == nil {
				return envelope, http.StatusNotFound, ErrNotFound
			}
			if resource.Name == "" {
				resource.Name = spec.Name
			}
			if resource.Type == "" {
				resource.Type = spec.Type
			}
			envelope.Resources[spec.Name] = resource
		}
	}

	if plan.Security.Mode != SecurityModePublic && plan.Action != "" {
		if h.auth.Authorizer == nil {
			return envelope, http.StatusInternalServerError, fmt.Errorf("planned route %s %s requires authorizer", plan.Method, plan.Pattern)
		}
		resource := firstPlannedResource(plan, envelope.Resources)
		decision, err := h.auth.Authorizer.Authorize(ctx, AuthorizationRequest{HTTPRequest: httpReq, Request: req, Actor: actor, Action: plan.Action, Resource: resource, Resources: envelope.Resources})
		if err != nil {
			return envelope, statusForAuthError(err), err
		}
		if !decision.Allowed {
			if decision.Reason != "" {
				return envelope, http.StatusForbidden, fmt.Errorf("%w: %s", ErrForbidden, decision.Reason)
			}
			return envelope, http.StatusForbidden, ErrForbidden
		}
	}
	return envelope, 0, nil
}

func (h *Host) writePlannedError(w http.ResponseWriter, res *Response, status int, err error) {
	if res.Sent() {
		return
	}
	if status == 0 {
		status = http.StatusInternalServerError
	}
	message := http.StatusText(status)
	if h.dev && err != nil && status >= 500 {
		message = err.Error()
	}
	http.Error(w, message, status)
}

func (h *Host) recordAudit(ctx context.Context, httpReq *http.Request, req *RequestDTO, plan *RoutePlan, envelope *secureEnvelope, outcome string, status int, err error) {
	if h.auth.Audit == nil || plan == nil || plan.Audit.Event == "" {
		return
	}
	var actor *Actor
	resources := map[string]*ResourceRef{}
	if envelope != nil {
		actor = envelope.Actor
		resources = copyResourceRefs(envelope.Resources)
	}
	resource := firstPlannedResource(plan, resources)
	reason := ""
	if err != nil {
		reason = err.Error()
	}
	_ = h.auth.Audit.RecordAudit(ctx, AuditEvent{
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
	})
}

func statusForAuthError(err error) int {
	switch {
	case errors.Is(err, ErrUnauthenticated):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrCSRF):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

func resolveValueSource(req *RequestDTO, source ValueSource) (string, error) {
	switch source.Kind {
	case ValueSourceParam:
		value, ok := req.Params[source.Key]
		if !ok || value == "" {
			return "", fmt.Errorf("missing route parameter %q", source.Key)
		}
		return value, nil
	case ValueSourceQuery:
		value, ok := req.Query[source.Key]
		if !ok {
			return "", fmt.Errorf("missing query value %q", source.Key)
		}
		return stringifySourceValue(value, source.Key)
	case ValueSourceBody:
		body, ok := req.Body.(map[string]any)
		if !ok {
			return "", fmt.Errorf("body source %q requires object body", source.Key)
		}
		value, ok := body[source.Key]
		if !ok {
			return "", fmt.Errorf("missing body value %q", source.Key)
		}
		return stringifySourceValue(value, source.Key)
	case ValueSourceLiteral:
		return source.Value, nil
	default:
		return "", fmt.Errorf("unsupported value source %q", source.Kind)
	}
}

func stringifySourceValue(value any, key string) (string, error) {
	switch v := value.(type) {
	case string:
		if v == "" {
			return "", fmt.Errorf("value %q is empty", key)
		}
		return v, nil
	case fmt.Stringer:
		return v.String(), nil
	default:
		return fmt.Sprint(v), nil
	}
}

func firstPlannedResource(plan *RoutePlan, resources map[string]*ResourceRef) *ResourceRef {
	if plan == nil || len(plan.Resources) == 0 {
		return nil
	}
	return resources[plan.Resources[0].Name]
}

func copyResourceRefs(resources map[string]*ResourceRef) map[string]*ResourceRef {
	out := make(map[string]*ResourceRef, len(resources))
	for name, resource := range resources {
		out[name] = resource
	}
	return out
}

func (e *secureEnvelope) JSObject(vm *goja.Runtime) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("request", e.Request.Map())
	_ = obj.Set("actor", actorJSMap(e.Actor))
	_ = obj.Set("body", e.Body)
	_ = obj.Set("params", e.Request.Params)
	_ = obj.Set("resources", resourceJSMap(e.Resources))
	_ = obj.Set("action", e.Plan.Action)
	_ = obj.Set("routeName", e.Plan.Name)
	_ = obj.Set("resource", func(name string) goja.Value {
		resource := e.Resources[name]
		if resource == nil {
			return goja.Null()
		}
		return vm.ToValue(resourceRefJSMap(resource))
	})
	return obj
}

func actorJSMap(actor *Actor) map[string]any {
	if actor == nil {
		return nil
	}
	return map[string]any{
		"id":        actor.ID,
		"kind":      actor.Kind,
		"tenantIds": actor.TenantIDs,
		"claims":    actor.Claims,
	}
}

func resourceJSMap(resources map[string]*ResourceRef) map[string]any {
	out := make(map[string]any, len(resources))
	for name, resource := range resources {
		out[name] = resourceRefJSMap(resource)
	}
	return out
}

func resourceRefJSMap(resource *ResourceRef) map[string]any {
	if resource == nil {
		return nil
	}
	return map[string]any{
		"name":     resource.Name,
		"type":     resource.Type,
		"id":       resource.ID,
		"tenantId": resource.TenantID,
		"claims":   resource.Claims,
	}
}
