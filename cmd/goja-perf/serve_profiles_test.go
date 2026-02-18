package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProfileManifest(t *testing.T) {
	yamlContent := `
generated_at: "2026-02-18T15:12:00-05:00"
artifacts:
  - id: engine_new_cpu_svg
    title: "EngineNew CPU"
    description: "CPU callgraph SVG"
    phase: "phase1"
    task_id: "p1-runtime-lifecycle"
    benchmark: "BenchmarkRuntimeSpawn/EngineNew_NoCallLog"
    kind: "flamegraph_svg"
    rel_path: "profiles/engine_new.svg"
    mime: "image/svg+xml"
    bytes: 170000
    tags: ["cpu", "baseline"]
  - id: engine_factory_cpu_svg
    title: "EngineFactory CPU"
    kind: "flamegraph_svg"
    rel_path: "profiles/engine_factory.svg"
    tags: ["cpu", "candidate"]
comparisons:
  - id: new_vs_factory
    title: "EngineNew vs EngineFactory"
    baseline_artifact_id: engine_new_cpu_svg
    candidate_artifact_id: engine_factory_cpu_svg
`
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := loadProfileManifest(path)
	if err != nil {
		t.Fatalf("loadProfileManifest: %v", err)
	}

	if m.GeneratedAt != "2026-02-18T15:12:00-05:00" {
		t.Errorf("GeneratedAt = %q", m.GeneratedAt)
	}
	if len(m.Artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(m.Artifacts))
	}
	if m.Artifacts[0].ID != "engine_new_cpu_svg" {
		t.Errorf("artifact[0].ID = %q", m.Artifacts[0].ID)
	}
	if len(m.Comparisons) != 1 {
		t.Fatalf("expected 1 comparison, got %d", len(m.Comparisons))
	}
	if m.Comparisons[0].BaselineArtifactID != "engine_new_cpu_svg" {
		t.Errorf("comparison[0].BaselineArtifactID = %q", m.Comparisons[0].BaselineArtifactID)
	}
}

func TestFindArtifact(t *testing.T) {
	m := &profileManifest{
		Artifacts: []profileArtifact{
			{ID: "a1", Title: "Artifact 1"},
			{ID: "a2", Title: "Artifact 2"},
		},
	}

	a := m.findArtifact("a1")
	if a == nil || a.Title != "Artifact 1" {
		t.Error("findArtifact(a1) failed")
	}

	a = m.findArtifact("nonexistent")
	if a != nil {
		t.Error("findArtifact(nonexistent) should be nil")
	}
}

func TestSafeResolvePath(t *testing.T) {
	dir := t.TempDir()

	// Create a valid file
	validFile := filepath.Join(dir, "data", "test.svg")
	if err := os.MkdirAll(filepath.Dir(validFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(validFile, []byte("<svg/>"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Valid path
	p, err := safeResolvePath(dir, "data/test.svg")
	if err != nil {
		t.Errorf("valid path failed: %v", err)
	}
	if p != validFile {
		t.Errorf("resolved = %q, want %q", p, validFile)
	}

	// Path traversal
	_, err = safeResolvePath(dir, "../../../etc/passwd")
	if err == nil {
		t.Error("path traversal should be rejected")
	}

	// Empty path
	_, err = safeResolvePath(dir, "")
	if err == nil {
		t.Error("empty path should be rejected")
	}

	// Non-existent file
	_, err = safeResolvePath(dir, "data/nonexistent.svg")
	if err == nil {
		t.Error("non-existent file should be rejected")
	}

	// Directory (not regular file)
	_, err = safeResolvePath(dir, "data")
	if err == nil {
		t.Error("directory should be rejected")
	}
}

func TestDetectMime(t *testing.T) {
	tests := []struct {
		art  profileArtifact
		want string
	}{
		{profileArtifact{Mime: "image/svg+xml"}, "image/svg+xml"},
		{profileArtifact{RelPath: "test.svg"}, "image/svg+xml"},
		{profileArtifact{RelPath: "test.pprof"}, "application/octet-stream"},
		{profileArtifact{RelPath: "test.txt"}, "text/plain; charset=utf-8"},
		{profileArtifact{RelPath: "test.yaml"}, "text/yaml; charset=utf-8"},
	}
	for _, tt := range tests {
		got := detectMime(&tt.art)
		if got != tt.want {
			t.Errorf("detectMime(%q, rel=%q) = %q, want %q", tt.art.Mime, tt.art.RelPath, got, tt.want)
		}
	}
}

func TestBuildProfilesViewData_NoManifest(t *testing.T) {
	data := buildProfilesViewData("phase1", nil)
	if data.HasManifest {
		t.Error("HasManifest should be false with nil manifest")
	}
}

func TestBuildProfilesViewData_WithManifest(t *testing.T) {
	m := &profileManifest{
		GeneratedAt: "2026-02-18",
		Artifacts: []profileArtifact{
			{ID: "baseline", Title: "Baseline", Kind: "flamegraph_svg", RelPath: "x.svg", Benchmark: "BenchmarkTest/Fast-8"},
			{ID: "candidate", Title: "Candidate", Kind: "flamegraph_svg", RelPath: "y.svg"},
			{ID: "diff", Title: "Diff", Kind: "flamegraph_svg", RelPath: "d.svg"},
		},
		Comparisons: []profileComparison{
			{
				ID:                  "cmp1",
				Title:               "Fast vs Slow",
				BaselineArtifactID:  "baseline",
				CandidateArtifactID: "candidate",
				DiffArtifactID:      "diff",
			},
		},
	}

	data := buildProfilesViewData("phase1", m)
	if !data.HasManifest {
		t.Fatal("HasManifest should be true")
	}
	if data.ArtifactCount != 3 {
		t.Errorf("ArtifactCount = %d, want 3", data.ArtifactCount)
	}
	if data.CompareCount != 1 {
		t.Errorf("CompareCount = %d, want 1", data.CompareCount)
	}

	// Check URLs
	if data.Artifacts[0].ViewURL != "/api/profile-artifact/phase1/baseline" {
		t.Errorf("ViewURL = %q", data.Artifacts[0].ViewURL)
	}

	// Check benchmark shortening
	if data.Artifacts[0].Benchmark != "Fast" {
		t.Errorf("Benchmark = %q, want Fast", data.Artifacts[0].Benchmark)
	}

	// Check comparisons resolved
	cmp := data.Comparisons[0]
	if cmp.BaselineView == nil || cmp.BaselineView.ID != "baseline" {
		t.Error("comparison baseline not resolved")
	}
	if cmp.DiffView == nil || cmp.DiffView.ID != "diff" {
		t.Error("comparison diff not resolved")
	}
}

func TestKindLabel(t *testing.T) {
	if kindLabel("flamegraph_svg") != "Flamegraph" {
		t.Error("flamegraph_svg label")
	}
	if kindLabel("pprof_cpu") != "CPU Profile" {
		t.Error("pprof_cpu label")
	}
	if kindLabel("unknown_thing") != "unknown_thing" {
		t.Error("unknown kind should pass through")
	}
}
