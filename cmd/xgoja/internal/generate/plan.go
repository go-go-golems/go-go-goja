package generate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/plan"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2"
)

func WriteAllPlan(dir string, compiled *plan.Plan, opts Options) error {
	if dir == "" {
		return fmt.Errorf("generate directory is required")
	}
	if compiled == nil {
		return fmt.Errorf("plan is nil")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create generate directory %s: %w", dir, err)
	}
	if err := copyEmbeddedJSVerbsPlan(dir, compiled); err != nil {
		return err
	}
	if err := copyEmbeddedHelpSourcesPlan(dir, compiled); err != nil {
		return err
	}
	if err := copyEmbeddedAssetsPlan(dir, compiled); err != nil {
		return err
	}
	files := map[string]string{
		"go.mod":             RenderGoModPlan(compiled, opts),
		"main.go":            RenderMainPlan(compiled),
		"xgoja.runtime.json": RenderRuntimePlanJSONFromPlan(compiled),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return fmt.Errorf("write generated %s: %w", name, err)
		}
	}
	return nil
}

func WritePackagePlan(dir string, compiled *plan.Plan, opts PackageOptions) error {
	if dir == "" {
		return fmt.Errorf("generate directory is required")
	}
	if compiled == nil {
		return fmt.Errorf("plan is nil")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create generate directory %s: %w", dir, err)
	}
	if err := copyEmbeddedJSVerbsPlan(dir, compiled); err != nil {
		return err
	}
	if err := copyEmbeddedHelpSourcesPlan(dir, compiled); err != nil {
		return err
	}
	if err := copyEmbeddedAssetsPlan(dir, compiled); err != nil {
		return err
	}
	packageName := strings.TrimSpace(opts.PackageName)
	if packageName == "" {
		packageName = InferPackageNameFromDir(dir)
	}
	content := RenderPackagePlan(compiled, packageName)
	if err := os.WriteFile(filepath.Join(dir, "xgoja_runtime.gen.go"), []byte(content), 0o644); err != nil {
		return fmt.Errorf("write generated package: %w", err)
	}
	return nil
}

func WriteSourceFragmentsPlan(dir string, compiled *plan.Plan, opts PackageOptions) error {
	if dir == "" {
		return fmt.Errorf("generate directory is required")
	}
	if compiled == nil {
		return fmt.Errorf("plan is nil")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create generate directory %s: %w", dir, err)
	}
	if err := copyEmbeddedJSVerbsPlan(dir, compiled); err != nil {
		return err
	}
	if err := copyEmbeddedHelpSourcesPlan(dir, compiled); err != nil {
		return err
	}
	if err := copyEmbeddedAssetsPlan(dir, compiled); err != nil {
		return err
	}
	packageName := strings.TrimSpace(opts.PackageName)
	if packageName == "" {
		packageName = InferPackageNameFromDir(dir)
	}
	for name, content := range RenderSourceFragmentsPlan(compiled, packageName) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return fmt.Errorf("write generated source fragment %s: %w", name, err)
		}
	}
	return nil
}

func WriteCustomTemplatePlan(outputFile string, compiled *plan.Plan, opts TemplateOptions) error {
	if strings.TrimSpace(outputFile) == "" {
		return fmt.Errorf("custom template output file is required")
	}
	if compiled == nil {
		return fmt.Errorf("plan is nil")
	}
	if err := os.MkdirAll(filepath.Dir(outputFile), 0o755); err != nil {
		return fmt.Errorf("create custom template output directory: %w", err)
	}
	packageName := strings.TrimSpace(opts.PackageName)
	if packageName == "" {
		packageName = InferPackageNameFromDir(filepath.Dir(outputFile))
	}
	content, err := loadCustomTemplate(opts.TemplatePath, packageTemplateDataFromPlan(compiled, packageName))
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputFile, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write custom template output %s: %w", outputFile, err)
	}
	return nil
}

func TemplateDataJSONFromPlan(compiled *plan.Plan, packageName string) (string, error) {
	if compiled == nil {
		return "", fmt.Errorf("plan is nil")
	}
	data, err := json.MarshalIndent(packageTemplateDataFromPlan(compiled, packageName), "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

func copyEmbeddedJSVerbsPlan(dir string, compiled *plan.Plan) error {
	paths := embeddedPlanPaths(compiled.Config)
	for _, source := range compiled.Config.Sources {
		root := paths.JSVerbRoots[source.ID]
		if root == "" {
			continue
		}
		src, err := resolveSourcePath(compiled.Config.BaseDir, source.From.Dir)
		if err != nil {
			return fmt.Errorf("resolve embedded jsverb source %s: %w", source.ID, err)
		}
		dst := filepath.Join(dir, filepath.FromSlash(root))
		if err := copyDir(dst, src); err != nil {
			return fmt.Errorf("copy embedded jsverb source %s: %w", source.ID, err)
		}
	}
	return nil
}

func copyEmbeddedHelpSourcesPlan(dir string, compiled *plan.Plan) error {
	paths := embeddedPlanPaths(compiled.Config)
	for _, source := range compiled.Config.Sources {
		root := paths.HelpRoots[source.ID]
		if root == "" {
			continue
		}
		src, err := resolveSourcePath(compiled.Config.BaseDir, source.From.Dir)
		if err != nil {
			return fmt.Errorf("resolve embedded help source %s: %w", source.ID, err)
		}
		dst := filepath.Join(dir, filepath.FromSlash(root))
		if err := copyDir(dst, src); err != nil {
			return fmt.Errorf("copy embedded help source %s: %w", source.ID, err)
		}
	}
	return nil
}

func copyEmbeddedAssetsPlan(dir string, compiled *plan.Plan) error {
	paths := embeddedPlanPaths(compiled.Config)
	for _, source := range compiled.Config.Sources {
		root := paths.AssetRoots[source.ID]
		if root == "" {
			continue
		}
		src, err := resolveSourcePath(compiled.Config.BaseDir, source.From.Dir)
		if err != nil {
			return fmt.Errorf("resolve embedded asset source %s: %w", source.ID, err)
		}
		dst := filepath.Join(dir, filepath.FromSlash(root))
		if err := copyDirWithOptions(dst, src, copyDirOptions{skipNodeModules: true}); err != nil {
			return fmt.Errorf("copy embedded asset source %s: %w", source.ID, err)
		}
	}
	return nil
}

func embeddedSourceIDsFromPlanArtifacts(artifacts []specv2.ArtifactSpec) map[string]bool {
	out := map[string]bool{}
	for _, artifact := range artifacts {
		switch artifact.Type {
		case "binary", "runtime-package", "source", "template", "adapter", "cobra":
			for _, sourceID := range artifact.Sources {
				if strings.TrimSpace(sourceID) != "" {
					out[sourceID] = true
				}
			}
		}
	}
	return out
}

func embeddedAssetIDsFromPlanArtifacts(artifacts []specv2.ArtifactSpec) map[string]bool {
	out := map[string]bool{}
	for _, artifact := range artifacts {
		if artifact.Type != "embedded-assets" {
			continue
		}
		for _, sourceID := range artifact.Sources {
			if strings.TrimSpace(sourceID) != "" {
				out[sourceID] = true
			}
		}
	}
	return out
}

func cloneStringMapFromPlan(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
