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
