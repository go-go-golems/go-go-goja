package main

import (
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/plan"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2"
)

func TestSelectPlanTargetSelectsCommandCompatiblePrimaryAndScopesPlan(t *testing.T) {
	compiled := artifactSelectionPlan(
		specv2.ArtifactSpec{ID: "runtime", Type: "runtime-package", Output: "internal/runtime", Sources: []string{"package-verbs"}},
		specv2.ArtifactSpec{ID: "assets", Type: "embedded-assets", Sources: []string{"webapp"}},
		specv2.ArtifactSpec{ID: "binary", Type: "binary", Output: "dist/tool", Sources: []string{"binary-verbs"}},
		specv2.ArtifactSpec{ID: "declarations", Type: "dts", Output: "types.d.ts"},
	)

	buildTarget, buildPlan, err := selectPlanTarget(compiled, artifactCommandBuild)
	if err != nil {
		t.Fatalf("select build target: %v", err)
	}
	if buildTarget.ID != "binary" || buildTarget.Type != "binary" || buildTarget.Kind != "xgoja" {
		t.Fatalf("unexpected build target: %#v", buildTarget)
	}
	assertArtifactIDs(t, buildPlan.Config.Artifacts, "binary", "assets", "declarations")
	assertArtifactPlanIDs(t, buildPlan.Artifacts, "binary", "assets", "declarations")

	generateTarget, generatePlan, err := selectPlanTarget(compiled, artifactCommandGenerate)
	if err != nil {
		t.Fatalf("select generate target: %v", err)
	}
	if generateTarget.ID != "runtime" || generateTarget.Type != "runtime-package" || generateTarget.Kind != "package" {
		t.Fatalf("unexpected generate target: %#v", generateTarget)
	}
	assertArtifactIDs(t, generatePlan.Config.Artifacts, "runtime", "assets", "declarations")
	assertArtifactPlanIDs(t, generatePlan.Artifacts, "runtime", "assets", "declarations")

	assertArtifactIDs(t, compiled.Config.Artifacts, "runtime", "assets", "binary", "declarations")
	assertArtifactPlanIDs(t, compiled.Artifacts, "runtime", "assets", "binary", "declarations")
}

func TestSelectPlanTargetNormalizesArtifactTypesInScopedPlan(t *testing.T) {
	compiled := artifactSelectionPlan(
		specv2.ArtifactSpec{ID: "runtime", Type: " runtime-package "},
		specv2.ArtifactSpec{ID: "assets", Type: " embedded-assets "},
		specv2.ArtifactSpec{ID: "binary", Type: " binary "},
	)

	buildTarget, buildPlan, err := selectPlanTarget(compiled, artifactCommandBuild)
	if err != nil {
		t.Fatalf("select whitespace-padded build target: %v", err)
	}
	if buildTarget.Type != "binary" || buildTarget.Kind != "xgoja" {
		t.Fatalf("unexpected normalized build target: %#v", buildTarget)
	}
	if got := buildPlan.Config.Artifacts[0].Type; got != "binary" {
		t.Fatalf("scoped config primary type = %q, want binary", got)
	}
	if got := buildPlan.Config.Artifacts[1].Type; got != "embedded-assets" {
		t.Fatalf("scoped config support type = %q, want embedded-assets", got)
	}
	if got := buildPlan.Artifacts[0].Spec.Type; got != "binary" {
		t.Fatalf("scoped plan primary type = %q, want binary", got)
	}
	if got := compiled.Config.Artifacts[2].Type; got != " binary " {
		t.Fatalf("original config was mutated: %q", got)
	}
}

func TestSelectPlanTargetRejectsNoCompatiblePrimary(t *testing.T) {
	compiled := artifactSelectionPlan(specv2.ArtifactSpec{ID: "runtime", Type: "runtime-package"})

	_, _, err := selectPlanTarget(compiled, artifactCommandBuild)
	if err == nil {
		t.Fatal("expected no-compatible-primary error")
	}
	for _, want := range []string{"xgoja build", "binary, adapter, or cobra", "runtime (runtime-package)"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err, want)
		}
	}
}

func TestSelectPlanTargetRejectsAmbiguousPrimary(t *testing.T) {
	compiled := artifactSelectionPlan(
		specv2.ArtifactSpec{ID: "runtime", Type: "runtime-package"},
		specv2.ArtifactSpec{ID: "template", Type: "template"},
	)

	_, _, err := selectPlanTarget(compiled, artifactCommandGenerate)
	if err == nil {
		t.Fatal("expected ambiguous-primary error")
	}
	for _, want := range []string{"xgoja generate", "found 2", "runtime (runtime-package)", "template (template)"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err, want)
		}
	}
}

func artifactSelectionPlan(artifacts ...specv2.ArtifactSpec) *plan.Plan {
	artifactPlans := make([]plan.ArtifactPlan, 0, len(artifacts))
	for _, artifact := range artifacts {
		artifactPlans = append(artifactPlans, plan.ArtifactPlan{Spec: artifact})
	}
	return &plan.Plan{
		Config:    specv2.Config{Artifacts: artifacts},
		Artifacts: artifactPlans,
	}
}

func assertArtifactIDs(t *testing.T, artifacts []specv2.ArtifactSpec, want ...string) {
	t.Helper()
	got := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		got = append(got, artifact.ID)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("artifact ids = %v, want %v", got, want)
	}
}

func assertArtifactPlanIDs(t *testing.T, artifacts []plan.ArtifactPlan, want ...string) {
	t.Helper()
	got := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		got = append(got, artifact.Spec.ID)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("artifact plan ids = %v, want %v", got, want)
	}
}
