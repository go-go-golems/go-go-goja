package hostauth

import (
	"fmt"
	"reflect"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

// LookupServiceFactory returns the hostauth service factory from host services,
// when one was configured.
func LookupServiceFactory(host providerapi.HostServices) (ServiceFactory, bool, error) {
	raw, ok := lookupHostService(host, ServiceFactoryKey)
	if !ok {
		return nil, false, nil
	}
	factory, ok := raw.(ServiceFactory)
	if !ok {
		return nil, false, fmt.Errorf("hostauth service %q must implement hostauth.ServiceFactory, got %T", ServiceFactoryKey, raw)
	}
	if isNil(raw) || isNil(factory) {
		return nil, false, fmt.Errorf("hostauth service %q is nil", ServiceFactoryKey)
	}
	return factory, true, nil
}

// LookupServices returns concrete hostauth services from host services, when
// they were configured for a runtime/module setup.
func LookupServices(host providerapi.HostServices) (*Services, bool, error) {
	raw, ok := lookupHostService(host, ServicesKey)
	if !ok {
		return nil, false, nil
	}
	services, ok := raw.(*Services)
	if !ok {
		return nil, false, fmt.Errorf("hostauth service %q must be *hostauth.Services, got %T", ServicesKey, raw)
	}
	if services == nil {
		return nil, false, fmt.Errorf("hostauth service %q is nil", ServicesKey)
	}
	return services, true, nil
}

func lookupHostService(host providerapi.HostServices, key string) (any, bool) {
	lookup, ok := host.(providerapi.HostServiceLookup)
	if !ok || lookup == nil {
		return nil, false
	}
	return lookup.HostService(key)
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	kind := v.Kind()
	if kind == reflect.Chan || kind == reflect.Func || kind == reflect.Interface || kind == reflect.Map || kind == reflect.Pointer || kind == reflect.Slice {
		return v.IsNil()
	}
	return false
}
