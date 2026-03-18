package main

import (
	"context"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/sdk"
)

func main() {
	sdk.Serve(
		sdk.MustModule(
			"plugin:examples:system-info",
			sdk.Version("v1"),
			sdk.Doc("Example plugin with mixed export shapes and nested responses"),
			sdk.Capabilities("examples", "mixed-exports", "nested-results"),
			sdk.Function("hostname", func(_ context.Context, _ *sdk.Call) (any, error) {
				return os.Hostname()
			}, sdk.ExportDoc("Return the current hostname")),
			sdk.Function("envSummary", func(_ context.Context, call *sdk.Call) (any, error) {
				prefix := call.StringDefault(0, "GO")
				values := map[string]any{}
				for _, entry := range os.Environ() {
					name, value, found := strings.Cut(entry, "=")
					if !found || !strings.HasPrefix(name, prefix) {
						continue
					}
					values[name] = value
				}
				return map[string]any{
					"prefix": prefix,
					"count":  len(values),
					"values": values,
				}, nil
			}, sdk.ExportDoc("Return a filtered environment summary")),
			sdk.Object("runtime",
				sdk.ObjectDoc("Go runtime and process details"),
				sdk.Method("snapshot", func(_ context.Context, _ *sdk.Call) (any, error) {
					wd, _ := os.Getwd()
					exe, _ := os.Executable()
					return map[string]any{
						"goVersion": runtime.Version(),
						"goos":      runtime.GOOS,
						"goarch":    runtime.GOARCH,
						"cpus":      runtime.NumCPU(),
						"pid":       os.Getpid(),
						"cwd":       wd,
						"executable": map[string]any{
							"path": exe,
						},
						"timestamp": time.Now().UTC().Format(time.RFC3339),
					}, nil
				}, sdk.MethodSummary("Return nested runtime and process information"), sdk.MethodDoc("Return nested runtime and process information"), sdk.MethodTags("runtime", "process")),
			),
		),
	)
}
