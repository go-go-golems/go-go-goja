package main

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/plan"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2"
)

type artifactCommand string

const (
	artifactCommandBuild    artifactCommand = "build"
	artifactCommandGenerate artifactCommand = "generate"
)

type v2PlanTarget struct {
	ID       string
	Type     string
	Kind     string
	Output   string
	Package  string
	Template string
}

func selectPlanTarget(compiled *plan.Plan, command artifactCommand) (v2PlanTarget, *plan.Plan, error) {
	if compiled == nil {
		return v2PlanTarget{}, nil, fmt.Errorf("xgoja %s: compiled plan is nil", command)
	}

	candidates := make([]plan.ArtifactPlan, 0, len(compiled.Artifacts))
	for _, artifact := range compiled.Artifacts {
		if isCompatiblePrimary(command, artifact.Spec.Type) {
			candidates = append(candidates, artifact)
		}
	}
	if len(candidates) == 0 {
		return v2PlanTarget{}, nil, fmt.Errorf(
			"xgoja %s found no compatible primary artifact; accepts %s; configured artifacts: %s",
			command,
			compatibleArtifactTypes(command),
			formatArtifacts(compiled.Artifacts),
		)
	}
	if len(candidates) > 1 {
		return v2PlanTarget{}, nil, fmt.Errorf(
			"xgoja %s requires exactly one compatible primary artifact; found %d: %s",
			command,
			len(candidates),
			formatArtifacts(candidates),
		)
	}

	selected := candidates[0].Spec
	scoped, err := scopePlanToPrimary(compiled, selected.ID)
	if err != nil {
		return v2PlanTarget{}, nil, err
	}
	return targetFromArtifact(selected), scoped, nil
}

func isCompatiblePrimary(command artifactCommand, artifactType string) bool {
	switch command {
	case artifactCommandBuild:
		return artifactType == "binary" || artifactType == "adapter" || artifactType == "cobra"
	case artifactCommandGenerate:
		return artifactType == "runtime-package" || artifactType == "source" || artifactType == "template"
	default:
		return false
	}
}

func isSupportArtifact(artifactType string) bool {
	return artifactType == "dts" || artifactType == "embedded-assets"
}

func compatibleArtifactTypes(command artifactCommand) string {
	switch command {
	case artifactCommandBuild:
		return "binary, adapter, or cobra"
	case artifactCommandGenerate:
		return "runtime-package, source, or template"
	default:
		return "no artifact types"
	}
}

func targetFromArtifact(artifact specv2.ArtifactSpec) v2PlanTarget {
	kind := artifact.Type
	switch kind {
	case "binary":
		kind = "xgoja"
	case "runtime-package":
		kind = "package"
	}
	return v2PlanTarget{
		ID:       artifact.ID,
		Type:     artifact.Type,
		Kind:     kind,
		Output:   artifact.Output,
		Package:  artifact.Package,
		Template: artifact.Template,
	}
}

func scopePlanToPrimary(compiled *plan.Plan, selectedID string) (*plan.Plan, error) {
	if compiled == nil {
		return nil, fmt.Errorf("scope plan: compiled plan is nil")
	}

	selectedConfig, ok := artifactByID(compiled.Config.Artifacts, selectedID)
	if !ok {
		return nil, fmt.Errorf("scope plan: selected artifact %q is missing from config", selectedID)
	}
	selectedPlan, ok := artifactPlanByID(compiled.Artifacts, selectedID)
	if !ok {
		return nil, fmt.Errorf("scope plan: selected artifact %q is missing from compiled artifacts", selectedID)
	}

	scoped := *compiled
	scoped.Config = compiled.Config
	scoped.Config.Artifacts = []specv2.ArtifactSpec{selectedConfig}
	scoped.Artifacts = []plan.ArtifactPlan{selectedPlan}
	for _, artifact := range compiled.Config.Artifacts {
		if isSupportArtifact(artifact.Type) {
			scoped.Config.Artifacts = append(scoped.Config.Artifacts, artifact)
		}
	}
	for _, artifact := range compiled.Artifacts {
		if isSupportArtifact(artifact.Spec.Type) {
			scoped.Artifacts = append(scoped.Artifacts, artifact)
		}
	}
	return &scoped, nil
}

func artifactByID(artifacts []specv2.ArtifactSpec, id string) (specv2.ArtifactSpec, bool) {
	for _, artifact := range artifacts {
		if artifact.ID == id {
			return artifact, true
		}
	}
	return specv2.ArtifactSpec{}, false
}

func artifactPlanByID(artifacts []plan.ArtifactPlan, id string) (plan.ArtifactPlan, bool) {
	for _, artifact := range artifacts {
		if artifact.Spec.ID == id {
			return artifact, true
		}
	}
	return plan.ArtifactPlan{}, false
}

func formatArtifacts(artifacts []plan.ArtifactPlan) string {
	if len(artifacts) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		parts = append(parts, fmt.Sprintf("%s (%s)", artifact.Spec.ID, artifact.Spec.Type))
	}
	return strings.Join(parts, ", ")
}
