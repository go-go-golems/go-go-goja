package app

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

const xgojaSectionSlug = "xgoja"
const xgojaDebugPanicStackField = "debug-panic-stack"

func xgojaRuntimeSection() (schema.Section, error) {
	return schema.NewSection(xgojaSectionSlug, "xgoja",
		schema.WithFields(
			fields.New(xgojaDebugPanicStackField, fields.TypeBool,
				fields.WithHelp("Include Go debug stacks in recovered runtime panic errors")),
		),
	)
}

func includeRecoveredPanicStack(vals *values.Values) (bool, error) {
	if vals == nil {
		return false, nil
	}
	field, ok := vals.GetField(xgojaSectionSlug, xgojaDebugPanicStackField)
	if !ok || field == nil || field.Value == nil {
		return false, nil
	}
	switch v := field.Value.(type) {
	case bool:
		return v, nil
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "", "false", "0", "no", "off":
			return false, nil
		case "true", "1", "yes", "on":
			return true, nil
		default:
			return false, fmt.Errorf("invalid xgoja debug panic stack value %q", v)
		}
	default:
		return false, fmt.Errorf("invalid xgoja debug panic stack value type %T", field.Value)
	}
}
