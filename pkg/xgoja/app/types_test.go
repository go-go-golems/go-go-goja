package app

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/dtsgen"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/core"
	"github.com/spf13/cobra"
)

func TestHostTypeScriptDeclarationsUsesSelectedAliases(t *testing.T) {
	t.Parallel()

	host := typedTestHost(t)
	result, err := host.TypeScriptDeclarations(dtsgen.Options{Strict: true})
	if err != nil {
		t.Fatalf("typescript declarations: %v", err)
	}
	if !strings.Contains(result.DTS, `declare module "path:runtime"`) {
		t.Fatalf("expected aliased module declaration, got:\n%s", result.DTS)
	}
}

func TestTypesCommandPrintsDeclarations(t *testing.T) {
	t.Parallel()

	host := typedTestHost(t)
	root := &cobra.Command{Use: "test"}
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	host.AttachTypes(root)
	root.SetArgs([]string{"types", "--strict"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute types command: %v", err)
	}
	if !strings.Contains(buf.String(), `declare module "path:runtime"`) {
		t.Fatalf("expected declarations on stdout, got:\n%s", buf.String())
	}
}

func typedTestHost(t *testing.T) *Host {
	t.Helper()
	registry := providerapi.NewProviderRegistry()
	if err := core.Register(registry); err != nil {
		t.Fatalf("register core provider: %v", err)
	}
	return NewHost(registry, &RuntimeSpec{
		Name: "typed-test",
		Modules: []ModuleInstanceSpec{{
			Package: core.PackageID,
			Name:    "path",
			As:      "path:runtime",
		}},
	})
}
