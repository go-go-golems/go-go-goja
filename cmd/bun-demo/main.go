package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/host"
)

//go:embed assets/*.cjs assets/tsx.html
var bundleFS embed.FS

func embeddedSourceLoader(path string) ([]byte, error) {
	cleaned := strings.TrimPrefix(path, "./")
	cleaned = strings.TrimPrefix(cleaned, "/")

	data, err := bundleFS.ReadFile(cleaned)
	if err == nil {
		return data, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return nil, require.ModuleFileDoesNotExistError
	}
	return nil, err
}

func main() {
	entry := flag.String("entry", "./assets/bundle.cjs", "bundle entrypoint to require")
	var pluginDirs host.StringSliceFlag
	var allowPluginModules host.StringSliceFlag
	flag.Var(&pluginDirs, "plugin-dir", fmt.Sprintf("directory containing HashiCorp go-plugin module binaries (defaults to %s/... when omitted)", host.DefaultDiscoveryRoot()))
	flag.Var(&allowPluginModules, "allow-plugin-module", "allow only the listed plugin module names (for example plugin:examples:greeter)")
	flag.Parse()
	pluginSetup := host.NewRuntimeSetup(pluginDirs, allowPluginModules)

	builder := pluginSetup.WithBuilder(engine.NewBuilder().
		WithRequireOptions(require.WithLoader(embeddedSourceLoader)).
		UseModuleMiddleware(engine.MiddlewareSafe()))

	factory, err := builder.Build()
	if err != nil {
		log.Fatalf("build engine factory: %v", err)
	}

	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		log.Fatalf("create runtime: %v", err)
	}
	defer func() {
		_ = rt.Close(context.Background())
	}()

	mod, err := rt.Require.Require(*entry)
	if err != nil {
		log.Fatalf("require %s: %v", *entry, err)
	}

	exports := mod.ToObject(rt.VM)
	runVal := exports.Get("run")
	run, ok := goja.AssertFunction(runVal)
	if !ok {
		log.Fatalf("bundle export 'run' is not a function")
	}

	result, err := run(goja.Undefined())
	if err != nil {
		log.Fatalf("run(): %v", err)
	}

	if !goja.IsUndefined(result) {
		fmt.Println(result.Export())
	}
}
