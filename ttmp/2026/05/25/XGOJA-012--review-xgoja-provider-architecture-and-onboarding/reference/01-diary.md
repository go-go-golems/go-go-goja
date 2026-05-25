---
Title: XGOJA-012 Review Diary
Ticket: XGOJA-012
Status: active
DocType: diary
Intent: investigation
---

# XGOJA-012 Review Diary

## Step 1: Ticket creation and review framing

The review ticket was created to take a large step back from the rapid XGOJA-007 through XGOJA-011 implementation sequence. The goal is not to implement another provider immediately, but to explain and assess what exists: the provider API, runtime profiles, module capabilities, command providers, module sections, runtime initializers, lifecycle hooks, generated examples, and the Discord bot integration.

The requested audience is a new intern taking over the system from the previous implementation agent. The report therefore needs two layers:

1. A clear, textbook-style explanation of how xgoja works and how all the introduced abstractions fit together.
2. A code-quality review that calls out where the architecture is solid, where it is confusing, where it may be over-abstracted, and where documentation/onboarding should improve.

## Step 2: Source inventory

Commands used:

```bash
rg --files pkg/xgoja cmd/xgoja modules/express pkg/gojahttp examples/xgoja | sort
rg -n "type .*Capability|type CommandSet|type Module|sectionsForRuntimeProfile|InitRuntimeFromSections|AttachCommandProviders|DecodeSectionInto" pkg/xgoja cmd/xgoja modules/express pkg/gojahttp -S
rg -n "CommandSetProvider|collectModuleSections|xgojaBotRuntimeFactory|SetOutboundOps|channels.list|ChannelList|NewRuntimeForVerb|WithRuntimeFactory" pkg/xgoja internal/jsdiscord internal/bot pkg/botcli examples/xgoja -S
```

Key source areas reviewed:

- `go-go-goja/pkg/xgoja/providerapi/*`
- `go-go-goja/pkg/xgoja/app/*`
- `go-go-goja/pkg/xgoja/providers/{core,host,http}`
- `go-go-goja/cmd/xgoja/doc/*`
- `go-go-goja/examples/xgoja/*`
- `go-go-goja/modules/express` and `go-go-goja/pkg/gojahttp`
- `discord-bot/pkg/xgoja/provider`
- `discord-bot/internal/jsdiscord`
- `discord-bot/examples/xgoja/discord-bot-provider`

## Step 3: Drafted report

The report was written as `design/01-xgoja-provider-architecture-review-and-onboarding-guide.md`. It covers:

- the core mental model;
- the API map;
- runtime flows for built-ins and provider-owned commands;
- the Discord bot case study;
- what is solid;
- what is confusing or messy;
- concrete cleanup opportunities with file references and solution sketches;
- documentation and onboarding recommendations;
- suggested implementation sequence for the next maintainer.

## Step 4: Added detailed follow-up implementation plan

The user clarified the desired direction for the follow-up work:

- Keep capabilities conceptually understood, but simplify naming and public API where possible.
- Explain `RuntimeFactory`: what it is, why it was temporarily typed as `any`, how it is created, and concrete examples from built-in xgoja and Discord adapter paths.
- Move duplicated section aggregation/runtime initialization helper logic into a shared provider-facing utility.
- Use Option A for capability naming: rename the package-scoped capability concept to `PackageCapability`.
- Remove `ComponentInitializerCapability` and `InitializedModule` unless a real provider needs them.
- Clarify discovery-vs-execution side effects, especially why `InitRuntimeFromSections` can currently see `vals == nil`.
- Fix stale provider docs.
- Rename/reorganize examples as a numbered learning path.
- Reorganize xgoja docs into overview, user guide/reference, and tutorials.

The ticket task list was expanded into a detailed multi-phase checklist so the next session can start from the task file without reconstructing context.

## Step 5: Typed provider-facing RuntimeFactory

### Intent

The user approved a hard cutover to a concrete provider-facing runtime factory shape:

```go
type RuntimeFactory interface {
    NewRuntime(ctx context.Context, profile string, opts ...require.Option) (*engine.Runtime, error)
}
```

This removes the temporary `any` field from `providerapi.CommandSetContext` and makes command-provider authors able to see exactly what runtime service xgoja offers.

### What changed

In `go-go-goja/pkg/xgoja/providerapi/commands.go`:

- Added `providerapi.RuntimeFactory`.
- Changed `CommandSetContext.RuntimeFactory` from `any` to `RuntimeFactory`.

In `discord-bot/pkg/xgoja/provider/provider.go`:

- Removed the local `xgojaRuntimeFactory` interface.
- Removed the runtime factory type assertion.
- The Discord command provider now uses `ctx.RuntimeFactory` directly when constructing `xgojaBotRuntimeFactory`.

### Why this is simpler

Before this change, the command-provider API said only “there may be some runtime factory-like object here.” The real contract was hidden in the Discord adapter as a local interface. That made the API harder to learn and easy to misuse.

After this change, the API says directly: command providers can create xgoja runtimes by calling `NewRuntime(ctx, profile, opts...)`.

### Validation

```bash
go test ./pkg/xgoja/providerapi ./pkg/xgoja/app -count=1
go test ./pkg/xgoja/provider -count=1   # in discord-bot
```

Result: passed.

## Step 6: Extracted providerutil section/init helpers

### Intent

The review identified duplicated logic between xgoja built-in commands and the Discord command provider. Both need to:

1. collect module-provided Glazed sections from selected module descriptors;
2. reject duplicate or malformed section slugs; and
3. run selected runtime initializer capabilities after runtime creation.

Because external adapters need this behavior too, the shared logic should not remain private to `pkg/xgoja/app`.

### What changed

Added `go-go-goja/pkg/xgoja/providerutil` with:

- `CollectConfigSections(...)`
- `AppendUniqueSections(...)`
- `InitRuntimeFromSections(...)`

Updated `go-go-goja/pkg/xgoja/app/module_sections.go` to use `providerutil` for config-section collection, duplicate checks, and runtime initializer invocation.

Updated `discord-bot/pkg/xgoja/provider/provider.go` to use `providerutil` instead of maintaining local copies of `collectModuleSections` and initializer walking logic.

### Tests added

`go-go-goja/pkg/xgoja/providerutil/sections_test.go` covers:

- duplicate section slug rejection;
- nil section rejection;
- empty slug rejection;
- runtime initializer invocation;
- runtime initializer error wrapping; and
- no-op behavior when there are no runtime initializers.

### Validation

```bash
go test ./pkg/xgoja/providerutil ./pkg/xgoja/app -count=1
go test ./pkg/xgoja/provider -count=1   # in discord-bot
```

Result: passed.

## Step 7: Renamed package-scoped capabilities and removed component initializer abstraction

### Intent

The review found that `ModuleCapability` was misleading: capabilities are registered at provider package level through `WithCapability(...)`, not on individual `Module` values. The user selected the hard-cutover route with no compatibility wrappers.

The review also found that `ComponentInitializerCapability` and `InitializedModule` were not used by a real provider and added conceptual weight.

### What changed

In `go-go-goja/pkg/xgoja/providerapi`:

- `ModuleCapability` -> `PackageCapability`.
- `WithCapability(...)` -> `WithPackageCapability(...)`.
- `ResolveCapabilities(...)` -> `ResolvePackageCapabilities(...)`.
- `ModuleDescriptor.Capabilities` -> `ModuleDescriptor.PackageCapabilities`.
- `Package.Capabilities` -> `Package.PackageCapabilities`.
- Removed `ComponentInitializerCapability`.
- Removed `InitializedModule`.

Updated uses in:

- `pkg/xgoja/app`.
- `pkg/xgoja/providerutil`.
- `pkg/xgoja/providers/http`.
- `pkg/xgoja/testprovider`.
- `pkg/xgoja/providerapi` tests.
- `discord-bot/pkg/xgoja/provider` tests.

### Why this is simpler

The API now says what it does. Capabilities are package capabilities that xgoja attaches to selected module descriptors from that package. There is no longer a misleading “module capability” name suggesting module-local registration.

Removing component initializers narrows the public API to concepts with real users: config sections, runtime initializers, runtime handles/closers, modules, and command sets.

### Validation

```bash
go test ./pkg/xgoja/providerapi ./pkg/xgoja/providerutil ./pkg/xgoja/app ./pkg/xgoja/providers/http -count=1
go test ./pkg/xgoja/provider -count=1   # in discord-bot
```

Result: passed.

## Step 8: Tested discovery-vs-execution HTTP settings behavior

### Intent

The HTTP provider uses an implicit convention: when `InitRuntimeFromSections` receives `vals == nil`, the runtime is being created without parsed command values. This can happen during discovery/preload paths in host-owned runners. Providers should avoid irreversible side effects in that mode.

For the HTTP provider this means: do not enable the HTTP server when values are nil.

### What changed

Added
 focused tests in `pkg/xgoja/providers/http/http_test.go`:

- `TestCapabilityDisablesHTTPWhenValuesAreNil` verifies nil values force disabled settings.
- `TestCapabilityEnablesHTTPByDefaultWhenValuesArePresent` verifies ordinary command values preserve the default `enabled=true` and default listen address.
- `TestCapabilityAllowsExplicitHTTPDisable` verifies explicit disable wins even when parsed command values exist.

### Validation

```bash
go test ./pkg/xgoja/providers/http -count=1
```

Result: passed.

## Step 9: Updated stale provider documentation signatures

### Intent

The provider docs still showed old capability signatures and referenced component initializers after the API cleanup. That would mislead the next provider author.

### What changed

Updated:

- `cmd/xgoja/doc/04-providers.md`
  - `ConfigSections(providerapi.SectionContext)` signature.
  - `InitRuntimeFromSections(context.Context, *values.Values, providerapi.RuntimeHandle)` argument order.
  - `WithPackageCapability(...)` registration helper.
  - nil-values discovery/preload side-effect convention.
  - `providerutil` as the shared helper package.
  - typed `providerapi.RuntimeFactory` for command providers.
- `cmd/xgoja/doc/02-buildspec.md`
  - command providers now mention selected module descriptors plus runtime initializers/runtime factory, not component initializers.

### Validation

Ran `go test ./cmd/xgoja/... -count=1`; result: passed. Also grepped docs and `pkg/xgoja` for stale capability/component-initializer names; result: clean.

## Step 10: Added runtime factory regression coverage and refreshed the review/report docs

### Intent

After typing `providerapi.RuntimeFactory`, I wanted a direct regression test that command set providers receive a non-nil typed factory and can use it to instantiate the selected runtime profile. I also needed to update the long-form XGOJA-012 report so it no longer described the old API as current reality.

### What changed

Code/test changes:

- Extended `pkg/xgoja/app/command_providers_test.go`.
- The command provider test now asserts `ctx.RuntimeFactory != nil`.
- It calls `ctx.RuntimeFactory.NewRuntime(ctx.Context, ctx.RuntimeProfile)` and verifies the returned runtime and VM are non-nil.
- It closes the runtime after the assertion.
- Cleaned the `WithPackageCapability` comment in `providerapi/capabilities.go` to avoid saying “module capability”.

Docs/report changes:

- Updated the XGOJA-012 architecture review to describe the implemented typed runtime factory, `PackageCapability`, and `providerutil` changes.
- Removed the old domain-object initializer concept as a public recommendation and replaced it with a note that provider-owned command code should own non-runtime Go state directly.
- Updated the provider guide with a decision table covering:
  - simple modules,
  - static module config,
  - command-time config sections,
  - runtime initializers,
  - runtime closers,
  - command set providers.
- Fixed the command set provider example signature to `New: func(c providerapi.CommandSetContext) ...`.
- Added concrete RuntimeFactory explanation for built-ins and the Discord adapter.

### Validation

```bash
go test ./pkg/xgoja/app ./pkg/xgoja/providerapi ./cmd/xgoja/... -count=1
```

Result: passed.

## Step 11: Numbered xgoja examples as an onboarding curriculum

### Intent

The examples were useful but read like a smoke-test pile. Phase 7 called for turning them into a numbered path that starts with simple provider composition and ends with JS verb distribution variants.

### What changed

Renamed the examples in one breaking pass, without compatibility directories:

- `core-provider` -> `01-core-provider`
- `host-provider` -> `02-host-provider`
- `multiple-runtimes` -> `03-multiple-runtimes`
- `module-sections` -> `04-module-sections`
- `command-provider` -> `05-command-provider`
- `runtime-filesystem` -> `06-runtime-filesystem`
- `embedded-jsverbs` -> `07-embedded-jsverbs`
- `provider-shipped-jsverbs` -> `08-provider-shipped-jsverbs`

Updated references in:

- `examples/xgoja/README.md`
- `cmd/xgoja/doc/04-providers.md`
- `cmd/xgoja/cmd_build.go`
- per-example README command snippets

The new `examples/xgoja/README.md` explains the examples as a learning path and includes an all-example smoke loop using the numbered names.

### Validation

Ran all numbered example smokes:

```bash
for dir in 01-core-provider 02-host-provider 03-multiple-runtimes 04-module-sections 05-command-provider 06-runtime-filesystem 07-embedded-jsverbs 08-provider-shipped-jsverbs; do
  make -C examples/xgoja/$dir smoke
done
```

Result: passed.

## Step 12: Reorganized xgoja help docs into overview, user guide, and tutorials

### Intent

Phase 8 called for moving from numbered historical docs to a clearer help structure: overview, user guide/reference, and focused tutorials.

### What changed

Renamed and retitled the bundled help pages:

- `01-overview.md` remains `overview`.
- `02-buildspec.md` -> `02-user-guide.md` with slug `user-guide`.
- `03-tutorial.md` -> `03-tutorial-using-xgoja-yaml.md` with slug `tutorial-using-xgoja-yaml`.
- `04-providers.md` -> `04-tutorial-providing-package-and-modules.md` with slug `tutorial-providing-package-and-modules`.
- Added `05-tutorial-providing-commands.md` with slug `tutorial-providing-commands`.
- Added `06-buildspec-reference.md` as a quick-reference pointer to the full user guide.

Updated references in provider docs and the xgoja help test to use the new `user-guide` topic. The embed pattern already loads `*.md`, so no code-level doc registration change was needed beyond filenames/slugs.

### Validation

```bash
go test ./cmd/xgoja/... -count=1
GOWORK=off go run ./cmd/xgoja help user-guide
GOWORK=off go run ./cmd/xgoja help tutorial-providing-commands
```

Result: passed.

## Step 13: Broad validation after cleanup phases

### Commands

```bash
go test ./pkg/xgoja/provider ./internal/jsdiscord ./pkg/botcli -count=1   # in discord-bot
go test ./pkg/xgoja/... ./cmd/xgoja/... -count=1                         # in go-go-goja
```

### Result

Both passed.

## Step 14: Final docmgr doctor and reMarkable upload

### Commands

```bash
docmgr doctor --ticket XGOJA-012 --stale-after 30
remarquee upload bundle \
  design/01-xgoja-provider-architecture-review-and-onboarding-guide.md \
  tasks.md \
  changelog.md \
  reference/01-diary.md \
  --name "XGOJA-012 xgoja provider cleanup final" \
  --remote-dir "/ai/2026/05/25/XGOJA-012" \
  --toc-depth 2 \
  --non-interactive
```

### Result

- `docmgr doctor`: passed.
- reMarkable upload: `OK: uploaded XGOJA-012 xgoja provider cleanup final.pdf -> /ai/2026/05/25/XGOJA-012`.
