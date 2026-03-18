package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/sdk"
)

func main() {
	mod := sdk.MustModule(
		"plugin:echo",
		sdk.Version("v1"),
		sdk.Function("ping", func(_ context.Context, call *sdk.Call) (any, error) {
			if call.Len() == 0 {
				return nil, nil
			}
			return call.Value(0)
		}),
		sdk.Function("pid", func(context.Context, *sdk.Call) (any, error) {
			return os.Getpid(), nil
		}),
		sdk.Object("math",
			sdk.Method("add", func(_ context.Context, call *sdk.Call) (any, error) {
				if call.Len() != 2 {
					return nil, fmt.Errorf("math.add expects 2 arguments")
				}
				a, err := call.Float64(0)
				if err != nil {
					return nil, err
				}
				b, err := call.Float64(1)
				if err != nil {
					return nil, err
				}
				return a + b, nil
			}),
		),
	)

	sdk.Serve(mod)
}
