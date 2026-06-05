package generate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildspec"
)

func RenderMain(buildSpec *buildspec.BuildSpec) string {
	rendered, err := renderMainTemplate(mainTemplateDataFromSpec(buildSpec))
	if err != nil {
		panic(err)
	}
	return rendered
}

func RenderPackage(buildSpec *buildspec.BuildSpec, packageName string) string {
	rendered, err := renderPackageTemplate(packageTemplateDataFromSpec(buildSpec, packageName))
	if err != nil {
		panic(err)
	}
	return rendered
}

func RenderEmbeddedSpec(buildSpec *buildspec.BuildSpec) string {
	buildSpec = runtimeSpec(buildSpec)
	var helpSpec *buildspec.HelpSpec
	if len(buildSpec.Help.Sources) > 0 {
		helpSpec = &buildSpec.Help
	}
	payload := struct {
		Name             string                                  `json:"name"`
		AppName          string                                  `json:"appName,omitempty"`
		EnvPrefix        string                                  `json:"envPrefix,omitempty"`
		ConfigFile       *buildspec.ConfigFileSpec               `json:"configFile,omitempty"`
		Target           buildspec.TargetSpec                    `json:"target"`
		Packages         []buildspec.PackageSpec                 `json:"packages"`
		Modules          []buildspec.ModuleInstanceSpec          `json:"modules"`
		Commands         buildspec.CommandsSpec                  `json:"commands"`
		CommandProviders []buildspec.CommandProviderInstanceSpec `json:"commandProviders,omitempty"`
		JSVerbs          []buildspec.JSVerbSourceSpec            `json:"jsverbs,omitempty"`
		Help             *buildspec.HelpSpec                     `json:"help,omitempty"`
		Assets           []buildspec.AssetSourceSpec             `json:"assets,omitempty"`
	}{
		Name:             buildSpec.Name,
		AppName:          buildSpec.AppName,
		EnvPrefix:        buildSpec.EnvPrefix,
		ConfigFile:       buildSpec.ConfigFile,
		Target:           buildSpec.Target,
		Packages:         buildSpec.Packages,
		Modules:          buildSpec.Modules,
		Commands:         buildSpec.Commands,
		CommandProviders: buildSpec.CommandProviders,
		JSVerbs:          buildSpec.JSVerbs,
		Help:             helpSpec,
		Assets:           buildSpec.Assets,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(data) + "\n"
}

func runtimeSpec(buildSpec *buildspec.BuildSpec) *buildspec.BuildSpec {
	if buildSpec == nil {
		return nil
	}
	clone := *buildSpec
	if len(buildSpec.JSVerbs) > 0 {
		clone.JSVerbs = append([]buildspec.JSVerbSourceSpec(nil), buildSpec.JSVerbs...)
		roots := embeddedJSVerbRoots(buildSpec)
		for i := range clone.JSVerbs {
			if root := roots[i]; root != "" {
				clone.JSVerbs[i].Path = root
			}
		}
	}
	if len(buildSpec.Help.Sources) > 0 {
		clone.Help.Sources = append([]buildspec.HelpSourceSpec(nil), buildSpec.Help.Sources...)
		roots := embeddedHelpRoots(buildSpec)
		for i := range clone.Help.Sources {
			if root := roots[i]; root != "" {
				clone.Help.Sources[i].Path = root
			}
		}
	}
	if len(buildSpec.Assets) > 0 {
		clone.Assets = append([]buildspec.AssetSourceSpec(nil), buildSpec.Assets...)
		roots := embeddedAssetRoots(buildSpec)
		for i := range clone.Assets {
			if root := roots[i]; root != "" {
				clone.Assets[i].Path = root
			}
		}
	}
	return &clone
}

func hasEmbeddedJSVerbSources(buildSpec *buildspec.BuildSpec) bool {
	if buildSpec == nil {
		return false
	}
	for _, source := range buildSpec.JSVerbs {
		if source.Embed && source.Path != "" && source.Package == "" && source.Source == "" {
			return true
		}
	}
	return false
}

func hasEmbeddedHelpSources(buildSpec *buildspec.BuildSpec) bool {
	if buildSpec == nil {
		return false
	}
	for _, source := range buildSpec.Help.Sources {
		if source.Embed && source.Path != "" && source.Package == "" && source.Source == "" {
			return true
		}
	}
	return false
}

func hasEmbeddedAssetSources(buildSpec *buildspec.BuildSpec) bool {
	if buildSpec == nil {
		return false
	}
	for _, source := range buildSpec.Assets {
		if source.Embed && source.Path != "" {
			return true
		}
	}
	return false
}

func embeddedJSVerbRoots(buildSpec *buildspec.BuildSpec) map[int]string {
	roots := map[int]string{}
	if buildSpec == nil {
		return roots
	}
	used := map[string]struct{}{}
	for i, source := range buildSpec.JSVerbs {
		if !source.Embed || strings.TrimSpace(source.Path) == "" || strings.TrimSpace(source.Package) != "" || strings.TrimSpace(source.Source) != "" {
			continue
		}
		base := sanitizeIdentifier(source.ID)
		if base == "" {
			base = "source"
		}
		name := base
		for suffix := 2; ; suffix++ {
			if _, ok := used[name]; !ok {
				break
			}
			name = fmt.Sprintf("%s_%d", base, suffix)
		}
		used[name] = struct{}{}
		roots[i] = "xgoja_embed/jsverbs/" + name
	}
	return roots
}

func embeddedHelpRoots(buildSpec *buildspec.BuildSpec) map[int]string {
	roots := map[int]string{}
	if buildSpec == nil {
		return roots
	}
	used := map[string]struct{}{}
	for i, source := range buildSpec.Help.Sources {
		if !source.Embed || strings.TrimSpace(source.Path) == "" || strings.TrimSpace(source.Package) != "" || strings.TrimSpace(source.Source) != "" {
			continue
		}
		base := sanitizeIdentifier(source.ID)
		if base == "" {
			base = "source"
		}
		name := base
		for suffix := 2; ; suffix++ {
			if _, ok := used[name]; !ok {
				break
			}
			name = fmt.Sprintf("%s_%d", base, suffix)
		}
		used[name] = struct{}{}
		roots[i] = "xgoja_embed/help/" + name
	}
	return roots
}

func embeddedAssetRoots(buildSpec *buildspec.BuildSpec) map[int]string {
	roots := map[int]string{}
	if buildSpec == nil {
		return roots
	}
	used := map[string]struct{}{}
	for i, source := range buildSpec.Assets {
		if !source.Embed || strings.TrimSpace(source.Path) == "" {
			continue
		}
		base := sanitizeIdentifier(source.ID)
		if base == "" {
			base = "source"
		}
		name := base
		for suffix := 2; ; suffix++ {
			if _, ok := used[name]; !ok {
				break
			}
			name = fmt.Sprintf("%s_%d", base, suffix)
		}
		used[name] = struct{}{}
		roots[i] = "xgoja_embed/assets/" + name
	}
	return roots
}

func importAliases(packages []buildspec.PackageSpec) map[string]string {
	aliases := map[string]string{}
	used := map[string]struct{}{}
	for _, pkg := range packages {
		base := sanitizeIdentifier(pkg.ID)
		if base == "" {
			parts := strings.Split(strings.Trim(pkg.Import, "/"), "/")
			base = sanitizeIdentifier(parts[len(parts)-1])
		}
		if base == "" {
			base = "provider"
		}
		alias := base
		for i := 2; ; i++ {
			if _, ok := used[alias]; !ok {
				break
			}
			alias = fmt.Sprintf("%s%d", base, i)
		}
		used[alias] = struct{}{}
		aliases[pkg.ID] = alias
	}
	return aliases
}

func sanitizeIdentifier(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	for i, r := range value {
		valid := r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || i > 0 && r >= '0' && r <= '9'
		if valid {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 {
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return ""
	}
	if out[0] >= '0' && out[0] <= '9' {
		out = "provider_" + out
	}
	return out
}

func escapeRawString(value string) string {
	return strings.ReplaceAll(value, "`", "` + \"`\" + `")
}
