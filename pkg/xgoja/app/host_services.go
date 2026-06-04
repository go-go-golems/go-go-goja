package app

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

type hostServiceCollector struct {
	base     providerapi.HostServices
	services map[string][]any
}

var _ providerapi.HostServiceSink = (*hostServiceCollector)(nil)

func newHostServiceCollector(base providerapi.HostServices) *hostServiceCollector {
	collector := &hostServiceCollector{
		base:     base,
		services: map[string][]any{},
	}
	if concrete, ok := base.(HostServices); ok {
		for key, values := range concrete.Services {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			collector.services[key] = append([]any(nil), values...)
		}
	}
	return collector
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

func (c *hostServiceCollector) servicesForRuntime() providerapi.HostServices {
	if c == nil {
		return nil
	}
	return contributedHostServices{base: c.base, services: cloneHostServiceMap(c.services)}
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
