package jsverbs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Command struct {
	*cmds.CommandDescription
	registry *Registry
	verb     *VerbSpec
}

type WriterCommand struct {
	*cmds.CommandDescription
	registry *Registry
	verb     *VerbSpec
}

var _ cmds.GlazeCommand = (*Command)(nil)
var _ cmds.WriterCommand = (*WriterCommand)(nil)

func (r *Registry) Commands() ([]cmds.Command, error) {
	commands := make([]cmds.Command, 0, len(r.verbs))
	for _, verb := range r.verbs {
		cmd, err := r.commandForVerb(verb)
		if err != nil {
			return nil, err
		}
		commands = append(commands, cmd)
	}
	return commands, nil
}

func (r *Registry) commandForVerb(verb *VerbSpec) (cmds.Command, error) {
	description, err := r.buildDescription(verb)
	if err != nil {
		return nil, err
	}
	switch verb.OutputMode {
	case OutputModeGlaze:
		return &Command{
			CommandDescription: description,
			registry:           r,
			verb:               verb,
		}, nil
	case OutputModeText:
		return &WriterCommand{
			CommandDescription: description,
			registry:           r,
			verb:               verb,
		}, nil
	default:
		return nil, fmt.Errorf("%s has unsupported output mode %q", verb.SourceRef(), verb.OutputMode)
	}
}

func (r *Registry) buildDescription(verb *VerbSpec) (*cmds.CommandDescription, error) {
	plan, err := buildVerbBindingPlan(r, verb)
	if err != nil {
		return nil, err
	}

	sections := map[string]*schema.SectionImpl{}
	ordered := []string{}

	ensureSection := func(slug, title, description string) (*schema.SectionImpl, error) {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			slug = schema.DefaultSlug
		}
		if section, ok := sections[slug]; ok {
			if section.Name == "" && title != "" {
				section.Name = title
			}
			if section.Description == "" && description != "" {
				section.Description = description
			}
			return section, nil
		}
		if title == "" {
			if slug == schema.DefaultSlug {
				title = "Arguments"
			} else {
				title = prettySectionTitle(slug)
			}
		}
		section, err := schema.NewSection(slug, title, schema.WithDescription(description))
		if err != nil {
			return nil, err
		}
		sections[slug] = section
		ordered = append(ordered, slug)
		return section, nil
	}

	if _, err := ensureSection(schema.DefaultSlug, "Arguments", ""); err != nil {
		return nil, err
	}

	for _, slug := range plan.ReferencedSections {
		spec, ok := r.ResolveSection(verb, slug)
		if !ok {
			return nil, fmt.Errorf("%s references unknown section %q", verb.SourceRef(), slug)
		}
		section, err := ensureSection(spec.Slug, spec.Title, spec.Description)
		if err != nil {
			return nil, err
		}
		fieldNames := make([]string, 0, len(spec.Fields))
		for name := range spec.Fields {
			fieldNames = append(fieldNames, name)
		}
		sort.Strings(fieldNames)
		for _, name := range fieldNames {
			field, err := buildFieldDefinition(spec.Fields[name])
			if err != nil {
				return nil, fmt.Errorf("%s section %s field %s: %w", verb.SourceRef(), slug, name, err)
			}
			section.AddFields(field)
		}
	}

	addField := func(sectionSlug string, fieldSpec *FieldSpec) error {
		section, err := ensureSection(sectionSlug, "", "")
		if err != nil {
			return err
		}
		field, err := buildFieldDefinition(fieldSpec)
		if err != nil {
			return err
		}
		if field.IsArgument && sectionSlug != schema.DefaultSlug {
			return fmt.Errorf("arguments are only supported in the default section (field %s in %s)", field.Name, sectionSlug)
		}
		section.AddFields(field)
		return nil
	}

	for _, binding := range plan.Parameters {
		if binding.Mode != BindingModePositional {
			continue
		}
		if err := addField(binding.SectionSlug, binding.Field); err != nil {
			return nil, fmt.Errorf("%s field %s: %w", verb.SourceRef(), binding.Param.Name, err)
		}
	}

	for _, extraField := range plan.ExtraFields {
		if err := addField(extraField.SectionSlug, extraField.Field); err != nil {
			return nil, fmt.Errorf("%s field %s: %w", verb.SourceRef(), extraField.Name, err)
		}
	}

	description := cmds.NewCommandDescription(
		verb.Name,
		cmds.WithShort(verb.Short),
		cmds.WithLong(verb.Long),
		cmds.WithParents(verb.Parents...),
		cmds.WithSource("jsverbs:"+verb.SourceRef()),
	)
	for _, slug := range ordered {
		description.Schema.Set(slug, sections[slug])
	}
	return description, nil
}

func inferFieldFromParam(param ParameterSpec) *FieldSpec {
	field := &FieldSpec{Name: param.Name}
	if param.Rest {
		field.Argument = true
		field.Type = "stringList"
		return field
	}
	switch param.Kind {
	case ParameterIdentifier, ParameterUnknown:
		field.Type = "string"
		return field
	case ParameterObject, ParameterArray:
		return field
	}

	return field
}

func buildFieldDefinition(spec *FieldSpec) (*fields.Definition, error) {
	if spec == nil {
		return nil, fmt.Errorf("field spec is nil")
	}
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return nil, fmt.Errorf("field name is empty")
	}
	fieldType, err := glazedFieldType(spec)
	if err != nil {
		return nil, err
	}
	options := []fields.Option{}
	if spec.Help != "" {
		options = append(options, fields.WithHelp(spec.Help))
	}
	if spec.Short != "" {
		options = append(options, fields.WithShortFlag(spec.Short))
	}
	if spec.Required || (spec.Argument && spec.Default == nil) {
		options = append(options, fields.WithRequired(true))
	}
	if spec.Argument {
		options = append(options, fields.WithIsArgument(true))
	}
	if len(spec.Choices) > 0 {
		options = append(options, fields.WithChoices(spec.Choices...))
	}
	if spec.Default != nil {
		value, err := normalizeDefaultValue(spec.Default, fieldType)
		if err != nil {
			return nil, err
		}
		options = append(options, fields.WithDefault(value))
	}
	return fields.New(name, fieldType, options...), nil
}

func glazedFieldType(spec *FieldSpec) (fields.Type, error) {
	typeName := strings.ToLower(strings.TrimSpace(spec.Type))
	if typeName == "" && len(spec.Choices) > 0 {
		typeName = "choice"
	}
	switch typeName {
	case "", "string":
		return fields.TypeString, nil
	case "bool", "boolean":
		return fields.TypeBool, nil
	case "int", "integer":
		return fields.TypeInteger, nil
	case "float", "number":
		return fields.TypeFloat, nil
	case "stringlist", "list", "[]string":
		return fields.TypeStringList, nil
	case "choice":
		return fields.TypeChoice, nil
	case "choicelist":
		return fields.TypeChoiceList, nil
	default:
		return "", fmt.Errorf("unsupported field type %q", spec.Type)
	}
}

func normalizeDefaultValue(value interface{}, fieldType fields.Type) (interface{}, error) {
	switch fieldType {
	case fields.TypeString, fields.TypeChoice:
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string default, got %T", value)
		}
		return s, nil
	case fields.TypeBool:
		b, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool default, got %T", value)
		}
		return b, nil
	case fields.TypeInteger:
		switch v := value.(type) {
		case float64:
			return int(v), nil
		case int:
			return v, nil
		case int64:
			return v, nil
		default:
			return nil, fmt.Errorf("expected integer default, got %T", value)
		}
	case fields.TypeFloat:
		switch v := value.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		default:
			return nil, fmt.Errorf("expected float default, got %T", value)
		}
	case fields.TypeStringList, fields.TypeChoiceList:
		switch v := value.(type) {
		case []interface{}:
			out := make([]string, 0, len(v))
			for _, item := range v {
				s, ok := item.(string)
				if !ok {
					return nil, fmt.Errorf("expected string list default, got %T", item)
				}
				out = append(out, s)
			}
			return out, nil
		case []string:
			return append([]string{}, v...), nil
		default:
			return nil, fmt.Errorf("expected string list default, got %T", value)
		}
	case fields.TypeSecret,
		fields.TypeStringFromFile,
		fields.TypeStringFromFiles,
		fields.TypeFile,
		fields.TypeFileList,
		fields.TypeObjectListFromFile,
		fields.TypeObjectListFromFiles,
		fields.TypeObjectFromFile,
		fields.TypeStringListFromFile,
		fields.TypeStringListFromFiles,
		fields.TypeKeyValue,
		fields.TypeDate,
		fields.TypeIntegerList,
		fields.TypeFloatList:
		return value, nil
	}

	return value, nil
}

func (c *Command) RunIntoGlazeProcessor(ctx context.Context, parsedValues *values.Values, gp middlewares.Processor) error {
	result, err := c.registry.invoke(ctx, c.verb, parsedValues)
	if err != nil {
		return err
	}
	rows, err := rowsFromResult(result)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := gp.AddRow(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

func (c *WriterCommand) RunIntoWriter(ctx context.Context, parsedValues *values.Values, w io.Writer) error {
	result, err := c.registry.invoke(ctx, c.verb, parsedValues)
	if err != nil {
		return err
	}
	text, err := renderTextResult(result)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, text)
	return err
}

func renderTextResult(result interface{}) (string, error) {
	if result == nil {
		return "", nil
	}
	switch v := result.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case fmt.Stringer:
		return v.String(), nil
	}

	value := reflect.ValueOf(result)
	if value.IsValid() && (value.Kind() == reflect.Slice || value.Kind() == reflect.Array) {
		if value.Type().Elem().Kind() == reflect.Uint8 {
			return string(value.Bytes()), nil
		}
	}

	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func rowsFromResult(result interface{}) ([]types.Row, error) {
	if result == nil {
		return nil, nil
	}
	if row, ok := toRow(result); ok {
		return []types.Row{row}, nil
	}
	value := reflect.ValueOf(result)
	if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		rows := make([]types.Row, 0, value.Len())
		for i := 0; i < value.Len(); i++ {
			item := value.Index(i).Interface()
			if row, ok := toRow(item); ok {
				rows = append(rows, row)
			} else {
				rows = append(rows, types.NewRow(types.MRP("value", item)))
			}
		}
		return rows, nil
	}

	return []types.Row{types.NewRow(types.MRP("value", result))}, nil
}

func prettySectionTitle(slug string) string {
	return cases.Title(language.English).String(strings.ReplaceAll(slug, "-", " "))
}

func toRow(value interface{}) (types.Row, bool) {
	switch v := value.(type) {
	case map[string]interface{}:
		return types.NewRowFromMap(v), true
	case map[string]string:
		row := types.NewRow()
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			row.Set(key, v[key])
		}
		return row, true
	default:
		rv := reflect.ValueOf(value)
		if !rv.IsValid() || rv.Kind() != reflect.Map || rv.Type().Key().Kind() != reflect.String {
			return nil, false
		}
		row := types.NewRow()
		iter := rv.MapRange()
		keys := []string{}
		valuesByKey := map[string]interface{}{}
		for iter.Next() {
			key := iter.Key().String()
			keys = append(keys, key)
			valuesByKey[key] = iter.Value().Interface()
		}
		sort.Strings(keys)
		for _, key := range keys {
			row.Set(key, valuesByKey[key])
		}
		return row, true
	}
}
