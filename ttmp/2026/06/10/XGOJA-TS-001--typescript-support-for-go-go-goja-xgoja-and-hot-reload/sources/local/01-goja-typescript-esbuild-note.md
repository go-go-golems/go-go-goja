---
Title: goja TypeScript esbuild note
Ticket: XGOJA-TS-001
Status: active
Topics:
    - goja
    - xgoja
    - typescript
    - tooling
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - /tmp/goja-ts.md
Summary: "Imported note explaining esbuild Go API usage for transpiling and bundling TypeScript for goja."
LastUpdated: 2026-06-10T21:45:00-04:00
WhatFor: "Use as source material for the TypeScript support design."
WhenToUse: "When reviewing why esbuild's Go API is the proposed compiler backend."
---

Yes. You can make it self-contained in **one Go binary** by importing esbuild’s Go package:

```bash
go get github.com/evanw/esbuild
```

Then:

```go
import "github.com/evanw/esbuild/pkg/api"
```

This does **not** require Node, npm, or an external `esbuild` executable at runtime. The Go API is a library API. esbuild’s Go package exposes two main operations: **Build** and **Transform**. The docs describe `Build` as an end-to-end build over entry points and dependencies, and `Transform` as a source-string-to-JavaScript operation for things like TypeScript/JSX conversion or minification. ([Go Packages][1])

## Mental model

Use this mapping:

| Need                                | Use                                        |
| ----------------------------------- | ------------------------------------------ |
| One `.ts` file, no imports          | `api.Transform`                            |
| `.ts` file with `import` / `export` | `api.Build` with `Bundle: true`            |
| Return JS in memory                 | `Write: false`                             |
| Run result in goja                  | `vm.RunScript(...)` or `goja.Compile(...)` |

esbuild has built-in TypeScript parsing for `.ts`, `.tsx`, `.mts`, and `.cts` files, but it strips types rather than type-checking. For type checking, run `tsc --noEmit` separately during development or CI. ([esbuild][2])

## Simple case: transpile one TypeScript string

This is fully in-process and self-contained.

```go
package main

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/evanw/esbuild/pkg/api"
)

func runTS(vm *goja.Runtime, filename string, source string) (goja.Value, error) {
	result := api.Transform(source, api.TransformOptions{
		Loader:     api.LoaderTS,
		Sourcefile: filename,

		// Safer target for goja than modern JS output.
		Target: api.ES2015,
		Format: api.FormatIIFE,
	})

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("esbuild transform failed: %s", result.Errors[0].Text)
	}

	return vm.RunScript(filename, string(result.Code))
}

func main() {
	vm := goja.New()

	_, err := runTS(vm, "example.ts", `
		type User = { name: string }

		const user: User = { name: "Manuel" }
		globalThis.result = "hello " + user.name
	`)
	if err != nil {
		panic(err)
	}

	fmt.Println(vm.Get("result"))
}
```

`Transform` is the right API when you already have the file contents and do not need esbuild to resolve imports.

## File with imports: bundle first, then run in goja

Use `api.Build` when TypeScript imports other files. The Build API follows entry points and dependencies. ([Go Packages][1])

```go
package scripts

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/evanw/esbuild/pkg/api"
)

func bundleTS(entry string) ([]byte, error) {
	result := api.Build(api.BuildOptions{
		EntryPoints: []string{entry},
		Bundle:     true,

		// Return output in memory instead of writing files.
		Write: false,

		// Good shape for goja: one executable script.
		Format: api.FormatIIFE,

		// Adjust based on what your goja version supports.
		Target: api.ES2015,

		Platform: api.PlatformNeutral,
	})

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("esbuild build failed: %s", result.Errors[0].Text)
	}
	if len(result.OutputFiles) == 0 {
		return nil, fmt.Errorf("esbuild produced no output")
	}

	return result.OutputFiles[0].Contents, nil
}

func RunTSEntry(vm *goja.Runtime, entry string) error {
	js, err := bundleTS(entry)
	if err != nil {
		return err
	}

	_, err = vm.RunScript(entry, string(js))
	return err
}
```

The Go Build API can either write output to disk or return in-memory output files. The docs note that the Go API does **not** write by default, and `result.OutputFiles` contains the generated content when using in-memory output. ([esbuild][3])

## Fully self-contained binary with embedded TypeScript

You have two choices.

### Option A: embed TypeScript and transpile at runtime

Good if users may edit scripts before compilation is not desired, or you want runtime plugin behavior.

```go
package main

import (
	"embed"
	"fmt"

	"github.com/dop251/goja"
	"github.com/evanw/esbuild/pkg/api"
)

//go:embed scripts/*.ts
var scriptsFS embed.FS

func main() {
	src, err := scriptsFS.ReadFile("scripts/main.ts")
	if err != nil {
		panic(err)
	}

	result := api.Transform(string(src), api.TransformOptions{
		Loader:     api.LoaderTS,
		Sourcefile: "scripts/main.ts",
		Target:     api.ES2015,
		Format:     api.FormatIIFE,
	})

	if len(result.Errors) > 0 {
		panic(result.Errors[0].Text)
	}

	vm := goja.New()

	_, err = vm.RunScript("scripts/main.ts", string(result.Code))
	if err != nil {
		panic(err)
	}

	fmt.Println("done")
}
```

This produces one Go binary containing:

```text
your Go app
goja
esbuild
embedded .ts files
```

No Node. No npm. No external esbuild binary.

### Option B: precompile TypeScript at Go build time and embed JavaScript

This is usually better for production:

```bash
esbuild scripts/main.ts --bundle --format=iife --target=es2015 --outfile=internal/assets/main.js
```

Then:

```go
//go:embed internal/assets/main.js
var mainJS string
```

At runtime you only execute JS:

```go
vm := goja.New()
_, err := vm.RunScript("main.js", mainJS)
```

This gives a smaller runtime surface: no TypeScript compilation step while your app runs. But it does require esbuild during your build process.

## Important caveats

`goja` is a pure-Go ECMAScript engine, but it is not Node.js. It implements ECMAScript 5.1+ rather than the full modern Node/browser runtime. ([GitHub][4]) So avoid emitting very modern JavaScript unless you know your `goja` version supports it. In practice, start with:

```go
Target: api.ES2015
```

Then lower or raise the target based on actual tests.

Also, TypeScript imports such as this:

```ts
import fs from "fs"
```

will not magically work in `goja`. esbuild can bundle files, but `goja` will not provide Node built-ins unless you implement/provide them. For embedded scripting, prefer application-level APIs:

```go
vm.Set("app", map[string]any{
	"log": func(s string) {
		fmt.Println(s)
	},
})
```

Then TypeScript can call:

```ts
app.log("hello")
```

## Recommendation for your case

For a `goja` scripting system, I’d use this structure:

```text
internal/scriptengine/
  engine.go       // goja setup
  compile.go      // esbuild Transform/Build wrapper
scripts/
  main.ts
  helpers.ts
```

During development, allow `.ts` files on disk:

```go
js, err := bundleTS("scripts/main.ts")
```

For release builds, either embed the `.ts` files and transpile at runtime, or pre-bundle to JS and embed that. The most robust production setup is:

```text
TypeScript source
    ↓ esbuild during build/CI
single JS bundle
    ↓ go:embed
single Go binary
    ↓ goja.RunScript
```

That keeps your shipped binary self-contained while avoiding runtime TypeScript compilation costs.

[1]: https://pkg.go.dev/github.com/evanw/esbuild/pkg/api "api package - github.com/evanw/esbuild/pkg/api - Go Packages"
[2]: https://esbuild.github.io/content-types/ "esbuild - Content Types"
[3]: https://esbuild.github.io/api/ "esbuild - API"
[4]: https://github.com/dop251/goja?utm_source=chatgpt.com "dop251/goja: ECMAScript/JavaScript engine in pure Go"

