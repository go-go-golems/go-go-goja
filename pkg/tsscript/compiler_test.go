package tsscript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/evanw/esbuild/pkg/api"
)

func TestTransformSourceStripsTypesAndRunsInGoja(t *testing.T) {
	artifact, err := TransformSource(Source{
		Path: "example.ts",
		Contents: []byte(`
			type User = { name: string }
			const user: User = { name: "Manuel" }
			globalThis.result = "hello " + user.name
		`),
	}, Options{Format: api.FormatIIFE})
	if err != nil {
		t.Fatalf("TransformSource() error = %v", err)
	}
	if strings.Contains(string(artifact.Code), "type User") {
		t.Fatalf("transformed code still contains TypeScript type declaration:\n%s", artifact.Code)
	}
	vm := goja.New()
	if _, err := vm.RunScript("example.js", string(artifact.Code)); err != nil {
		t.Fatalf("RunScript() error = %v\n%s", err, artifact.Code)
	}
	if got := vm.Get("result").String(); got != "hello Manuel" {
		t.Fatalf("result = %q, want %q", got, "hello Manuel")
	}
}

func TestBundleVirtualEntryFollowsTypeScriptImports(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "helper.ts"), []byte(`export const answer: number = 42`), 0o644); err != nil {
		t.Fatal(err)
	}

	artifact, err := BundleVirtualEntry(Source{
		Path:       "entry.ts",
		ResolveDir: dir,
		Contents: []byte(`
			import { answer } from "./helper"
			globalThis.answer = answer
		`),
	}, Options{Format: api.FormatIIFE})
	if err != nil {
		t.Fatalf("BundleVirtualEntry() error = %v", err)
	}
	vm := goja.New()
	if _, err := vm.RunScript("entry.js", string(artifact.Code)); err != nil {
		t.Fatalf("RunScript() error = %v\n%s", err, artifact.Code)
	}
	if got := vm.Get("answer").ToInteger(); got != 42 {
		t.Fatalf("answer = %d, want 42", got)
	}
}

func TestBundleEntryPreservesExternalRequire(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "entry.ts")
	if err := os.WriteFile(entry, []byte(`
		import native from "native-module"
		globalThis.value = native.value
	`), 0o644); err != nil {
		t.Fatal(err)
	}

	artifact, err := BundleEntry(entry, Options{External: []string{"native-module"}})
	if err != nil {
		t.Fatalf("BundleEntry() error = %v", err)
	}
	code := string(artifact.Code)
	if !strings.Contains(code, `require("native-module")`) {
		t.Fatalf("bundle did not preserve external require for native-module:\n%s", code)
	}
}

func TestTransformSourceReportsDiagnostics(t *testing.T) {
	_, err := TransformSource(Source{Path: "broken.ts", Contents: []byte(`const =`)}, Options{})
	if err == nil {
		t.Fatalf("TransformSource() expected error")
	}
	if !strings.Contains(err.Error(), "broken.ts") {
		t.Fatalf("error %q does not mention source file", err.Error())
	}
}

func TestLoaderForPath(t *testing.T) {
	cases := map[string]api.Loader{
		"a.ts":   api.LoaderTS,
		"a.tsx":  api.LoaderTSX,
		"a.mts":  api.LoaderTS,
		"a.cts":  api.LoaderTS,
		"a.jsx":  api.LoaderJSX,
		"a.json": api.LoaderJSON,
		"a.js":   api.LoaderJS,
	}
	for path, want := range cases {
		if got := LoaderForPath(path); got != want {
			t.Fatalf("LoaderForPath(%q) = %v, want %v", path, got, want)
		}
	}
}
