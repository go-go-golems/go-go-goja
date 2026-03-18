package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/sdk"
)

func main() {
	mod := sdk.MustModule(
		"plugin:examples:greeter",
		sdk.Version("v1"),
		sdk.Doc("Example greeter plugin"),
		sdk.Capabilities("examples", "strings"),
		sdk.Function("greet", func(_ context.Context, call *sdk.Call) (any, error) {
			name := call.StringDefault(0, "world")
			return fmt.Sprintf("hello, %s", name), nil
		}, sdk.ExportDoc("Return a greeting for the provided name")),
		sdk.Object("strings",
			sdk.ObjectDoc("String helpers"),
			sdk.Method("upper", func(_ context.Context, call *sdk.Call) (any, error) {
				return strings.ToUpper(call.StringDefault(0, "")), nil
			}, sdk.MethodSummary("Uppercase the first argument"), sdk.MethodDoc("Uppercase the first argument"), sdk.MethodTags("strings", "uppercase")),
			sdk.Method("lower", func(_ context.Context, call *sdk.Call) (any, error) {
				return strings.ToLower(call.StringDefault(0, "")), nil
			}),
		),
		sdk.Object("meta",
			sdk.ObjectDoc("Runtime metadata"),
			sdk.Method("pid", func(context.Context, *sdk.Call) (any, error) {
				return os.Getpid(), nil
			}),
		),
	)

	sdk.Serve(mod)
}
