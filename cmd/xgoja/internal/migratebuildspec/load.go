package migratebuildspec

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

	unsupportedReport, err := unsupportedFieldsReport(data)
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

func unsupportedFieldsReport(data []byte) (*Report, error) {
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
		switch key.Value {
		case "runtimes":
			report.AddError(
				"runtime-profile-field",
				"runtimes",
				"runtime profiles are no longer supported; move the single runtime's modules to top-level modules",
			)
		case "commands":
			addUnsupportedCommandRuntimeFields(report, value)
		case "commandProviders":
			addUnsupportedCommandProviderRuntimeProfileFields(report, value)
		case "assets":
			addUnsupportedAssetFilterFields(report, value)
		}
	}
	return report, nil
}

func addUnsupportedAssetFilterFields(report *Report, value *yaml.Node) {
	if value == nil || value.Kind != yaml.SequenceNode {
		return
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

func addUnsupportedCommandRuntimeFields(report *Report, value *yaml.Node) {
	if value == nil || value.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(value.Content); i += 2 {
		commandKey := value.Content[i]
		commandValue := value.Content[i+1]
		if commandValue.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j+1 < len(commandValue.Content); j += 2 {
			field := commandValue.Content[j]
			if field.Value != "runtime" {
				continue
			}
			report.AddError(
				"command-runtime-field",
				fmt.Sprintf("commands.%s.runtime", commandKey.Value),
				"command runtime selectors are no longer supported; all commands use top-level modules",
			)
		}
	}
}

func addUnsupportedCommandProviderRuntimeProfileFields(report *Report, value *yaml.Node) {
	if value == nil || value.Kind != yaml.SequenceNode {
		return
	}
	for providerIndex, provider := range value.Content {
		if provider.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j+1 < len(provider.Content); j += 2 {
			field := provider.Content[j]
			if field.Value != "runtimeProfile" {
				continue
			}
			report.AddError(
				"command-provider-runtime-profile-field",
				fmt.Sprintf("commandProviders[%d].runtimeProfile", providerIndex),
				"command provider runtime profiles are no longer supported; all command providers use top-level modules",
			)
		}
	}
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
		buildSpec.Go.Module = "xgoja.generated/" + sanitizeModulePathPart(buildSpec.Name)
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
	for i := range buildSpec.JSVerbs {
		applyTypeScriptDefaults(buildSpec.JSVerbs[i].TypeScript)
	}
}

func applyTypeScriptDefaults(spec *TypeScriptSpec) {
	if spec == nil {
		return
	}
	spec.Target = strings.TrimSpace(spec.Target)
	if spec.Enabled && spec.Target == "" {
		spec.Target = "es2015"
	}
	spec.Format = strings.TrimSpace(spec.Format)
	if spec.Enabled && spec.Format == "" {
		spec.Format = "cjs"
	}
	spec.Platform = strings.TrimSpace(spec.Platform)
	if spec.Enabled && spec.Platform == "" {
		spec.Platform = "neutral"
	}
	spec.Tsconfig = strings.TrimSpace(spec.Tsconfig)
	spec.Sourcemap = strings.TrimSpace(spec.Sourcemap)
	for i := range spec.External {
		spec.External[i] = strings.TrimSpace(spec.External[i])
	}
	for i := range spec.CheckCommand {
		spec.CheckCommand[i] = strings.TrimSpace(spec.CheckCommand[i])
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
