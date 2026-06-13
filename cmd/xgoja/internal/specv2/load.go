package specv2

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type SchemaKind string

const (
	SchemaKindUnknown SchemaKind = "unknown"
	SchemaKindV1      SchemaKind = "xgoja/v1"
	SchemaKindV2      SchemaKind = Schema
)

func LoadFile(path string) (*Config, error) {
	if strings.TrimSpace(path) == "" {
		path = "xgoja.yaml"
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve xgoja v2 spec path %q: %w", path, err)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("read xgoja v2 spec %s: %w", abs, err)
	}
	cfg, err := LoadData(data)
	if err != nil {
		return nil, fmt.Errorf("parse xgoja v2 spec %s: %w", abs, err)
	}
	cfg.BaseDir = filepath.Dir(abs)
	return cfg, nil
}

func LoadData(data []byte) (*Config, error) {
	kind, rawSchema, err := DetectSchema(data)
	if err != nil {
		return nil, err
	}
	if kind != SchemaKindV2 {
		if kind == SchemaKindV1 {
			return nil, fmt.Errorf("xgoja.yaml appears to be v1; run xgoja migrate-spec -f xgoja.yaml --out xgoja.v2.yaml")
		}
		return nil, fmt.Errorf("unsupported xgoja schema %q; expected %q", rawSchema, Schema)
	}

	cfg := &Config{}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(cfg); err != nil {
		return nil, err
	}
	ApplyDefaults(cfg)
	report := Validate(cfg)
	if report.HasErrors() {
		return cfg, &ValidationError{Report: report}
	}
	return cfg, nil
}

func DetectSchema(data []byte) (SchemaKind, string, error) {
	root := yaml.Node{}
	if err := yaml.Unmarshal(data, &root); err != nil {
		return SchemaKindUnknown, "", err
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return SchemaKindUnknown, "", nil
	}
	mapping := root.Content[0]
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key := mapping.Content[i]
		value := mapping.Content[i+1]
		if key.Value != "schema" {
			continue
		}
		schema := strings.TrimSpace(value.Value)
		switch schema {
		case Schema:
			return SchemaKindV2, schema, nil
		case "", "xgoja/v1":
			return SchemaKindV1, schema, nil
		default:
			return SchemaKindUnknown, schema, nil
		}
	}
	return SchemaKindV1, "", nil
}
