package specv2

import (
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildspec"
)

func TestMigrateV1TypeScriptJSVerbs(t *testing.T) {
	v1 := &buildspec.BuildSpec{
		Name:    "typescript-jsverbs",
		AppName: "typescript-jsverbs",
		Go: buildspec.GoSpec{
			Version: "1.26",
			Module:  "xgoja.generated/typescript-jsverbs",
		},
		Target: buildspec.TargetSpec{Kind: "xgoja", Output: "dist/typescript-jsverbs"},
		Packages: []buildspec.PackageSpec{{
			ID:       "go-go-goja-http",
			Import:   "github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http",
			Register: "Register",
		}},
		Modules: []buildspec.ModuleInstanceSpec{{
			Package: "go-go-goja-http",
			Name:    "express",
		}},
		Commands: buildspec.CommandsSpec{
			Run:     buildspec.CommandSpec{Enabled: true, Name: "run"},
			JSVerbs: buildspec.CommandSpec{Enabled: true, Name: "verbs"},
		},
		CommandProviders: []buildspec.CommandProviderInstanceSpec{{
			ID:      "http-serve",
			Package: "go-go-goja-http",
			Name:    "serve",
			Mount:   "serve",
		}},
		JSVerbs: []buildspec.JSVerbSourceSpec{{
			ID:         "local-sites",
			Path:       "./verbs",
			Extensions: []string{".ts"},
			Include:    []string{"**/*.ts"},
			Exclude:    []string{"**/*.test.ts"},
			TypeScript: &buildspec.TypeScriptSpec{
				Enabled:  true,
				Bundle:   true,
				Target:   "es2015",
				Format:   "cjs",
				Platform: "neutral",
				External: []string{"express"},
			},
		}},
	}

	result := MigrateV1(v1)
	cfg := result.Config
	if cfg.Schema != Schema {
		t.Fatalf("schema = %q", cfg.Schema)
	}
	if got := cfg.Providers[0].ID; got != "go-go-goja-http" {
		t.Fatalf("provider id = %q", got)
	}
	if got := cfg.Runtime.Modules[0].Alias(); got != "express" {
		t.Fatalf("runtime alias = %q", got)
	}
	if got := cfg.Sources[0].Language; got != "typescript" {
		t.Fatalf("source language = %q", got)
	}
	if cfg.Sources[0].Compile == nil || !cfg.Sources[0].Compile.Bundle {
		t.Fatalf("compile bundle not migrated: %#v", cfg.Sources[0].Compile)
	}
	if got := cfg.Commands[1].Sources; len(got) != 1 || got[0] != "local-sites" {
		t.Fatalf("jsverbs command sources = %#v", got)
	}
	if got := cfg.Commands[2].Type; got != "provider.command-set" {
		t.Fatalf("provider command type = %q", got)
	}
	if len(result.Warnings) < 4 {
		t.Fatalf("expected TypeScript profile warnings, got %#v", result.Warnings)
	}
	if !warningsContain(result.Warnings, "runtime module alias \"express\" is derived automatically") {
		t.Fatalf("missing runtime alias warning: %#v", result.Warnings)
	}

	rendered, err := Render(cfg)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	text := string(rendered)
	for _, want := range []string{
		"schema: xgoja/v2",
		"type: provider.command-set",
		"language: typescript",
		"bundle: true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered migration missing %q:\n%s", want, text)
		}
	}
}

func TestMigrateV1EmbeddedJSVerbsBecomeArtifactSources(t *testing.T) {
	v1 := &buildspec.BuildSpec{
		Name:   "embedded-verbs",
		Target: buildspec.TargetSpec{Kind: "xgoja", Output: "dist/embedded-verbs"},
		JSVerbs: []buildspec.JSVerbSourceSpec{{
			ID:    "local-verbs",
			Path:  "./verbs",
			Embed: true,
		}},
	}

	result := MigrateV1(v1)
	if len(result.Config.Artifacts) != 1 {
		t.Fatalf("artifacts = %#v", result.Config.Artifacts)
	}
	artifact := result.Config.Artifacts[0]
	if artifact.Type != "binary" || len(artifact.Sources) != 1 || artifact.Sources[0] != "local-verbs" {
		t.Fatalf("embedded jsverb source not attached to binary artifact: %#v", artifact)
	}
	if !warningsContain(result.Warnings, "embedded jsverb source is represented as an artifact source dependency") {
		t.Fatalf("missing embedded source warning: %#v", result.Warnings)
	}
}

func TestMigrateV1AssetsAndRuntimePackage(t *testing.T) {
	v1 := &buildspec.BuildSpec{
		Name:   "runtime-assets",
		Target: buildspec.TargetSpec{Kind: "package", Output: "internal/xgojaruntime", Package: "xgojaruntime"},
		Packages: []buildspec.PackageSpec{{
			ID:      "host",
			Import:  "github.com/example/host",
			Replace: "../host",
		}},
		Assets: []buildspec.AssetSourceSpec{{
			ID:    "app-assets",
			Path:  "./assets",
			Embed: true,
		}},
	}

	result := MigrateV1(v1)
	if got := result.Config.Providers[0].Module.Replace; got != "../host" {
		t.Fatalf("provider replace = %q", got)
	}
	if got := result.Config.Artifacts[0].Type; got != "runtime-package" {
		t.Fatalf("artifact type = %q", got)
	}
	if len(result.Config.Artifacts) != 2 || result.Config.Artifacts[1].Type != "embedded-assets" {
		t.Fatalf("embedded assets artifact not migrated: %#v", result.Config.Artifacts)
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0].Message, "workspace.mode=auto") {
		t.Fatalf("replace warning not emitted: %#v", result.Warnings)
	}
}

func warningsContain(warnings []MigrationWarning, needle string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning.String(), needle) {
			return true
		}
	}
	return false
}
