package specv2

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

func Render(cfg Config) ([]byte, error) {
	ApplyDefaults(&cfg)
	if cfg.Schema != Schema {
		cfg.Schema = Schema
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal xgoja v2 spec: %w", err)
	}
	return bytes.TrimSpace(data), nil
}
