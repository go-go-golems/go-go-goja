package buildspec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadFile(path string) (*BuildSpec, *Report, error) {
	if strings.TrimSpace(path) == "" {
		path = "xgoja.yaml"
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve build spec path %q: %w", path, err)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, nil, fmt.Errorf("read build spec %s: %w", abs, err)
	}

	unsupportedReport, err := unsupportedAssetFieldsReport(data)
	if err != nil {
		return nil, nil, fmt.Errorf("parse build spec %s: %w", abs, err)
	}

	buildSpec := &BuildSpec{}
	if err := yaml.Unmarshal(data, buildSpec); err != nil {
		return nil, nil, fmt.Errorf("parse build spec %s: %w", abs, err)
	}
	buildSpec.BaseDir = filepath.Dir(abs)
	applyDefaults(buildSpec)

	report := Validate(buildSpec)
	report.Checks = append(report.Checks, unsupportedReport.Checks...)
	if report.HasErrors() {
		return buildSpec, report, &ValidationError{Report: report}
	}
	return buildSpec, report, nil
}

func unsupportedAssetFieldsReport(data []byte) (*Report, error) {
	report := &Report{}
	root := yaml.Node{}
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return report, nil
	}
	mapping := root.Content[0]
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key := mapping.Content[i]
		value := mapping.Content[i+1]
		if key.Value != "assets" || value.Kind != yaml.SequenceNode {
			continue
		}
		for assetIndex, asset := range value.Content {
			if asset.Kind != yaml.MappingNode {
				continue
			}
			for j := 0; j+1 < len(asset.Content); j += 2 {
				field := asset.Content[j]
				if field.Value != "include" && field.Value != "exclude" {
					continue
				}
				report.AddError(
					"asset-filter-field",
					fmt.Sprintf("assets[%d].%s", assetIndex, field.Value),
					"asset include/exclude filters are not supported yet; remove this field",
				)
			}
		}
	}
	return report, nil
}

func applyDefaults(buildSpec *BuildSpec) {
	if buildSpec == nil {
		return
	}
	buildSpec.Name = strings.TrimSpace(buildSpec.Name)
	buildSpec.AppName = strings.TrimSpace(buildSpec.AppName)
	buildSpec.EnvPrefix = strings.TrimSpace(buildSpec.EnvPrefix)
	if buildSpec.ConfigFile != nil && buildSpec.ConfigFile.Enabled {
		if strings.TrimSpace(buildSpec.ConfigFile.FileName) == "" {
			buildSpec.ConfigFile.FileName = "config.yaml"
		}
	}
	if buildSpec.Name == "" {
		buildSpec.Name = "xgoja-app"
	}
	if strings.TrimSpace(buildSpec.Go.Version) == "" {
		buildSpec.Go.Version = "1.26"
	}
	if strings.TrimSpace(buildSpec.Go.Module) == "" {
		buildSpec.Go.Module = "example.com/generated/" + sanitizeModulePathPart(buildSpec.Name)
	}
	if strings.TrimSpace(buildSpec.Target.Kind) == "" {
		buildSpec.Target.Kind = "xgoja"
	}
	if strings.TrimSpace(buildSpec.Target.Output) == "" {
		buildSpec.Target.Output = filepath.ToSlash(filepath.Join("dist", sanitizeModulePathPart(buildSpec.Name)))
	}
	for i := range buildSpec.Packages {
		if strings.TrimSpace(buildSpec.Packages[i].Register) == "" {
			buildSpec.Packages[i].Register = "Register"
		}
	}
	if buildSpec.Commands.Eval.Enabled && strings.TrimSpace(buildSpec.Commands.Eval.Name) == "" {
		buildSpec.Commands.Eval.Name = "eval"
	}
	if buildSpec.Commands.Run.Enabled && strings.TrimSpace(buildSpec.Commands.Run.Name) == "" {
		buildSpec.Commands.Run.Name = "run"
	}
	if buildSpec.Commands.Repl.Enabled && strings.TrimSpace(buildSpec.Commands.Repl.Name) == "" {
		buildSpec.Commands.Repl.Name = "repl"
	}
	if buildSpec.Commands.JSVerbs.Enabled && strings.TrimSpace(buildSpec.Commands.JSVerbs.Name) == "" {
		buildSpec.Commands.JSVerbs.Name = "verbs"
	}
}

func sanitizeModulePathPart(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ' || r == '.':
			if !lastDash && b.Len() > 0 {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "xgoja-app"
	}
	return out
}
