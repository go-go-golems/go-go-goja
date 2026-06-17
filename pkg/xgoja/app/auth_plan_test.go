package app

import (
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestNewHostInstallsRuntimePlanAuthServiceFactory(t *testing.T) {
	runtimePlan := &RuntimePlan{
		Schema: RuntimePlanSchema,
		Name:   "auth-app",
		Auth: &hostauth.Config{
			Mode: hostauth.ModeDev,
			Stores: hostauth.StoresConfig{Default: hostauth.StoreConfig{
				Driver: string(hostauth.StoreDriverSQLite),
				DSN:    "file:auth.db",
			}},
		},
	}

	host := NewHost(providerapi.NewProviderRegistry(), runtimePlan)
	lookup, ok := host.Services.HostService(hostauth.ServiceFactoryKey)
	if !ok {
		t.Fatalf("expected host service %q", hostauth.ServiceFactoryKey)
	}
	factory, ok := lookup.(hostauth.ServiceFactory)
	if !ok {
		t.Fatalf("hostauth service = %T, want hostauth.ServiceFactory", lookup)
	}
	defaultsProvider, ok := factory.(hostauth.ConfigDefaultsProvider)
	if !ok {
		t.Fatalf("hostauth factory = %T, want ConfigDefaultsProvider", factory)
	}
	defaults := defaultsProvider.AuthConfigDefaults()
	if defaults.Mode != hostauth.ModeDev {
		t.Fatalf("auth mode = %q, want %q", defaults.Mode, hostauth.ModeDev)
	}
	if defaults.Stores.Default.DSN != "file:auth.db" {
		t.Fatalf("default store dsn = %q, want file:auth.db", defaults.Stores.Default.DSN)
	}
}
