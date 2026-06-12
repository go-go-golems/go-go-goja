package migratebuildspec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateCommandProvidersAcceptsKnownPackage(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.CommandProviders = []CommandProviderInstanceSpec{{
		ID:      "fixture-tools",
		Package: "fixture",
		Name:    "tools",
	}}

	report := Validate(buildSpec)
	if report.HasErrors() {
		t.Fatalf("expected no validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusOK, "command-providers", "commandProviders")
}

func TestValidateCommandProvidersRejectsInvalidEntries(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.CommandProviders = []CommandProviderInstanceSpec{
		{ID: "dup", Package: "missing", Name: "tools"},
		{ID: "dup", Package: "fixture"},
	}

	report := Validate(buildSpec)
	if !report.HasErrors() {
		t.Fatalf("expected validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusError, "command-provider-package", "commandProviders[0].package")
	assertCheck(t, report, StatusError, "command-provider-id", "commandProviders[1].id")
	assertCheck(t, report, StatusError, "command-provider-name", "commandProviders[1].name")
}

func TestValidateGoImportsAcceptsBlankAndNamedImports(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.Go.Imports = []GoImportSpec{
		{Import: "github.com/lib/pq", Alias: "_", Version: "v1.10.9"},
		{Import: "example.com/app/hooks", Alias: "hooks", Module: "example.com/app", Version: "v0.1.0"},
	}

	report := Validate(buildSpec)
	if report.HasErrors() {
		t.Fatalf("expected no validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusOK, "go-imports", "go.imports")
	assertCheck(t, report, StatusOK, "go-import", "go.imports[0].import")
	assertCheck(t, report, StatusOK, "go-import-alias", "go.imports[0].alias")
}

func TestValidateGoImportsRejectsInvalidEntries(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.Go.Imports = []GoImportSpec{
		{Alias: "_"},
		{Import: "github.com/lib/pq", Alias: "bad-alias"},
		{Import: "example.com/one", Alias: "dup"},
		{Import: "example.com/two", Alias: "dup"},
		{Import: "github.com/lib/pq", Alias: "_"},
		{Import: "github.com/lib/pq", Alias: "_"},
	}

	report := Validate(buildSpec)
	if !report.HasErrors() {
		t.Fatalf("expected validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusError, "go-import", "go.imports[0].import")
	assertCheck(t, report, StatusError, "go-import-alias", "go.imports[1].alias")
	assertCheck(t, report, StatusError, "go-import-alias", "go.imports[3].alias")
	assertCheck(t, report, StatusError, "go-import", "go.imports[5].import")
}

func TestValidateHelpSourcesAcceptsProviderAndEmbeddedLocalSources(t *testing.T) {
	dir := t.TempDir()
	helpDir := filepath.Join(dir, "docs", "help")
	if err := os.MkdirAll(helpDir, 0o755); err != nil {
		t.Fatalf("mkdir help dir: %v", err)
	}
	buildSpec := validSpec()
	buildSpec.BaseDir = dir
	buildSpec.Help.Sources = []HelpSourceSpec{
		{ID: "fixture-docs", Package: "fixture", Source: "docs"},
		{ID: "local-docs", Path: "docs/help", Embed: true},
		{ID: "dev-docs", Path: "../runtime-docs"},
	}

	report := Validate(buildSpec)
	if report.HasErrors() {
		t.Fatalf("expected no validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusOK, "help-provider-source", "help.sources[0]")
	assertCheck(t, report, StatusOK, "help-path", "help.sources[1].path")
	assertCheck(t, report, StatusOK, "help-path", "help.sources[2].path")
}

func TestValidateAssetsAcceptsEmbeddedLocalSources(t *testing.T) {
	dir := t.TempDir()
	assetsDir := filepath.Join(dir, "assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatalf("mkdir assets dir: %v", err)
	}
	buildSpec := validSpec()
	buildSpec.BaseDir = dir
	buildSpec.Assets = []AssetSourceSpec{{
		ID:          "app-assets",
		Path:        "assets",
		Embed:       true,
		Description: "application assets",
	}}

	report := Validate(buildSpec)
	if report.HasErrors() {
		t.Fatalf("expected no validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusOK, "asset-path", "assets[0].path")
}

func TestValidateAssetsRejectsInvalidEntries(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "not-a-dir.txt")
	if err := os.WriteFile(filePath, []byte("asset"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	buildSpec := validSpec()
	buildSpec.BaseDir = dir
	buildSpec.Assets = []AssetSourceSpec{
		{ID: "dup", Path: "not-a-dir.txt", Embed: true},
		{ID: "dup", Path: "missing", Embed: true},
		{ID: "runtime", Path: "runtime", Embed: false},
		{ID: "missing-path", Embed: true},
		{Path: "assets", Embed: true},
	}

	report := Validate(buildSpec)
	if !report.HasErrors() {
		t.Fatalf("expected validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusError, "asset-path", "assets[0].path")
	assertCheck(t, report, StatusError, "asset-id", "assets[1].id")
	assertCheck(t, report, StatusError, "asset-path", "assets[1].path")
	assertCheck(t, report, StatusError, "asset-embed", "assets[2].embed")
	assertCheck(t, report, StatusError, "asset-path", "assets[3].path")
	assertCheck(t, report, StatusError, "asset-id", "assets[4].id")
}

func TestValidateHelpSourcesRejectsInvalidEntries(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "not-a-dir.md")
	if err := os.WriteFile(filePath, []byte("docs"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	buildSpec := validSpec()
	buildSpec.BaseDir = dir
	buildSpec.Help.Sources = []HelpSourceSpec{
		{ID: "dup", Package: "fixture"},
		{ID: "dup", Package: "missing", Source: "docs"},
		{ID: "mixed", Path: "docs", Package: "fixture", Source: "docs"},
		{ID: "missing-path"},
		{ID: "file", Path: "not-a-dir.md", Embed: true},
		{Path: "docs"},
	}

	report := Validate(buildSpec)
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

func validSpec() *BuildSpec {
	return &BuildSpec{
		Name: "fixture",
		Target: TargetSpec{
			Kind:   "xgoja",
			Output: "dist/fixture",
		},
		Packages: []PackageSpec{{
			ID:     "fixture",
			Import: "github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider",
		}},
		Modules: []ModuleInstanceSpec{{
			Package: "fixture",
			Name:    "hello",
			As:      "hello",
		}},
		Commands: CommandsSpec{
			Run: CommandSpec{Enabled: false},
		},
	}
}

func TestValidatePackageTargetAcceptsPackageOutput(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.Target = TargetSpec{Kind: "package", Output: "internal/xgojaruntime", Package: "xgojaruntime"}

	report := Validate(buildSpec)
	if report.HasErrors() {
		t.Fatalf("expected no validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusOK, "target-kind", "target.kind")
	assertCheck(t, report, StatusOK, "target-package", "target.package")
}

func TestValidatePackageTargetRejectsInvalidPackageName(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.Target = TargetSpec{Kind: "package", Output: "internal/bad-name", Package: "bad-name"}

	report := Validate(buildSpec)
	if !report.HasErrors() {
		t.Fatalf("expected validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusError, "target-package", "target.package")
}

func TestValidateSourceTargetAcceptsPackageOutput(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.Target = TargetSpec{Kind: "source", Output: "internal/xgojaruntime", Package: "xgojaruntime"}

	report := Validate(buildSpec)
	if report.HasErrors() {
		t.Fatalf("expected no validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusOK, "target-kind", "target.kind")
	assertCheck(t, report, StatusOK, "target-package", "target.package")
}

func TestValidateTemplateTargetRequiresExistingTemplate(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.Target = TargetSpec{Kind: "template", Output: "internal/xgojaruntime/custom.gen.go", Package: "xgojaruntime"}

	report := Validate(buildSpec)
	if !report.HasErrors() {
		t.Fatalf("expected validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusError, "target-template", "target.template")
}

func TestValidateTemplateTargetAcceptsExistingTemplate(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "runtime.go.tmpl")
	if err := os.WriteFile(templatePath, []byte("package {{ .PackageName }}\n"), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}
	buildSpec := validSpec()
	buildSpec.BaseDir = dir
	buildSpec.Target = TargetSpec{Kind: "template", Output: "internal/xgojaruntime/custom.gen.go", Package: "xgojaruntime", Template: "runtime.go.tmpl"}

	report := Validate(buildSpec)
	if report.HasErrors() {
		t.Fatalf("expected no validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusOK, "target-template", "target.template")
}

func TestValidateJSVerbCommandMountAcceptsRootAliases(t *testing.T) {
	for _, mount := range []string{"root", "/", "."} {
		buildSpec := validSpec()
		buildSpec.Commands.JSVerbs = CommandSpec{Enabled: true, Mount: mount}

		report := Validate(buildSpec)
		if report.HasErrors() {
			t.Fatalf("mount %q: expected no validation errors, got %#v", mount, report.Checks)
		}
		assertCheck(t, report, StatusOK, "command-mount", "commands.jsverbs.mount")
	}
}

func TestValidateJSVerbCommandMountRejectsUnknownValue(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.Commands.JSVerbs = CommandSpec{Enabled: true, Mount: "top-level"}

	report := Validate(buildSpec)
	if !report.HasErrors() {
		t.Fatalf("expected validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusError, "command-mount", "commands.jsverbs.mount")
}

func TestValidateJSVerbFiltersRejectEmptyPatterns(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.JSVerbs = []JSVerbSourceSpec{{ID: "site", Path: "verbs", Include: []string{""}, Exclude: []string{"assets/**"}, Extensions: []string{" "}}}

	report := Validate(buildSpec)
	if !report.HasErrors() {
		t.Fatalf("expected validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusError, "jsverb-include", "jsverbs[0].include[0]")
	assertCheck(t, report, StatusError, "jsverb-extension", "jsverbs[0].extensions[0]")
}

func TestValidateJSVerbFiltersAcceptNonEmptyPatterns(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.JSVerbs = []JSVerbSourceSpec{{ID: "site", Path: "verbs", Include: []string{"site.js"}, Exclude: []string{"assets/**"}, Extensions: []string{"js", ".cjs"}}}

	report := Validate(buildSpec)
	if report.HasErrors() {
		t.Fatalf("expected no validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusOK, "jsverb-filters", "jsverbs[0]")
}

func TestValidateJSVerbFiltersRejectMalformedGlobs(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.JSVerbs = []JSVerbSourceSpec{{ID: "site", Path: "verbs", Include: []string{"verbs/["}, Exclude: []string{"assets/["}}}

	report := Validate(buildSpec)
	if !report.HasErrors() {
		t.Fatalf("expected validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusError, "jsverb-include", "jsverbs[0].include[0]")
	assertCheck(t, report, StatusError, "jsverb-exclude", "jsverbs[0].exclude[0]")
}

func TestValidateTypeScriptSpecAcceptsSupportedValues(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.JSVerbs = []JSVerbSourceSpec{{
		ID:         "site",
		Path:       "verbs",
		Extensions: []string{".ts", ".tsx"},
		TypeScript: &TypeScriptSpec{
			Enabled:      true,
			Bundle:       true,
			Target:       "es2015",
			Format:       "cjs",
			Platform:     "neutral",
			Sourcemap:    "inline",
			External:     []string{"express"},
			CheckCommand: []string{"tsc", "--noEmit"},
		},
	}}

	report := Validate(buildSpec)
	if report.HasErrors() {
		t.Fatalf("expected no validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusOK, "typescript", "jsverbs[0].typescript")
	assertCheck(t, report, StatusOK, "typescript-bundle", "jsverbs[0].typescript.bundle")
}

func TestValidateTypeScriptSpecRejectsUnsupportedValues(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.JSVerbs = []JSVerbSourceSpec{{
		ID:   "site",
		Path: "verbs",
		TypeScript: &TypeScriptSpec{
			Enabled:      true,
			Target:       "future-js",
			Format:       "amd",
			Platform:     "deno",
			Sourcemap:    "surprise",
			External:     []string{""},
			CheckCommand: []string{"tsc", ""},
		},
	}}

	report := Validate(buildSpec)
	if !report.HasErrors() {
		t.Fatalf("expected validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusError, "typescript-target", "jsverbs[0].typescript.target")
	assertCheck(t, report, StatusError, "typescript-format", "jsverbs[0].typescript.format")
	assertCheck(t, report, StatusError, "typescript-platform", "jsverbs[0].typescript.platform")
	assertCheck(t, report, StatusError, "typescript-sourcemap", "jsverbs[0].typescript.sourcemap")
	assertCheck(t, report, StatusError, "typescript-external", "jsverbs[0].typescript.external[0]")
	assertCheck(t, report, StatusError, "typescript-check-command", "jsverbs[0].typescript.checkCommand[1]")
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

func TestValidateAppSettingsAcceptsAppNameAndEnvPrefix(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.AppName = "my-app"
	buildSpec.EnvPrefix = "MY_APP"

	report := Validate(buildSpec)
	if report.HasErrors() {
		t.Fatalf("expected no validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusOK, "app-name", "appName")
	assertCheck(t, report, StatusOK, "env-prefix", "envPrefix")
}

func TestValidateAppSettingsRejectsInvalidEnvPrefix(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.EnvPrefix = "my-app"

	report := Validate(buildSpec)
	if !report.HasErrors() {
		t.Fatalf("expected validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusError, "env-prefix", "envPrefix")
}

func TestValidateConfigRequiresAppNameForAppScopedLayers(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.ConfigFile = &ConfigFileSpec{Enabled: true, Layers: []string{"system"}}

	report := Validate(buildSpec)
	if !report.HasErrors() {
		t.Fatalf("expected validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusError, "config-app-name", "config")
}

func TestValidateConfigAllowsLocalLayersWithoutAppName(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.ConfigFile = &ConfigFileSpec{Enabled: true, Layers: []string{"cwd", "git-root", "explicit"}}

	report := Validate(buildSpec)
	if report.HasErrors() {
		t.Fatalf("expected no validation errors, got %#v", report.Checks)
	}
}

func TestValidateConfigRejectsUnknownLayers(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.AppName = "my-app"
	buildSpec.ConfigFile = &ConfigFileSpec{Enabled: true, Layers: []string{"cwd", "unknown", "xdg"}}

	report := Validate(buildSpec)
	if !report.HasErrors() {
		t.Fatalf("expected validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusError, "config-layer", "config.layers[1]")
	assertCheck(t, report, StatusOK, "config-layer", "config.layers[0]")
	assertCheck(t, report, StatusOK, "config-layer", "config.layers[2]")
}

func TestValidateConfigAcceptsValidLayers(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.AppName = "my-app"
	buildSpec.ConfigFile = &ConfigFileSpec{Enabled: true, Layers: []string{"system", "xdg", "home", "git-root", "cwd", "explicit"}}

	report := Validate(buildSpec)
	if report.HasErrors() {
		t.Fatalf("expected no validation errors, got %#v", report.Checks)
	}
}

func TestValidateConfigRequiresLayers(t *testing.T) {
	buildSpec := validSpec()
	buildSpec.AppName = "my-app"
	buildSpec.ConfigFile = &ConfigFileSpec{Enabled: true, Layers: []string{}}

	report := Validate(buildSpec)
	if !report.HasErrors() {
		t.Fatalf("expected validation errors, got %#v", report.Checks)
	}
	assertCheck(t, report, StatusError, "config-layers", "config.layers")
}
