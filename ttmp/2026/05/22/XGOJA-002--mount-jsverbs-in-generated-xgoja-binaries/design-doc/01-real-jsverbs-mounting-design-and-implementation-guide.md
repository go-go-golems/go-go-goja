---
Title: Real jsverbs Mounting Design and Implementation Guide
Ticket: XGOJA-002
Status: active
Topics:
    - xgoja
    - jsverbs
    - goja
    - cli
    - glazed
    - documentation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/jsverbs/command.go
      Note: Existing Glazed command wrappers for JS verbs
    - Path: pkg/jsverbs/runtime.go
      Note: Existing jsverbs invocation path and planned direct runtime API
    - Path: pkg/xgoja/app/factory.go
      Note: Xgoja minimal runtime factory to extend with require options
    - Path: pkg/xgoja/app/root.go
      Note: Generated app placeholder verbs command to replace
ExternalSources: []
Summary: Design and implementation guide for mounting configured JS verb repositories as executable commands inside generated xgoja binaries.
LastUpdated: 2026-05-22T19:06:53-04:00
WhatFor: Use when implementing or reviewing real jsverbs command mounting in generated xgoja binaries.
WhenToUse: Read before changing pkg/xgoja/app verb handling, pkg/jsverbs invocation APIs, or generated xgoja runtime behavior.
---


# Real jsverbs Mounting Design and Implementation Guide

## Executive summary

`xgoja` currently proves compile-time Go provider composition: a generated binary imports provider packages, registers native modules, and can evaluate JavaScript that calls `require("...")`. The `verbs` command, however, is still a placeholder. It lists configured verb source IDs rather than scanning JavaScript files and mounting them as executable Glazed/Cobra commands.

Real jsverbs mounting means the generated binary should turn configured JavaScript verb repositories into actual subcommands. A spec that declares a `jsverbs` source and a `commands.jsverbs` runtime profile should produce commands like:

```bash
my-xgoja verbs tools greet --name intern
```

That command should create the configured xgoja runtime profile, register the provider modules selected by the spec, load the JS verb file through the jsverbs source loader, invoke the selected JavaScript function, and emit the result through the existing Glazed command machinery.

## Current state

The current generated app runtime lives under:

- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app`

The current `verbs` command is intentionally minimal. It only prints configured source IDs. It does not scan files, build commands, or invoke JavaScript.

Existing jsverbs support lives under:

- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbs/scan.go` — scans `.js` and `.cjs` files from directories, embedded filesystems, and in-memory sources.
- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbs/command.go` — converts discovered verb specs into Glazed commands.
- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbs/runtime.go` — invokes a verb inside an `engine.Runtime`.
- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbscli/command.go` — scans repositories and mounts commands for the standalone jsverbs CLI path.

The mismatch is runtime shape. `jsverbs.Registry.InvokeInRuntime` currently expects `*engine.Runtime`, while the generated xgoja app currently uses a lighter `*app.JSRuntime` because importing the full engine still hits the repository's existing `goja_nodejs` / `goja` dependency mismatch.

## Proposed solution

Implement a direct jsverbs invocation path that accepts a goja runtime and require module directly:

```go
func (r *Registry) InvokeInGojaRuntime(
    ctx context.Context,
    vm *goja.Runtime,
    req *require.RequireModule,
    verb *VerbSpec,
    parsedValues *values.Values,
) (interface{}, error)
```

Then extend `pkg/xgoja/app.RuntimeFactory` so callers can pass extra `require.Option` values when creating a runtime. The real jsverbs mounting path needs to pass `require.WithLoader(registry.RequireLoader())` so `require("/verb-file.js")` loads the scanned source with the jsverbs overlay injected.

Finally, replace `newVerbsCommand` in `pkg/xgoja/app/root.go` with real mounting:

1. Build a Cobra parent command using `commands.jsverbs.name`.
2. Scan each configured filesystem source with `jsverbs.ScanDir`.
3. For each discovered verb, build a Glazed command with `registry.CommandForVerbWithInvoker`.
4. Use an invoker that creates an xgoja runtime for `commands.jsverbs.runtime` and invokes the verb through the new direct jsverbs invocation API.
5. Attach the generated Glazed commands to the parent with `glazed/pkg/cli.AddCommandsToRootCommand`.

## Scope for this ticket

In scope:

- Filesystem JS verb sources from `jsverbs[].path`.
- Runtime profile selection through `commands.jsverbs.runtime`.
- JS verbs requiring xgoja provider modules.
- Synchronous JS verb execution and basic promise polling compatible with the existing jsverbs behavior.
- Tests using the fixture xgoja provider module.

Out of scope for the first pass:

- Copying and embedding `embed: true` verb directories into generated source files.
- Package-provided verb sources with non-nil `fs.FS`.
- Full interactive REPL integration.
- Replacing xgoja's minimal runtime with `engine.Factory` before the dependency mismatch is fixed.

## Implementation plan

### Task 1: Add direct jsverbs invocation

Add a public method in `pkg/jsverbs/runtime.go` that reuses the existing argument binding and overlay behavior but does not require `engine.Runtime`.

Pseudocode:

```go
func (r *Registry) InvokeInGojaRuntime(ctx context.Context, vm *goja.Runtime, req *require.RequireModule, verb *VerbSpec, vals *values.Values) (any, error) {
    plan := buildVerbBindingPlan(r, verb)
    args := buildArguments(vals, plan, r.RootDir)
    req.Require(verb.File.ModulePath)
    fn := globalThis.__glazedVerbRegistry[verb.File.ModulePath][verb.FunctionName]
    result := fn(undefined, args...)
    if promise, ok := result.Export().(*goja.Promise); ok { return waitForPromiseDirect(ctx, promise) }
    return result.Export(), nil
}
```

### Task 2: Add require options to xgoja runtime factory

Change `RuntimeFactory.NewRuntime` to accept optional `require.Option` values:

```go
func (f *RuntimeFactory) NewRuntime(ctx context.Context, profile string, opts ...require.Option) (*JSRuntime, error)
```

The eval path can call it without options. The jsverbs path calls it with the scanned registry loader.

### Task 3: Mount filesystem sources as commands

Replace the placeholder verb-source listing command with real command construction. Keep a `list` subcommand if useful, but the important behavior is executable verbs.

### Task 4: Add tests

Add an app-level test that creates a temp verb directory with a JS file such as:

```js
__package__({ name: "tools" })
__verb__("greet", { name: "greet", output: "text", fields: { name: { type: "string", required: true } } })
function greet(name) {
  const hello = require("hello")
  return hello.greet(name)
}
```

Then assert:

```bash
verbs tools greet --name intern
```

prints:

```text
hello intern
```

## Risks and review points

- The direct invocation path duplicates some behavior from `InvokeInRuntime`. Keep the shared pieces small and avoid diverging argument binding semantics.
- Async promise behavior may be limited without the full engine event loop. Record that limitation clearly.
- Glazed command mounting can route output through framework processors rather than the root's output writer. Tests should validate execution and, where possible, output behavior.
- Embedded verb sources need a separate generation task because they require copying source trees and adding generated `go:embed` declarations.

## Acceptance criteria

- A generated xgoja app can mount a filesystem jsverbs source as commands.
- A mounted JS verb can call `require("hello")` for a module provided by an xgoja provider package.
- Targeted tests pass.
- Diary and changelog explain implementation details and limitations.

## References

- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbs/runtime.go`
- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbs/command.go`
- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbs/scan.go`
- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/root.go`
- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/factory.go`
