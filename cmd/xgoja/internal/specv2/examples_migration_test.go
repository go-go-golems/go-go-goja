package specv2

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/migratebuildspec"
	"gopkg.in/yaml.v3"
)

func TestMigrateV1ExamplesToValidV2(t *testing.T) {
	examples := []string{
		"01-core-provider",
		"02-host-provider",
		"03-single-runtime-modules",
		"04-module-sections",
		"05-command-provider",
		"06-runtime-filesystem",
		"07-embedded-jsverbs",
		"08-provider-shipped-jsverbs",
		"09-provider-shipped-help-docs",
		"10-embedded-assets-fs",
		"11-config-env",
		"12-geppetto-host-services",
		"13-http-serve-jsverbs",
		"14-generated-runtime-package",
		"15-typescript-jsverbs",
	}

	for _, example := range examples {
		t.Run(example, func(t *testing.T) {
			path := filepath.Join("..", "..", "..", "..", "examples", "xgoja", example, "xgoja.yaml")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read example: %v", err)
			}
			kind, _, err := DetectSchema(data)
			if err != nil {
				t.Fatalf("detect schema: %v", err)
			}
			if kind == SchemaKindV2 {
				loaded, err := LoadData(data)
				if err != nil {
					t.Fatalf("load native v2 example: %v", err)
				}
				if loaded.Schema != Schema {
					t.Fatalf("schema = %q", loaded.Schema)
				}
				return
			}
			v1, err := loadV1ExampleForMigrationTest(path)
			if err != nil {
				t.Fatalf("load v1 example: %v", err)
			}
			result := MigrateV1(v1)
			rendered, err := Render(result.Config)
			if err != nil {
				t.Fatalf("render migrated config: %v", err)
			}
			loaded, err := LoadData(rendered)
			if err != nil {
				t.Fatalf("load rendered v2 config:\n%s\nerror: %v", rendered, err)
			}
			if loaded.Schema != Schema {
				t.Fatalf("schema = %q", loaded.Schema)
			}
			if len(loaded.Providers) != len(v1.Packages) {
				t.Fatalf("providers = %d, want %d", len(loaded.Providers), len(v1.Packages))
			}
			if strings.Contains(string(rendered), "platform:") || strings.Contains(string(rendered), "target: es") || strings.Contains(string(rendered), "format: cjs") {
				t.Fatalf("rendered v2 leaked low-level TypeScript profile fields:\n%s", rendered)
			}
		})
	}
}

func loadV1ExampleForMigrationTest(path string) (*migratebuildspec.BuildSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	v1 := &migratebuildspec.BuildSpec{}
	if err := yaml.Unmarshal(data, v1); err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	v1.BaseDir = filepath.Dir(abs)
	return v1, nil
}
