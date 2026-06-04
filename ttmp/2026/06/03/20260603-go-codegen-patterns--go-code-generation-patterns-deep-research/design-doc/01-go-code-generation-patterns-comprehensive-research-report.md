---
Title: 'Go Code Generation Patterns: Comprehensive Research Report'
Ticket: 20260603-go-codegen-patterns
Status: active
Topics:
    - go
    - code-generation
    - patterns
    - metaprogramming
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/06/03/20260603-go-codegen-patterns--go-code-generation-patterns-deep-research/sources/articles/01-go-wiki-generate-tools.md
      Note: Canonical list of go generate tools from Go Wiki
    - Path: ttmp/2026/06/03/20260603-go-codegen-patterns--go-code-generation-patterns-deep-research/sources/articles/04-eli-bendersky-ast.md
      Note: Deep AST rewriting tutorial with astutil
    - Path: ttmp/2026/06/03/20260603-go-codegen-patterns--go-code-generation-patterns-deep-research/sources/articles/07-xcaddy-builder-go.md
      Note: xcaddy builder.go API and Build method
    - Path: ttmp/2026/06/03/20260603-go-codegen-patterns--go-code-generation-patterns-deep-research/sources/articles/10-xcaddy-environment.go.md
      Note: xcaddy environment.go showing template-based binary builder pattern
    - Path: ttmp/2026/06/03/20260603-go-codegen-patterns--go-code-generation-patterns-deep-research/sources/articles/17-stringer-source.md
      Note: Canonical stringer generator source for go generate pattern
ExternalSources: []
Summary: ""
LastUpdated: 2026-06-03T17:20:00Z
WhatFor: ""
WhenToUse: ""
---


# Go Code Generation Patterns: Comprehensive Research Report

## Executive Summary

Go does not have macros, a preprocessor, or runtime metaprogramming in the style of Lisp or Ruby. Instead, the Go ecosystem has converged on a family of **compile-time code generation** patterns that are robust, debuggable, and version-controllable. This document surveys the widely accepted patterns, explains their mechanics, and provides concrete API references and pseudocode so that a new intern can understand and choose among them.

The six dominant pattern categories are:

1. **`go:generate` + CLI tools** — One-shot generators triggered by `go generate` (e.g., `stringer`, `mockgen`).
2. **`text/template` source generation** — Emitting `.go` files from templates (e.g., `xcaddy`, `templ`, many internal tools).
3. **`go/ast` AST-based generation** — Parsing existing Go source, transforming the syntax tree, and re-emitting code.
4. **Schema-first generators** — Protocol Buffers, OpenAPI, JSON Schema, SQL DDL → Go structs + methods.
5. **Compile-time plugin/binary builders** — Programmatically creating a `main.go` + `go.mod` in a temp directory, importing selected packages, and running `go build` to produce a custom binary (exemplified by `xcaddy`).
6. **Compile-time dependency injection** — `google/wire` and similar tools that generate wiring code from provider graphs.

Each pattern trades off between **expressiveness**, **tooling complexity**, **debuggability**, and **reviewability**. The report includes decision records for choosing among them.

---

## Problem Statement and Scope

### What problem are we solving?

As Go projects grow, developers repeatedly encounter the following needs:

- **Boilerplate reduction**: Writing `String()`, `MarshalJSON()`, getters, or error-wrapping code for dozens of types.
- **Type-safe APIs**: Ensuring that protocol contracts (protobuf, OpenAPI, SQL) are reflected in Go types without manual drift.
- **Plugin/binary composition**: Building a single binary that includes a dynamic set of packages (e.g., a web server with optional plugins) without maintaining a separate `main.go` for every permutation.
- **Compile-time wiring**: Assembling dependency graphs at build time rather than using runtime reflection and service locators.

### What is in scope?

- Patterns that generate Go source code or binaries from Go source, templates, schemas, or metadata.
- Tools and libraries that are widely used, actively maintained, and representative of an accepted pattern.
- The mechanics of writing custom generators (both template-based and AST-based).

### What is out of scope?

- Cgo and foreign-function interface generation (e.g., SWIG).
- Runtime code generation (e.g., `plugin` package, `yaegi`, `gomacro`).
- Non-Go generators that happen to produce Go (e.g., purely external OpenAPI generators not written in Go).

---

## Current-State Architecture: Pattern Taxonomy

Go code generation tools cluster into a small number of architectural families. Understanding these families is more important than memorizing individual tools, because new tools are created constantly but they reuse the same mechanics.

| Pattern Family | Input | Output | Key Libraries / Tools | Typical Use Case |
|---|---|---|---|---|
| **Directive + CLI** | Go source comments (`//go:generate`) | `.go` files adjacent to source | `stringer`, `mockgen`, `enumer`, `sqlc` | Generate methods for enums, mocks, DB queries |
| **Template Source Gen** | Metadata / config files | `.go` files via `text/template` | `xcaddy` (main.go), `templ`, internal tools | Emitting struct suites, wiring files, or custom binaries |
| **AST Rewrite** | Existing `.go` source | Modified `.go` source | `go/ast`, `go/parser`, `go/printer`, `astutil` | Adding imports, injecting methods, refactoring |
| **Schema-First** | `.proto`, `.json`, `.sql`, `.yaml` | `.go` types + stubs | `protoc-gen-go`, `oapi-codegen`, `go-jsonschema`, `sqlc` | Contract-first API / DB design |
| **Binary Builder** | Module list + version spec | Custom compiled binary | `xcaddy`, `ko`, custom builders | Composing a binary from optional plugins |
| **Compile-Time DI** | Provider graph declarations | `wire_gen.go` wiring code | `google/wire` | Eliminating hand-written constructor wiring |

### API References: Core Go Packages for Code Generation

```go
// go/token — Source positions and file sets
import "go/token"
// go/ast — Abstract syntax tree nodes
import "go/ast"
// go/parser — Parse source files into AST
import "go/parser"
// go/printer — Format AST back to source
import "go/printer"
// text/template — Emit source from templates
import "text/template"
// go/format — Go-aware source formatter (uses printer + tabwriter)
import "go/format"
// golang.org/x/tools/go/ast/astutil — AST helpers (AddImport, etc.)
import "golang.org/x/tools/go/ast/astutil"
// golang.org/x/tools/go/packages — Load packages with type information
import "golang.org/x/tools/go/packages"
```

---

## Pattern 1: `go:generate` + CLI Tools

### How it works

`go generate` is a standard Go command (not part of `go build`) that scans source files for lines matching:

```go
//go:generate <command> [arguments...]
```

When you run `go generate ./...`, Go executes each such command in the directory containing the directive. The command can be any executable: a tool from `golang.org/x/tools/cmd/...`, a vendored binary, or a `go run` invocation.

### Example: `stringer`

```go
package painttype

//go:generate go run golang.org/x/tools/cmd/stringer@latest -type=Paint

type Paint int

const (
    Acrylic Paint = iota
    Oil
    Watercolor
)
```

Running `go generate` produces `paint_string.go`:

```go
func (i Paint) String() string { /* ... */ }
```

### Internals of `stringer` (simplified)

`stringer` uses `golang.org/x/tools/go/packages` to load and type-check the target package, finds constants of the named type, and emits a file. It writes the output by hand (fmt.Fprintf) rather than using a template, which is common for small generators.

```go
// pseudocode: stringer's core logic
pkgs, _ := packages.Load(&packages.Config{Mode: packages.NeedTypes}, ".")
for _, pkg := range pkgs {
    obj := pkg.Types.Scope().Lookup("Paint")
    if obj == nil { continue }
    typ := obj.Type().Underlying()
    // collect named constants of this type
    // emit fmt.Fprintf(out, "func (i %s) String() string { ... }", typeName)
}
```

### Decision: When to use this pattern

- **Context**: You need to generate a small, localized piece of code (one method, one mock file) next to the source that defines the input.
- **Options considered**: Template files in a separate directory; AST rewriting.
- **Decision**: Use `go:generate` when the output is tightly coupled to a specific type/file and should live next to the input.
- **Rationale**: Version-controllable, reviewable, and the directive is self-documenting.
- **Consequences**: CI must run `go generate` and fail if `git diff` shows drift.
- **Status**: accepted

---

## Pattern 2: `text/template` Source Generation

### How it works

Go's built-in `text/template` (and `html/template`) engine lets you define a template string or file, execute it against a data structure, and write the result to a `.go` file. This is the workhorse for medium-complexity generators that emit many similar declarations.

### Example: Generating enum helpers from a YAML spec

Imagine a YAML file defining all error codes:

```yaml
codes:
  - name: NotFound
    http: 404
  - name: Conflict
    http: 409
```

A generator reads the YAML and executes a template:

```go
package main

import (
    "os"
    "text/template"
)

const codeTmpl = `package errors

{{range .Codes}}
const {{.Name}} = {{.HTTP}}
{{end}}

func HTTPStatusFor(code int) int {
    switch code {
    {{range .Codes}}case {{.Name}}: return {{.HTTP}}
    {{end}}default: return 500
    }
}
`

func main() {
    type Code struct { Name string; HTTP int }
    data := struct{ Codes []Code }{
        Codes: []Code{{Name: "NotFound", HTTP: 404}, {Name: "Conflict", HTTP: 409}},
    }
    t := template.Must(template.New("codes").Parse(codeTmpl))
    f, _ := os.Create("codes_gen.go")
    defer f.Close()
    t.Execute(f, data)
}
```

### Important: always pipe through `go/format`

Templates are easy to misindent. The standard practice is:

```go
import "go/format"

var buf bytes.Buffer
t.Execute(&buf, data)
formatted, err := format.Source(buf.Bytes())
if err != nil { /* ... */ }
os.WriteFile("output.go", formatted, 0644)
```

### Decision: When to use this pattern

- **Context**: You have structured metadata (YAML, JSON, SQL, protobuf descriptors) and want to emit many related declarations.
- **Options considered**: AST construction (more verbose but type-safe); hand-writing strings (brittle).
- **Decision**: Use `text/template` when the output is mostly declarative and the shape is regular.
- **Rationale**: Templates are readable by non-Go-experts; easy to iterate.
- **Consequences**: Template syntax errors surface at generator runtime, not compile time. Always format output.
- **Status**: accepted

---

## Pattern 3: `go/ast` AST-Based Generation

### How it works

Go provides a full parser in the standard library. You can:

1. Parse a file with `parser.ParseFile`.
2. Traverse or mutate the `ast.File`.
3. Print it back with `go/printer` or `go/format`.

### Example: Parse a file and list all function names

```go
package main

import (
    "go/ast"
    "go/parser"
    "go/token"
    "fmt"
)

func main() {
    fset := token.NewFileSet()
    node, err := parser.ParseFile(fset, "input.go", nil, parser.ParseComments)
    if err != nil { panic(err) }

    ast.Inspect(node, func(n ast.Node) bool {
        if fn, ok := n.(*ast.FuncDecl); ok {
            fmt.Println(fn.Name.Name)
        }
        return true
    })
}
```

### Example: Add an import with astutil

```go
import "golang.org/x/tools/go/ast/astutil"

// After parsing into 'f' (an *ast.File)
astutil.AddImport(fset, f, "context")
```

### Example: Generate a function declaration from scratch

```go
fn := &ast.FuncDecl{
    Name: ast.NewIdent("NewService"),
    Type: &ast.FuncType{
        Params: &ast.FieldList{
            List: []*ast.Field{
                {Names: []*ast.Ident{ast.NewIdent("cfg")}, Type: ast.NewIdent("Config")},
            },
        },
        Results: &ast.FieldList{
            List: []*ast.Field{
                {Type: ast.NewIdent("*Service")},
            },
        },
    },
    Body: &ast.BlockStmt{
        List: []ast.Stmt{
            &ast.ReturnStmt{
                Results: []ast.Expr{
                    &ast.UnaryExpr{Op: token.AND, X: &ast.CompositeLit{
                        Type: ast.NewIdent("Service"),
                        Elts: []ast.Expr{ast.NewIdent("cfg")},
                    }},
                },
            },
        },
    },
}
```

This is verbose, which is why many AST-based tools use a **hybrid** approach: parse existing code to extract metadata, then feed that metadata into a `text/template` to emit the new file.

### Decision: When to use this pattern

- **Context**: You need to modify existing source in place, preserve comments and formatting, or you need type-checked accuracy.
- **Options considered**: Textual search/replace (brittle); full template rewrite.
- **Decision**: Use AST when you are rewriting existing `.go` files rather than generating from external metadata.
- **Rationale**: AST preserves structure and formatting identity.
- **Consequences**: Code is more verbose than templates. `x/tools/go/packages` is needed for type information.
- **Status**: accepted

---

## Pattern 4: Schema-First Generators

### How it works

A schema-first generator treats a non-Go contract file (`.proto`, `.json`, `.sql`, `.yaml`) as the source of truth and emits Go code that satisfies the contract. This is the most widely adopted pattern in distributed systems and API design.

### 4a. Protocol Buffers (`protoc-gen-go`)

Protobuf is the canonical example. A `.proto` file defines messages and services. The `protoc` compiler delegates to a Go plugin (`protoc-gen-go`), which emits `.pb.go` files.

```protobuf
syntax = "proto3";
package example;

message User {
    string id = 1;
    string email = 2;
}
```

Generated Go (simplified):

```go
type User struct {
    state         protoimpl.MessageState
    sizeCache     protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields
    Id    string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
    Email string `protobuf:"bytes,2,opt,name=email,proto3" json:"email,omitempty"`
}
// ... accessor methods, ProtoMessage(), Reset(), String(), etc.
```

### 4b. OpenAPI (`oapi-codegen`)

OpenAPI specs describe HTTP APIs. `oapi-codegen` generates:

- Go struct models from schema definitions.
- Server interface stubs.
- Client SDK methods.

```yaml
paths:
  /users/{id}:
    get:
      operationId: getUser
      parameters:
        - name: id
          in: path
          schema:
            type: string
```

### 4c. JSON Schema → Go (`go-jsonschema`)

Tools like `github.com/atombender/go-jsonschema` or `omissis/go-jsonschema` read JSON Schema and emit Go structs with `json` tags and unmarshaling validation.

### 4d. SQL → Go (`sqlc`)

`sqlc` is unique: you write SQL queries in `.sql` files, and it generates type-safe Go functions that execute those queries against `database/sql` or `pgx`.

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = $1;
```

Generated:

```go
func (q *Queries) GetUser(ctx context.Context, id int64) (User, error)
```

### Decision: When to use schema-first

- **Context**: Your system has an external contract (API, DB, wire format) that is already defined or should be defined independently of Go.
- **Options considered**: Code-first ORMs; hand-written structs.
- **Decision**: Use schema-first when interoperability, multi-language support, or contract review is important.
- **Rationale**: The schema becomes a single source of truth; generated code eliminates drift.
- **Consequences**: Toolchain complexity (must install `protoc`, `sqlc`, etc. in CI). Generated code should be checked in.
- **Status**: accepted

---

## Pattern 5: Compile-Time Plugin / Custom Binary Builder

### How it works

This is the pattern the user explicitly asked about, exemplified by `xcaddy`. Instead of runtime plugins (`.so`/`.dll`), the tool **generates a temporary Go module** that imports a configurable set of packages, then compiles it.

The phases are:

1. **Define inputs**: Which packages (with versions) should be compiled in.
2. **Generate `main.go`**: Use a `text/template` that imports all selected packages as blank imports (`_`).
3. **Generate `go.mod` / bootstrap it**: Run `go mod init <name>` in a temp directory.
4. **Pin versions**: Run `go get <pkg>@<version>` for each dependency.
5. **Apply replacements**: Run `go mod edit -replace` for local development overrides.
6. **Build**: Run `go build -o <output>`.
7. **Cleanup**: Optionally delete the temp directory.

### Why blank imports?

In Go, `_ "package/path"` causes the package's `init()` functions to run. If the package registers itself with a central registry (e.g., `caddy.RegisterModule`), the binary now includes that plugin without any explicit usage in `main()`.

### Walkthrough: xcaddy's builder (simplified)

xcaddy (from `builder.go` and `environment.go`) defines:

```go
type Dependency struct {
    PackagePath string // e.g. "github.com/caddyserver/ntlm-transport"
    Version     string // e.g. "v0.1.1"
}

type Builder struct {
    CaddyVersion string
    Plugins      []Dependency
    Replacements []Replace
    // ... build flags, timeouts, etc.
}
```

It then defines a template for `main.go`:

```go
const mainModuleTemplate = `package main

import (
    caddycmd "{{.CaddyModule}}/cmd"
    _ "{{.CaddyModule}}/modules/standard"
    {{- range .Plugins}}
    _ "{{.}}"
    {{- end}}
)

func main() {
    caddycmd.Main()
}
`
```

And the build flow (condensed):

```go
func (b Builder) Build(ctx context.Context, outputFile string) error {
    // 1. Determine module paths with semantic import versioning (SIV)
    caddyModulePath := versionedModulePath(defaultCaddyModulePath, b.CaddyVersion)

    // 2. Prepare template context
    tplCtx := goModTemplateContext{
        CaddyModule: caddyModulePath,
        Plugins:     pluginPaths,
    }

    // 3. Render main.go into a temp folder
    tpl := template.Must(template.New("main").Parse(mainModuleTemplate))
    var buf bytes.Buffer
    tpl.Execute(&buf, tplCtx)
    tempFolder, _ := newTempFolder()
    os.WriteFile(filepath.Join(tempFolder, "main.go"), buf.Bytes(), 0644)

    // 4. Initialize Go module
    runCmd(tempFolder, "go", "mod", "init", "caddy")

    // 5. Apply replacements (for local development)
    for _, r := range b.Replacements {
        runCmd(tempFolder, "go", "mod", "edit", "-replace", fmt.Sprintf("%s=%s", r.Old, r.New))
    }

    // 6. Pin plugin versions (prevents accidental upgrades)
    for _, p := range b.Plugins {
        runCmd(tempFolder, "go", "get", "-v", p.PackagePath+"@"+p.Version)
    }

    // 7. Build the binary
    runCmd(tempFolder, "go", "build", "-o", outputFile, ".")
    return nil
}
```

### Why this pattern is powerful

- **Static compilation**: The resulting binary is a normal Go binary. No runtime linking issues, no `.so` version mismatches.
- **Cross-compilation**: Works with `GOOS`/`GOARCH` because it's just `go build`.
- **Registry pattern**: Any package that self-registers in `init()` can be included by a blank import.
- **Version pinning**: Each plugin can be pinned to a specific version, avoiding "dependency hell" in a monolithic codebase.

### Diagram: xcaddy-style build flow

```
+-------------+     +--------------------+     +------------------+
|  User CLI   | --> | Builder (struct)   | --> |  text/template   |
|  inputs:    |     | - CaddyVersion     |     |  main.go template|
|  plugins,   |     | - Plugins[]        |     |                  |
|  versions,  |     | - Replacements[]   |     |                  |
|  output     |     | - BuildFlags       |     |                  |
+-------------+     +--------------------+     +------------------+
                                                          |
                                                          v
                                            +-----------------------+
                                            |  Temp folder          |
                                            |  - main.go            |
                                            |  - go.mod (init)      |
                                            |  - go.sum (get)       |
                                            +-----------------------+
                                                          |
                                                          v
                                            +-----------------------+
                                            |  os/exec: go build    |
                                            |  -o <output>          |
                                            +-----------------------+
                                                          |
                                                          v
                                            +-----------------------+
                                            |  Custom binary        |
                                            |  with plugins linked  |
                                            +-----------------------+
```

### Decision: When to use this pattern

- **Context**: You need to ship a single binary that can include a variable set of plugins/modules without maintaining a separate repo per combination.
- **Options considered**: Go `plugin` package (runtime `.so` loading); static monorepo with all plugins imported always.
- **Decision**: Use custom binary builders when the set of plugins is large or user-configurable, and runtime `.so` is unacceptable (portability, security, or deployment constraints).
- **Rationale**: Go's `plugin` package is unsupported on Windows and has cross-version fragility. Static compilation is the Go way.
- **Consequences**: The build process requires network access (for `go get`). Build times are longer than a normal `go build` because dependencies are fetched fresh. CI must cache module downloads.
- **Status**: accepted

---

## Pattern 6: Compile-Time Dependency Injection (Wire)

### How it works

`google/wire` is a code generator for dependency injection. You write "provider" functions and "injector" signatures in a `wire.go` file (tagged with `//go:build wireinject`), then run `wire` to generate `wire_gen.go`.

```go
// wire.go
//go:build wireinject

func InitializeApp(cfg Config) (*App, error) {
    wire.Build(NewDB, NewCache, NewHandler, NewApp)
    return nil, nil // stub; wire ignores this body
}
```

Generated `wire_gen.go`:

```go
func InitializeApp(cfg Config) (*App, error) {
    db, err := NewDB(cfg.DatabaseURL)
    if err != nil { return nil, err }
    cache := NewCache(db)
    handler := NewHandler(cache)
    app := NewApp(handler)
    return app, nil
}
```

### Decision: When to use this pattern

- **Context**: You have a deep object graph and want to avoid hand-writing constructor wiring.
- **Options considered**: Manual constructors; runtime DI containers (e.g., `uber-go/dig`).
- **Decision**: Use Wire when you want compile-time failure for missing dependencies and zero runtime reflection cost.
- **Rationale**: Wire-generated code is plain Go; no magic at runtime.
- **Consequences**: Generated file must be checked in. If providers change, re-run `wire`. Slight learning curve for the `wire.Build` DSL.
- **Status**: accepted

---

## Decision Records Summary

### DR-1: Template vs AST for custom generators

- **Context**: Building an internal tool that emits Go structs from a company-specific schema format.
- **Options considered**: `text/template`; `go/ast` construction; hybrid (parse schema, emit via template).
- **Decision**: Hybrid. Parse schema into Go structs, then feed into `text/template` with `go/format` post-processing.
- **Rationale**: Template is readable and maintainable by the whole team. AST is overkill for purely additive generation. Pure string concatenation is too brittle.
- **Consequences**: We must ship a small CLI that reads the schema, runs the template, and formats the output.
- **Status**: accepted

### DR-2: Blank-import registration vs explicit wiring

- **Context**: Plugin system where packages self-register.
- **Options considered**: Blank import (`_ "plugin"`) with `init()` registration; explicit `RegisterPlugin()` call in `main()`; Wire-style DI graph.
- **Decision**: Blank import for optional modules that register into a global typed registry (like Caddy). Explicit wiring for required application core.
- **Rationale**: Blank import keeps `main.go` short and plugin list easily templated. Explicit wiring avoids hidden side effects in core startup.
- **Consequences**: Global registry needs thread-safe registration and clear ordering rules.
- **Status**: accepted

### DR-3: Checked-in generated code vs CI-only generation

- **Context**: Should `.pb.go`, `wire_gen.go`, or `*_gen.go` files be committed?
- **Options considered**: Always generate in CI (clean repo); always check in generated code.
- **Decision**: Check in generated code for public libraries and for any file a developer might need to read (protobuf, wire). Allow CI-only for purely internal tooling outputs that are never imported.
- **Rationale**: `go get` should work without running the generator. Reviewers must see generated changes. 
- **Consequences**: Repo size grows slightly. Need a CI check that `go generate` + diff produces no changes.
- **Status**: accepted

---

## Implementation Plan (for a team adopting these patterns)

### Phase 1: Tooling & CI hygiene

1. Pin generator versions (e.g., `//go:generate go run github.com/.../cmd/stringer@v0.15.0`).
2. Add a CI step `go generate ./... && git diff --exit-code` to prevent drift.
3. Document which files are hand-written and which are generated (a `// Code generated by ... DO NOT EDIT.` header is standard).

### Phase 2: Pick one pattern per use case

| Use case | Pattern family | Tool example |
|---|---|---|
| Enum string methods | `go:generate` | `stringer` |
| Mock interfaces | `go:generate` | `mockgen` or `mockery` |
| REST API contract | Schema-first | `oapi-codegen` |
| RPC services | Schema-first | `protoc-gen-go` + `connect-go` |
| SQL queries | Schema-first | `sqlc` |
| Custom binary with plugins | Binary builder | `xcaddy`-style temp module |
| DI wiring | Compile-time DI | `wire` |
| Boilerplate reduction from YAML/JSON | Template gen | Hand-written template + `go/format` |
| Source refactoring / migration | AST rewrite | `go/ast` + `astutil` |

### Phase 3: Build a custom binary builder (if needed)

If the project needs to compose a binary from optional packages, implement the xcaddy pattern:

1. Define a `Builder` struct with slice fields for plugins and versions.
2. Write a `main.go` template that blank-imports each plugin.
3. Create a temp folder, `go mod init`, run `go mod edit -replace`, and `go get` pinned versions.
4. Run `go build -o <output>`.
5. Return the output path and optionally clean up the temp folder.

### Phase 4: Testing & validation

- **Unit tests for generators**: Feed known input schemas, generate code, `format.Source()` must not error, and the generated code must `go test` successfully.
- **Golden file tests**: Check in expected output and use `go test` diff against it.
- **Integration tests for binary builders**: Build a minimal binary with oneplugin, run it, assert it starts and registers the plugin.

---

## Risks, Alternatives, and Open Questions

### Risks

1. **Generator drift**: If `go generate` is not enforced in CI, hand-edited generated code will diverge from the source of truth.
2. **Tool dependency rot**: Pinning generator versions prevents breakage, but also means security patches may lag.
3. **Temp-folder binary builders are slow**: Each build fetches modules. Network latency and module download time dominate.
4. **Build reproducibility**: `go mod edit -replace` for local paths can produce binaries that are not reproducible on another machine.

### Alternatives

- **Runtime plugins (`plugin` package)**: Supported only on Linux. Unsuitable for Windows/macOS deployments.
- **WASM plugins**: Sandboxed but adds runtime complexity. Good for security-critical extensibility.
- **Scripting languages (Lua/JS) embedded in Go**: No recompilation needed, but performance and type safety tradeoffs.

### Open Questions

1. Should the generator be a standalone CLI or a `go:generate` directive calling `go run`?
2. How do we version-control the generator itself when it lives in the same repo as the generated code?
3. For binary builders, can `GOMODCACHE` reuse and caching make builds fast enough for interactive CLI use?
4. Are there emerging Go 1.24 `go tool` features that change the recommendation for generator installation?

---

## References

### External Resources (saved in ticket `sources/`)

| File | Description |
|---|---|
| `sources/articles/01-go-wiki-generate-tools.md` | Go Wiki list of `go generate` tools |
| `sources/articles/02-dolthub-go-generate.md` | DoltHub 2025 blog on `go generate` usage |
| `sources/articles/03-metaprogramming-go-devto.md` | dev.to post on metaprogramming with `go/ast` |
| `sources/articles/04-eli-bendersky-ast.md` | Eli Bendersky on AST rewriting |
| `sources/articles/05-pluggable-library-go.md` | Jose Sitanggang on pluggable libraries |
| `sources/articles/06-caddy-extending.md` | Caddy plugin extension docs |
| `sources/articles/07-xcaddy-github.md` | xcaddy README |
| `sources/articles/07-xcaddy-readme.md` | xcaddy raw README |
| `sources/articles/07-xcaddy-builder-go.md` | xcaddy `builder.go` source |
| `sources/articles/08-zakaria-ast.md` | AST function comments tutorial |
| `sources/articles/09-go-cookbook-ast.md` | AST manipulation cookbook |
| `sources/articles/10-xcaddy-environment.go.md` | xcaddy `environment.go` source |
| `sources/articles/11-xcaddy-cmd-main.md` | xcaddy `cmd/xcaddy/main.go` |
| `sources/articles/12-xcaddy-environment_test.md` | xcaddy environment tests |
| `sources/articles/13-masterminds-sprig.md` | Sprig template functions |
| `sources/articles/14-google-wire-readme.md` | Google Wire README |
| `sources/articles/17-stringer-source.md` | `golang.org/x/tools/cmd/stringer` source |

### Key APIs and Packages

- `go/token` — File sets and positions.
- `go/ast` — Syntax tree nodes.
- `go/parser` — Source file parser.
- `go/printer` — AST back to text.
- `go/format` — Canonical formatting.
- `text/template` — Template execution.
- `os/exec` — Running `go build`, `go get`, `go mod init`.
- `golang.org/x/tools/go/ast/astutil` — AST helper functions.
- `golang.org/x/tools/go/packages` — Package loading with type info.

### Related Tools

- `stringer` — Enum `String()` methods.
- `enumer` — Enum marshaling for json/yaml/sql/text.
- `mockgen` / `mockery` — Mock generation.
- `protoc-gen-go` — Protocol Buffers → Go.
- `oapi-codegen` — OpenAPI → Go server/client/types.
- `sqlc` — SQL queries → Go types + functions.
- `wire` — DI wiring code generation.
- `go-jsonschema` — JSON Schema → Go structs.
- `xcaddy` — Custom binary builder for Caddy.
