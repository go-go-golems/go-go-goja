package main

import (
	"context"
	"fmt"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/sdk"
)

func main() {
	sdk.Serve(
		sdk.MustModule(
			"plugin:examples:failing",
			sdk.Version("v1"),
			sdk.Doc("Example plugin that demonstrates explicit handler errors"),
			sdk.Capabilities("examples", "errors"),
			sdk.Function("always", func(_ context.Context, call *sdk.Call) (any, error) {
				reason := call.StringDefault(0, "forced failure")
				return nil, fmt.Errorf("always failed: %s", reason)
			}, sdk.ExportDoc("Always return an error")),
			sdk.Function("sometimes", func(_ context.Context, call *sdk.Call) (any, error) {
				mode := call.StringDefault(0, "ok")
				if mode == "fail" {
					return nil, fmt.Errorf("sometimes failed because mode=%q", mode)
				}
				return map[string]any{
					"ok":   true,
					"mode": mode,
				}, nil
			}, sdk.ExportDoc("Return an error only for the fail mode")),
			sdk.Object("checks",
				sdk.ObjectDoc("Validation methods that reject bad inputs"),
				sdk.Method("requirePositive", func(_ context.Context, call *sdk.Call) (any, error) {
					value, err := call.Float64(0)
					if err != nil {
						return nil, err
					}
					if value <= 0 {
						return nil, fmt.Errorf("value must be positive, got %v", value)
					}
					return map[string]any{
						"value": value,
						"ok":    true,
					}, nil
				}, sdk.MethodSummary("Reject zero or negative numeric arguments"), sdk.MethodDoc("Reject zero or negative numeric arguments"), sdk.MethodTags("validation", "numbers")),
			),
		),
	)
}
