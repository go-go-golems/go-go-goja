---
Title: loupedeck xgoja command provider implementation guide
Ticket: XGOJA-014
Status: active
Topics:
  - xgoja
  - command-providers
  - loupedeck
  - jsverbs
DocType: design-doc
Intent: implementation
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Design for exposing loupedeck scene and verb commands through an xgoja CommandSetProvider."
LastUpdated: 2026-05-25T20:05:00-04:00
WhatFor: "Guide the loupedeck portion of XGOJA-014."
WhenToUse: "When adding or reviewing the loupedeck xgoja command provider."
---

# loupedeck xgoja command provider implementation guide

## Goal

Expose package-owned Loupedeck scene commands in generated xgoja binaries. A generated binary that already mounts the safe `gfx`/`easing` modules should also be able to mount a `loupedeck.scenes` command provider and run scene files or annotated scene verbs without hand-recreating the standalone `loupedeck` CLI.

## Current state

- `runtime/js/provider` registers safe `easing` and `gfx` modules for xgoja.
- `pkg/xgoja/provider` re-exports the runtime provider.
- `cmd/loupedeck/cmds/run` exposes a Glazed `run` command that executes a raw JavaScript scene on real hardware.
- `cmd/loupedeck/cmds/verbs` discovers annotated jsverbs repositories and wraps them as Glazed commands, but its command-list constructor is currently internal to the package.

## Provider shape

Add a command provider to the existing `loupedeck` package registration:

```go
providerapi.CommandSetProvider{
  Name:         "scenes",
  DefaultMount: "loupedeck",
  Description:  "Run Loupedeck JavaScript scenes and annotated scene verbs",
  New:          newScenesCommandSet,
}
```

Provider config:

```yaml
commandProviders:
  - package: loupedeck
    name: scenes
    mount: loupe
    config:
      includeRun: true
      repositories:
        - ./examples/js
```

Implementation notes:

- Export a small command-list helper from `cmd/loupedeck/cmds/verbs`, e.g. `NewCommands(bootstrap Bootstrap) ([]cmds.Command, error)`, so the command provider can reuse discovery without Cobra.
- Build a command set containing:
  - optional `run` command from `cmd/loupedeck/cmds/run.NewCommand()`;
  - discovered annotated verb commands from the helper.
- Decode config into a typed struct and convert repository paths into the existing `verbs.Bootstrap` structure.
- Preserve existing `Parents` from jsverbs command descriptions; xgoja will prepend the mount.
- This provider intentionally remains package-owned and hardware-aware. It does not turn xgoja into a Loupedeck host; it merely makes Loupedeck CLI capabilities mountable in generated binaries.

## Runtime behavior

The first implementation should reuse the existing Loupedeck live-scene runtime path. That path knows about device sessions, timing flags, render flushing, and signal handling. A future integration could add a command-provider runtime factory that calls `ctx.RuntimeFactory.NewRuntime(...)`, but the immediate goal is to make the existing package-owned commands available without changing device semantics.

## Tests

Add tests in `runtime/js/provider` or `pkg/xgoja/provider`:

1. Registry resolution finds command provider `loupedeck.scenes`.
2. Provider builds at least the `run` command when `includeRun` is omitted or true.
3. Provider can be configured with `includeRun: false` and an empty repository list without failing.

Avoid tests that open hardware sessions; construction-only tests are sufficient for this ticket.

## Validation

Run:

```bash
go test ./runtime/js/provider ./pkg/xgoja/provider ./cmd/loupedeck/cmds/verbs ./cmd/loupedeck/cmds/run -count=1
```

Use `github.com/go-go-golems/go-go-goja v0.5.0` for the command-provider APIs.
