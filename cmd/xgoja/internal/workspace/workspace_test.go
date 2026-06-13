package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindGoWorkSearchesUpward(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.work"), "go 1.26\n\nuse ./mod\n")
	leaf := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(leaf, 0o755); err != nil {
		t.Fatalf("mkdir leaf: %v", err)
	}
	got, err := FindGoWork(leaf)
	if err != nil {
		t.Fatalf("FindGoWork: %v", err)
	}
	if got != filepath.Join(root, "go.work") {
		t.Fatalf("FindGoWork = %q", got)
	}
}

func TestParseGoWorkMapsModulePathsToDirs(t *testing.T) {
	root := t.TempDir()
	modA := filepath.Join(root, "mod-a")
	modB := filepath.Join(root, "nested", "mod-b")
	writeGoMod(t, modA, "example.com/a")
	writeGoMod(t, modB, "example.com/b")
	writeFile(t, filepath.Join(root, "go.work"), "go 1.26\n\nuse (\n\t./mod-a\n\t./nested/mod-b\n)\n")

	modules, err := ParseGoWork(filepath.Join(root, "go.work"))
	if err != nil {
		t.Fatalf("ParseGoWork: %v", err)
	}
	if modules["example.com/a"].Dir != modA {
		t.Fatalf("module a dir = %q", modules["example.com/a"].Dir)
	}
	if modules["example.com/b"].Dir != modB {
		t.Fatalf("module b dir = %q", modules["example.com/b"].Dir)
	}
}

func TestResolvePrecedence(t *testing.T) {
	root := t.TempDir()
	workspaceMod := filepath.Join(root, "workspace-mod")
	explicitMod := filepath.Join(root, "explicit-mod")
	cliMod := filepath.Join(root, "cli-mod")
	writeGoMod(t, workspaceMod, "example.com/workspace")
	writeFile(t, filepath.Join(root, "go.work"), "go 1.26\n\nuse ./workspace-mod\n")

	plan, err := Resolve([]Requirement{
		{ModulePath: "example.com/explicit", Version: "v1.0.0", ExplicitReplace: explicitMod, RequiredBy: []string{"provider:explicit"}},
		{ModulePath: "example.com/cli", Version: "v1.0.0", RequiredBy: []string{"provider:cli"}},
		{ModulePath: "example.com/workspace", Version: "v1.0.0", RequiredBy: []string{"provider:workspace"}},
		{ModulePath: "example.com/versioned", Version: "v1.0.0", RequiredBy: []string{"provider:versioned"}},
	}, Options{
		Spec:     Spec{Mode: ModeAuto},
		StartDir: root,
		CLIReplace: map[string]string{
			"example.com/cli": cliMod,
		},
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	byModule := map[string]GoModulePlan{}
	for _, module := range plan.Modules {
		byModule[module.ModulePath] = module
	}
	assertResolution(t, byModule["example.com/explicit"], ResolutionReplace, SourceExplicitReplace, explicitMod)
	assertResolution(t, byModule["example.com/cli"], ResolutionReplace, SourceCLIReplace, cliMod)
	assertResolution(t, byModule["example.com/workspace"], ResolutionWorkspace, SourceGoWork, workspaceMod)
	assertResolution(t, byModule["example.com/versioned"], ResolutionVersioned, SourceVersion, "")
}

func TestResolveWorkspaceOff(t *testing.T) {
	root := t.TempDir()
	workspaceMod := filepath.Join(root, "workspace-mod")
	writeGoMod(t, workspaceMod, "example.com/workspace")
	writeFile(t, filepath.Join(root, "go.work"), "go 1.26\n\nuse ./workspace-mod\n")

	plan, err := Resolve([]Requirement{{ModulePath: "example.com/workspace", Version: "v1.0.0"}}, Options{Spec: Spec{Mode: ModeOff}, StartDir: root})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := plan.Modules[0].ResolutionSource; got != SourceVersion {
		t.Fatalf("resolution source = %q", got)
	}
}

func assertResolution(t *testing.T, module GoModulePlan, kind ResolutionKind, source ResolutionSource, dir string) {
	t.Helper()
	if module.ResolutionKind != kind || module.ResolutionSource != source {
		t.Fatalf("%s resolution = %s/%s, want %s/%s", module.ModulePath, module.ResolutionKind, module.ResolutionSource, kind, source)
	}
	if dir != "" && module.LocalDir != dir {
		t.Fatalf("%s dir = %q, want %q", module.ModulePath, module.LocalDir, dir)
	}
}

func writeGoMod(t *testing.T, dir string, modulePath string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir module: %v", err)
	}
	writeFile(t, filepath.Join(dir, "go.mod"), "module "+modulePath+"\n\ngo 1.26\n")
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
