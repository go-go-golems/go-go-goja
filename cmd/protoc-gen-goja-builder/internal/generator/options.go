package generator

import (
	"fmt"
	"strconv"
	"strings"
)

// Options captures protoc-gen-goja-builder plugin parameters.
type Options struct {
	ModuleName     string
	Paths          string
	EmitDTS        bool
	EmitProvider   bool
	RegisterGlobal bool
	BuilderSuffix  string
	MessageRefName string
}

// DefaultOptions returns the generator defaults used when protoc parameters are
// omitted.
func DefaultOptions() Options {
	return Options{
		Paths:          "import",
		EmitDTS:        true,
		EmitProvider:   false,
		RegisterGlobal: false,
		BuilderSuffix:  "Builder",
		MessageRefName: "ProtoMessage",
	}
}

// ParseParameter parses the comma-separated parameter string passed by protoc.
func ParseParameter(parameter string) (Options, error) {
	opts := DefaultOptions()
	if strings.TrimSpace(parameter) == "" {
		return opts, nil
	}
	for _, part := range strings.Split(parameter, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, "=")
		key = strings.TrimSpace(key)
		if !ok {
			value = "true"
		}
		value = strings.TrimSpace(value)
		if err := opts.set(key, value); err != nil {
			return Options{}, err
		}
	}
	return opts, nil
}

func (o *Options) set(key, value string) error {
	switch key {
	case "module_name":
		o.ModuleName = value
	case "paths":
		if value != "import" && value != "source_relative" {
			return fmt.Errorf("invalid paths option %q", value)
		}
		o.Paths = value
	case "emit_dts":
		parsed, err := parseBoolOption(key, value)
		if err != nil {
			return err
		}
		o.EmitDTS = parsed
	case "emit_provider":
		parsed, err := parseBoolOption(key, value)
		if err != nil {
			return err
		}
		o.EmitProvider = parsed
	case "register_global":
		parsed, err := parseBoolOption(key, value)
		if err != nil {
			return err
		}
		o.RegisterGlobal = parsed
	case "builder_suffix":
		o.BuilderSuffix = value
	case "message_ref_name":
		o.MessageRefName = value
	case "module":
		// protoc-gen-go owns this option. Parse it leniently so tests and callers
		// can share one parameter string with protogen.
	case "annotate_code", "default_api_level":
		// protoc-gen-go/protogen options are accepted but not interpreted here.
	default:
		if strings.HasPrefix(key, "M") || strings.HasPrefix(key, "apilevelM") {
			return nil
		}
		return fmt.Errorf("unknown option %q", key)
	}
	return nil
}

func parseBoolOption(key, value string) (bool, error) {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid %s option %q: %w", key, value, err)
	}
	return parsed, nil
}
