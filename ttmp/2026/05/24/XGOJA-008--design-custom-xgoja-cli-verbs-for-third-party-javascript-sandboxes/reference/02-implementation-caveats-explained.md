---
Title: Implementation caveats explained
Ticket: XGOJA-008
Status: complete
Topics:
    - xgoja
    - goja
    - providers
    - jsverbs
    - command-registration
    - architecture
    - geppetto
    - loupedeck
    - go-minitrace
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/pkg/xgoja/app/command_providers.go
      Note: Contains command provider mounting and mount-prefix mutation behavior
    - Path: go-go-goja/pkg/xgoja/app/host.go
    - Path: go-go-goja/pkg/xgoja/app/module_sections.go
      Note: Contains runtime-profile section aggregation and package capability de-duplication
    - Path: go-go-goja/pkg/xgoja/app/root.go
      Note: Contains generated root/jsverbs command construction relevant to built-in aggregation caveats
    - Path: go-go-goja/pkg/xgoja/app/run.go
    - Path: go-go-goja/pkg/xgoja/app/tui.go
    - Path: go-go-goja/pkg/xgoja/providerapi/commands.go
      Note: Defines CommandSetContext including the current RuntimeFactory any field
ExternalSources: []
Summary: 'Explains remaining XGOJA-008 caveats: raw Cobra eval, static runtime schemas, mutable mount prefixing, and RuntimeFactory typing.'
LastUpdated: 2026-05-25T11:29:20-04:00
WhatFor: Explain known implementation caveats left after the first XGOJA-008 rollout.
WhenToUse: Use before extending eval, command-provider mounting, runtime profile overrides, or real provider adapters.
---


# Implementation caveats explained

## Goal

Explain the caveats that remained after the first XGOJA-008 implementation pass:

1. `eval` is still a raw Cobra command.
2. Command schemas are static for the configured runtime profile.
3. Command provider mount prefixing mutates command descriptions in place.
4. `CommandSetContext.RuntimeFactory` is currently typed as `any`.

These are not release blockers for the implemented provider-capability work. They are design seams to understand before building the next layer of provider integrations.

## Context

XGOJA-008 added two related extension surfaces:

- **Module capabilities**: providers can expose Glazed sections and runtime/component initializers.
- **Command set providers**: providers can return Glazed commands that xgoja mounts into the generated CLI.

The implementation now wires module-provided sections into generated `run`, `repl`, and `jsverbs` commands. It also supports `commandProviders` in `xgoja.yaml` and generated binaries.

The remaining caveats mostly come from older xgoja command paths or deliberately conservative API choices made before real package adapters exist.

## Quick Reference

| Caveat | What it means | Why it exists | User-visible effect | Recommended next step |
| --- | --- | --- | --- | --- |
| Raw Cobra `eval` | `eval` is built directly as a Cobra command, not as a Glazed command description. | `eval` predates the module-section aggregation design. | Provider module sections like `--fixture-value` are available on `run`, `repl`, and `jsverbs`, but not on `eval`. | Convert `eval` to a Glazed `BareCommand` and reuse the same section aggregation/init helpers. |
| Static runtime schemas | xgoja builds command flags from the runtime profile configured in `xgoja.yaml`, not from runtime overrides supplied later at invocation time. | CLI flags must be known when Cobra/Glazed constructs the command tree. | If a future command supports `--runtime other`, its available provider section flags will still reflect the configured default runtime, not `other`. | Treat runtime profile as schema-defining, or generate subcommands per runtime if dynamic schemas are needed. |
| Mutable mount prefixing | `applyMountToCommands` modifies returned command descriptions to add mount parents. | It was the simplest way to mount Glazed commands into a parent path for the first implementation slice. | A provider that returns the same command object repeatedly could accumulate or leak parent prefixes. | Providers should return fresh command objects today; xgoja should clone descriptions before mutation later. |
| `RuntimeFactory any` | `CommandSetContext.RuntimeFactory` is intentionally untyped. | The correct minimal interface was unclear before adapting real packages like loupedeck or discord-bot. | Provider authors do not get a strongly typed runtime-factory API from the context yet. | Replace `any` with a narrow interface once real command providers prove which methods are needed. |

## Caveat 1: raw Cobra `eval`

### What “raw Cobra” means

Most newly extended commands now follow this shape:

1. Build a Glazed `CommandDescription`.
2. Append provider module sections to that description.
3. Let Glazed/Cobra expose the fields as flags.
4. Decode parsed `values.Values`.
5. Run provider runtime initializers before JavaScript evaluation.

`eval` does not follow that path yet. It is still attached directly as a Cobra command. That means it does not naturally receive Glazed sections and does not participate in the new `values.Values.DecodeSectionInto` initializer flow.

### Why it matters

If a provider exposes this section:

```text
fixture section -> --fixture-value
```

then these work today:

```bash
my-xgoja run script.js --fixture-value run-ok
my-xgoja verbs tools check-fixture --fixture-value verb-ok
my-xgoja repl --fixture-value repl-ok
```

But this does not work yet:

```bash
my-xgoja eval 'globalThis.fixtureValue' --fixture-value eval-ok
```

The flag is not registered on `eval`, because `eval` has not been converted to the shared Glazed command path.

### How to fix it

Convert `eval` into a Glazed `BareCommand` that mirrors `run`:

1. Build `evalCommand` with a `CommandDescription`.
2. Call the same helper used by `run`/`repl`/`jsverbs` to append runtime-profile module sections.
3. Parse values through Glazed.
4. Create the runtime.
5. Call `initRuntimeFromSections(...)`.
6. Evaluate the JavaScript expression.

The key is not “add more Cobra flags manually.” The key is to move `eval` into the same Glazed-based command construction model as the other built-ins.

## Caveat 2: static runtime-profile schemas

### What “static schema” means

The command tree is built before a user invokes a specific command. Cobra needs to know available flags during command construction. Glazed therefore builds a command schema from the runtime profile selected in the buildspec.

Example:

```yaml
commands:
  run:
    enabled: true
    runtime: safe
```

If runtime `safe` selects provider modules with a `safe-*` section, `run --help` can show `--safe-*` flags.

If a future `run` command also allows this:

```bash
my-xgoja run --runtime host script.js
```

then the flags cannot automatically change after `--runtime host` is parsed, because the command parser has already been constructed.

### User-visible consequence

A runtime override can change which modules are loaded, but it cannot safely change which flags exist on the command unless xgoja designs a different CLI shape.

This means the following can become confusing if implemented naively:

```bash
my-xgoja run --runtime host script.js --host-token abc
```

If the configured schema came from `safe`, `--host-token` is unknown even though runtime `host` would have wanted it.

### Recommended design options

#### Option A: runtime profile defines the schema

Keep today’s model: the buildspec-selected runtime defines the command flags. Runtime overrides either remain unsupported or are limited to profiles with compatible section schemas.

This is simple and predictable.

#### Option B: generate one subcommand per runtime

Generate commands such as:

```bash
my-xgoja run safe script.js --safe-value x
my-xgoja run host script.js --host-token y
```

Each subcommand has a static schema, but the user can choose among schemas at the command level.

#### Option C: use generic key/value config for dynamic runtimes

Allow dynamic runtime overrides, but pass provider config through generic repeated flags:

```bash
my-xgoja run --runtime host --set host.token=abc script.js
```

This avoids dynamic Cobra schemas but loses typed Glazed help for provider sections.

### Recommendation

Prefer option A for now. If users need frequent runtime switching with different provider sections, add option B later.

## Caveat 3: mutable command-provider mount prefixing

### What mutates

Command providers return Glazed commands. xgoja supports mounting those commands under a path from `commandProviders[].mount`.

Example:

```yaml
commandProviders:
  - id: fixture-tools
    package: fixture
    name: tools
    mount: fixture
```

The provider might return a command named `rows`. xgoja adjusts its command description so the generated CLI exposes:

```bash
my-xgoja fixture rows
```

The first implementation does this by modifying the returned command description in place.

### Why this is acceptable for now

The provider API expects the provider factory to construct commands when called. If each `New(...)` call returns fresh command objects, mutation is local to that generated host.

That is the normal Glazed command construction pattern.

### What can go wrong

If a provider caches and reuses command objects, mutation can leak:

1. First host mounts command under `fixture`.
2. xgoja mutates the command description to add parent `fixture`.
3. Second host asks for the same cached command and mounts under `tools`.
4. The command might now carry stale or duplicated parent metadata.

This is especially risky in tests or long-lived processes that build multiple hosts from the same provider instance.

### Recommended rule for provider authors

Do this:

```go
New: func(ctx context.Context, c providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
    cmd, err := newFreshCommand()
    if err != nil {
        return nil, err
    }
    return &providerapi.CommandSet{Commands: []cmds.Command{cmd}}, nil
}
```

Avoid this:

```go
var cached cmds.Command

New: func(ctx context.Context, c providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
    return &providerapi.CommandSet{Commands: []cmds.Command{cached}}, nil
}
```

### Recommended xgoja fix

Before applying mount prefixes, xgoja should clone command descriptions instead of mutating provider-owned objects. The current implementation is fine for the first rollout, but cloning is the safer long-term host-boundary behavior.

## Caveat 4: `CommandSetContext.RuntimeFactory any`

### What it means

`providerapi.CommandSetContext` currently includes a runtime factory field typed as `any` rather than a narrow interface.

Conceptually, command providers may need to create JavaScript runtimes or inspect runtime-profile state. But the exact API they should receive is not obvious yet.

### Why it was left loose

The implementation intentionally avoided guessing too early. Real providers will clarify the needs:

- loupedeck may need device/controller services plus JS runtime creation.
- discord-bot may need bot/session services and interaction lifecycle hooks.
- css-visual-diff may need screenshot/diff services.
- go-minitrace may need trace database/query context.

A too-large public interface would freeze accidental internals. A too-small one might immediately need breaking changes.

### User-visible consequence

This is mostly a provider-author API caveat, not an end-user caveat. Generated binaries work. But provider authors cannot yet rely on a stable typed runtime-factory interface from `CommandSetContext`.

### Recommended next step

After one or two real adapters are implemented, replace `any` with a minimal interface, for example something like:

```go
type RuntimeFactory interface {
    NewRuntime(ctx context.Context, profile string) (*engine.Runtime, error)
    SelectedModules(profile string) ([]providerapi.ModuleDescriptor, error)
}
```

Do not adopt that exact interface blindly. First implement a real provider adapter and record what methods it actually needs.

## Practical review checklist

Before extending XGOJA-008 further, ask:

- Does this command need module-provided flags?
  - If yes, it should be Glazed-based, not raw Cobra.
- Does this command allow selecting a runtime dynamically?
  - If yes, how will its flags remain statically knowable?
- Does this provider command factory return fresh command objects?
  - If no, mount prefix mutation can leak.
- Does this provider truly need the runtime factory?
  - If yes, note the exact method shape before widening the public API.

## Usage Examples

### Verify the caveat with `eval`

Use the module-sections example:

```bash
make -C examples/xgoja/module-sections build
examples/xgoja/module-sections/dist/module-sections run \
  examples/xgoja/module-sections/scripts/check-fixture.js \
  --fixture-value run-ok
```

The `run` command accepts `--fixture-value` because it participates in module-section aggregation.

By contrast, `eval` does not currently expose that provider section flag.

### Verify provider command sections

Use the command-provider example:

```bash
make -C examples/xgoja/command-provider build
examples/xgoja/command-provider/dist/command-provider fixture rows \
  --message rows-ok \
  --fixture-value shared \
  --output json
```

This proves provider-shipped Glazed commands can consume provider module sections.

## Related

- `design-doc/01-custom-xgoja-cli-verbs-for-third-party-javascript-sandboxes.md`
- `reference/01-diary.md`
- `cmd/xgoja/doc/02-buildspec.md`
- `cmd/xgoja/doc/04-providers.md`
- `examples/xgoja/module-sections/README.md`
- `examples/xgoja/command-provider/README.md`
