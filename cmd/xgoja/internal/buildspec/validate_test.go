package buildspec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateCommandProvidersAcceptsKnownPackageAndRuntime(t *testing.T) {
	spec := validSpec()
	spec.CommandProviders = []CommandProviderInstance{{
		ID:             "fixture-tools",
		Package:        "fixture",
		Name:           "tools",
		RuntimeProfile: "main",
	}}

	report := Validate(spec)
	if report.HasErrors() {
		t.Fatalf("expected no validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusOK, "command-provider-runtime", "commandProviders[0].runtimeProfile")
	assertCheck(t, report, StatusOK, "command-providers", "commandProviders")
}

func TestValidateCommandProvidersRejectsInvalidEntries(t *testing.T) {
	spec := validSpec()
	spec.CommandProviders = []CommandProviderInstance{
		{ID: "dup", Package: "missing", Name: "tools", RuntimeProfile: "missing"},
		{ID: "dup", Package: "fixture"},
	}

	report := Validate(spec)
	if !report.HasErrors() {
		t.Fatalf("expected validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusError, "command-provider-package", "commandProviders[0].package")
	assertCheck(t, report, StatusError, "command-provider-runtime", "commandProviders[0].runtimeProfile")
	assertCheck(t, report, StatusError, "command-provider-id", "commandProviders[1].id")
	assertCheck(t, report, StatusError, "command-provider-name", "commandProviders[1].name")
}

func TestValidateHelpSourcesAcceptsProviderAndEmbeddedLocalSources(t *testing.T) {
	dir := t.TempDir()
	helpDir := filepath.Join(dir, "docs", "help")
	if err := os.MkdirAll(helpDir, 0o755); err != nil {
		t.Fatalf("mkdir help dir: %v", err)
	}
	spec := validSpec()
	spec.BaseDir = dir
	spec.Help.Sources = []HelpSourceSpec{
		{ID: "fixture-docs", Package: "fixture", Source: "docs"},
		{ID: "local-docs", Path: "docs/help", Embed: true},
		{ID: "dev-docs", Path: "../runtime-docs"},
	}

	report := Validate(spec)
	if report.HasErrors() {
		t.Fatalf("expected no validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusOK, "help-provider-source", "help.sources[0]")
	assertCheck(t, report, StatusOK, "help-path", "help.sources[1].path")
	assertCheck(t, report, StatusOK, "help-path", "help.sources[2].path")
}

func TestValidateHelpSourcesRejectsInvalidEntries(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "not-a-dir.md")
	if err := os.WriteFile(filePath, []byte("docs"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	spec := validSpec()
	spec.BaseDir = dir
	spec.Help.Sources = []HelpSourceSpec{
		{ID: "dup", Package: "fixture"},
		{ID: "dup", Package: "missing", Source: "docs"},
		{ID: "mixed", Path: "docs", Package: "fixture", Source: "docs"},
		{ID: "missing-path"},
		{ID: "file", Path: "not-a-dir.md", Embed: true},
		{Path: "docs"},
	}

	report := Validate(spec)
	if !report.HasErrors() {
		t.Fatalf("expected validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusError, "help-provider-source", "help.sources[0]")
	assertCheck(t, report, StatusError, "help-source-id", "help.sources[1].id")
	assertCheck(t, report, StatusError, "help-provider-source", "help.sources[1].package")
	assertCheck(t, report, StatusError, "help-source-shape", "help.sources[2]")
	assertCheck(t, report, StatusError, "help-path", "help.sources[3].path")
	assertCheck(t, report, StatusError, "help-path", "help.sources[4].path")
	assertCheck(t, report, StatusError, "help-source-id", "help.sources[5].id")
}

func validSpec() *Spec {
	return &Spec{
		Name: "fixture",
		Target: TargetSpec{
			Kind:   "xgoja",
			Output: "dist/fixture",
		},
		Packages: []PackageSpec{{
			ID:     "fixture",
			Import: "github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider",
		}},
		Runtimes: map[string]Runtime{
			"main": {
				Modules: []ModuleInstance{{
					Package: "fixture",
					Name:    "hello",
					As:      "hello",
				}},
			},
		},
		Commands: CommandsSpec{
			Run: CommandSpec{Enabled: false},
		},
	}
}

func assertCheck(t *testing.T, report *Report, status Status, name string, path string) {
	t.Helper()
	for _, check := range report.Checks {
		if check.Status == status && check.Name == name && check.Path == path {
			return
		}
	}
	t.Fatalf("missing %s check %s at %s in %#v", status, name, path, report.Checks)
}
