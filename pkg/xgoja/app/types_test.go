package app

import (
	"bytes"
	"os"
	"path/filepath"
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

	buf := &bytes.Buffer{}
	if err := executeTypesCommand(t, buf, "types", "--strict"); err != nil {
		t.Fatalf("execute types command: %v", err)
	}
	if !strings.Contains(buf.String(), `declare module "path:runtime"`) {
		t.Fatalf("expected declarations on stdout, got:\n%s", buf.String())
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Fatalf("expected declarations to end with newline")
	}
}

func TestTypesCommandCheckAcceptsTrailingNewline(t *testing.T) {
	t.Parallel()

	host := typedTestHost(t)
	result, err := host.TypeScriptDeclarations(dtsgen.Options{Strict: true})
	if err != nil {
		t.Fatalf("typescript declarations: %v", err)
	}
	path := filepath.Join(t.TempDir(), "xgoja-modules.d.ts")
	if err := os.WriteFile(path, []byte(result.DTS+"\n"), 0o644); err != nil {
		t.Fatalf("write declarations: %v", err)
	}
	if err := executeTypesCommand(t, nil, "types", "--strict", "--check", path); err != nil {
		t.Fatalf("check declarations with trailing newline: %v", err)
	}
}

func executeTypesCommand(t *testing.T, out *bytes.Buffer, args ...string) error {
	t.Helper()
	host := typedTestHost(t)
	root := &cobra.Command{Use: "test"}
	if out == nil {
		out = &bytes.Buffer{}
	}
	root.SetOut(out)
	host.AttachTypes(root)
	root.SetArgs(args)
	return root.Execute()
}

func typedTestHost(t *testing.T) *Host {
	t.Helper()
	registry := providerapi.NewProviderRegistry()
	if err := core.Register(registry); err != nil {
		t.Fatalf("register core provider: %v", err)
	}
	return NewHost(registry, &RuntimePlan{
		Name: "typed-test",
		Modules: []RuntimeModulePlan{{
			Package: core.PackageID,
			Name:    "path",
			As:      "path:runtime",
		}},
	})
}
