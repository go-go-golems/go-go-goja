package jsverbs

import (
	"fmt"
	"sort"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds/schema"
)

type BindingMode string

const (
	BindingModePositional BindingMode = "positional"
	BindingModeSection    BindingMode = "section"
	BindingModeAll        BindingMode = "all"
	BindingModeContext    BindingMode = "context"
)

type ParameterBinding struct {
	Param       ParameterSpec
	Field       *FieldSpec
	Mode        BindingMode
	SectionSlug string
}

type ExtraFieldBinding struct {
	Name        string
	Field       *FieldSpec
	SectionSlug string
}

type VerbBindingPlan struct {
	Verb               *VerbSpec
	Parameters         []ParameterBinding
	ExtraFields        []ExtraFieldBinding
	ReferencedSections []string
}

func buildVerbBindingPlan(r *Registry, verb *VerbSpec) (*VerbBindingPlan, error) {
	if r == nil {
		return nil, fmt.Errorf("registry is nil")
	}

	referencedSections := map[string]struct{}{}
	for _, slug := range verb.UseSections {
		referencedSections[slug] = struct{}{}
	}
	for _, fieldMeta := range verb.Fields {
		if fieldMeta == nil {
			continue
		}
		if fieldMeta.Section != "" {
			referencedSections[fieldMeta.Section] = struct{}{}
		}
		switch bind := normalizeBind(fieldMeta.Bind); bind {
		case "", "all", "context":
		default:
			referencedSections[bind] = struct{}{}
		}
	}

	plan := &VerbBindingPlan{
		Verb:       verb,
		Parameters: make([]ParameterBinding, 0, len(verb.Params)),
	}

	usedFields := map[string]struct{}{}
	for _, param := range verb.Params {
		fieldSpec := verb.Field(param.Name)
		if fieldSpec != nil {
			usedFields[param.Name] = struct{}{}
			fieldSpec = fieldSpec.Clone()
		} else {
			fieldSpec = inferFieldFromParam(param)
		}
		if fieldSpec == nil {
			continue
		}
		if fieldSpec.Name == "" {
			fieldSpec.Name = param.Name
		}
		binding, err := resolveParameterBinding(verb, param, fieldSpec)
		if err != nil {
			return nil, err
		}
		plan.Parameters = append(plan.Parameters, binding)
	}

	extraFieldNames := make([]string, 0, len(verb.Fields))
	for name := range verb.Fields {
		if _, ok := usedFields[name]; ok {
			continue
		}
		extraFieldNames = append(extraFieldNames, name)
	}
	sort.Strings(extraFieldNames)
	for _, name := range extraFieldNames {
		fieldSpec := verb.Fields[name]
		if fieldSpec == nil {
			continue
		}
		fieldSpec = fieldSpec.Clone()
		if fieldSpec.Name == "" {
			fieldSpec.Name = name
		}
		if normalizeBind(fieldSpec.Bind) != "" {
			continue
		}
		sectionSlug := schema.DefaultSlug
		if fieldSpec.Section != "" {
			sectionSlug = fieldSpec.Section
		}
		plan.ExtraFields = append(plan.ExtraFields, ExtraFieldBinding{
			Name:        name,
			Field:       fieldSpec,
			SectionSlug: sectionSlug,
		})
	}

	for slug := range referencedSections {
		if _, ok := r.ResolveSection(verb, slug); !ok {
			return nil, fmt.Errorf("%s references unknown section %q", verb.SourceRef(), slug)
		}
	}

	plan.ReferencedSections = make([]string, 0, len(referencedSections))
	seenSections := map[string]struct{}{}
	appendSection := func(slug string) {
		if _, ok := referencedSections[slug]; !ok {
			return
		}
		if _, ok := seenSections[slug]; ok {
			return
		}
		plan.ReferencedSections = append(plan.ReferencedSections, slug)
		seenSections[slug] = struct{}{}
	}

	for _, slug := range verb.File.SectionOrder {
		appendSection(slug)
	}
	for _, slug := range r.SharedSectionOrder {
		appendSection(slug)
	}

	remaining := make([]string, 0, len(referencedSections))
	for slug := range referencedSections {
		if _, ok := seenSections[slug]; ok {
			continue
		}
		remaining = append(remaining, slug)
	}
	sort.Strings(remaining)
	for _, slug := range remaining {
		appendSection(slug)
	}

	return plan, nil
}

func resolveParameterBinding(verb *VerbSpec, param ParameterSpec, fieldSpec *FieldSpec) (ParameterBinding, error) {
	binding := ParameterBinding{
		Param:       param,
		Field:       fieldSpec,
		Mode:        BindingModePositional,
		SectionSlug: schema.DefaultSlug,
	}
	switch bind := normalizeBind(fieldSpec.Bind); bind {
	case "":
		if param.Kind != ParameterIdentifier && param.Kind != ParameterUnknown {
			return ParameterBinding{}, fmt.Errorf("%s parameter %q requires a bind because it is %s", verb.SourceRef(), param.Name, param.Kind)
		}
		if fieldSpec.Section != "" {
			binding.SectionSlug = fieldSpec.Section
		}
	case "all":
		binding.Mode = BindingModeAll
		binding.SectionSlug = ""
	case "context":
		binding.Mode = BindingModeContext
		binding.SectionSlug = ""
	default:
		binding.Mode = BindingModeSection
		binding.SectionSlug = bind
	}
	return binding, nil
}

func normalizeBind(bind string) string {
	switch cleaned := strings.TrimSpace(bind); cleaned {
	case "":
		return ""
	case "all", "context":
		return cleaned
	default:
		return cleanCommandWord(cleaned)
	}
}
