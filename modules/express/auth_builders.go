package express

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

type builderStore struct {
	authSpecs      sync.Map // map[*goja.Object]*gojahttp.SecuritySpec
	resourceSpecs  sync.Map // map[*goja.Object]*gojahttp.ResourceSpec
	rateLimitSpecs sync.Map // map[*goja.Object]*gojahttp.RateLimitSpec
}

func newBuilderStore() *builderStore { return &builderStore{} }

func (s *builderStore) newUserBuilder(vm *goja.Runtime) goja.Value {
	spec := &gojahttp.SecuritySpec{Mode: gojahttp.SecurityModeUser, Required: true}
	obj := vm.NewObject()
	s.authSpecs.Store(obj, spec)
	_ = obj.Set("required", func() goja.Value {
		spec.Required = true
		return obj
	})
	_ = obj.Set("mfaFresh", func(raw string) (goja.Value, error) {
		d, err := time.ParseDuration(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("express.user().mfaFresh(%q): %w", raw, err)
		}
		spec.MFAFreshWithin = d
		return obj, nil
	})
	return obj
}

func (s *builderStore) authSpec(vm *goja.Runtime, value goja.Value) (gojahttp.SecuritySpec, error) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return gojahttp.SecuritySpec{}, fmt.Errorf(".auth(...) expects value returned by express.user()")
	}
	obj := value.ToObject(vm)
	raw, ok := s.authSpecs.Load(obj)
	if !ok {
		return gojahttp.SecuritySpec{}, fmt.Errorf(".auth(...) expects value returned by express.user(); got %s", valueString(value))
	}
	spec, ok := raw.(*gojahttp.SecuritySpec)
	if !ok || spec == nil {
		return gojahttp.SecuritySpec{}, fmt.Errorf("internal auth spec has invalid type")
	}
	return *spec, nil
}

func (s *builderStore) newResourceBuilder(vm *goja.Runtime, resourceType string) (goja.Value, error) {
	resourceType = strings.TrimSpace(resourceType)
	if resourceType == "" {
		return nil, fmt.Errorf("express.resource(type) requires a non-empty type")
	}
	spec := &gojahttp.ResourceSpec{Name: resourceType, Type: resourceType}
	obj := vm.NewObject()
	s.resourceSpecs.Store(obj, spec)
	_ = obj.Set("named", func(name string) (goja.Value, error) {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("resource.named(name) requires a non-empty name")
		}
		spec.Name = name
		return obj, nil
	})
	setIDFromParam := func(param string) (goja.Value, error) {
		param = strings.TrimSpace(param)
		if param == "" {
			return nil, fmt.Errorf("resource.idFromParam(param) requires a non-empty param")
		}
		spec.ID = gojahttp.ValueSource{Kind: gojahttp.ValueSourceParam, Key: param}
		return obj, nil
	}
	_ = obj.Set("idFromParam", setIDFromParam)
	_ = obj.Set("fromParam", setIDFromParam)
	setTenantFromParam := func(param string) (goja.Value, error) {
		param = strings.TrimSpace(param)
		if param == "" {
			return nil, fmt.Errorf("resource.tenantFromParam(param) requires a non-empty param")
		}
		source := gojahttp.ValueSource{Kind: gojahttp.ValueSourceParam, Key: param}
		spec.Tenant = &source
		return obj, nil
	}
	_ = obj.Set("tenantFromParam", setTenantFromParam)
	_ = obj.Set("withinTenantParam", setTenantFromParam)
	_ = obj.Set("mustExist", func() goja.Value {
		spec.MustExist = true
		return obj
	})
	return obj, nil
}

func (s *builderStore) newRateLimitBuilder(vm *goja.Runtime, policy string) (goja.Value, error) {
	policy = strings.TrimSpace(policy)
	if policy == "" {
		return nil, fmt.Errorf("express.rateLimit(policy) requires a non-empty policy")
	}
	spec := &gojahttp.RateLimitSpec{Policy: policy}
	obj := vm.NewObject()
	s.rateLimitSpecs.Store(obj, spec)
	_ = obj.Set("limit", func(count int, window string) (goja.Value, error) {
		d, err := time.ParseDuration(strings.TrimSpace(window))
		if err != nil {
			return nil, fmt.Errorf("rateLimit.limit(%d, %q): %w", count, window, err)
		}
		spec.Limit = count
		spec.Window = d
		return obj, nil
	})
	_ = obj.Set("window", func(window string) (goja.Value, error) {
		d, err := time.ParseDuration(strings.TrimSpace(window))
		if err != nil {
			return nil, fmt.Errorf("rateLimit.window(%q): %w", window, err)
		}
		spec.Window = d
		return obj, nil
	})
	_ = obj.Set("perSecond", func(count int) goja.Value { spec.Limit = count; spec.Window = time.Second; return obj })
	_ = obj.Set("perMinute", func(count int) goja.Value { spec.Limit = count; spec.Window = time.Minute; return obj })
	_ = obj.Set("perHour", func(count int) goja.Value { spec.Limit = count; spec.Window = time.Hour; return obj })
	_ = obj.Set("burst", func(count int) goja.Value { spec.Burst = count; return obj })
	_ = obj.Set("byIP", func() goja.Value {
		spec.KeyParts = append(spec.KeyParts, gojahttp.RateLimitKeyPart{Kind: gojahttp.RateLimitKeyIP})
		return obj
	})
	_ = obj.Set("byRoute", func() goja.Value {
		spec.KeyParts = append(spec.KeyParts, gojahttp.RateLimitKeyPart{Kind: gojahttp.RateLimitKeyRoute})
		return obj
	})
	_ = obj.Set("byActor", func() goja.Value {
		spec.KeyParts = append(spec.KeyParts, gojahttp.RateLimitKeyPart{Kind: gojahttp.RateLimitKeyActor})
		return obj
	})
	_ = obj.Set("byParam", func(param string) goja.Value {
		spec.KeyParts = append(spec.KeyParts, gojahttp.RateLimitKeyPart{Kind: gojahttp.RateLimitKeyParam, Key: strings.TrimSpace(param)})
		return obj
	})
	_ = obj.Set("byTenantParam", func(param string) goja.Value {
		spec.KeyParts = append(spec.KeyParts, gojahttp.RateLimitKeyPart{Kind: gojahttp.RateLimitKeyTenantParam, Key: strings.TrimSpace(param)})
		return obj
	})
	_ = obj.Set("byHeader", func(header string) goja.Value {
		spec.KeyParts = append(spec.KeyParts, gojahttp.RateLimitKeyPart{Kind: gojahttp.RateLimitKeyHeader, Key: strings.TrimSpace(header)})
		return obj
	})
	_ = obj.Set("byBodyField", func(field string) goja.Value {
		spec.KeyParts = append(spec.KeyParts, gojahttp.RateLimitKeyPart{Kind: gojahttp.RateLimitKeyBodyField, Key: strings.TrimSpace(field)})
		return obj
	})
	_ = obj.Set("byResource", func(name string) goja.Value {
		spec.KeyParts = append(spec.KeyParts, gojahttp.RateLimitKeyPart{Kind: gojahttp.RateLimitKeyResource, Key: strings.TrimSpace(name)})
		return obj
	})
	_ = obj.Set("failOpen", func(value bool) goja.Value { spec.FailOpen = value; return obj })
	return obj, nil
}

func (s *builderStore) rateLimitSpec(vm *goja.Runtime, value goja.Value) (gojahttp.RateLimitSpec, error) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return gojahttp.RateLimitSpec{}, fmt.Errorf(".rateLimit(...) expects value returned by express.rateLimit(policy)")
	}
	obj := value.ToObject(vm)
	raw, ok := s.rateLimitSpecs.Load(obj)
	if !ok {
		return gojahttp.RateLimitSpec{}, fmt.Errorf(".rateLimit(...) expects value returned by express.rateLimit(policy); got %s", valueString(value))
	}
	spec, ok := raw.(*gojahttp.RateLimitSpec)
	if !ok || spec == nil {
		return gojahttp.RateLimitSpec{}, fmt.Errorf("internal rate limit spec has invalid type")
	}
	return *spec, nil
}

func (s *builderStore) resourceSpec(vm *goja.Runtime, value goja.Value) (gojahttp.ResourceSpec, error) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return gojahttp.ResourceSpec{}, fmt.Errorf(".resource(...) expects value returned by express.resource(type)")
	}
	obj := value.ToObject(vm)
	raw, ok := s.resourceSpecs.Load(obj)
	if !ok {
		return gojahttp.ResourceSpec{}, fmt.Errorf(".resource(...) expects value returned by express.resource(type); got %s", valueString(value))
	}
	spec, ok := raw.(*gojahttp.ResourceSpec)
	if !ok || spec == nil {
		return gojahttp.ResourceSpec{}, fmt.Errorf("internal resource spec has invalid type")
	}
	return *spec, nil
}

type routeBuilder struct {
	registrar *Registrar
	store     *builderStore
	vm        *goja.Runtime
	plan      gojahttp.RoutePlan
}

func newRouteBuilder(vm *goja.Runtime, registrar *Registrar, store *builderStore, method, pattern string) goja.Value {
	b := &routeBuilder{registrar: registrar, store: store, vm: vm, plan: gojahttp.RoutePlan{Method: method, Pattern: pattern}}
	return b.needsSecurityObject()
}

func (b *routeBuilder) needsSecurityObject() goja.Value {
	obj := b.vm.NewObject()
	_ = obj.Set("name", func(name string) goja.Value {
		b.plan.Name = strings.TrimSpace(name)
		return obj
	})
	_ = obj.Set("public", func() goja.Value {
		b.plan.Security = gojahttp.SecuritySpec{Mode: gojahttp.SecurityModePublic}
		return b.needsHandlerObject()
	})
	_ = obj.Set("auth", func(value goja.Value) (goja.Value, error) {
		spec, err := b.store.authSpec(b.vm, value)
		if err != nil {
			return nil, err
		}
		b.plan.Security = spec
		return b.needsPolicyObject(), nil
	})
	return obj
}

func (b *routeBuilder) needsPolicyObject() goja.Value {
	obj := b.vm.NewObject()
	_ = obj.Set("resource", func(value goja.Value) (goja.Value, error) {
		spec, err := b.store.resourceSpec(b.vm, value)
		if err != nil {
			return nil, err
		}
		b.plan.Resources = append(b.plan.Resources, spec)
		return obj, nil
	})
	b.attachCSRFMethod(obj)
	b.attachAuditMethod(obj)
	b.attachRateLimitMethod(obj)
	_ = obj.Set("allow", func(action string) (goja.Value, error) {
		action = strings.TrimSpace(action)
		if action == "" {
			return nil, fmt.Errorf(".allow(action) requires a non-empty action")
		}
		b.plan.Action = action
		return b.needsHandlerObject(), nil
	})
	return obj
}

func (b *routeBuilder) needsHandlerObject() goja.Value {
	obj := b.vm.NewObject()
	b.attachCSRFMethod(obj)
	b.attachAuditMethod(obj)
	b.attachRateLimitMethod(obj)
	_ = obj.Set("handle", func(handler goja.Value) error {
		fn, ok := goja.AssertFunction(handler)
		if !ok {
			return fmt.Errorf("planned route .handle(...) requires a function")
		}
		return b.registrar.host.RegisterPlanned(b.plan, fn)
	})
	return obj
}

func (b *routeBuilder) attachCSRFMethod(obj *goja.Object) {
	_ = obj.Set("csrf", func(call goja.FunctionCall) goja.Value {
		required := true
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Argument(0)) && !goja.IsNull(call.Argument(0)) {
			required = call.Argument(0).ToBoolean()
		}
		b.plan.CSRF.Required = required
		return obj
	})
}

func (b *routeBuilder) attachRateLimitMethod(obj *goja.Object) {
	_ = obj.Set("rateLimit", func(value goja.Value) (goja.Value, error) {
		spec, err := b.store.rateLimitSpec(b.vm, value)
		if err != nil {
			return nil, err
		}
		b.plan.RateLimits = append(b.plan.RateLimits, spec)
		return obj, nil
	})
}

func (b *routeBuilder) attachAuditMethod(obj *goja.Object) {
	_ = obj.Set("audit", func(event string) (goja.Value, error) {
		event = strings.TrimSpace(event)
		if event == "" {
			return nil, fmt.Errorf(".audit(event) requires a non-empty event")
		}
		b.plan.Audit.Event = event
		return obj, nil
	})
}

func valueString(value goja.Value) string {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return "undefined"
	}
	return value.String()
}
