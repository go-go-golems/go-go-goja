package generate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/buildspec"
)

func RenderMain(spec *buildspec.Spec) string {
	rendered, err := renderMainTemplate(mainTemplateDataFromSpec(spec))
	if err != nil {
		panic(err)
	}
	return rendered
}

func RenderEmbeddedSpec(spec *buildspec.Spec) string {
	spec = runtimeSpec(spec)
	var helpSpec *buildspec.HelpSpec
	if len(spec.Help.Sources) > 0 {
		helpSpec = &spec.Help
	}
	payload := struct {
		Name             string                              `json:"name"`
		Target           buildspec.TargetSpec                `json:"target"`
		Packages         []buildspec.PackageSpec             `json:"packages"`
		Runtimes         map[string]buildspec.Runtime        `json:"runtimes"`
		Commands         buildspec.CommandsSpec              `json:"commands"`
		CommandProviders []buildspec.CommandProviderInstance `json:"commandProviders,omitempty"`
		JSVerbs          []buildspec.JSVerbSourceSpec        `json:"jsverbs,omitempty"`
		Help             *buildspec.HelpSpec                 `json:"help,omitempty"`
	}{
		Name:             spec.Name,
		Target:           spec.Target,
		Packages:         spec.Packages,
		Runtimes:         spec.Runtimes,
		Commands:         spec.Commands,
		CommandProviders: spec.CommandProviders,
		JSVerbs:          spec.JSVerbs,
		Help:             helpSpec,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(data) + "\n"
}

func runtimeSpec(spec *buildspec.Spec) *buildspec.Spec {
	if spec == nil {
		return nil
	}
	clone := *spec
	if len(spec.JSVerbs) > 0 {
		clone.JSVerbs = append([]buildspec.JSVerbSourceSpec(nil), spec.JSVerbs...)
		roots := embeddedJSVerbRoots(spec)
		for i := range clone.JSVerbs {
			if root := roots[i]; root != "" {
				clone.JSVerbs[i].Path = root
			}
		}
	}
	if len(spec.Help.Sources) > 0 {
		clone.Help.Sources = append([]buildspec.HelpSourceSpec(nil), spec.Help.Sources...)
		roots := embeddedHelpRoots(spec)
		for i := range clone.Help.Sources {
			if root := roots[i]; root != "" {
				clone.Help.Sources[i].Path = root
			}
		}
	}
	return &clone
}

func hasEmbeddedJSVerbSources(spec *buildspec.Spec) bool {
	if spec == nil {
		return false
	}
	for _, source := range spec.JSVerbs {
		if source.Embed && source.Path != "" && source.Package == "" && source.Source == "" {
			return true
		}
	}
	return false
}

func hasEmbeddedHelpSources(spec *buildspec.Spec) bool {
	if spec == nil {
		return false
	}
	for _, source := range spec.Help.Sources {
		if source.Embed && source.Path != "" && source.Package == "" && source.Source == "" {
			return true
		}
	}
	return false
}

func embeddedJSVerbRoots(spec *buildspec.Spec) map[int]string {
	roots := map[int]string{}
	if spec == nil {
		return roots
	}
	used := map[string]struct{}{}
	for i, source := range spec.JSVerbs {
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

func embeddedHelpRoots(spec *buildspec.Spec) map[int]string {
	roots := map[int]string{}
	if spec == nil {
		return roots
	}
	used := map[string]struct{}{}
	for i, source := range spec.Help.Sources {
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
