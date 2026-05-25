---
Title: css-visual-diff xgoja provider and command provider implementation guide
Ticket: XGOJA-014
Status: active
Topics:
  - xgoja
  - command-providers
  - css-visual-diff
  - jsverbs
DocType: design-doc
Intent: implementation
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Design for adding a css-visual-diff xgoja package provider and mounting its workflow verbs as command providers."
LastUpdated: 2026-05-25T20:05:00-04:00
WhatFor: "Guide the css-visual-diff portion of XGOJA-014."
WhenToUse: "When adding or reviewing css-visual-diff xgoja provider support."
---

# css-visual-diff xgoja provider and command provider implementation guide

## Goal

Make css-visual-diff usable from generated xgoja binaries in two ways:

1. as runtime modules (`require("css-visual-diff")`, and optionally compatibility modules like `diff`/`report`);
2. as package-owned workflow commands loaded from built-in and external jsverb repositories.

## Current state

- `internal/cssvisualdiff/jsapi` installs the native `css-visual-diff` module, but only through an engine runtime-module registrar.
- `internal/cssvisualdiff/dsl` registers helper modules (`diff`, `css-visual-diff`, `report`) and scans embedded workflow scripts.
- `internal/cssvisualdiff/verbcli` already discovers repositories and turns verbs into Glazed commands, but it builds runtimes with the css-visual-diff local factory.
- There is no public `pkg/xgoja/provider` package yet.

## Provider shape

Add `pkg/xgoja/provider` with package ID `css-visual-diff`:

```go
providerapi.Module{Name: "css-visual-diff", DefaultAs: "css-visual-diff", ...}
providerapi.Module{Name: "diff", DefaultAs: "diff", ...}
providerapi.Module{Name: "report", DefaultAs: "report", ...}
providerapi.CommandSetProvider{Name: "verbs", DefaultMount: "css-diff", ...}
```

Provider config:

```yaml
commandProviders:
  - package: css-visual-diff
    name: verbs
    mount: css
    runtimeProfile: browser
    config:
      repositories:
        - ./examples/verbs
```

## Module loader refactor

The xgoja module provider API returns `require.ModuleLoader`, while the existing css-visual-diff module installer expects an `engine.RuntimeModuleContext`. Extract module installation helpers that can be called from either path:

- Existing registrar path: build a real `RuntimeModuleContext` and register modules into the require registry.
- xgoja provider path: create loaders that use `runtimebridge.Lookup(vm)` to recover runtime owner/context bindings and then install exports into the current module object.

This is enough for async browser APIs because they only require `Context` and `Owner`. If future APIs require `AddCloser`, add a package capability or runtime-aware module spec later.

## Command provider behavior

Expose jsverb workflow commands through `css-visual-diff.verbs`:

- Decode `repositories` from provider config.
- Reuse `verbcli.ScanRepositories` and `verbcli.CollectDiscoveredVerbs`.
- Export a `verbcli.NewCommandsWithInvokerFactory(...)` helper so the provider can supply an xgoja-aware invoker.
- For each invocation, call `ctx.RuntimeFactory.NewRuntime(ctx, ctx.RuntimeProfile, require.WithLoader(repo.Registry.RequireLoader()))` when available, then invoke the selected verb in that runtime.
- Fall back to the standalone css-visual-diff runtime factory only if no xgoja runtime factory is available.

## Tests

Add tests for:

1. Provider registration resolves modules and command provider.
2. `css-visual-diff` loader can be installed in an xgoja-created runtime and exposes basic sync APIs.
3. Command provider builds at least the built-in verb commands without starting a browser.
4. Runtime-profile name from `CommandSetContext.RuntimeProfile` is used when constructing runtimes.

## Validation

Run:

```bash
go test ./pkg/xgoja/provider ./internal/cssvisualdiff/jsapi ./internal/cssvisualdiff/verbcli ./internal/cssvisualdiff/dsl -count=1
```

Use published `github.com/go-go-golems/go-go-goja v0.5.0` to avoid local replace directives for provider APIs.
