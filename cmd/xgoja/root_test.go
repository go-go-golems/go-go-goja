package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2"
)

func TestRootHelp(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute help: %v", err)
	}
	rendered := out.String()
	for _, want := range []string{"xgoja", "build", "generate", "gen-dts", "doctor", "inspect", "list-modules", "migrate-spec"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected help to contain %q, got %q", want, rendered)
		}
	}
}

func TestBundledHelpTopic(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	root.SetArgs([]string{"help", "user-guide"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute help topic: %v", err)
	}
	rendered := out.String()
	for _, want := range []string{"xgoja user guide and v2 spec reference", "Source sets", "Providers and runtime modules"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected bundled help to contain %q, got %q", want, rendered)
		}
	}
}

func TestBuildCommandWired(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeValidSpec(t)
	workDir := filepath.Join(t.TempDir(), "work")
	root.SetArgs([]string{"build", "-f", specPath, "--output", "./dist/fixture", "--work-dir", workDir, "--dry-run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute build: %v", err)
	}
	rendered := out.String()
	for _, want := range []string{
		"generated build workspace",
		"generated module: xgoja.generated/fixture",
		"xgoja builds from the generated module root",
		"release note: if you check this generated host into a repository as a nested Go module",
		"xgoja dry run ok",
		"./dist/fixture",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected build output to contain %q, got %q", want, rendered)
		}
	}
	for _, name := range []string{"go.mod", "main.go", "xgoja.runtime.json"} {
		if _, err := os.Stat(filepath.Join(workDir, name)); err != nil {
			t.Fatalf("expected generated %s: %v", name, err)
		}
	}
}

func TestBuildAndGenerateSelectCompatibleArtifactsRegardlessOfOrder(t *testing.T) {
	for _, packageFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "binary-first", true: "package-first"}[packageFirst], func(t *testing.T) {
			specPath := writeBinaryAndPackageSpec(t, packageFirst)

			buildOut := &bytes.Buffer{}
			buildRoot, err := newRootCommand(buildOut)
			if err != nil {
				t.Fatalf("new build root command: %v", err)
			}
			workDir := filepath.Join(t.TempDir(), "build")
			buildRoot.SetArgs([]string{"build", "-f", specPath, "--artifact", "binary", "--work-dir", workDir, "--dry-run"})
			if err := buildRoot.Execute(); err != nil {
				t.Fatalf("execute build: %v\n%s", err, buildOut.String())
			}
			buildPlan, err := os.ReadFile(filepath.Join(workDir, "xgoja.runtime.json"))
			if err != nil {
				t.Fatalf("read generated build runtime plan: %v", err)
			}
			if !strings.Contains(string(buildPlan), `"kind": "xgoja"`) || strings.Contains(string(buildPlan), `"kind": "package"`) {
				t.Fatalf("build runtime target did not select binary:\n%s", buildPlan)
			}

			generateOut := &bytes.Buffer{}
			generateRoot, err := newRootCommand(generateOut)
			if err != nil {
				t.Fatalf("new generate root command: %v", err)
			}
			packageDir := filepath.Join(t.TempDir(), "runtime")
			generateRoot.SetArgs([]string{"generate", "-f", specPath, "--artifact", "runtime", "--output", packageDir})
			if err := generateRoot.Execute(); err != nil {
				t.Fatalf("execute generate: %v\n%s", err, generateOut.String())
			}
			generatedPackage, err := os.ReadFile(filepath.Join(packageDir, "xgoja_runtime.gen.go"))
			if err != nil {
				t.Fatalf("read generated package: %v", err)
			}
			if !strings.Contains(string(generatedPackage), `"kind": "package"`) || strings.Contains(string(generatedPackage), `"kind": "xgoja"`) {
				t.Fatalf("generated runtime target did not select package:\n%s", generatedPackage)
			}
		})
	}
}

func TestBuildAndGenerateScopeEmbeddedSourcesToSelectedPrimary(t *testing.T) {
	specPath, binaryVerbs, packageVerbs, assets := writeScopedArtifactSourcesSpec(t)

	buildOut := &bytes.Buffer{}
	buildRoot, err := newRootCommand(buildOut)
	if err != nil {
		t.Fatalf("new build root command: %v", err)
	}
	workDir := filepath.Join(t.TempDir(), "build")
	buildRoot.SetArgs([]string{"build", "-f", specPath, "--work-dir", workDir, "--dry-run"})
	if err := buildRoot.Execute(); err != nil {
		t.Fatalf("execute build: %v\n%s", err, buildOut.String())
	}
	assertExists(t, filepath.Join(workDir, "xgoja_embed", "jsverbs", "binary_verbs", filepath.Base(binaryVerbs)))
	assertNotExists(t, filepath.Join(workDir, "xgoja_embed", "jsverbs", "package_verbs", filepath.Base(packageVerbs)))
	assertExists(t, filepath.Join(workDir, "xgoja_embed", "assets", "webapp", filepath.Base(assets)))

	generateOut := &bytes.Buffer{}
	generateRoot, err := newRootCommand(generateOut)
	if err != nil {
		t.Fatalf("new generate root command: %v", err)
	}
	packageDir := filepath.Join(t.TempDir(), "runtime")
	generateRoot.SetArgs([]string{"generate", "-f", specPath, "--output", packageDir})
	if err := generateRoot.Execute(); err != nil {
		t.Fatalf("execute generate: %v\n%s", err, generateOut.String())
	}
	assertExists(t, filepath.Join(packageDir, "xgoja_embed", "jsverbs", "package_verbs", filepath.Base(packageVerbs)))
	assertNotExists(t, filepath.Join(packageDir, "xgoja_embed", "jsverbs", "binary_verbs", filepath.Base(binaryVerbs)))
	assertExists(t, filepath.Join(packageDir, "xgoja_embed", "assets", "webapp", filepath.Base(assets)))
}

func TestGenerateCommandWired(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writePackageSpec(t)
	outputDir := filepath.Join(t.TempDir(), "xgojaruntime")
	root.SetArgs([]string{"generate", "-f", specPath, "--output", outputDir, "--package", "xgojaruntime"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute generate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "xgoja_runtime.gen.go")); err != nil {
		t.Fatalf("expected generated package: %v", err)
	}
	if !strings.Contains(out.String(), "xgoja generate ok") {
		t.Fatalf("expected generate output, got %q", out.String())
	}
}

func TestGenerateCommandLetsGeneratorSanitizeInferredPackageName(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writePackageSpecWithoutPackageName(t)
	outputDir := filepath.Join(t.TempDir(), "xgoja-runtime")
	root.SetArgs([]string{"generate", "-f", specPath, "--output", outputDir})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute generate: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(outputDir, "xgoja_runtime.gen.go"))
	if err != nil {
		t.Fatalf("read generated package: %v", err)
	}
	if !strings.Contains(string(data), "package xgoja_runtime") {
		t.Fatalf("expected sanitized inferred package name, got:\n%s", data)
	}
}

func TestGenerateCommandPrintsTemplateData(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writePackageSpecWithoutPackageName(t)
	outputDir := filepath.Join(t.TempDir(), "xgoja-runtime")
	root.SetArgs([]string{"generate", "-f", specPath, "--output", outputDir, "--template-data"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute generate template-data: %v", err)
	}
	rendered := out.String()
	for _, want := range []string{`"PackageName": "xgoja_runtime"`, `"ProviderImports"`, `"RuntimePlanJSON"`} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected template data to contain %s, got %s", want, rendered)
		}
	}
	if _, err := os.Stat(filepath.Join(outputDir, "xgoja_runtime.gen.go")); !os.IsNotExist(err) {
		t.Fatalf("template-data should not write output, stat err=%v", err)
	}
}

func TestGenerateCommandCleanRemovesKnownGeneratedFiles(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeSourceSpec(t)
	outputDir := filepath.Join(t.TempDir(), "xgojaruntime")
	if err := os.MkdirAll(filepath.Join(outputDir, "xgoja_embed"), 0o755); err != nil {
		t.Fatalf("mkdir stale embed: %v", err)
	}
	staleGenerated := filepath.Join(outputDir, "xgoja_runtime.gen.go")
	if err := os.WriteFile(staleGenerated, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale generated file: %v", err)
	}
	keep := filepath.Join(outputDir, "keep.go")
	if err := os.WriteFile(keep, []byte("package keep"), 0o644); err != nil {
		t.Fatalf("write keep file: %v", err)
	}
	root.SetArgs([]string{"generate", "-f", specPath, "--output", outputDir, "--package", "xgojaruntime", "--clean"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute generate clean source: %v", err)
	}
	if _, err := os.Stat(staleGenerated); !os.IsNotExist(err) {
		t.Fatalf("expected stale generated file removed, err=%v", err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("expected non-generated file preserved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "runtime_plan.gen.go")); err != nil {
		t.Fatalf("expected regenerated source fragment: %v", err)
	}
}

func TestGenerateCommandWritesSourceFragments(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeSourceSpec(t)
	outputDir := filepath.Join(t.TempDir(), "xgojaruntime")
	root.SetArgs([]string{"generate", "-f", specPath, "--output", outputDir, "--package", "xgojaruntime"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute generate source: %v", err)
	}
	for _, name := range []string{"runtime_plan.gen.go", "providers.gen.go", "bundle.gen.go"} {
		if _, err := os.Stat(filepath.Join(outputDir, name)); err != nil {
			t.Fatalf("expected source fragment %s: %v", name, err)
		}
	}
}

func TestGenerateCommandWritesCustomTemplate(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeTemplateSpec(t)
	outputPath := filepath.Join(t.TempDir(), "custom.gen.go")
	root.SetArgs([]string{"generate", "-f", specPath, "--output", outputPath, "--package", "customruntime"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute generate template: %v", err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read custom output: %v", err)
	}
	if !strings.Contains(string(data), "package customruntime") || !strings.Contains(string(data), "const ProviderCount = 1") {
		t.Fatalf("unexpected custom output: %s", data)
	}
}

func TestBuildCommandLoadsV2SpecDryRun(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeV2Spec(t)
	root.SetArgs([]string{"build", "-f", specPath, "--dry-run", "--keep-work"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute v2 build dry-run: %v", err)
	}
	if !strings.Contains(out.String(), "validated xgoja/v2 plan") {
		t.Fatalf("expected v2 validation output, got %q", out.String())
	}
}

func TestBuildCommandBuildsBinary(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeBuildableSpec(t)
	outputPath := filepath.Join(t.TempDir(), "fixture")
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	root.SetArgs([]string{"build", "-f", specPath, "--output", outputPath, "--xgoja-replace", repoRoot})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute build: %v", err)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected output binary: %v", err)
	}
}

func TestBuildCommandProviderServeUsesCommandScopedSources(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a generated xgoja binary")
	}
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeHTTPServeScopedSourcesSpec(t)
	outputPath := filepath.Join(t.TempDir(), "scoped-serve")
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	root.SetArgs([]string{"build", "-f", specPath, "--output", outputPath, "--xgoja-replace", repoRoot})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute build: %v\noutput:\n%s", err, out.String())
	}

	help := runGeneratedBinary(t, outputPath, "serve", "sitea", "start", "--help", "--long-help")
	for _, want := range []string{"Start A", "--http-listen", "--hot-reload"} {
		if !strings.Contains(help, want) {
			t.Fatalf("expected scoped serve help to contain %q, got:\n%s", want, help)
		}
	}

	serveHelp := runGeneratedBinary(t, outputPath, "serve", "--help")
	if strings.Contains(serveHelp, "siteb") || strings.Contains(serveHelp, "Start B") {
		t.Fatalf("unexpectedly exposed site-b command outside command sources; output:\n%s", serveHelp)
	}
}

func TestDoctorCommandWired(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeValidSpec(t)
	root.SetArgs([]string{"doctor", "-f", specPath, "--output", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute doctor: %v", err)
	}
}

func TestDoctorCommandLoadsV2Spec(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeV2Spec(t)
	root.SetArgs([]string{"doctor", "-f", specPath, "--output", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute doctor: %v", err)
	}
}

func TestDoctorCommandReportsReadErrors(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "missing.yaml")
	handled, err := (&doctorCommand{}).runV2Doctor(context.Background(), specPath, nil)
	if !handled {
		t.Fatalf("expected missing file to be handled as a doctor error")
	}
	if err == nil {
		t.Fatalf("expected missing file error")
	}
	if strings.Contains(err.Error(), "legacy xgoja spec") {
		t.Fatalf("expected read error, got legacy migration hint: %v", err)
	}
}

func TestDoctorCommandReportsMalformedYAML(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte("schema: [\n"), 0o644); err != nil {
		t.Fatalf("write malformed spec: %v", err)
	}
	handled, err := (&doctorCommand{}).runV2Doctor(context.Background(), specPath, nil)
	if !handled {
		t.Fatalf("expected malformed YAML to be handled as a doctor error")
	}
	if err == nil {
		t.Fatalf("expected malformed YAML error")
	}
	if strings.Contains(err.Error(), "legacy xgoja spec") {
		t.Fatalf("expected parse error, got legacy migration hint: %v", err)
	}
}

func TestMigrateSpecCommandWritesOutput(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeValidSpec(t)
	outputPath := filepath.Join(t.TempDir(), "xgoja.v2.yaml")
	root.SetArgs([]string{"migrate-spec", "-f", specPath, "--out", outputPath})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute migrate-spec: %v", err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read migrated spec: %v", err)
	}
	text := string(data)
	for _, want := range []string{"schema: xgoja/v2", "providers:", "runtime:", "artifacts:"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected migrated spec to contain %q, got:\n%s", want, text)
		}
	}
	if !strings.Contains(out.String(), "wrote migrated xgoja/v2 spec") {
		t.Fatalf("expected migrate output, got %q", out.String())
	}
}

func TestMigrateSpecCommandInPlaceBackup(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeValidSpec(t)
	root.SetArgs([]string{"migrate-spec", "-f", specPath, "--in-place", "--backup"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute migrate-spec in-place: %v", err)
	}
	if _, err := os.Stat(specPath + ".bak"); err != nil {
		t.Fatalf("expected backup: %v", err)
	}
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read in-place spec: %v", err)
	}
	if !strings.Contains(string(data), "schema: xgoja/v2") {
		t.Fatalf("expected v2 schema after in-place migration, got:\n%s", data)
	}
}

func TestMigrateSpecCommandCheckAlreadyV2(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	rendered, err := specv2.Render(specv2.Config{
		Name: "already-v2",
		Providers: []specv2.ProviderSpec{{
			ID:     "core",
			Import: "github.com/example/core",
		}},
		Artifacts: []specv2.ArtifactSpec{{
			ID:   "binary",
			Type: "binary",
		}},
	})
	if err != nil {
		t.Fatalf("render v2 spec: %v", err)
	}
	specPath := writeFile(t, "xgoja.v2.yaml", string(rendered)+"\n")
	root.SetArgs([]string{"migrate-spec", "-f", specPath, "--check"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute migrate-spec check: %v", err)
	}
	if !strings.Contains(out.String(), "already in rendered xgoja/v2 form") {
		t.Fatalf("expected check output, got %q", out.String())
	}
}

func TestMigrateSpecCommandPrintsWarnings(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeFile(t, "xgoja.yaml", `name: warnings
appName: warnings
packages:
  - id: http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
modules:
  - package: http
    name: express
commands:
  jsverbs:
    enabled: true
jsverbs:
  - id: local-sites
    path: ./verbs
    typescript:
      enabled: true
      bundle: true
      target: es2015
      format: cjs
      platform: neutral
      external:
        - express
`)
	outputPath := filepath.Join(t.TempDir(), "xgoja.v2.yaml")
	root.SetArgs([]string{"migrate-spec", "-f", specPath, "--out", outputPath})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute migrate-spec warnings: %v", err)
	}
	if !strings.Contains(out.String(), "warning:") || !strings.Contains(out.String(), "runtime module alias") {
		t.Fatalf("expected warning output, got %q", out.String())
	}
}

func TestInspectCommandReadsCurrentBinary(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	root.SetArgs([]string{"inspect", os.Args[0], "--output", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute inspect: %v", err)
	}
}

func TestGenDTSCommandLoadsV2Spec(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeTypedCoreV2Spec(t)
	outputPath := filepath.Join(filepath.Dir(specPath), "xgoja-modules.d.ts")
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	root.SetArgs([]string{"gen-dts", "-f", specPath, "--xgoja-replace", repoRoot})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute v2 gen-dts: %v\n%s", err, out.String())
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read generated dts: %v", err)
	}
	if !strings.Contains(string(data), `declare module "path:typed"`) {
		t.Fatalf("expected aliased path declaration, got:\n%s", data)
	}
	if !strings.Contains(out.String(), "validated xgoja/v2 plan") {
		t.Fatalf("expected v2 validation output, got %q", out.String())
	}
}

func TestGenDTSCommandWired(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeTypedCoreSpec(t)
	outputPath := filepath.Join(t.TempDir(), "xgoja-modules.d.ts")
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	root.SetArgs([]string{"gen-dts", "-f", specPath, "--out", outputPath, "--strict", "--xgoja-replace", repoRoot})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute gen-dts: %v\n%s", err, out.String())
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read generated dts: %v", err)
	}
	if !strings.Contains(string(data), `declare module "path:typed"`) {
		t.Fatalf("expected aliased path declaration, got:\n%s", data)
	}
}

func TestListModulesCommandWired(t *testing.T) {
	out := &bytes.Buffer{}
	root, err := newRootCommand(out)
	if err != nil {
		t.Fatalf("new root command: %v", err)
	}
	specPath := writeValidSpec(t)
	root.SetArgs([]string{"list-modules", "-f", specPath, "--output", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute list-modules: %v", err)
	}
}

func writeBinaryAndPackageSpec(t *testing.T, packageFirst bool) string {
	t.Helper()
	binary := `
  - id: binary
    type: binary
    output: dist/fixture
`
	runtimePackage := `
  - id: runtime
    type: runtime-package
    output: internal/xgojaruntime
    package: xgojaruntime
`
	artifacts := binary + runtimePackage
	if packageFirst {
		artifacts = runtimePackage + binary
	}
	return writeFile(t, "xgoja.yaml", `
schema: xgoja/v2
name: binary-and-package
go:
  module: xgoja.generated/binary-and-package
  version: "1.26"
workspace:
  mode: off
providers:
  - id: fixture
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider
    register: Register
runtime:
  modules:
    - provider: fixture
      name: hello
      as: hello
artifacts:`+artifacts)
}

func writeScopedArtifactSourcesSpec(t *testing.T) (string, string, string, string) {
	t.Helper()
	dir := t.TempDir()
	binaryDir := filepath.Join(dir, "binary-verbs")
	packageDir := filepath.Join(dir, "package-verbs")
	assetsDir := filepath.Join(dir, "assets")
	binaryVerbs := filepath.Join(binaryDir, "binary.js")
	packageVerbs := filepath.Join(packageDir, "package.js")
	assets := filepath.Join(assetsDir, "app.css")
	for path, content := range map[string]string{
		binaryVerbs:  "export const binary = true;\n",
		packageVerbs: "export const runtimePackage = true;\n",
		assets:       "body { color: black; }\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	specPath := writeFile(t, "xgoja.yaml", `
schema: xgoja/v2
name: scoped-artifact-sources
go:
  module: xgoja.generated/scoped-artifact-sources
  version: "1.26"
workspace:
  mode: off
sources:
  - id: binary-verbs
    kind: jsverbs
    from:
      dir: `+filepath.ToSlash(binaryDir)+`
  - id: package-verbs
    kind: jsverbs
    from:
      dir: `+filepath.ToSlash(packageDir)+`
  - id: webapp
    kind: assets
    from:
      dir: `+filepath.ToSlash(assetsDir)+`
artifacts:
  - id: runtime
    type: runtime-package
    output: internal/xgojaruntime
    package: xgojaruntime
    sources: [package-verbs]
  - id: assets
    type: embedded-assets
    sources: [webapp]
  - id: binary
    type: binary
    output: dist/fixture
    sources: [binary-verbs]
`)
	return specPath, binaryVerbs, packageVerbs, assets
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s: %v", path, err)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be absent, err=%v", path, err)
	}
}

func writePackageSpec(t *testing.T) string {
	t.Helper()
	return writeV2ArtifactSpec(t, "fixture-package", "runtime-package", "internal/xgojaruntime", "xgojaruntime", "")
}

func writePackageSpecWithoutPackageName(t *testing.T) string {
	t.Helper()
	return writeV2ArtifactSpec(t, "fixture-package", "runtime-package", "internal/xgoja-runtime", "", "")
}

func writeSourceSpec(t *testing.T) string {
	t.Helper()
	return writeV2ArtifactSpec(t, "fixture-source", "source", "internal/xgojaruntime", "xgojaruntime", "")
}

func writeTemplateSpec(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "runtime.go.tmpl")
	if err := os.WriteFile(templatePath, []byte(`// Code generated by custom xgoja template; DO NOT EDIT.
package {{ .PackageName }}

const ProviderCount = {{ len .ProviderImports }}
`), 0o644); err != nil {
		t.Fatalf("write custom template: %v", err)
	}
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
schema: xgoja/v2
name: fixture-template
go:
  module: xgoja.generated/fixture-template
  version: "1.26"
workspace:
  mode: off
providers:
  - id: fixture
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider
    register: Register
runtime:
  modules:
    - provider: fixture
      name: hello
      as: hello
artifacts:
  - id: template
    type: template
    output: internal/xgojaruntime/custom.gen.go
    package: customruntime
    template: runtime.go.tmpl
`), 0o644); err != nil {
		t.Fatalf("write template spec: %v", err)
	}
	return specPath
}

func writeBuildableSpec(t *testing.T) string {
	t.Helper()
	return writeV2ArtifactSpec(t, "fixture", "binary", "dist/fixture", "", "")
}

func writeHTTPServeScopedSourcesSpec(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	siteADir := filepath.Join(dir, "site-a")
	siteBDir := filepath.Join(dir, "site-b")
	if err := os.MkdirAll(siteADir, 0o755); err != nil {
		t.Fatalf("mkdir site-a: %v", err)
	}
	if err := os.MkdirAll(siteBDir, 0o755); err != nil {
		t.Fatalf("mkdir site-b: %v", err)
	}
	if err := os.WriteFile(filepath.Join(siteADir, "a.js"), []byte(`
__package__({ name: "sitea" });
__verb__("start", { name: "start", short: "Start A", output: "text" });
function start() { return "a"; }
`), 0o644); err != nil {
		t.Fatalf("write site-a jsverb: %v", err)
	}
	if err := os.WriteFile(filepath.Join(siteBDir, "b.js"), []byte(`
__package__({ name: "siteb" });
__verb__("start", { name: "start", short: "Start B", output: "text" });
function start() { return "b"; }
`), 0o644); err != nil {
		t.Fatalf("write site-b jsverb: %v", err)
	}
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
schema: xgoja/v2
name: scoped-serve
go:
  module: xgoja.generated/scoped-serve
  version: "1.26"
workspace:
  mode: off
providers:
  - id: go-go-goja-http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
runtime:
  modules:
    - provider: go-go-goja-http
      name: express
      as: express
sources:
  - id: site-a
    kind: jsverbs
    from:
      dir: `+filepath.ToSlash(siteADir)+`
    language: javascript
  - id: site-b
    kind: jsverbs
    from:
      dir: `+filepath.ToSlash(siteBDir)+`
    language: javascript
commands:
  - id: serve
    type: provider.command-set
    provider: go-go-goja-http
    name: serve
    mount: serve
    sources: [site-a]
artifacts:
  - id: binary
    type: binary
    output: dist/scoped-serve
    sources: [site-a, site-b]
`), 0o644); err != nil {
		t.Fatalf("write scoped serve v2 spec: %v", err)
	}
	return specPath
}

func writeTypedCoreV2Spec(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	specPath := filepath.Join(dir, "xgoja.yaml")
	dtsPath := filepath.ToSlash(filepath.Join(dir, "xgoja-modules.d.ts"))
	if err := os.WriteFile(specPath, []byte(`
schema: xgoja/v2
name: typed-core
go:
  module: xgoja.generated/typed-core
  version: "1.26"
workspace:
  mode: off
providers:
  - id: go-go-goja-core
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/core
    register: Register
runtime:
  modules:
    - provider: go-go-goja-core
      name: path
      as: path:typed
commands:
  - id: eval
    type: builtin.eval
    name: eval
artifacts:
  - id: declarations
    type: dts
    output: `+dtsPath+`
    strict: true
`), 0o644); err != nil {
		t.Fatalf("write typed core v2 spec: %v", err)
	}
	return specPath
}

func writeTypedCoreSpec(t *testing.T) string {
	t.Helper()
	return writeTypedCoreV2Spec(t)
}

func writeV2ArtifactSpec(t *testing.T, name, artifactType, output, packageName, template string) string {
	t.Helper()
	dir := t.TempDir()
	specPath := filepath.Join(dir, "xgoja.yaml")
	packageLine := ""
	if packageName != "" {
		packageLine = "    package: " + packageName + "\n"
	}
	templateLine := ""
	if template != "" {
		templateLine = "    template: " + template + "\n"
	}
	content := `
schema: xgoja/v2
name: ` + name + `
go:
  module: xgoja.generated/` + name + `
  version: "1.26"
workspace:
  mode: off
providers:
  - id: fixture
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider
    register: Register
runtime:
  modules:
    - provider: fixture
      name: hello
      as: hello
commands:
  - id: repl
    type: builtin.repl
    name: repl
artifacts:
  - id: artifact
    type: ` + artifactType + `
    output: ` + output + `
` + packageLine + templateLine
	if err := os.WriteFile(specPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write v2 artifact spec: %v", err)
	}
	return specPath
}

func runGeneratedBinary(t *testing.T, binary string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	data, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("generated binary timed out: %v\noutput:\n%s", ctx.Err(), data)
	}
	if err != nil {
		t.Fatalf("run generated binary %v: %v\noutput:\n%s", args, err, data)
	}
	return string(data)
}

func writeV2Spec(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	verbsDir := filepath.Join(dir, "verbs")
	if err := os.Mkdir(verbsDir, 0o755); err != nil {
		t.Fatalf("mkdir verbs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(verbsDir, "site.js"), []byte(`__package__({ name: "sites" })`), 0o644); err != nil {
		t.Fatalf("write verb: %v", err)
	}
	specPath := filepath.Join(dir, "xgoja.yaml")
	if err := os.WriteFile(specPath, []byte(`
schema: xgoja/v2
name: fixture
go:
  module: xgoja.generated/fixture
  version: "1.26"
workspace:
  mode: off
providers:
  - id: fixture
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider
    register: Register
runtime:
  modules:
    - provider: fixture
      name: hello
      as: hello
sources:
  - id: local
    kind: jsverbs
    from:
      dir: ./verbs
    extensions: [.js]
commands:
  - id: verbs
    type: builtin.jsverbs
    sources: [local]
artifacts:
  - id: bin
    type: binary
    output: dist/fixture
    sources: [local]
`), 0o644); err != nil {
		t.Fatalf("write v2 spec: %v", err)
	}
	return specPath
}

func writeValidSpec(t *testing.T) string {
	t.Helper()
	return writeV2Spec(t)
}

func writeFile(t *testing.T, name string, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}
