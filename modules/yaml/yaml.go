package yamlmod

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	"gopkg.in/yaml.v3"
)

// m implements the yaml native module for go-go-goja.
// It is stateless; an empty struct is sufficient.
type m struct{}

var _ modules.NativeModule = (*m)(nil)
var _ modules.TypeScriptDeclarer = (*m)(nil)

func (m) Name() string { return "yaml" }

func (m) TypeScriptModule() *spec.Module {
	return &spec.Module{
		Name: "yaml",
		Functions: []spec.Function{
			{
				Name: "parse",
				Params: []spec.Param{
					{Name: "input", Type: spec.String()},
				},
				Returns: spec.Any(),
			},
			{
				Name: "stringify",
				Params: []spec.Param{
					{Name: "value", Type: spec.Any()},
					{
						Name:     "options",
						Type:     spec.Object(spec.Field{Name: "indent", Type: spec.Number(), Optional: true}),
						Optional: true,
					},
				},
				Returns: spec.String(),
			},
			{
				Name: "validate",
				Params: []spec.Param{
					{Name: "input", Type: spec.String()},
				},
				Returns: spec.Object(
					spec.Field{Name: "valid", Type: spec.Boolean()},
					spec.Field{Name: "errors", Type: spec.Array(spec.String()), Optional: true},
				),
			},
		},
	}
}

func (m) Doc() string {
	return `
The yaml module provides YAML parsing and serialization.

Functions:
  parse(input): Parse a YAML string into a JavaScript value. Throws on parse errors.
  stringify(value, options?): Serialize a JavaScript value into a YAML string.
    Options:
      indent (number): Indentation spaces. Default is 2.
  validate(input): Validate YAML syntax. Returns { valid: boolean, errors?: string[] }.
`
}

func (mod m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)

	// parse(input: string) -> any | throws
	modules.SetExport(exports, mod.Name(), "parse", func(input string) (any, error) {
		var out any
		if err := yaml.Unmarshal([]byte(input), &out); err != nil {
			return nil, fmt.Errorf("yaml.parse: %w", err)
		}
		return out, nil
	})

	// stringify(value: any, options?: { indent?: number }) -> string | throws
	modules.SetExport(exports, mod.Name(), "stringify", func(value any, options map[string]any) (string, error) {
		indent := 2
		if options != nil {
			if v, ok := options["indent"]; ok {
				switch n := v.(type) {
				case int64:
					indent = int(n)
				case int:
					indent = n
				case float64:
					indent = int(n)
				default:
					return "", fmt.Errorf("yaml.stringify: indent must be a number, got %T", v)
				}
				if indent < 0 {
					return "", fmt.Errorf("yaml.stringify: indent must be >= 0")
				}
			}
			for k := range options {
				if k != "indent" {
					return "", fmt.Errorf("yaml.stringify: unknown option %q", k)
				}
			}
		}

		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(indent)
		if err := enc.Encode(value); err != nil {
			return "", fmt.Errorf("yaml.stringify: %w", err)
		}
		if err := enc.Close(); err != nil {
			return "", fmt.Errorf("yaml.stringify: %w", err)
		}
		return buf.String(), nil
	})

	// validate(input: string) -> { valid: boolean, errors?: string[] }
	modules.SetExport(exports, mod.Name(), "validate", func(input string) map[string]any {
		decoder := yaml.NewDecoder(strings.NewReader(input))
		var errors []string
		var out any
		for {
			if err := decoder.Decode(&out); err != nil {
				if err == io.EOF {
					break
				}
				errors = append(errors, err.Error())
				break
			}
		}
		if len(errors) > 0 {
			return map[string]any{"valid": false, "errors": errors}
		}
		return map[string]any{"valid": true}
	})
}

func init() {
	modules.Register(&m{})
}
