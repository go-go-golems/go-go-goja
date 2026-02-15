# go-go-goja

`go-go-goja` is a Go + JavaScript tooling workspace centered on:
- goja native module execution (`cmd/repl`, `modules/*`, `engine`)
- JavaScript parsing/indexing/completion (`pkg/jsparse`)
- inspector domain primitives (`pkg/inspector/*`)
- user-facing inspector orchestration (`pkg/inspectorapi`)
- terminal inspector adapters (`cmd/inspector`, `cmd/smalltalk-inspector`)

## Quick Start

From repo root:

```bash
go run ./cmd/repl testdata/hello.js
go run ./cmd/inspector ../goja/testdata/sample.js
go run ./cmd/smalltalk-inspector ../goja/testdata/sample.js
```

## Project Layout

```text
go-go-goja/
├── cmd/
│   ├── repl/                  # goja runtime REPL + script runner
│   ├── inspector/             # jsparse-oriented AST inspector example
│   └── smalltalk-inspector/   # Smalltalk-style object inspector UI
├── engine/                    # runtime bootstrap helpers (goja + require)
├── modules/                   # native modules exposed through require()
├── pkg/
│   ├── jsparse/               # low-level parser/index/resolution/completion
│   ├── inspector/             # reusable inspector core/runtime/navigation/tree
│   ├── inspectorapi/          # high-level service facade for adapters
│   └── doc/                   # glazed help pages
├── testdata/
└── ttmp/                      # ticket docs, plans, diaries, reviews
```

## Architecture Boundaries

Use `pkg/jsparse` when you need parser-level control:
- AST indexing and source mapping
- lexical binding resolution
- completion context extraction and candidate resolution

Use `pkg/inspectorapi` when you need adapter-facing workflows:
- document/session lifecycle
- globals/members/jump orchestration
- runtime merge helpers
- tree/source sync wrappers

Command adapters (`cmd/*`) should stay focused on input, key handling, and rendering. Business orchestration should live in `pkg/inspectorapi` and lower packages.

## Native Module Authoring

Add a module under `modules/<name>/` and register it during `init()`:

```go
package uuidmod

import (
    "github.com/dop251/goja"
    "github.com/go-go-golems/go-go-goja/modules"
    "github.com/google/uuid"
)

type Module struct{}

var _ modules.NativeModule = (*Module)(nil)

func (m *Module) Name() string { return "uuid" }

func (m *Module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)
    exports.Set("v4", func() string { return uuid.NewString() })
}

func init() { modules.Register(&Module{}) }
```

Ensure the package is imported once (usually in `engine/runtime.go`) so the registration `init()` runs.

## Testing

Run the full suite:

```bash
go test ./... -count=1
```

Focused checks:

```bash
go test ./pkg/jsparse/... -count=1
go test ./pkg/inspectorapi/... -count=1
go test ./cmd/smalltalk-inspector/... -count=1
```

## Documentation

Glazed help pages live in `pkg/doc`. Useful entry points:

```bash
glaze help jsparse-framework-reference
glaze help inspectorapi-hybrid-service-guide
glaze help inspector-example-user-guide
```

## License

MIT (see `LICENSE`).

[dop251/goja]: https://github.com/dop251/goja
[goja_nodejs/require]: https://pkg.go.dev/github.com/dop251/goja_nodejs/require
