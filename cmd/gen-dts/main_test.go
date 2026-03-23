package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
)

type fakeModule struct {
	name       string
	descriptor *spec.Module
}

var _ modules.NativeModule = (*fakeModule)(nil)
var _ modules.TypeScriptDeclarer = (*fakeModule)(nil)

func (m *fakeModule) Name() string { return m.name }
func (m *fakeModule) Doc() string  { return "" }
func (m *fakeModule) Loader(*goja.Runtime, *goja.Object) {
}
func (m *fakeModule) TypeScriptModule() *spec.Module {
	return m.descriptor
}

type fakeModuleNoTypes struct {
	name string
}

var _ modules.NativeModule = (*fakeModuleNoTypes)(nil)

func (m *fakeModuleNoTypes) Name() string { return m.name }
func (m *fakeModuleNoTypes) Doc() string  { return "" }
func (m *fakeModuleNoTypes) Loader(*goja.Runtime, *goja.Object) {
}

func TestGenerateFromModules(t *testing.T) {
	t.Parallel()

	all := []modules.NativeModule{
		&fakeModuleNoTypes{name: "untagged"},
		&fakeModule{
			name: "fs",
			descriptor: &spec.Module{
				Name: "fs",
				Functions: []spec.Function{
					{Name: "readFileSync", Returns: spec.String()},
				},
			},
		},
	}

	t.Run("strict errors on missing descriptors", func(t *testing.T) {
		t.Parallel()
		_, err := generateFromModules(all, options{Strict: true})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("module filter renders selected descriptor", func(t *testing.T) {
		t.Parallel()
		out, err := generateFromModules(all, options{Modules: []string{"fs"}, Strict: true})
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		if !strings.Contains(out, `declare module "fs"`) {
			t.Fatalf("expected fs declaration, got: %s", out)
		}
		if strings.Contains(out, `declare module "untagged"`) {
			t.Fatalf("unexpected untagged declaration in output: %s", out)
		}
	})

	t.Run("missing selected module fails", func(t *testing.T) {
		t.Parallel()
		_, err := generateFromModules(all, options{Modules: []string{"missing"}})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

func TestWriteOrCheck(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "types.d.ts")
	content := "declare module \"x\" {}\n"

	if err := writeOrCheck(path, content, false); err != nil {
		t.Fatalf("write mode failed: %v", err)
	}
	if err := writeOrCheck(path, content, true); err != nil {
		t.Fatalf("check mode failed: %v", err)
	}

	if err := os.WriteFile(path, []byte("different\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := writeOrCheck(path, content, true); err == nil {
		t.Fatalf("expected check mismatch error")
	}
}

func TestParseOptions(t *testing.T) {
	t.Parallel()

	opts, err := parseOptions([]string{
		"--out", "/tmp/out.d.ts",
		"--module", "fs, exec,fs",
		"--strict",
		"--check",
	})
	if err != nil {
		t.Fatalf("parse options: %v", err)
	}

	if opts.Out != "/tmp/out.d.ts" {
		t.Fatalf("unexpected out: %s", opts.Out)
	}
	if !opts.Strict || !opts.Check {
		t.Fatalf("strict/check flags should be true: %+v", opts)
	}

	if len(opts.Modules) != 2 || opts.Modules[0] != "exec" || opts.Modules[1] != "fs" {
		t.Fatalf("unexpected modules parse result: %#v", opts.Modules)
	}

	if _, err := parseOptions([]string{}); err == nil {
		t.Fatalf("expected error when out is missing")
	}
}
