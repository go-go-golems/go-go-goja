package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/sdk"
)

func main() {
	sdk.Serve(
		sdk.MustModule(
			"plugin:validator",
			sdk.Version("v1"),
			sdk.Doc("Example plugin that demonstrates sdk.Call helpers and validation errors"),
			sdk.Capabilities("examples", "call-helpers", "validation"),
			sdk.Function("greet", func(_ context.Context, call *sdk.Call) (any, error) {
				name := call.StringDefault(0, "world")
				excited := false
				if call.Len() > 1 {
					var err error
					excited, err = call.Bool(1)
					if err != nil {
						return nil, err
					}
				}
				if excited {
					return "hello, " + strings.ToUpper(name) + "!", nil
				}
				return "hello, " + name, nil
			}, sdk.ExportDoc("Use StringDefault and Bool helpers together")),
			sdk.Function("grade", func(_ context.Context, call *sdk.Call) (any, error) {
				score, err := call.Float64(0)
				if err != nil {
					return nil, err
				}
				strict := false
				if call.Len() > 1 {
					strict, err = call.Bool(1)
					if err != nil {
						return nil, err
					}
				}
				passing := score >= 0.5
				if strict {
					passing = score >= 0.8
				}
				return map[string]any{
					"score":   score,
					"strict":  strict,
					"passing": passing,
				}, nil
			}, sdk.ExportDoc("Validate numeric and boolean arguments")),
			sdk.Object("profiles",
				sdk.ObjectDoc("Map-based validation helpers"),
				sdk.Method("check", func(_ context.Context, call *sdk.Call) (any, error) {
					profile, err := call.Map(0)
					if err != nil {
						return nil, err
					}
					name, _ := profile["name"].(string)
					name = strings.TrimSpace(name)
					if name == "" {
						return nil, fmt.Errorf("profile.name must be a non-empty string")
					}
					tags, err := call.Slice(1)
					if err != nil {
						return nil, err
					}
					return map[string]any{
						"name":      name,
						"tagCount":  len(tags),
						"hasAdmin":  containsString(tags, "admin"),
						"validated": true,
					}, nil
				}, sdk.ExportDoc("Validate a profile object and an accompanying tag list")),
			),
		),
	)
}

func containsString(values []any, needle string) bool {
	for _, value := range values {
		if s, ok := value.(string); ok && s == needle {
			return true
		}
	}
	return false
}
