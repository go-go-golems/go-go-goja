package lazy

import "github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"

// TypeScriptModule declares the TypeScript surface for require("lazy").
func (module) TypeScriptModule() *spec.Module {
	return &spec.Module{
		Name:        "lazy",
		Description: "Lazily-materialised objects backed by Go via goja Proxy.",
		RawDTS: []string{
			"interface LazyMapOptions {",
			"  keys: string[];",
			"  get: (key: string) => any;",
			"  mutable?: boolean;",
			"}",
			"interface LazyArrayOptions {",
			"  length: number;",
			"  get: (index: number) => any;",
			"}",
			"interface LazyRecordOptions {",
			"  keys: () => string[];",
			"  get: (key: string) => any;",
			"  has?: (key: string) => boolean;",
			"  mutable?: boolean;",
			"}",
			"// Lazy values: reading a key/index runs `get` once (then cached).",
			"// Object.keys() enumerates keys without calling `get`.",
			"type LazyMap = Record<string, any>;",
			"type LazyRecord = Record<string, any>;",
		},
		Functions: []spec.Function{
			{
				Name:        "map",
				Description: "Build a lazily-valued string-keyed object with known keys.",
				Params:      []spec.Param{{Name: "options", Type: spec.Named("LazyMapOptions")}},
				Returns:     spec.Named("LazyMap"),
			},
			{
				Name:        "array",
				Description: "Build a lazily-valued integer-indexed array-like with a length.",
				Params:      []spec.Param{{Name: "options", Type: spec.Named("LazyArrayOptions")}},
				Returns:     spec.Any(),
			},
			{
				Name:        "record",
				Description: "Build a lazily-valued object whose key set is produced on demand.",
				Params:      []spec.Param{{Name: "options", Type: spec.Named("LazyRecordOptions")}},
				Returns:     spec.Named("LazyRecord"),
			},
		},
	}
}
