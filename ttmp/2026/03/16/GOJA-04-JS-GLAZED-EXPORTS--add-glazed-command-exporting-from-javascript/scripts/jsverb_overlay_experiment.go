package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/engine"
)

func main() {
	sources := map[string]string{
		"entry.js": `
const helper = require("./helper.js");

function listIssues(repo) {
  return helper.prefix(repo);
}

const hidden = () => "hidden";

module.exports = {
  exported: true
};
`,
		"helper.js": `
module.exports = {
  prefix(value) {
    return "repo:" + value;
  }
};
`,
	}

	loader := func(path string) ([]byte, error) {
		cleaned := strings.TrimPrefix(path, "./")
		src, ok := sources[cleaned]
		if !ok {
			return nil, require.ModuleFileDoesNotExistError
		}
		if cleaned == "entry.js" {
			src = injectOverlay(cleaned, src, []string{"listIssues", "hidden"})
		}
		return []byte(src), nil
	}

	factory, err := engine.NewBuilder(
		engine.WithRequireOptions(require.WithLoader(loader)),
	).Build()
	if err != nil {
		panic(err)
	}

	rt, err := factory.NewRuntime(context.Background())
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = rt.Close(context.Background())
	}()

	mod, err := rt.Require.Require("./entry.js")
	if err != nil {
		panic(err)
	}

	registryValue := rt.VM.Get("__glazedVerbRegistry")
	registryObject := registryValue.ToObject(rt.VM)
	entryValue := registryObject.Get("entry.js")
	entryObject := entryValue.ToObject(rt.VM)

	listIssuesValue := entryObject.Get("listIssues")
	listIssuesFn, ok := goja.AssertFunction(listIssuesValue)
	if !ok {
		panic("listIssues was not captured as a function")
	}

	out, err := listIssuesFn(goja.Undefined(), rt.VM.ToValue("openai/openai"))
	if err != nil {
		panic(err)
	}

	result := map[string]any{
		"moduleExports": mod.Export(),
		"registryKeys":  objectKeys(entryObject),
		"listIssues":    out.Export(),
		"hiddenType":    entryObject.Get("hidden").ExportType().String(),
	}

	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}

func injectOverlay(modulePath string, source string, functionNames []string) string {
	var b strings.Builder
	b.WriteString(`
globalThis.__glazedVerbRegistry = globalThis.__glazedVerbRegistry || {};
globalThis.__package__ = globalThis.__package__ || function() {};
globalThis.__section__ = globalThis.__section__ || function() {};
globalThis.__verb__ = globalThis.__verb__ || function() {};
globalThis.doc = globalThis.doc || function() { return ""; };
`)
	b.WriteString(source)
	b.WriteString("\n")
	b.WriteString(`globalThis.__glazedVerbRegistry["`)
	b.WriteString(modulePath)
	b.WriteString(`"] = {`)
	for i, name := range functionNames {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(name)
		b.WriteString(`: typeof `)
		b.WriteString(name)
		b.WriteString(` === "function" ? `)
		b.WriteString(name)
		b.WriteString(` : undefined`)
	}
	b.WriteString("};\n")
	return b.String()
}

func objectKeys(obj *goja.Object) []string {
	keys := make([]string, 0, len(obj.Keys()))
	keys = append(keys, obj.Keys()...)
	return keys
}
