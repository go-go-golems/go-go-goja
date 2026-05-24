package core

import (
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestRegisterCoreProvider(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register core provider: %v", err)
	}
	pkg := registry.Packages()
	if len(pkg) != 1 {
		t.Fatalf("packages len = %d, want 1", len(pkg))
	}
	if pkg[0].ID != PackageID {
		t.Fatalf("package ID = %q, want %q", pkg[0].ID, PackageID)
	}
	for _, name := range []string{"path", "node:path", "yaml", "crypto", "node:crypto", "time", "timer", "events", "node:events"} {
		if _, ok := registry.ResolveModule(PackageID, name); !ok {
			t.Fatalf("expected core module %q", name)
		}
	}
}
