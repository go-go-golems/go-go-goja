---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: engine/module_specs.go
      Note: DefaultRegistryModules spec
    - Path: engine/runtime.go
      Note: Blank imports for default module registration
    - Path: modules/common.go
      Note: NativeModule interface and DefaultRegistry
    - Path: modules/fs/fs.go
      Note: Simplest module pattern to follow
    - Path: modules/timer/timer_test.go
      Note: Integration test pattern
    - Path: pkg/doc/04-repl-usage.md
      Note: REPL usage doc with yaml examples
    - Path: pkg/doc/16-yaml-module.md
      Note: Glazed help entry
    - Path: pkg/tsgen/spec/helpers.go
      Note: TypeScript declaration helpers
    - Path: testdata/yaml.js
      Note: Example script
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---



# YAML Primitive Module: Analysis, Design and Implementation Guide

## Executive Summary

This document provides a comprehensive, intern-friendly guide for adding **YAML primitive support** to go-go-goja as a native module named `yaml`. The module will be **enabled by default** in all runtime configurations that use `engine.DefaultRegistryModules()`, meaning every go-go-goja runtime will be able to `require("yaml")` without additional setup.

The module exposes three core functions:

- `yaml.parse(input)` — parse a YAML string into a JavaScript value
- `yaml.stringify(value, options?)` — serialize a JavaScript value into a YAML string
- `yaml.validate(input)` — validate YAML syntax without full parsing

This guide walks through every subsystem you need to understand: the go-go-goja module registry, the native module interface, the engine factory pipeline, TypeScript declaration generation, testing conventions, and the exact file-level changes required. Each section contains prose explanations, bullet points, pseudocode, file references, and concrete API contracts.

---

## Table of Contents

1. [What is go-go-goja?](#1-what-is-go-go-goja)
2. [The Module System: A 10,000-Foot View](#2-the-module-system-a-10000-foot-view)
3. [Problem Statement and Scope](#3-problem-statement-and-scope)
4. [Current-State Architecture (Evidence-Based)](#4-current-state-architecture-evidence-based)
5. [Gap Analysis](#5-gap-analysis)
6. [Proposed Architecture and APIs](#6-proposed-architecture-and-apis)
7. [Pseudocode and Key Flows](#7-pseudocode-and-key-flows)
8. [Implementation Phases](#8-implementation-phases)
9. [Test Strategy](#9-test-strategy)
10. [Risks, Alternatives, and Open Questions](#10-risks-alternatives-and-open-questions)
11. [References](#11-references)

---

## 1. What is go-go-goja?

go-go-goja is a Go project that embeds the **goja JavaScript engine** (a pure-Go ECMAScript 5.1+ runtime by dop251) and layers on top of it a production-ready runtime environment. It is part of the go-go-golems ecosystem and is used to execute JavaScript scripts with access to native Go capabilities: file system operations, process execution, database queries, timers, and more.

Think of go-go-goja as "Node.js-like capabilities for Go-embedded JavaScript." It does not run in a browser. It runs inside a Go process. Scripts can `require()` native modules written in Go, and those modules can expose any Go functionality to JavaScript.

### Key architectural layers

- **goja core** (`github.com/dop251/goja`): The ECMAScript engine. It parses JS, compiles to bytecode, and executes in a VM. It provides `goja.Runtime`.
- **goja_nodejs** (`github.com/dop251/goja_nodejs`): Adds Node.js-compatible APIs: `require()`, `console`, `eventloop`, etc.
- **go-go-goja engine** (`./engine/`): Our factory/runtime layer. Composes goja + goja_nodejs + our native modules into an owned, lifecycle-managed `Runtime`.
- **go-go-goja modules** (`./modules/`): Native Go modules exposed to JS via `require("<name>")`.
- **go-go-goja packages** (`./pkg/`): Supporting libraries: REPL, inspectors, TypeScript generation, etc.

---

## 2. The Module System: A 10,000-Foot View

Before we design the YAML module, you must understand how a native module goes from "Go code" to "available in JavaScript." This section explains the full lifecycle.

### 2.1 The `NativeModule` interface

Every native module in go-go-goja implements the `NativeModule` interface, defined in `./modules/common.go` (lines 18–24):

```go
type NativeModule interface {
    Name() string
    Doc() string
    Loader(*goja.Runtime, *goja.Object)
}
```

What each method means:

- **`Name()`** returns the string used in `require("name")`. For our module, this will be `"yaml"`.
- **`Doc()`** returns a human-readable documentation string. This is surfaced in help systems and introspection APIs.
- **`Loader(vm, moduleObj)`** is the function goja_nodejs calls when your module is first required. You populate `moduleObj.Get("exports")` with the functions and values you want JS to see.

### 2.2 Registration via `init()`

Go packages can declare `init()` functions that run automatically when the package is imported. Every go-go-goja module package has an `init()` that calls `modules.Register(&module{})`. This adds the module to a **global default registry** (`modules.DefaultRegistry`).

This is a **passive** registration: the module announces its existence, but nothing is enabled yet. No JavaScript runtime can see it until the registry is explicitly wired into a `require.Registry`.

### 2.3 From registry to runtime: the factory pipeline

The `engine` package is where modules become available to JavaScript. Here is the exact flow, traced through the source code.

#### Step A: Blank imports in `engine/runtime.go`

Look at `./engine/runtime.go`, lines 17–24:

```go
import (
    // ...
    _ "github.com/go-go-golems/go-go-goja/modules/database"
    _ "github.com/go-go-golems/go-go-goja/modules/exec"
    _ "github.com/go-go-golems/go-go-goja/modules/fs"
    _ "github.com/go-go-golems/go-go-goja/modules/timer"
)
```

These are **blank imports** (the underscore prefix). They exist solely to trigger the `init()` functions in those packages, which register the modules into `modules.DefaultRegistry`. Without these imports, the modules would compile but never register.

#### Step B: `DefaultRegistryModules()` returns a `ModuleSpec`

In `./engine/module_specs.go` (lines 102–115):

```go
type defaultRegistryModulesSpec struct{}

func (s defaultRegistryModulesSpec) ID() string {
    return "default-registry-modules"
}

func (s defaultRegistryModulesSpec) Register(reg *require.Registry) error {
    modules.EnableAll(reg)
    return nil
}

func DefaultRegistryModules() ModuleSpec {
    return defaultRegistryModulesSpec{}
}
```

`DefaultRegistryModules()` returns a spec that, when registered with a `require.Registry`, calls `modules.EnableAll(reg)`. This iterates over every module in `modules.DefaultRegistry` and calls `reg.RegisterNativeModule(name, loader)` for each.

#### Step C: Factory builds the composition

In `./engine/factory.go`, `FactoryBuilder` collects module specs, then `Build()` validates and freezes them into a `Factory`. `Factory.NewRuntime()` creates a `goja.Runtime`, creates a `require.NewRegistry(...)`, and then calls `mod.Register(reg)` for every module spec.

#### Step D: Runtime initialization completes

After all modules are registered, `NewRuntime()` enables the require module on the VM (`reg.Enable(vm)`), enables console logging, and runs any `RuntimeInitializer` hooks.

### 2.4 How `require("yaml")` will work

Once we add the YAML module, this is what happens inside JavaScript:

```javascript
const yaml = require("yaml");
const doc = yaml.parse("hello: world");
console.log(doc.hello); // "world"
```

Under the hood:

1. goja_nodejs sees `require("yaml")`.
2. It looks up "yaml" in the `require.Registry`.
3. It finds the native module loader we registered.
4. It creates a fresh module object and calls our `Loader(vm, moduleObj)`.
5. Our loader attaches `parse`, `stringify`, and `validate` to `moduleObj.exports`.
6. The `require()` call returns the exports object to JavaScript.

### 2.5 TypeScript declarations

Some modules also implement `modules.TypeScriptDeclarer`, defined in `./modules/typing.go`:

```go
type TypeScriptDeclarer interface {
    TypeScriptModule() *spec.Module
}
```

This allows go-go-goja's `gen-dts` tooling to emit TypeScript `.d.ts` files for native modules, giving JavaScript developers autocomplete and type checking in their editors.

---

## 3. Problem Statement and Scope

### 3.1 The problem

go-go-goja currently has no built-in YAML support. JavaScript scripts running in the runtime can parse JSON natively (`JSON.parse`), but if they encounter a YAML configuration file, a YAML API response, or a YAML data payload, they have no path forward without implementing a YAML parser in pure JavaScript (which is slow, large, and error-prone).

### 3.2 Why a native module?

A native Go module is the correct choice because:

- **Performance**: Go's `gopkg.in/yaml.v3` is a heavily optimized, battle-tested parser. A pure-JS YAML parser would be orders of magnitude slower.
- **Correctness**: YAML 1.2 is a complex specification. Reimplementing it in JS is a recipe for subtle bugs.
- **Consistency**: go-go-goja already provides native modules for `fs`, `exec`, `timer`, and `database`. YAML is a natural "primitive" alongside these.
- **Dependency reuse**: `gopkg.in/yaml.v3` is already in our `go.mod` (indirect dependency), so adding it as a direct dependency has minimal cost.

### 3.3 Scope

**In scope:**

- A `yaml` native module with `parse`, `stringify`, and `validate` functions.
- Registration in `modules.DefaultRegistry`.
- Blank import in `engine/runtime.go` so it is available by default.
- TypeScript declaration support (`TypeScriptDeclarer`).
- Integration tests covering happy paths and error cases.
- Documentation in module `Doc()` and this design guide.

**Out of scope (future work):**

- Streaming YAML parser (for very large documents).
- YAML schema validation (against JSON Schema or custom schemas).
- Custom tags and anchors manipulation APIs.
- `yaml.loadAll` for multi-document streams (can be added later; we will design with this in mind).

---

## 4. Current-State Architecture (Evidence-Based)

This section maps the exact files and code patterns that the YAML module must align with. Every claim is anchored to source code.

### 4.1 The module registration machinery

**File:** `./modules/common.go`

- Lines 18–24: `NativeModule` interface.
- Lines 28–32: `Registry` struct holds a slice of `NativeModule`.
- Lines 35–38: `NewRegistry()` creates an empty registry.
- Lines 41–43: `Register(m)` appends to the slice.
- Lines 66–73: `Enable(gojaRegistry)` iterates and calls `gojaRegistry.RegisterNativeModule(m.Name(), m.Loader)`.
- Lines 76–79: `DefaultRegistry` is the global singleton.
- Lines 82–95: Package-level helpers `Register()`, `GetModule()`, `ListDefaultModules()`, `EnableAll()`.

**Key insight:** There is no magic. `Register()` appends to a slice. `EnableAll()` iterates that slice and calls goja_nodejs's registration API. This is explicit and easy to trace.

### 4.2 A simple module: `fs`

**File:** `./modules/fs/fs.go`

This is the simplest existing module. Study it first.

```go
type m struct{}

var _ modules.NativeModule = (*m)(nil)
var _ modules.TypeScriptDeclarer = (*m)(nil)

func (m) Name() string { return "fs" }

func (m) TypeScriptModule() *spec.Module {
    return &spec.Module{
        Name: "fs",
        Functions: []spec.Function{
            { Name: "readFileSync", Params: []spec.Param{
                {Name: "path", Type: spec.String()},
            }, Returns: spec.String() },
            { Name: "writeFileSync", Params: []spec.Param{
                {Name: "path", Type: spec.String()},
                {Name: "data", Type: spec.String()},
            }, Returns: spec.Void() },
        },
    }
}

func (m) Doc() string { /* ... */ }

func (mod m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)
    modules.SetExport(exports, mod.Name(), "readFileSync", func(path string) (string, error) {
        b, err := os.ReadFile(path)
        return string(b), err
    })
    modules.SetExport(exports, mod.Name(), "writeFileSync", func(path, data string) error {
        return os.WriteFile(path, []byte(data), 0o644)
    })
}

func init() { modules.Register(&m{}) }
```

**Patterns to copy:**

1. Empty struct `m` as the concrete type.
2. Compile-time interface checks with `var _ modules.NativeModule = (*m)(nil)`.
3. `Loader` receives `*goja.Runtime` and `*goja.Object`, gets `exports`, and calls `modules.SetExport`.
4. Go functions return `(T, error)` — goja automatically turns errors into thrown JS exceptions.
5. `init()` calls `modules.Register(&m{})`.

### 4.3 A stateful module: `database`

**File:** `./modules/database/database.go`

The database module is more complex because it holds state (the DB connection). It uses an options pattern for construction:

```go
type DBModule struct { /* fields */ }

func New(options ...Option) *DBModule { /* ... */ }

func (m *DBModule) Name() string { return m.name }
func (m *DBModule) Loader(vm *goja.Runtime, moduleObj *goja.Object) { /* ... */ }

func init() { modules.Register(New()) }
```

**Key insight for YAML:** Our module is stateless, like `fs`. We do not need per-instance configuration. We can use the empty-struct pattern.

### 4.4 The engine runtime wiring

**File:** `./engine/runtime.go`

Lines 17–24 show the blank imports that trigger registration. This is the **only** place in the engine that knows about specific module packages. When we add the YAML module, we **must** add a blank import here.

**File:** `./engine/module_specs.go`

Lines 102–115 show `DefaultRegistryModules()`. This spec is used in test code and application code alike. By adding our module to `modules.DefaultRegistry`, it becomes available to any code that uses `DefaultRegistryModules()`.

**File:** `./engine/factory.go`

Lines 162–176 show `NewRuntime()` registering each module spec. There is nothing YAML-specific to change here — the generic spec system handles it.

### 4.5 Existing YAML dependency

**File:** `./go.mod`

```
gopkg.in/yaml.v3 v3.0.1
```

This is already an indirect dependency (pulled in by testify and other libraries). We will make it a direct dependency by importing it in our new module.

### 4.6 TypeScript declaration spec helpers

**File:** `./pkg/tsgen/spec/helpers.go`

```go
func String() TypeRef  { return TypeRef{Kind: TypeKindString} }
func Number() TypeRef  { return TypeRef{Kind: TypeKindNumber} }
func Boolean() TypeRef { return TypeRef{Kind: TypeKindBoolean} }
func Any() TypeRef     { return TypeRef{Kind: TypeKindAny} }
func Unknown() TypeRef { return TypeRef{Kind: TypeKindUnknown} }
func Void() TypeRef    { return TypeRef{Kind: TypeKindVoid} }
func Named(name string) TypeRef { /* ... */ }
func Array(item TypeRef) TypeRef { /* ... */ }
func Union(items ...TypeRef) TypeRef { /* ... */ }
func Object(fields ...Field) TypeRef { /* ... */ }
```

**File:** `./pkg/tsgen/spec/types.go`

Defines `Module`, `Function`, `Param`, `TypeRef`, `Field`, and `TypeKind` constants. We will use these to describe the YAML module's API to the TypeScript generator.

### 4.7 Testing conventions

**File:** `./modules/timer/timer_test.go`

This is the best model for our integration tests. Key patterns:

1. Use `gggengine.NewBuilder().WithModules(gggengine.DefaultRegistryModules()).Build()` to create a factory.
2. Call `factory.NewRuntime(ctx)` to get a `*gggengine.Runtime`.
3. Use `rt.Owner.Call(ctx, name, fn)` to safely execute JS on the runtime's event loop.
4. Use `t.Cleanup()` to close the runtime.
5. Assert on exported values, state objects, or thrown errors.

```go
func newDefaultRuntime(t *testing.T) *gggengine.Runtime {
    t.Helper()
    factory, err := gggengine.NewBuilder().
        WithModules(gggengine.DefaultRegistryModules()).
        Build()
    require.NoError(t, err)
    rt, err := factory.NewRuntime(context.Background())
    require.NoError(t, err)
    t.Cleanup(func() { require.NoError(t, rt.Close(context.Background())) })
    return rt
}
```

---

## 5. Gap Analysis

| # | Gap | Impact | Mitigation |
|---|---|---|---|
| 1 | No YAML parsing in JS runtime | Users cannot read YAML config files or payloads | Add `yaml.parse()` |
| 2 | No YAML serialization in JS runtime | Users cannot produce YAML output | Add `yaml.stringify()` |
| 3 | No YAML validation without exceptions | Users cannot check YAML validity in control flow | Add `yaml.validate()` |
| 4 | No TypeScript types for YAML module | IDE autocomplete missing | Implement `TypeScriptDeclarer` |
| 5 | No blank import in `engine/runtime.go` | Module would compile but not register by default | Add blank import |

---

## 6. Proposed Architecture and APIs

### 6.1 Package layout

We will add exactly one new file:

```
modules/
  yaml/
    yaml.go          # NativeModule impl + parse/stringify/validate
    yaml_test.go     # Integration tests
```

No service/adapter split is needed because the module is stateless and thin. All logic is directly in `yaml.go`.

### 6.2 Module API (JavaScript-facing)

```typescript
declare module "yaml" {
    /**
     * Parse a YAML string into a JavaScript value.
     * Throws on parse errors.
     */
    export function parse(input: string): any;

    /**
     * Serialize a JavaScript value into a YAML string.
     */
    export function stringify(
        value: any,
        options?: { indent?: number }
    ): string;

    /**
     * Validate YAML syntax without producing a value.
     * Returns an object with `valid` boolean and optional `errors` array.
     */
    export function validate(input: string): {
        valid: boolean;
        errors?: string[];
    };
}
```

### 6.3 Go implementation design

#### Function: `parse(input string) (any, error)`

Uses `gopkg.in/yaml.v3` to unmarshal into `any`. The YAML library decodes into `map[string]any`, `[]any`, `string`, `int`, `float64`, `bool`, and `nil`. These map cleanly to JavaScript objects, arrays, strings, numbers, booleans, and `null`.

One subtlety: `yaml.v3` decodes integers as `int` by default. goja handles `int` fine, but for consistency with JSON (which uses `float64` for all numbers), we may want to normalize numeric types. We will document this behavior and align with what `yaml.v3` does natively.

#### Function: `stringify(value any, options map[string]any) (string, error)`

Uses `yaml.v3` to marshal from `any`. goja values exported to Go are already standard Go types (`map[string]any`, `[]any`, `string`, `float64`, `int64`, `bool`, `nil`), so we can pass them directly to `yaml.Marshal`.

Options handling:

- Read `options["indent"]` as an integer. If absent, use default indent (2 spaces).
- Ignore unknown options silently (defensive). Alternatively, error on unknown options. We will error on unknown options for strictness.

#### Function: `validate(input string) map[string]any`

We attempt to decode the input using `yaml.Decoder` but discard the result. If any error occurs, we collect error strings. The return value is always a JS object:

```javascript
{ valid: true }                           // on success
{ valid: false, errors: ["line 2: ..."] } // on failure
```

We do not throw on invalid YAML; this is an intentional design choice so scripts can do:

```javascript
const result = yaml.validate(raw);
if (!result.valid) {
    console.warn("Bad YAML:", result.errors.join("; "));
}
```

### 6.4 TypeScript declarations (Go side)

```go
func (m) TypeScriptModule() *spec.Module {
    return &spec.Module{
        Name: "yaml",
        Functions: []spec.Function{
            {
                Name: "parse",
                Params: []spec.Param{
                    {Name: "input", Type: spec.String()},
                },
                Returns: spec.Any(),
            },
            {
                Name: "stringify",
                Params: []spec.Param{
                    {Name: "value", Type: spec.Any()},
                    {Name: "options", Type: spec.Object(
                        spec.Field{Name: "indent", Type: spec.Number(), Optional: true},
                    ), Optional: true},
                },
                Returns: spec.String(),
            },
            {
                Name: "validate",
                Params: []spec.Param{
                    {Name: "input", Type: spec.String()},
                },
                Returns: spec.Object(
                    spec.Field{Name: "valid", Type: spec.Boolean()},
                    spec.Field{Name: "errors", Type: spec.Array(spec.String()), Optional: true},
                ),
            },
        },
    }
}
```

---

## 7. Pseudocode and Key Flows

### 7.1 Module loader pseudocode

```go
func (mod m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)

    // parse(input: string) -> any | throws
    modules.SetExport(exports, mod.Name(), "parse", func(input string) (any, error) {
        var out any
        if err := yaml.Unmarshal([]byte(input), &out); err != nil {
            return nil, fmt.Errorf("yaml.parse: %w", err)
        }
        return out, nil
    })

    // stringify(value: any, options?: { indent?: number }) -> string | throws
    modules.SetExport(exports, mod.Name(), "stringify", func(value any, options map[string]any) (string, error) {
        indent := 2
        if options != nil {
            if v, ok := options["indent"]; ok {
                switch n := v.(type) {
                case int64:
                    indent = int(n)
                case int:
                    indent = n
                case float64:
                    indent = int(n)
                default:
                    return "", fmt.Errorf("yaml.stringify: indent must be a number, got %T", v)
                }
                if indent < 0 {
                    return "", fmt.Errorf("yaml.stringify: indent must be >= 0")
                }
            }
            // Strictness: reject unknown keys
            for k := range options {
                if k != "indent" {
                    return "", fmt.Errorf("yaml.stringify: unknown option %q", k)
                }
            }
        }

        var buf bytes.Buffer
        enc := yaml.NewEncoder(&buf)
        enc.SetIndent(indent)
        if err := enc.Encode(value); err != nil {
            return "", fmt.Errorf("yaml.stringify: %w", err)
        }
        if err := enc.Close(); err != nil {
            return "", fmt.Errorf("yaml.stringify: %w", err)
        }
        return buf.String(), nil
    })

    // validate(input: string) -> { valid: boolean, errors?: string[] }
    modules.SetExport(exports, mod.Name(), "validate", func(input string) map[string]any {
        decoder := yaml.NewDecoder(strings.NewReader(input))
        var errors []string
        var out any
        for {
            if err := decoder.Decode(&out); err != nil {
                if err == io.EOF {
                    break
                }
                errors = append(errors, err.Error())
                break
            }
        }
        if len(errors) > 0 {
            return map[string]any{"valid": false, "errors": errors}
        }
        return map[string]any{"valid": true}
    })
}
```

### 7.2 Runtime creation flow (with YAML)

```
[Application code]
    |
    v
factory, err := engine.NewBuilder()
    .WithModules(engine.DefaultRegistryModules())
    .Build()
    |
    v
[engine/runtime.go blank import: _ "modules/yaml"]
    |
    v
[modules/yaml/yaml.go init() runs]
    modules.Register(&m{})
    |
    v
[modules.DefaultRegistry now contains "yaml"]
    |
    v
[factory.NewRuntime(ctx)]
    |
    +---> create goja.Runtime
    +---> create require.Registry
    +---> for each ModuleSpec:
    |       mod.Register(reg)
    |           |
    |           v
    |       defaultRegistryModulesSpec.Register(reg)
    |           calls modules.EnableAll(reg)
    |               for each module in DefaultRegistry:
    |                   reg.RegisterNativeModule(name, loader)
    |                   // "yaml" is registered here
    +---> reg.Enable(vm)
    +---> return Runtime
    |
    v
[JS: const yaml = require("yaml")]
    goja_nodejs looks up "yaml" in reg
    finds loader -> calls Loader(vm, moduleObj)
    exports now has parse/stringify/validate
```

### 7.3 Data type mapping: YAML <-> Go <-> JS

| YAML tag | yaml.v3 Go type | goja JS type |
|---|---|---|
| `!!str` | `string` | `string` |
| `!!int` | `int` | `number` |
| `!!float` | `float64` | `number` |
| `!!bool` | `bool` | `boolean` |
| `!!null` | `nil` | `null` |
| `!!map` | `map[string]any` | `object` |
| `!!seq` | `[]any` | `array` |

**Important note:** `yaml.v3` preserves `int` as Go `int`, not `float64`. goja converts Go `int` to JS `number` transparently, so this is fine. However, if a script serializes and then re-parses, the type may round-trip differently. We will document this.

---

## 8. Implementation Phases

### Phase 1: Create the module file

**File:** `modules/yaml/yaml.go`

- Implement `m struct{}` with `NativeModule` and `TypeScriptDeclarer`.
- Implement `parse`, `stringify`, `validate` as described.
- Add `init()` with `modules.Register(&m{})`.
- Add package documentation comment.

**Verification:** `go build ./modules/yaml` should succeed.

### Phase 2: Wire into the engine

**File:** `engine/runtime.go`

- Add blank import: `_ "github.com/go-go-golems/go-go-goja/modules/yaml"`

**Verification:** `go build ./engine` should succeed.

### Phase 3: Add integration tests

**File:** `modules/yaml/yaml_test.go`

Tests to write:

1. `TestYamlParseSimple` — parse `hello: world`, assert result.
2. `TestYamlParseNested` — parse nested maps and arrays.
3. `TestYamlParseInvalid` — parse invalid YAML, assert error is thrown.
4. `TestYamlStringifySimple` — stringify a JS object, assert output contains expected keys.
5. `TestYamlStringifyWithIndent` — pass `indent: 4`, verify indentation.
6. `TestYamlStringifyUnknownOption` — pass `foo: 1`, assert error thrown.
7. `TestYamlValidateValid` — validate correct YAML, assert `{ valid: true }`.
8. `TestYamlValidateInvalid` — validate broken YAML, assert `valid: false` and error messages.
9. `TestYamlRoundTrip` — parse then stringify, assert structural equality.
10. `TestDefaultRuntimeCanRequireYamlModule` — minimal smoke test that `require("yaml")` works.

Use `newDefaultRuntime(t)` helper from the timer test pattern.

**Verification:** `go test ./modules/yaml/... -count=1 -v`

### Phase 4: Update dependency status

**File:** `go.mod`

Run `go mod tidy` to promote `gopkg.in/yaml.v3` from indirect to direct.

**Verification:** `gopkg.in/yaml.v3` appears in the direct `require` block.

### Phase 5: Add TypeScript declaration generation test (optional)

If the repository has a `gen-dts` command or test that exercises `TypeScriptDeclarer`, run it and verify the YAML module appears in output.

**Verification:** Search generated `.d.ts` for `declare module "yaml"`.

### Phase 6: Documentation and handoff

- Update module `Doc()` with concise usage examples.
- Update `README.md` if there is a module list.
- Run full test suite: `go test ./... -count=1`
- Run linter: `make lint` or `golangci-lint run`

---

## 9. Test Strategy

### 9.1 Unit vs integration

The YAML module has no complex internal state, so **integration tests** are the primary validation. We test through the goja runtime because:

- The public API is the JS-facing API.
- We must verify goja's type conversions work correctly (Go `int` → JS `number`, JS object → Go `map[string]any`, etc.).
- We must verify `require("yaml")` actually resolves.

### 9.2 Test data patterns

Use table-driven tests for coverage:

```go
var parseCases = []struct {
    name     string
    input    string
    expected any
}{
    {"scalar string", "hello", "hello"},
    {"simple map", "a: 1\nb: 2", map[string]any{"a": 1, "b": 2}},
    {"nested map", "a:\n  b: 1", map[string]any{"a": map[string]any{"b": 1}}},
    {"array", "- 1\n- 2\n- 3", []any{1, 2, 3}},
    {"null", "null", nil},
}
```

### 9.3 Error testing

For functions that throw (parse, stringify), assert on the error message:

```go
_, err := vm.RunString(`yaml.parse("[bad")`)
require.Error(t, err)
require.Contains(t, err.Error(), "yaml.parse")
```

For `validate`, assert on the return value structure:

```javascript
const result = yaml.validate("[bad");
if (result.valid) throw new Error("expected invalid");
if (!result.errors || result.errors.length === 0) throw new Error("expected errors");
```

### 9.4 CI validation

Before merging, run:

```bash
go test ./... -count=1
GOWORK=off go test ./... -count=1
go mod tidy && git diff --exit-code go.mod go.sum
make lint  # or golangci-lint run
```

---

## 10. Risks, Alternatives, and Open Questions

### 10.1 Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `yaml.v3` decodes ints as `int`, causing subtle JS interop issues | Low | Medium | Document behavior; integration tests verify round-trips |
| Multi-document YAML input to `parse` only returns the first document | Medium | Low | Document that `parse` reads the first document; future `parseAll` can address |
| Large YAML inputs cause memory pressure | Low | Medium | Out of scope for initial PR; document size guidance |
| Options struct design becomes a breaking change if we add more later | Medium | Low | Keep options minimal; only `indent` for now |

### 10.2 Alternatives considered

**Alternative A: Pure-JS YAML library**

- We could vendor a JavaScript YAML parser (like `js-yaml`) and load it as a user module.
- **Rejected**: Performance is poor compared to native Go. Also increases bundle size and complexity.

**Alternative B: Add YAML to an existing module (e.g., `fs`)**

- Add `readYamlSync(path)` to the `fs` module.
- **Rejected**: YAML is a distinct concern from file I/O. It should be reusable for strings, network responses, and inline data, not just files.

**Alternative C: Use `gopkg.in/yaml.v2` instead of v3**

- v2 is older and more widely used.
- **Rejected**: v3 is already in our dependency tree, has better anchor/alias support, and is the maintained line.

### 10.3 Open questions

1. Should `stringify` support custom tags (e.g., `!!timestamp`) or always use generic tags?
2. Should we expose `parseAll` for multi-document streams in the initial PR?
3. Should `validate` collect all errors in a multi-document stream or stop at the first?

**Recommendation:** Keep the initial API minimal. Open questions can be resolved in follow-up tickets.

---

## 11. References

### 11.1 Source files (exact paths)

| File | Relevance |
|---|---|
| `/home/manuel/code/wesen/go-go-golems/go-go-goja/modules/common.go` | `NativeModule` interface and `DefaultRegistry` |
| `/home/manuel/code/wesen/go-go-golems/go-go-goja/modules/exports.go` | `SetExport` helper |
| `/home/manuel/code/wesen/go-go-golems/go-go-goja/modules/typing.go` | `TypeScriptDeclarer` interface |
| `/home/manuel/code/wesen/go-go-golems/go-go-goja/modules/fs/fs.go` | Simplest module pattern |
| `/home/manuel/code/wesen/go-go-golems/go-go-goja/modules/timer/timer.go` | Promise-based module pattern |
| `/home/manuel/code/wesen/go-go-golems/go-go-goja/modules/timer/timer_test.go` | Integration test pattern |
| `/home/manuel/code/wesen/go-go-golems/go-go-goja/modules/database/database.go` | Stateful module with options pattern |
| `/home/manuel/code/wesen/go-go-golems/go-go-goja/engine/runtime.go` | Blank imports and `Runtime` struct |
| `/home/manuel/code/wesen/go-go-golems/go-go-goja/engine/factory.go` | Factory builder and `NewRuntime()` |
| `/home/manuel/code/wesen/go-go-golems/go-go-goja/engine/module_specs.go` | `ModuleSpec`, `DefaultRegistryModules()` |
| `/home/manuel/code/wesen/go-go-golems/go-go-goja/engine/options.go` | `Option` and `builderSettings` |
| `/home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/tsgen/spec/types.go` | TypeScript declaration types |
| `/home/manuel/code/wesen/go-go-golems/go-go-goja/pkg/tsgen/spec/helpers.go` | TypeScript declaration helpers |
| `/home/manuel/code/wesen/go-go-golems/go-go-goja/go.mod` | Dependency list |

### 11.2 External references

- `gopkg.in/yaml.v3` documentation: https://pkg.go.dev/gopkg.in/yaml.v3
- goja documentation: https://pkg.go.dev/github.com/dop251/goja
- goja_nodejs require: https://pkg.go.dev/github.com/dop251/goja_nodejs/require
- YAML 1.2 spec: https://yaml.org/spec/1.2.2/

### 11.3 New files to create

| File | Description |
|---|---|
| `modules/yaml/yaml.go` | Native module implementation |
| `modules/yaml/yaml_test.go` | Integration tests |

### 11.4 Existing files to modify

| File | Change |
|---|---|
| `engine/runtime.go` | Add blank import `_ "github.com/go-go-golems/go-go-goja/modules/yaml"` |
| `go.mod` / `go.sum` | Promote `gopkg.in/yaml.v3` to direct dependency (via `go mod tidy`) |

---

## Appendix A: Quick-Start Checklist for the Implementer

- [ ] Create `modules/yaml/yaml.go` with `parse`, `stringify`, `validate`
- [ ] Implement `NativeModule` and `TypeScriptDeclarer`
- [ ] Add `init()` with `modules.Register(&m{})`
- [ ] Add blank import in `engine/runtime.go`
- [ ] Create `modules/yaml/yaml_test.go` with 10+ test cases
- [ ] Run `go test ./modules/yaml/... -count=1 -v`
- [ ] Run `go test ./... -count=1`
- [ ] Run `go mod tidy`
- [ ] Run `make lint` or `golangci-lint run`
- [ ] Verify TypeScript declarations if gen-dts is available
- [ ] Update `README.md` module list if applicable
- [ ] Commit with message: `feat(modules): add yaml primitive support (enabled by default)`
