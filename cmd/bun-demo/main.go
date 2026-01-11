package main

import (
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
)

//go:embed assets/bundle.cjs assets-split/* assets-split/modules/*
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
	flag.Parse()

	vm, req := engine.NewWithOptions(require.WithLoader(embeddedSourceLoader))
	mod, err := req.Require(*entry)
	if err != nil {
		log.Fatalf("require %s: %v", *entry, err)
	}

	exports := mod.ToObject(vm)
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
