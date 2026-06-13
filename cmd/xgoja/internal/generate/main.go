package generate

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/plan"
)

func RenderMainPlan(compiled *plan.Plan) string {
	rendered, err := renderMainTemplate(mainTemplateDataFromPlan(compiled))
	if err != nil {
		panic(err)
	}
	return rendered
}

func RenderPackagePlan(compiled *plan.Plan, packageName string) string {
	rendered, err := renderPackageTemplate(packageTemplateDataFromPlan(compiled, packageName))
	if err != nil {
		panic(err)
	}
	return rendered
}

func RenderSourceFragmentsPlan(compiled *plan.Plan, packageName string) map[string]string {
	data := packageTemplateDataFromPlan(compiled, packageName)
	fragments := map[string]func(packageTemplateData) (string, error){
		"spec.gen.go":      renderSpecFragmentTemplate,
		"providers.gen.go": renderProvidersFragmentTemplate,
		"bundle.gen.go":    renderBundleFragmentTemplate,
	}
	if data.HasEmbedded {
		fragments["embed.gen.go"] = renderEmbedFragmentTemplate
	}
	out := make(map[string]string, len(fragments))
	for name, render := range fragments {
		rendered, err := render(data)
		if err != nil {
			panic(err)
		}
		out[name] = rendered
	}
	return out
}

func RenderDTSGenMainPlan(compiled *plan.Plan, strict bool) string {
	rendered, err := renderDTSGenMainTemplate(dtsGenTemplateDataFromPlan(compiled, strict))
	if err != nil {
		panic(err)
	}
	return rendered
}

func RenderCustomTemplatePlan(compiled *plan.Plan, packageName string, templatePath string) string {
	rendered, err := loadCustomTemplate(templatePath, packageTemplateDataFromPlan(compiled, packageName))
	if err != nil {
		panic(err)
	}
	return rendered
}

type importAliasSeed struct {
	ID     string
	Import string
}

func importAliases(packages []importAliasSeed) map[string]string {
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
