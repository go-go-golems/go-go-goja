package main

import (
	"fmt"
	"os"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/plan"
	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/specv2"
)

func loadV2Plan(file string) (*plan.Plan, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	kind, _, err := specv2.DetectSchema(data)
	if err != nil {
		return nil, err
	}
	if kind != specv2.SchemaKindV2 {
		return nil, v1SpecRejectedError(file)
	}
	cfg, err := specv2.LoadFile(file)
	if err != nil {
		return nil, err
	}
	compiled, err := plan.Compile(plan.Options{Config: *cfg, Providers: syntheticProviderRegistryFromV2(cfg), StartDir: cfg.BaseDir})
	if err != nil {
		return nil, err
	}
	return compiled, nil
}

func v1SpecRejectedError(file string) error {
	return fmt.Errorf("%s appears to be a legacy xgoja spec; run xgoja migrate-spec -f %s --out xgoja.v2.yaml", file, file)
}
