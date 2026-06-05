---
Title: Sobek ECMAScript Modules and xgoja Native Module Architecture
Ticket: GOJA-053
Status: active
Topics:
    - xgoja
    - goja
    - javascript
    - module
    - architecture
    - engine
    - research
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/others/sobek/AGENTS.md
      Note: Sobek repository guidance summarizing architecture and ESM/event-loop caveats.
    - Path: ../../../../../../../../../../code/others/sobek/README.md
      Note: |-
        Sobek README documenting experimental ESM support, lack of event loop, and tests-as-documentation status.
        Sobek ESM caveats and event-loop/documentation warnings
    - Path: ../../../../../../../../../../code/others/sobek/modules.go
      Note: |-
        Sobek ModuleRecord/CyclicModuleRecord interfaces, linking/evaluation algorithms, dynamic import hooks.
        Sobek ModuleRecord/CyclicModuleRecord and dynamic import APIs
    - Path: ../../../../../../../../../../code/others/sobek/modules_integration_test.go
      Note: |-
        Sobek custom native ModuleRecord examples and dynamic-import event-loop test patterns.
        Sobek custom native ModuleRecord examples
    - Path: ../../../../../../../../../../code/others/sobek/modules_sourcetext.go
      Note: |-
        Sobek SourceTextModuleRecord parsing, import/export entries, Link/Evaluate entrypoints.
        Sobek ParseModule/SourceTextModuleRecord implementation
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/go-go-parc/Projects/2026/04/12/PROJ - Goja vs Sobek Deep Analysis.md
      Note: |-
        Prior local analysis comparing Goja and Sobek, with the conclusion that Sobek's main differentiator is experimental ESM support.
        Prior local Sobek/Goja ESM comparison
    - Path: geppetto/pkg/js/modules/geppetto/module.go
      Note: |-
        Complex current provider module implemented as a require.ModuleLoader with runtimebridge and owner scheduling.
        Complex current require.ModuleLoader provider module
    - Path: go-go-goja/engine/factory.go
      Note: |-
        Current Goja/goja_nodejs runtime construction and require registry installation.
        Current goja_nodejs require/eventloop runtime construction
    - Path: go-go-goja/engine/module_specs.go
      Note: Current NativeModuleSpec and default registry registration path for require() modules.
    - Path: go-go-goja/modules/common.go
      Note: |-
        Current native CommonJS module interface and global module registry.
        Current CommonJS NativeModule interface
    - Path: go-go-goja/pkg/xgoja/app/factory.go
      Note: |-
        Current xgoja provider Module.New to require.ModuleLoader adapter.
        Current provider Module.New to require.ModuleLoader adapter
    - Path: go-go-goja/pkg/xgoja/providerapi/module.go
      Note: providerapi.ModuleFactory and ModuleContext are currently tied to goja_nodejs require.ModuleLoader.
ExternalSources: []
Summary: An intern-facing analysis of whether Sobek ECMAScript Modules can reduce xgoja module machinery complexity, what native modules as ESM would require, and why a full migration is not currently recommended as a simplification move.
LastUpdated: 2026-06-04T00:00:00Z
WhatFor: Use when evaluating whether to replace or supplement xgoja's CommonJS require-based native module system with Sobek ECMAScript Modules.
WhenToUse: Before implementing Sobek support, ESM script execution, native ESM ModuleRecord adapters, or a goja/sobek engine abstraction.
---


# Sobek ECMAScript Modules and xgoja Native Module Architecture

## Executive summary

Sobek's ECMAScript Module support is real enough to run source-text modules, static imports, dynamic imports, module namespace objects, `import.meta`, and top-level await, but it does **not** make xgoja's module machinery disappear. It replaces one set of machinery with another:

- Today's xgoja uses **CommonJS-style `require()`** through `goja_nodejs/require.Registry` and `require.ModuleLoader`.
- Sobek ESM uses **ECMAScript ModuleRecords**, host import resolution, link/instantiate/evaluate phases, module namespace objects, live bindings, and Promise-based evaluation.
- Native provider modules can be exposed as ESM modules, but only if xgoja implements a **synthetic/native `ModuleRecord` adapter** or rewrites provider APIs to expose static ESM export names and values.
- Sobek does **not** provide a Node-style event loop or `goja_nodejs` replacement. xgoja would need to port/fork `goja_nodejs` pieces or replace them.

The most important recommendation is: **do not switch to Sobek/ESM now if the goal is simply reducing module-system complexity**. Sobek may be a good targeted experiment for modern JavaScript source modules, but it is not a drop-in simplification for xgoja's provider/native module architecture.

A practical migration strategy is:

1. Keep the current Goja/CommonJS path as production default.
2. Use bundling-to-CommonJS for modern TypeScript/ESM user code when possible.
3. Add an isolated Sobek ESM spike package that can run `.mjs` source modules from a small filesystem resolver.
4. Prototype native modules as **default-only ESM facades** first: `import fs from "fs"` returns the same object that `require("fs")` would have returned.
5. Only after that, consider typed/named native ESM exports: `import { readFile } from "fs"`.

The reason for this staged path is that ESM answers a user-facing JavaScript syntax problem (`import`/`export`), not the xgoja provider lifecycle/configuration problem. xgoja still needs provider selection, runtime profile selection, host services, runtime owner scheduling, config merging, runtimebridge services, and cleanup.

---

## 1. Background: two JavaScript module worlds

### 1.1 CommonJS: what xgoja uses today

CommonJS is the Node-era module style:

```javascript
const fs = require("fs");
const { readFileSync } = require("fs");
module.exports = { run };
```

Properties:

- `require(...)` is a normal function call.
- It happens at runtime, not at parse time.
- A module returns an object, usually `module.exports`.
- Native Go modules can populate that object dynamically.
- Cycles are handled through mutable module objects and caching.
- It maps naturally to Go functions such as `Loader(vm, moduleObj)`.

Current xgoja machinery is built around this model:

```text
providerapi.Module.New(ModuleContext)
  → require.ModuleLoader
  → require.Registry.RegisterNativeModule(alias, loader)
  → JavaScript calls require(alias)
  → loader populates module.exports
```

Key source files:

- `go-go-goja/pkg/xgoja/providerapi/module.go`
- `go-go-goja/pkg/xgoja/app/factory.go`
- `go-go-goja/engine/factory.go`
- `go-go-goja/engine/module_specs.go`
- `go-go-goja/modules/common.go`

### 1.2 ECMAScript Modules: what Sobek adds

ECMAScript Modules, or ESM, are the standard modern JavaScript module system:

```javascript
import fs from "fs";
import { readFile } from "fs";
export function run() {}
export default run;
```

Properties:

- Static imports are parsed before execution.
- The host must resolve import specifiers during linking.
- Export names are part of module linkage.
- Exports are **live bindings**, not merely object properties.
- Evaluation returns a Promise because top-level await can make module execution asynchronous.
- Dynamic `import(...)` is asynchronous and requires host scheduling/event-loop support.

Sobek implements this through types such as:

- `ModuleRecord`
- `CyclicModuleRecord`
- `ModuleInstance`
- `SourceTextModuleRecord`
- `HostResolveImportedModuleFunc`

Source files:

- `/home/manuel/code/others/sobek/modules.go`
- `/home/manuel/code/others/sobek/modules_sourcetext.go`
- `/home/manuel/code/others/sobek/modules_namespace.go`
- `/home/manuel/code/others/sobek/modules_test.go`
- `/home/manuel/code/others/sobek/modules_integration_test.go`

### 1.3 The conceptual mismatch

CommonJS native modules say:

```text
Give me a runtime and a module object; I will populate exports now.
```

ESM native modules say:

```text
Tell me your export names before linking; later provide live binding values when those exports are requested.
```

That difference matters. Existing xgoja modules often build an object dynamically inside a loader. ESM wants export names and bindings to be knowable during the link phase.

---

## 2. What Sobek ESM actually provides

### 2.1 Sobek is a Goja fork with experimental ESM

The prior local analysis at `/home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/04/12/PROJ - Goja vs Sobek Deep Analysis.md` concluded:

- Sobek tracks Goja closely.
- ESM is the major differentiator.
- Sobek adds roughly 3,600 lines across module files.
- Sobek is k6-driven.
- Prefer Goja unless ESM is specifically needed.

Sobek's own README says ESM is experimental, likely to change in breaking ways, has no concrete documentation, and currently relies on tests as examples. It also says Sobek does not provide an event loop and likely never will.

### 2.2 Core Sobek ESM APIs

From `/home/manuel/code/others/sobek/modules.go`:

```go
type HostResolveImportedModuleFunc func(
    referencingScriptOrModule interface{},
    specifier string,
) (ModuleRecord, error)

type ModuleRecord interface {
    GetExportedNames(callback func([]string), resolveset ...ModuleRecord) bool
    ResolveExport(exportName string, resolveset ...ResolveSetElement) (*ResolvedBinding, bool)
    Link() error
    Evaluate(*Runtime) *Promise
}

type CyclicModuleRecord interface {
    ModuleRecord
    RequestedModules() []string
    InitializeEnvironment() error
    Instantiate(rt *Runtime) (CyclicModuleInstance, error)
}

type ModuleInstance interface {
    GetBindingValue(string) Value
}

type CyclicModuleInstance interface {
    ModuleInstance
    HasTLA() bool
    ExecuteModule(rt *Runtime, res, rej func(interface{}) error) (CyclicModuleInstance, error)
}
```

From `/home/manuel/code/others/sobek/modules_sourcetext.go`:

```go
func ParseModule(
    name, sourceText string,
    resolveModule HostResolveImportedModuleFunc,
    opts ...parser.Option,
) (*SourceTextModuleRecord, error)

func (module *SourceTextModuleRecord) Link() error
func (module *SourceTextModuleRecord) Evaluate(rt *Runtime) *Promise
func (module *SourceTextModuleRecord) RequestedModules() []string
```

For dynamic import:

```go
type ImportModuleDynamicallyCallback func(
    referencingScriptOrModule interface{},
    specifier Value,
    promiseCapability interface{},
)

func (r *Runtime) SetImportModuleDynamically(callback ImportModuleDynamicallyCallback)
func (r *Runtime) FinishLoadingImportModule(...)
```

For `import.meta`:

```go
func (r *Runtime) SetGetImportMetaProperties(fn func(ModuleRecord) []MetaProperty)
func (r *Runtime) SetFinalImportMeta(fn func(*Object, ModuleRecord))
```

### 2.3 Basic source module flow

A simple Sobek ESM host does roughly this:

```go
resolver := newResolver(fs, nativeModules)

mainRecord, err := resolver.Resolve(nil, "main.mjs")
if err != nil { return err }

if err := mainRecord.Link(); err != nil { return err }

vm := sobek.New()
promise := mainRecord.Evaluate(vm)
awaitOrPumpEventLoopUntilSettled(promise)
```

The resolver is the host-defined part. Sobek calls it whenever an import needs resolving.

```go
func resolve(referrer interface{}, specifier string) (sobek.ModuleRecord, error) {
    if native, ok := nativeModules[specifier]; ok {
        return native, nil
    }

    path := resolvePath(referrer, specifier)
    if cached, ok := cache[path]; ok {
        return cached.record, cached.err
    }

    source := readFile(path)
    record, err := sobek.ParseModule(path, source, resolve)
    cache[path] = {record, err}
    return record, err
}
```

### 2.4 Static ESM imports are link-time work

Static import syntax:

```javascript
import { readFile } from "fs";
```

requires the host and engine to answer before execution:

1. What does specifier `"fs"` resolve to?
2. Does that module export `readFile`?
3. Is `readFile` ambiguous because of star exports?
4. Which module instance and binding does `readFile` refer to?

This is why a CommonJS object loader is not automatically enough. A CJS loader says what properties exist only after it runs. ESM linking wants export names before evaluation.

---

## 3. Current xgoja module machinery

### 3.1 Built-in native module registry

`go-go-goja/modules/common.go` defines the current `NativeModule` interface:

```go
type NativeModule interface {
    Name() string
    Doc() string
    Loader(*goja.Runtime, *goja.Object)
}
```

Modules register themselves into `modules.DefaultRegistry` from `init()`:

```go
func init() {
    modules.Register(New(WithName("fs")))
    modules.Register(New(WithName("node:fs")))
}
```

`engine/module_specs.go` later looks modules up by name and calls:

```go
reg.RegisterNativeModule(mod.Name(), mod.Loader)
```

This is CommonJS-oriented: a loader populates a `module.exports` object.

### 3.2 xgoja provider module registry

xgoja generated binaries use provider packages. A provider registers with `providerapi.Registry`:

```go
registry.Package(PackageID,
    providerapi.Module{
        Name: "geppetto",
        New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
            ...
        },
    },
)
```

Important types:

- `providerapi.Registry` in `go-go-goja/pkg/xgoja/providerapi/registry.go`
- `providerapi.Module` and `ModuleContext` in `go-go-goja/pkg/xgoja/providerapi/module.go`
- `providerRuntimeModuleSpec` in `go-go-goja/pkg/xgoja/app/factory.go`

Current flow:

```text
xgoja runtime profile selects provider module
  ↓
app.RuntimeFactory resolves providerapi.Module
  ↓
providerRuntimeModuleSpec.RegisterRuntimeModule
  ↓
module.New(ModuleContext) returns require.ModuleLoader
  ↓
require.Registry.RegisterNativeModule(alias, loader)
```

This path is also CommonJS-oriented.

### 3.3 Runtime services are not module-system-specific

Several confusing xgoja pieces are **not** specifically CommonJS:

- `runtimeowner.RuntimeOwner`: serializes access to the JS runtime owner thread.
- `runtimebridge.RuntimeServices`: lets native module code find runtime lifetime context and owner scheduling from a VM.
- `HostServices`: passes host capabilities such as assets and Geppetto services to provider module construction.
- `RuntimeFactory`: selects modules and creates runtime instances.

These remain necessary even with ESM. ESM does not eliminate:

- runtime lifecycle,
- cancellation,
- async scheduling,
- host services,
- provider config,
- security/capability selection,
- cleanup.

### 3.4 Current code is strongly typed to Goja and goja_nodejs

A migration to Sobek is not just swapping import paths in one file. A quick search found:

- about 230 files importing `github.com/dop251/goja`,
- about 106 files importing `github.com/dop251/goja_nodejs`.

Important examples:

- `engine/factory.go` imports `goja`, `goja_nodejs/buffer`, `console`, `eventloop`, `require`, and `url`.
- `engine/runtime.go` stores `*goja.Runtime`, `*eventloop.EventLoop`, and `*require.RequireModule`.
- `modules/common.go` defines loaders using `*goja.Runtime` and `*goja.Object`.
- `providerapi/module.go` returns `require.ModuleLoader`.
- Geppetto uses `goja` throughout `geppetto/pkg/js/modules/geppetto`.

Sobek's package path is `github.com/grafana/sobek`. Even if many API names are similar, Go treats `*goja.Runtime` and `*sobek.Runtime` as unrelated types. `goja_nodejs` cannot be reused directly unless it is ported/forked for Sobek.

---

## 4. Can native/provider modules be exposed as ESM?

Yes, but not for free. There are three possible levels.

### 4.1 Level 1: default-only ESM facade over a native object

JavaScript:

```javascript
import fs from "fs";
fs.readFileSync("/tmp/a.txt", "utf8");
```

The native module exposes only a default export. That default export is an object containing functions, similar to today's `module.exports`.

Pros:

- Lowest semantic mismatch.
- No need for named export discovery.
- Existing dynamic object-shaped native modules map well.
- Users can still write ESM syntax.

Cons:

- No `import { readFileSync } from "fs"`.
- Not idiomatic for all ESM users.
- Still requires Sobek native module adapter and resolver.

Pseudocode:

```go
type DefaultObjectModuleRecord struct {
    specifier string
    buildObject func(rt *sobek.Runtime) sobek.Value
}

func (m *DefaultObjectModuleRecord) GetExportedNames(cb func([]string), _ ...sobek.ModuleRecord) bool {
    cb([]string{"default"})
    return true
}

func (m *DefaultObjectModuleRecord) ResolveExport(name string, _ ...sobek.ResolveSetElement) (*sobek.ResolvedBinding, bool) {
    if name != "default" { return nil, false }
    return &sobek.ResolvedBinding{Module: m, BindingName: "default"}, false
}

func (m *DefaultObjectModuleRecord) Link() error { return nil }

func (m *DefaultObjectModuleRecord) Evaluate(rt *sobek.Runtime) *sobek.Promise {
    p, resolve, reject := rt.NewPromise()
    inst := &DefaultObjectModuleInstance{value: m.buildObject(rt)}
    if err := resolve(inst); err != nil { _ = reject(err) }
    return p
}

type DefaultObjectModuleInstance struct { value sobek.Value }
func (i *DefaultObjectModuleInstance) GetBindingValue(name string) sobek.Value {
    if name == "default" { return i.value }
    return nil
}
```

This pattern is close to Sobek's `simpleModuleImpl` in `modules_integration_test.go`, where a custom `ModuleRecord` evaluates to a custom `ModuleInstance`.

### 4.2 Level 2: named native ESM exports

JavaScript:

```javascript
import { readFileSync, writeFileSync } from "fs";
```

The native module must expose a stable export list before linking:

```go
type NativeESMModuleRecord struct {
    specifier string
    exports []NativeExportSpec
}

type NativeExportSpec struct {
    Name string
    Build func(rt *sobek.Runtime) sobek.Value
}
```

Then `GetExportedNames` returns all names, and `ResolveExport` maps each export name to a binding.

Pros:

- Idiomatic ESM imports.
- Enables static checking during linking.
- Better TypeScript declaration alignment.

Cons:

- Existing loaders must be refactored to declare export names.
- Dynamic exports or object replacement do not fit well.
- Live binding semantics must be thought through.
- It is harder for complex modules such as Geppetto that build nested APIs and classes dynamically.

### 4.3 Level 3: full ESM provider API

Provider API might become:

```go
type ProviderModule interface {
    Name() string
    CommonJS(ctx ModuleContext) (require.ModuleLoader, error) // old backend
    ESM(ctx ESMModuleContext) (ESMModuleRecordFactory, error) // new backend
}

type ESMModuleRecordFactory interface {
    ExportNames() []string
    NewRecord(ctx ESMRecordContext) (sobek.ModuleRecord, error)
}
```

This is the cleanest long-term shape, but it is the most work. It also does not by itself reduce the number of concepts: it adds ESM concepts beside current CommonJS concepts.

---

## 5. What using Sobek would require in xgoja

### 5.1 Replace or abstract the runtime type

Today many APIs directly mention Goja:

```go
*goja.Runtime
*goja.Object
goja.Value
goja.FunctionCall
require.ModuleLoader
*require.Registry
*eventloop.EventLoop
```

Sobek would require either:

1. a hard migration from Goja to Sobek; or
2. a backend abstraction layer; or
3. an isolated Sobek-only experimental runtime path.

A hard migration is risky because `goja_nodejs` is tied to Goja types. A backend abstraction is a lot of design work. An isolated Sobek-only spike is the safest first step.

### 5.2 Replace `goja_nodejs/require.Registry`

Sobek ESM does not use `require.Registry`. xgoja would need a new resolver/cache:

```go
type ESMResolver struct {
    fsRoots []fs.FS
    native map[string]NativeModuleRecordFactory
    cache map[string]sobek.ModuleRecord
}

func (r *ESMResolver) Resolve(ref interface{}, specifier string) (sobek.ModuleRecord, error) {
    if f, ok := r.native[specifier]; ok {
        return f.Record(), nil
    }
    path := r.resolvePath(ref, specifier)
    if rec, ok := r.cache[path]; ok {
        return rec, nil
    }
    src := r.read(path)
    rec, err := sobek.ParseModule(path, src, r.Resolve)
    r.cache[path] = rec
    return rec, err
}
```

This resolver would replace Node/CommonJS resolution for ESM execution. If xgoja still wants CommonJS `require()`, then it needs both systems or a CJS compatibility bridge.

### 5.3 Replace or port `goja_nodejs` built-ins

Current engine setup installs:

- `require(...)`,
- `console`,
- `Buffer`,
- `URL`,
- `URLSearchParams`,
- `util`,
- an event loop,
- native module loaders.

Source: `go-go-goja/engine/factory.go` and `go-go-goja/engine/module_specs.go`.

Sobek does not provide these Node primitives. xgoja would need to:

- port/fork the relevant `goja_nodejs` packages to Sobek;
- replace them with custom xgoja implementations;
- or intentionally not support them in Sobek mode.

This is one of the biggest reasons Sobek is not a simplification today.

### 5.4 Provide an event loop for dynamic import and top-level await

Sobek README and AGENTS both state that ESM requires the embedder to provide an event loop and Sobek does not provide one.

Sobek tests show minimal ad-hoc event-loop queues, for example in `modules_test.go` and `modules_integration_test.go`:

```go
eventLoopQueue := make(chan func(), 2)
vm.SetImportModuleDynamically(func(ref interface{}, specifier sobek.Value, pcap interface{}) {
    eventLoopQueue <- func() {
        m, err := resolver.resolve(ref, specifier.String())
        vm.FinishLoadingImportModule(ref, specifier, pcap, m, err)
    }
})
```

xgoja already has `runtimeowner` and a `goja_nodejs/eventloop`, but those are typed to Goja. A Sobek backend would need a Sobek-compatible owner loop and task queue.

### 5.5 Decide CJS/ESM interop policy

If xgoja supports ESM scripts, users will ask:

```javascript
import fs from "fs";
const fs = require("fs");
import { readFile } from "fs";
const mod = await import("fs");
```

We need a policy.

Recommended v1 policy:

- ESM entrypoints can `import defaultExport from "native-module"`.
- Native modules expose default object only by default.
- Named native exports require an explicit provider declaration.
- CommonJS `require()` is not available in pure Sobek ESM mode unless a compatibility module is intentionally installed.
- Existing Goja/CommonJS mode remains the default.

### 5.6 Rework TypeScript declaration generation

Current modules often implement `TypeScriptDeclarer`. That describes CommonJS-like module exports today. ESM named/default export declarations would need additional metadata:

```ts
// Default-object mode
declare module "fs" {
  const fs: {
    readFileSync(path: string, encoding?: string): string | Buffer;
  };
  export default fs;
}

// Named export mode
declare module "fs" {
  export function readFileSync(path: string, encoding?: string): string | Buffer;
}
```

The type declaration generator would need to know which surface the runtime supports.

---

## 6. Architecture options

### Option A: Keep Goja/CommonJS and improve naming/docs

This means:

- Keep `require()` as the production module system.
- Rename confusing xgoja capability names such as `ConfigSectionCapability`.
- Rename/document `ModuleContext.Context` as setup/startup context.
- Use Glazed `SectionValues` for GOJA-053 config timing.
- Use bundling-to-CJS for modern JS/TS projects.

Pros:

- Lowest risk.
- Keeps current modules working.
- Keeps `goja_nodejs` primitives.
- Directly addresses current confusion with docs and API naming.

Cons:

- Users cannot write unbundled ESM entrypoints with static imports.
- Modern JS still needs a bundling step.

Best when: the goal is reducing xgoja complexity now.

### Option B: Keep Goja/CommonJS runtime, add build-time ESM-to-CJS bundling

This is already documented by `go-go-goja/pkg/doc/bun-goja-bundling-playbook.md`.

Flow:

```text
TypeScript/ESM source
  → Bun/esbuild bundle
  → CommonJS output
  → current goja require runtime
```

Pros:

- Lets users author ESM/TypeScript.
- Avoids Sobek migration.
- Uses mature bundlers to solve package/module resolution.
- Keeps native modules external via `require("fs")`, `require("geppetto")`, etc.

Cons:

- Runtime still uses `require()`.
- Requires build step.
- Dynamic runtime imports are limited by bundler choices.

Best when: user-facing ESM syntax is desired, but runtime architecture simplification is the priority.

### Option C: Add isolated Sobek ESM runner experiment

Create a separate package, e.g.:

```text
go-go-goja/pkg/sobekesm/
```

It would not replace engine/xgoja immediately. It would implement:

- source module resolver,
- native default-object module records,
- basic event loop queue for dynamic imports,
- import.meta support,
- tests for `.mjs` entrypoints.

Pros:

- Low blast radius.
- Produces real evidence.
- Lets us learn Sobek's ESM API without breaking xgoja.

Cons:

- Duplicates runtime concepts temporarily.
- Does not integrate with current xgoja provider config yet.

Best when: we want evidence before deciding.

### Option D: Full Sobek backend for xgoja

Replace or abstract engine internals so xgoja can run on Sobek.

Would require:

- replacing `goja` imports or adding abstraction,
- porting/forking `goja_nodejs` packages,
- implementing ESM resolver/cache,
- implementing native ESM module records,
- adapting provider APIs,
- adapting runtimebridge/runtimeowner to Sobek,
- updating tests and docs.

Pros:

- Native ESM runtime possible.
- Modern module semantics at runtime.

Cons:

- Highest risk.
- Large migration surface.
- Sobek ESM API is experimental.
- Not likely to reduce conceptual complexity.

Best when: ESM runtime semantics become a hard requirement.

---

## 7. Proposed implementation guide for an intern: Sobek ESM spike

This section describes a safe experiment, not an immediate production migration.

### Phase 1: Create a tiny ESM resolver package

Create:

```text
go-go-goja/pkg/sobekesm/resolver.go
```

Pseudocode:

```go
type Resolver struct {
    fsys fs.FS
    native map[string]sobek.ModuleRecord
    cache map[string]cacheEntry
}

type cacheEntry struct {
    record sobek.ModuleRecord
    err error
}

func NewResolver(fsys fs.FS) *Resolver {
    return &Resolver{
        fsys: fsys,
        native: map[string]sobek.ModuleRecord{},
        cache: map[string]cacheEntry{},
    }
}

func (r *Resolver) RegisterNative(specifier string, record sobek.ModuleRecord) {
    r.native[specifier] = record
}

func (r *Resolver) Resolve(ref interface{}, specifier string) (sobek.ModuleRecord, error) {
    if native, ok := r.native[specifier]; ok {
        return native, nil
    }

    path := r.resolvePath(ref, specifier)
    if cached, ok := r.cache[path]; ok {
        return cached.record, cached.err
    }

    b, err := fs.ReadFile(r.fsys, path)
    if err != nil {
        r.cache[path] = cacheEntry{err: err}
        return nil, err
    }

    rec, err := sobek.ParseModule(path, string(b), r.Resolve)
    r.cache[path] = cacheEntry{record: rec, err: err}
    return rec, err
}
```

Tests:

- `main.mjs` imports `./dep.mjs`.
- Circular imports do not duplicate records.
- Missing import returns a useful error.

### Phase 2: Add default-object native module record

Create:

```text
go-go-goja/pkg/sobekesm/native_default.go
```

Pseudocode:

```go
type DefaultNativeModule struct {
    Name string
    Build func(*sobek.Runtime) sobek.Value
}
```

JavaScript test:

```javascript
import tools from "tools";
globalThis.answer = tools.add(2, 3);
```

Go test:

```go
resolver.RegisterNative("tools", sobekesm.NewDefaultNativeModule("tools", func(rt *sobek.Runtime) sobek.Value {
    obj := rt.NewObject()
    _ = obj.Set("add", func(a, b int) int { return a + b })
    return obj
}))
```

### Phase 3: Add named native exports

Create:

```go
type NamedNativeModule struct {
    Name string
    Exports []NativeExport
}

type NativeExport struct {
    Name string
    Build func(*sobek.Runtime) sobek.Value
}
```

JavaScript test:

```javascript
import { add } from "tools";
globalThis.answer = add(2, 3);
```

### Phase 4: Add dynamic import support

Dynamic import needs a queue. For tests:

```go
type EventQueue struct { q chan func() }

vm.SetImportModuleDynamically(func(ref interface{}, specifier sobek.Value, pcap interface{}) {
    queue.Post(func() {
        m, err := resolver.Resolve(ref, specifier.String())
        vm.FinishLoadingImportModule(ref, specifier, pcap, m, err)
    })
})
```

JavaScript:

```javascript
const tools = await import("tools");
globalThis.answer = tools.default.add(2, 3);
```

### Phase 5: Add import.meta support

Use Sobek's hook:

```go
vm.SetGetImportMetaProperties(func(m sobek.ModuleRecord) []sobek.MetaProperty {
    return []sobek.MetaProperty{
        {Key: "url", Value: vm.ToValue(resolver.URLFor(m))},
    }
})
```

Test:

```javascript
globalThis.url = import.meta.url;
```

### Phase 6: Compare with current CommonJS behavior

For each native module pattern, write equivalent tests:

```javascript
// CommonJS
const tools = require("tools");
globalThis.answer = tools.add(2, 3);

// ESM default
import tools from "tools";
globalThis.answer = tools.add(2, 3);

// ESM named
import { add } from "tools";
globalThis.answer = add(2, 3);
```

Use this to decide whether ESM actually reduces user friction.

---

## 8. What not to do first

Do not start with a full replacement of `engine.Factory`. That touches too much:

- require registry,
- event loop,
- runtime owner,
- runtimebridge,
- built-in modules,
- xgoja providers,
- Geppetto,
- jsverbs,
- TypeScript declarations,
- tests.

Do not attempt named ESM exports for every existing module immediately. Many existing modules are object-shaped and dynamic. Start with default-object exports.

Do not assume ESM removes config complexity. GOJA-053 still needs pre-`Module.New` config merging if provider module construction needs command/config/env values.

---

## 9. How this affects GOJA-053 specifically

GOJA-053 is about getting Glazed CLI/config/env values into provider module initialization before `Module.New`.

ESM does not remove that need. A Sobek native ESM provider would still need setup input before it creates a native module record or instance:

```text
xgoja.yaml static module config
  + command/env/config parsed values
  → provider setup config
  → native ESM ModuleRecord factory
  → resolver.RegisterNative(alias, record)
```

The equivalent of `Module.New(ModuleContext)` may become something like:

```go
type ESMModuleFactory func(ESMModuleContext) (sobek.ModuleRecord, error)

type ESMModuleContext struct {
    StartupContext context.Context
    Name string
    As string
    Config json.RawMessage // or SectionValues
    Host HostServices
    RuntimeOwner SobekRuntimeOwner
}
```

So GOJA-053's config design still applies. It would just feed an ESM module factory instead of a CommonJS loader factory.

---

## 10. Decision records

### Decision: Do not migrate xgoja wholesale to Sobek/ESM as a simplification move

- **Context:** xgoja has significant `require()` and module factory machinery. Sobek adds ESM, which may appear to replace `require()` complexity.
- **Options considered:** keep Goja/CommonJS; use bundling-to-CJS; add isolated Sobek spike; full Sobek backend.
- **Decision:** Keep Goja/CommonJS as production default and only pursue an isolated Sobek ESM spike if runtime ESM becomes important.
- **Rationale:** Sobek ESM is experimental, lacks event loop, does not replace `goja_nodejs`, and native modules still need synthetic ModuleRecord machinery.
- **Consequences:** Modern JS/TS users should continue using bundling-to-CJS for now. xgoja can experiment without destabilizing production runtime.
- **Status:** proposed.

### Decision: If experimenting, start with default-object native ESM modules

- **Context:** Existing native modules populate dynamic `module.exports` objects.
- **Options considered:** default-only export; named exports; full CJS/ESM interop bridge.
- **Decision:** Start with default-only native ESM modules.
- **Rationale:** It maps closest to current CommonJS exports and avoids needing static export lists for every existing module.
- **Consequences:** User code writes `import fs from "fs"` instead of `import { readFile } from "fs"` initially.
- **Status:** proposed.

### Decision: Keep GOJA-053 config work independent of ESM backend decisions

- **Context:** Provider module setup config is required before module factories run in either CommonJS or ESM.
- **Options considered:** solve config only in CommonJS path; solve config only after Sobek migration; keep config abstraction backend-agnostic.
- **Decision:** Continue with Glazed `SectionValues` config merging before module factory invocation.
- **Rationale:** The timing problem exists regardless of module syntax.
- **Consequences:** A future ESM backend can reuse the same config merge pipeline.
- **Status:** proposed.

---

## 11. Testing checklist

### Sobek ESM spike tests

- [ ] Static import from source module.
- [ ] Static import from default native module.
- [ ] Static named import from native module.
- [ ] Circular source imports.
- [ ] Circular custom native module imports.
- [ ] Top-level await in source module.
- [ ] Dynamic import of source module.
- [ ] Dynamic import of native module.
- [ ] import.meta.url for source modules.
- [ ] Missing export error.
- [ ] Missing module error.
- [ ] Runtime close cancels outstanding dynamic import work.

### xgoja compatibility tests

- [ ] Existing CommonJS `require("fs")` scripts still work.
- [ ] Existing xgoja generated binaries still build.
- [ ] Existing Geppetto scripts using `require("geppetto")` still work.
- [ ] Bundled CommonJS output from Bun still runs.
- [ ] ESM spike does not require changing `providerapi.Module` yet.

---

## 12. Intern mental model

If you remember only one thing:

```text
CommonJS native module:
  loader mutates module.exports at runtime

ESM native module:
  module record declares exports during link, then module instance supplies binding values during evaluation
```

This difference is why Sobek ESM can help users write `import` syntax, but it does not automatically simplify xgoja's provider module setup.

The current system is complex because xgoja is solving several independent problems:

- which Go provider packages are compiled into the target,
- which modules a runtime profile selects,
- which require alias each module gets,
- what static config each module receives,
- what command/env/config values affect runtime setup,
- how native Go code safely touches the JS runtime,
- how runtime resources are cleaned up,
- how JavaScript resolves dependencies.

ESM mainly changes the last item. It does not eliminate the others.

---

## 13. File reference

### Sobek and prior analysis

- `/home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/04/12/PROJ - Goja vs Sobek Deep Analysis.md`
- `/home/manuel/code/others/sobek/README.md`
- `/home/manuel/code/others/sobek/AGENTS.md`
- `/home/manuel/code/others/sobek/modules.go`
- `/home/manuel/code/others/sobek/modules_sourcetext.go`
- `/home/manuel/code/others/sobek/modules_namespace.go`
- `/home/manuel/code/others/sobek/modules_test.go`
- `/home/manuel/code/others/sobek/modules_integration_test.go`

### Current go-go-goja CommonJS machinery

- `go-go-goja/modules/common.go`
- `go-go-goja/modules/exports.go`
- `go-go-goja/modules/fs/fs.go`
- `go-go-goja/engine/factory.go`
- `go-go-goja/engine/module_specs.go`
- `go-go-goja/engine/module_roots.go`
- `go-go-goja/pkg/xgoja/app/factory.go`
- `go-go-goja/pkg/xgoja/providerapi/module.go`
- `go-go-goja/pkg/xgoja/providerapi/registry.go`
- `geppetto/pkg/js/modules/geppetto/module.go`
- `geppetto/pkg/js/modules/geppetto/provider/provider.go`

### Existing docs worth linking

- `go-go-goja/pkg/doc/bun-goja-bundling-playbook.md`
- `go-go-goja/pkg/doc/02-creating-modules.md`
- `go-go-goja/pkg/doc/16-nodejs-primitives.md`
- `go-go-goja/cmd/xgoja/doc/02-user-guide.md`
- `go-go-goja/cmd/xgoja/doc/04-tutorial-providing-package-and-modules.md`
