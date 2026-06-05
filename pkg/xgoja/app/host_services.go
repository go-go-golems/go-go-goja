package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

type hostServiceCollector struct {
	base     providerapi.HostServices
	services map[string][]any
	closers  []func(context.Context) error
}

var _ providerapi.HostServiceSink = (*hostServiceCollector)(nil)

func newHostServiceCollector(base providerapi.HostServices) *hostServiceCollector {
	return &hostServiceCollector{
		base:     base,
		services: map[string][]any{},
	}
}

func (c *hostServiceCollector) AddHostService(key string, value any) error {
	if c == nil {
		return fmt.Errorf("host service collector is nil")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("host service key is required")
	}
	if value == nil {
		return fmt.Errorf("host service %q value is nil", key)
	}
	if c.services == nil {
		c.services = map[string][]any{}
	}
	c.services[key] = append(c.services[key], value)
	return nil
}

func (c *hostServiceCollector) AddCloser(fn func(context.Context) error) error {
	if c == nil {
		return fmt.Errorf("host service collector is nil")
	}
	if fn == nil {
		return fmt.Errorf("host service closer is nil")
	}
	var once sync.Once
	var onceErr error
	c.closers = append(c.closers, func(ctx context.Context) error {
		once.Do(func() {
			onceErr = fn(ctx)
		})
		return onceErr
	})
	return nil
}

func (c *hostServiceCollector) servicesForRuntime() hostServicesForRuntime {
	if c == nil {
		return hostServicesForRuntime{}
	}
	return hostServicesForRuntime{
		services: contributedHostServices{base: c.base, services: cloneHostServiceMap(c.services)},
		closers:  append([]func(context.Context) error(nil), c.closers...),
	}
}

type hostServicesForRuntime struct {
	services providerapi.HostServices
	closers  []func(context.Context) error
}

type hostServiceCloserRegistrar struct {
	closers []func(context.Context) error
}

func (r hostServiceCloserRegistrar) ID() string { return "xgoja:host-service-closers" }

func (r hostServiceCloserRegistrar) RegisterRuntimeModule(ctx *engine.RuntimeModuleRegistrationContext, _ *require.Registry) error {
	if ctx == nil || ctx.AddCloser == nil {
		return fmt.Errorf("runtime closer registration is unavailable")
	}
	for i, closer := range r.closers {
		if closer == nil {
			continue
		}
		if err := ctx.AddCloser(closer); err != nil {
			return fmt.Errorf("register host service closer %d: %w", i, err)
		}
	}
	return nil
}

type contributedHostServices struct {
	base     providerapi.HostServices
	services map[string][]any
}

var _ providerapi.HostServices = contributedHostServices{}
var _ providerapi.HostServiceLookup = contributedHostServices{}

func (s contributedHostServices) AssetResolver() providerapi.AssetResolver {
	if s.base == nil {
		return nil
	}
	return s.base.AssetResolver()
}

func (s contributedHostServices) HostService(key string) (any, bool) {
	values := s.HostServiceValues(key)
	switch len(values) {
	case 0:
		return nil, false
	case 1:
		return values[0], true
	default:
		return values, true
	}
}

func (s contributedHostServices) HostServiceValues(key string) []any {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	out := []any{}
	if lookup, ok := s.base.(providerapi.HostServiceLookup); ok && lookup != nil {
		out = append(out, lookup.HostServiceValues(key)...)
	}
	if s.services != nil {
		out = append(out, s.services[key]...)
	}
	return append([]any(nil), out...)
}

func closeHostServiceClosers(ctx context.Context, closers []func(context.Context) error) error {
	var ret error
	for i := len(closers) - 1; i >= 0; i-- {
		if closers[i] == nil {
			continue
		}
		if err := closers[i](ctx); err != nil {
			ret = errors.Join(ret, err)
		}
	}
	return ret
}

func cloneHostServiceMap(in map[string][]any) map[string][]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string][]any, len(in))
	for key, values := range in {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = append([]any(nil), values...)
	}
	return out
}
