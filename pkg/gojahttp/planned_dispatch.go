package gojahttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/dop251/goja"
)

// SecureContext is the Go-native result of planned-route enforcement. It is
// built after authentication, CSRF verification, resource resolution, and
// authorization succeed, then passed to Go planned handlers or projected into a
// JavaScript ctx object for planned Goja handlers.
type SecureContext struct {
	Plan      RoutePlan
	Request   *RequestDTO
	Auth      AuthResult
	Actor     *Actor
	Resource  *ResourceRef
	Resources map[string]*ResourceRef
	Params    map[string]string
	Body      any
}

// secureEnvelope is the JavaScript adapter around SecureContext kept for the
// planned Goja route path. Go-native planned handlers receive SecureContext
// directly.
type secureEnvelope struct {
	*SecureContext
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
		result, err := route.GojaHandler(goja.Undefined(), envelope.JSObject(vm), res.JSObject(vm))
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
	sec, status, err := h.enforcer.Enforce(ctx, httpReq, req, plan)
	if sec == nil {
		return nil, status, err
	}
	return &secureEnvelope{SecureContext: sec}, status, err
}

func (h *Host) writePlannedError(w http.ResponseWriter, res *Response, status int, err error) {
	if res.Sent() {
		return
	}
	if status == 0 {
		status = http.StatusInternalServerError
	}
	if rateErr := (*RateLimitError)(nil); errors.As(err, &rateErr) && rateErr.RetryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(rateErr.RetryAfter.Seconds()+0.999)))
	}
	message := http.StatusText(status)
	if h.dev && err != nil && status >= 500 {
		message = err.Error()
	}
	http.Error(w, message, status)
}

func (h *Host) recordAudit(ctx context.Context, httpReq *http.Request, req *RequestDTO, plan *RoutePlan, envelope *secureEnvelope, outcome string, status int, err error) {
	var sec *SecureContext
	if envelope != nil {
		sec = envelope.SecureContext
	}
	h.enforcer.recordAudit(ctx, httpReq, req, plan, sec, outcome, status, err)
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
	case errors.Is(err, ErrRateLimited):
		return http.StatusTooManyRequests
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

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func (e *secureEnvelope) JSObject(vm *goja.Runtime) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("request", e.Request.Map())
	_ = obj.Set("auth", authJSMap(e.Auth))
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

func authJSMap(auth AuthResult) map[string]any {
	return map[string]any{
		"method":         string(auth.Method),
		"principalKind":  string(auth.PrincipalKind),
		"principalId":    auth.PrincipalID,
		"credentialId":   auth.CredentialID,
		"credentialHint": auth.CredentialHint,
		"scopes":         append([]string(nil), auth.Scopes...),
	}
}

func authAuditAttributes(auth AuthResult) map[string]any {
	if auth.Method == "" {
		return nil
	}
	out := map[string]any{"authMethod": string(auth.Method)}
	if auth.PrincipalKind != "" {
		out["principalKind"] = string(auth.PrincipalKind)
	}
	if auth.PrincipalID != "" {
		out["principalId"] = auth.PrincipalID
	}
	if auth.CredentialID != "" {
		out["credentialId"] = auth.CredentialID
	}
	if auth.CredentialHint != "" {
		out["credentialHint"] = auth.CredentialHint
	}
	if len(auth.Scopes) > 0 {
		out["scopes"] = append([]string(nil), auth.Scopes...)
	}
	return out
}

func actorJSMap(actor *Actor) map[string]any {
	if actor == nil {
		return nil
	}
	return map[string]any{
		"id":        actor.ID,
		"kind":      actor.Kind,
		"tenantIds": append([]string(nil), actor.TenantIDs...),
		"claims":    cloneAnyMap(actor.Claims),
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
		"claims":   cloneAnyMap(resource.Claims),
	}
}

func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneAnyValue(value)
	}
	return out
}

func cloneAnyValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return cloneAnyMap(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = cloneAnyValue(item)
		}
		return out
	case []string:
		return append([]string(nil), v...)
	default:
		return value
	}
}
