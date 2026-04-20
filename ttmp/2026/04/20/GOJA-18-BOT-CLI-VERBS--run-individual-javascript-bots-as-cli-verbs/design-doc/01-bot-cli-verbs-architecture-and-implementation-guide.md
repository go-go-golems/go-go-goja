---
Title: Bot CLI verbs architecture and implementation guide
Ticket: GOJA-18-BOT-CLI-VERBS
Status: active
Topics:
    - goja
    - javascript
    - cli
    - cobra
    - glazed
    - bots
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../loupedeck/cmd/loupedeck/cmds/verbs/bootstrap.go
      Note: Shows the strongest repository bootstrap and duplicate-detection pattern to borrow
    - Path: ../../../../../../../loupedeck/cmd/loupedeck/cmds/verbs/command.go
      Note: Shows how jsverbs can be wrapped into runtime-backed Cobra commands
    - Path: README.md
      Note: Explains the repo's explicit engine composition style and shows the absence of a canonical bot CLI today
    - Path: cmd/go-go-goja/main.go
      Note: Implements the canonical root CLI binary chosen in the design
    - Path: engine/factory.go
      Note: Defines the engine builder/factory/runtime lifecycle the bot CLI should reuse
    - Path: pkg/botcli/bootstrap.go
      Note: Implements repository bootstrap
    - Path: pkg/botcli/command.go
      Note: Implements the public bots list/run/help surface
    - Path: pkg/botcli/runtime.go
      Note: Implements runtime-backed single-verb execution and help wiring
    - Path: pkg/doc/12-bot-cli-verb-authoring-guide.md
      Note: Documents the explicit __verb__ authoring contract chosen by the implementation
    - Path: pkg/jsverbs/command.go
      Note: Provides command description generation from discovered VerbSpecs
    - Path: pkg/jsverbs/runtime.go
      Note: Provides runtime invocation and Promise settlement for selected verbs
    - Path: pkg/jsverbs/scan.go
      Note: Provides the scan stage for discovering filesystem-backed JavaScript verbs
    - Path: testdata/botcli/discord.js
      Note: Provides a dedicated bot fixture that reflects the intended v1 authoring style
ExternalSources: []
Summary: Intern-friendly architecture and implementation plan for a go-go-goja CLI surface that lists bot verbs, runs one bot verb on demand, and renders help for a selected verb by reusing jsverbs scanning and runtime execution patterns.
LastUpdated: 2026-04-20T12:45:00-04:00
WhatFor: Explain how to add a stable `go-go-goja bots list|run|help` command surface on top of the existing jsverbs and engine runtime layers.
WhenToUse: Read this before implementing or reviewing a CLI feature that exposes JavaScript bots as command-line verbs in go-go-goja.
---




# Bot CLI verbs architecture and implementation guide

## Executive summary

This ticket designs a new command-line surface for `go-go-goja` that lets an operator discover and run individual JavaScript bots as verbs without needing to know the internal `jsverbs` API. The user-facing shape is intentionally small and stable:

```text
go-go-goja bots list
go-go-goja bots run <verb>
go-go-goja bots help <verb>
```

The key implementation insight is that `go-go-goja` already has almost all of the machinery needed. The `pkg/jsverbs` package can already scan JavaScript source trees, infer or read verb metadata, compile verb schemas into Glazed command descriptions, and invoke the matching JavaScript function in a Goja runtime. The `loupedeck` repository proves that this machinery can be used to build a real CLI command surface. The missing piece in `go-go-goja` is not a new JavaScript execution engine. The missing piece is a small orchestration layer that turns scanning plus runtime invocation into the exact `bots list|run|help` experience the user wants.

The recommended design is to build a new `bots` command package that reuses `pkg/jsverbs` directly, borrows the repository bootstrap and lazy command-construction ideas from `loupedeck`, but deliberately keeps the v1 surface smaller. In particular, v1 should treat “bot” as “a JavaScript verb script discoverable by `jsverbs`”, not as “a sandbox `defineBot(...)` object.” The sandbox API remains useful, but it solves runtime capability registration, not CLI discovery.

## Problem statement and scope

The user wants to run individual bots from the command line using verbs, but does not want the current low-level or example-oriented workflow. Today, `go-go-goja` has:

- a reusable runtime engine,
- a reusable JavaScript verb scanner/compiler/runtime bridge,
- an example CLI called `jsverbs-example`, and
- a separate sandbox runtime API built around `defineBot(...)`.

What it does not yet have is a stable, product-shaped bot CLI entrypoint that feels like a normal application command tree rather than an internal demo.

The requested surface is:

- `go-go-goja bots list`
- `go-go-goja bots run <verb>`
- `go-go-goja bots help <verb>`

### In scope

- A design for a canonical CLI surface for bot verbs.
- A mapping from current `pkg/jsverbs` internals to the new commands.
- A recommended package layout and file-by-file implementation plan.
- Guidance for help generation, runtime invocation, output rendering, and testing.
- Explicit explanation of how this should relate to `loupedeck` and the new sandbox module.

### Out of scope for v1

- Discovering sandbox `defineBot(...)` declarations directly from the scanner.
- Replacing `pkg/jsverbs` with a new scanning model.
- Building permissioning, bot process management, or long-lived daemon orchestration.
- Designing a remote bot registry service.
- A full config system unless it is a thin copy of the already-proven repository bootstrap pattern.

## Reader orientation: the minimum concepts you must know first

Before touching implementation, an intern should understand five concepts.

### 1. `engine` owns Goja runtimes

The `engine` package is the low-level runtime composition system. `engine.NewBuilder()` accepts module and `require()` options, `Build()` freezes the composition, and `Factory.NewRuntime(...)` creates a concrete Goja runtime with an owner thread and `require` registry (`engine/factory.go:35-46`, `engine/factory.go:101-154`, `engine/factory.go:162-230`).

This matters because the bot CLI should not invent its own runtime lifecycle. It should reuse the engine.

### 2. `pkg/jsverbs` is a pipeline, not just a scanner

`pkg/jsverbs` has four major responsibilities:

- scan source trees into a `Registry` (`pkg/jsverbs/scan.go:17-74`, `pkg/jsverbs/scan.go:159-192`),
- store file/function/verb metadata in the registry model (`pkg/jsverbs/model.go:74-152`),
- compile a verb into a Glazed command description (`pkg/jsverbs/command.go:41-100`, `pkg/jsverbs/command.go:102-209`), and
- invoke the selected function in a runtime (`pkg/jsverbs/runtime.go:18-36`, `pkg/jsverbs/runtime.go:44-107`).

This matters because the new `bots` CLI should be thin. It should orchestrate these stages, not replace them.

### 3. `jsverbs-example` already proves the simple case

The example binary scans a directory, calls `registry.Commands()`, adds a `list` helper, and mounts the generated commands into Cobra via Glazed (`cmd/jsverbs-example/main.go:23-95`).

This matters because it is the smallest in-repo reference for “scan verbs and expose them as commands.”

### 4. `loupedeck` proves the advanced case

The `loupedeck` CLI adds a root-level `verbs` command, resolves repositories from embedded/config/env/CLI sources, scans them, resolves duplicates, builds command wrappers, and invokes verbs inside a live runtime session (`cmd/loupedeck/main.go:17-42`, `cmd/loupedeck/cmds/verbs/bootstrap.go:64-118`, `cmd/loupedeck/cmds/verbs/bootstrap.go:311-375`, `cmd/loupedeck/cmds/verbs/command.go:57-119`, `cmd/loupedeck/cmds/verbs/command.go:121-245`).

This matters because it is the best production-shaped reference for how to turn `jsverbs` into a real CLI.

### 5. The sandbox API is related, but different

The new sandbox module exposes `defineBot(...)` as a runtime-scoped CommonJS capability (`modules/sandbox/runtime.go:71-104`, `modules/sandbox/runtime.go:123-141`, `pkg/sandbox/registrar.go:35-64`). That is useful for host-managed bot runtimes, but it is not the same as static CLI verb discovery. `jsverbs` scans top-level functions and sentinel metadata; it does not inspect or understand sandbox bot registration by itself.

This matters because a lot of confusion disappears once you separate “CLI verb discovery” from “runtime bot capability API.”

## Current-state architecture

### A. Existing `go-go-goja` command surface

The repository has many small demo and utility binaries under `cmd/`, including `cmd/jsverbs-example/main.go` and `cmd/sandbox-demo/main.go`, but it does not currently expose one canonical `go-go-goja` root application binary with subcommands (`README.md`, `cmd/` tree, `cmd/jsverbs-example/main.go:23-95`).

That means the ticket needs to decide not only how `bots` should behave, but also where this command should live.

### B. `jsverbs` scanning model

The scanner can load from a real directory, an `fs.FS`, or in-memory sources (`pkg/jsverbs/scan.go:17-149`). It stores the results in a `Registry` that contains:

- `Files`,
- `Diagnostics`,
- shared sections,
- discovered verbs,
- and a module lookup map (`pkg/jsverbs/model.go:74-84`).

Each `VerbSpec` carries:

- function name,
- CLI name,
- parents,
- output mode,
- field/section metadata,
- source file,
- and parameter list (`pkg/jsverbs/model.go:140-152`).

This is the metadata backbone for the proposed bot CLI.

### C. `jsverbs` command compilation model

`Registry.CommandDescriptionForVerb(...)` and `Registry.CommandForVerbWithInvoker(...)` transform a discovered `VerbSpec` into Glazed command metadata and runnable wrappers (`pkg/jsverbs/command.go:61-100`). The description builder resolves field sections, parameter bindings, defaults, and help text (`pkg/jsverbs/command.go:102-209`).

This is important because `bots help <verb>` should use the same description-building logic as `bots run <verb>`. One metadata source should define both execution and help.

### D. `jsverbs` runtime model

`Registry.InvokeInRuntime(...)` does four critical things:

1. builds the argument list from parsed Glazed values,
2. ensures the source module is required,
3. looks up the captured JavaScript function in `__glazedVerbRegistry`, and
4. calls the function and normalizes Promise results (`pkg/jsverbs/runtime.go:44-107`).

It also injects a small runtime overlay that defines no-op sentinel functions and captures discovered functions into `globalThis.__glazedVerbRegistry` (`pkg/jsverbs/runtime.go:167-213`).

This is the exact invocation path the bot CLI should reuse.

### E. `loupedeck` repository bootstrap pattern

`loupedeck` gathers repositories from four places in deterministic order:

1. built-in embedded scripts,
2. config files,
3. environment variables,
4. CLI flags (`cmd/loupedeck/cmds/verbs/bootstrap.go:88-118`).

It then scans each repository, disables “include public functions” for stricter verb exposure, and rejects duplicate full verb paths (`cmd/loupedeck/cmds/verbs/bootstrap.go:311-351`).

This is a very strong pattern, but `go-go-goja` does not need to copy every part in v1.

### F. `loupedeck` command-building pattern

`loupedeck` uses a lazy `verbs` command that defers scanning until runtime (`cmd/loupedeck/cmds/verbs/command.go:61-81`). It then:

- scans repositories,
- builds descriptions per verb,
- augments descriptions with runtime sections,
- wraps execution in a runtime command adapter,
- creates parent Cobra commands dynamically,
- and finally invokes the selected verb (`cmd/loupedeck/cmds/verbs/command.go:97-206`, `cmd/loupedeck/cmds/verbs/command.go:221-245`).

This is extremely relevant, but the user asked for a different public surface: `bots list|run|help`, not `bots <verb>`.

## Gap analysis

There are four gaps between the current system and the desired CLI.

### Gap 1: there is no stable product-shaped bot command

The example command is useful for development, but it exposes discovered verbs directly as subcommands of `jsverbs-example`. The user wants a more application-like surface with stable nouns (`bots`) and actions (`list`, `run`, `help`).

### Gap 2: there is no single-verb lazy handoff command in `go-go-goja`

`loupedeck` dynamically builds a whole command tree under `verbs`. For the requested `bots run <verb>` surface, `go-go-goja` needs a command that resolves one verb name first, then builds a parser for only that verb, then forwards the remaining arguments.

### Gap 3: there is no explicit repository bootstrap API for bots

`jsverbs-example` takes one `--dir` and scans it (`cmd/jsverbs-example/main.go:24-31`, `cmd/jsverbs-example/main.go:53`). `loupedeck` has a richer bootstrap abstraction. The new bot CLI needs a small, reusable bootstrap layer that answers:

- where do bot repositories come from?
- how are duplicates handled?
- how do we label their source?

### Gap 4: sandbox bots are not automatically CLI verbs

The new sandbox API exposes `defineBot(...)` as a runtime capability, not as scanner metadata (`modules/sandbox/runtime.go:71-104`). Therefore, a design that assumes `bots run` can discover sandbox bots directly would be incorrect.

## Design goals

The design should satisfy the following goals.

1. **Stable user surface**
   - The public commands should remain `bots list`, `bots run`, and `bots help`.

2. **Reuse existing internals**
   - The feature should reuse `pkg/jsverbs` and `engine` rather than inventing a separate scanner or runtime.

3. **Small v1 scope**
   - v1 should support filesystem repositories first.
   - Config/env/embedded repositories can be added later if needed.

4. **One source of truth for help and execution**
   - The same `VerbSpec` and `CommandDescription` must drive both parsing and help output.

5. **Clear boundary with sandbox**
   - CLI bot verbs and sandbox-defined runtime bots should remain separate concepts.

## Proposed command surface

The public UX should be:

```text
go-go-goja bots list [--bot-repository DIR...]
go-go-goja bots run <verb> [verb flags and args...]
go-go-goja bots help <verb>
```

### Command semantics

#### `bots list`

Shows all discovered verb paths and their source files, for example:

```text
admin cleanup        scripts/admin.js
alerts ping          scripts/alerts.js
discord greet        bots/discord/greet.js
```

#### `bots run <verb>`

Resolves `<verb>` to exactly one `VerbSpec`, then:

- builds the Glazed schema for that verb,
- parses the remaining CLI arguments against that schema,
- creates a runtime with the correct module roots and `require` overlay,
- invokes the JavaScript function,
- prints structured or text output depending on `OutputMode`.

#### `bots help <verb>`

Resolves `<verb>` to exactly one `VerbSpec`, builds a single-verb Cobra command from the generated description, and renders that command’s help/usage text.

This gives the user “command-specific help” without exposing dynamic top-level subcommands.

## Recommended architecture

### High-level shape

```text
cobra root
  └── bots
      ├── list
      ├── run <verb>
      └── help <verb>
              |
              v
        bot bootstrap layer
              |
              v
        jsverbs.Registry
         /     |      \
        /      |       \
   scan.go  command.go  runtime.go
        \      |       /
         \     |      /
             engine
```

### Package layout recommendation

I recommend the following layout for implementation:

```text
cmd/go-go-goja/main.go              # new canonical root CLI (or equivalent host binary)
pkg/botcli/bootstrap.go             # repository discovery + scanning
pkg/botcli/model.go                 # repository / discovered bot metadata
pkg/botcli/list_command.go          # `bots list`
pkg/botcli/run_command.go           # `bots run <verb>`
pkg/botcli/help_command.go          # `bots help <verb>`
pkg/botcli/command_helpers.go       # shared resolution, parser, output helpers
```

### Why a new `pkg/botcli` package?

Because the new code is orchestration code, not core `jsverbs` code.

- `pkg/jsverbs` should remain generic.
- `pkg/botcli` can encode this repo’s opinionated UX.
- The package boundary makes future replacement or extension easier.

## Detailed design decisions

### Decision 1: bots are `jsverbs` in v1, not sandbox objects

**Decision:** treat an individual bot as a `jsverbs`-discoverable JavaScript function.

**Why:**

- `jsverbs` already supports scanning, schema generation, runtime invocation, and output.
- `defineBot(...)` is a runtime capability surface, not a static discovery format.
- A v1 implementation that tries to make the scanner understand sandbox registration would expand scope dramatically.

**Practical consequence:** if someone has a sandbox-defined bot today and wants a CLI verb, they should write a small `jsverbs` wrapper function that calls into the sandbox flow.

### Decision 2: use static action commands instead of dynamic top-level subcommands

**Decision:** prefer `bots run <verb>` over `bots <verb>`.

**Why:**

- This is the surface the user explicitly asked for.
- It gives a more stable CLI contract.
- It avoids rebuilding the entire command tree for shell completion or help in v1.
- It makes it easier to support wrappers like `bots help <verb>`.

**Tradeoff:** this is slightly less “pure Cobra” than dynamically mounting every discovered verb as a subcommand. That is acceptable because the user wants action-style dispatch.

### Decision 3: reuse `CommandDescriptionForVerb(...)` for help and parsing

**Decision:** a resolved verb should be converted to a single `cmds.CommandDescription`, and that description should power both argument parsing and help rendering.

**Why:**

- It prevents drift between “what help says” and “what run parses.”
- It keeps Glazed as the schema authority.

### Decision 4: scope repository discovery modestly in v1

**Decision:** support a repeatable `--bot-repository` flag first, with optional future env/config support.

**Why:**

- `loupedeck` proves the richer bootstrap model works.
- But `go-go-goja` does not need built-in repos, app config parsing, and embedded FS on day one.
- Simpler bootstrap means faster delivery and less review surface.

**Future extension path:** if users want parity with `loupedeck`, the same bootstrap package can later add environment/config layers.

### Decision 5: keep runtime creation caller-owned in the command layer

**Decision:** `bots run` should assemble engine options and create a runtime explicitly rather than relying on hidden global setup.

**Why:**

- This matches the current repository’s explicit engine style (`README.md`, `engine/factory.go:35-46`, `engine/factory.go:152-230`).
- It keeps module loading and lifecycle reviewable.

## Proposed API sketches

### Repository bootstrap model

```go
type Repository struct {
    Name     string
    RootDir  string
    Source   string   // e.g. "cli"
    SourceRef string  // e.g. "--bot-repository"
}

type Bootstrap struct {
    Repositories []Repository
}
```

### Bot discovery result

```go
type DiscoveredBot struct {
    Repository Repository
    Registry   *jsverbs.Registry
    Verb       *jsverbs.VerbSpec
}
```

### Core helper functions

```go
func DiscoverBootstrapFromCommand(cmd *cobra.Command) (Bootstrap, error)
func ScanRepositories(bootstrap Bootstrap) ([]ScannedRepository, error)
func CollectDiscoveredBots(repos []ScannedRepository) ([]DiscoveredBot, error)
func ResolveBot(selector string, discovered []DiscoveredBot) (DiscoveredBot, error)
func BuildVerbDescription(bot DiscoveredBot) (*cmds.CommandDescription, error)
func BuildRuntimeForBot(ctx context.Context, bot DiscoveredBot) (*engine.Runtime, error)
func PrintVerbResult(w io.Writer, outputMode string, value any) error
```

### Single-verb handoff helper

```go
func RunSelectedVerb(ctx context.Context, bot DiscoveredBot, rawArgs []string) error {
    desc := BuildVerbDescription(bot)
    parser := glazedcli.NewCobraParserFromSections(desc.Schema, cfg)
    parsed := parser.ParseArgs(rawArgs)
    rt := BuildRuntimeForBot(ctx, bot)
    result := bot.Registry.InvokeInRuntime(ctx, rt, bot.Verb, parsed)
    return PrintVerbResult(os.Stdout, bot.Verb.OutputMode, result)
}
```

## End-to-end flows

### Flow 1: `go-go-goja bots list`

```text
user runs command
    |
    v
read --bot-repository flags
    |
    v
scan each repository with jsverbs.ScanDir(...)
    |
    v
collect verbs + reject duplicates by FullPath()
    |
    v
print sorted table/list with source refs
```

#### Pseudocode

```go
func runList(cmd *cobra.Command) error {
    bootstrap := DiscoverBootstrapFromCommand(cmd)
    repos := ScanRepositories(bootstrap)
    bots := CollectDiscoveredBots(repos)
    for _, bot := range bots {
        fmt.Printf("%s\t%s\n", bot.Verb.FullPath(), bot.Verb.SourceRef())
    }
}
```

### Flow 2: `go-go-goja bots run <verb>`

```text
user runs bots run alerts ping --channel ops
    |
    v
resolve repositories
    |
    v
scan repositories
    |
    v
resolve selector "alerts ping" to one VerbSpec
    |
    v
build single-verb CommandDescription
    |
    v
parse remaining args against that schema
    |
    v
build engine runtime with:
  - registry RequireLoader overlay
  - default registry modules
  - module roots/global folders
    |
    v
registry.InvokeInRuntime(...)
    |
    v
print structured or text output
```

#### Pseudocode

```go
func runVerb(cmd *cobra.Command, selector string, rawArgs []string) error {
    bootstrap := DiscoverBootstrapFromCommand(cmd)
    repos := ScanRepositories(bootstrap)
    bots := CollectDiscoveredBots(repos)
    bot := ResolveBot(selector, bots)

    desc := bot.Registry.CommandDescriptionForVerb(bot.Verb)
    parser := glazedcli.NewCobraParserFromSections(desc.Schema, &glazedcli.CobraParserConfig{
        SkipCommandSettingsSection: true,
    })
    parsedValues := parser.Parse(cmd, rawArgs)

    runtimeOptions := buildRuntimeOptions(bot)
    factory := engine.NewBuilder(runtimeOptions...).WithModules(engine.DefaultRegistryModules()).Build()
    rt := factory.NewRuntime(cmd.Context())
    defer rt.Close(context.Background())

    result := bot.Registry.InvokeInRuntime(cmd.Context(), rt, bot.Verb, parsedValues)
    return PrintVerbResult(cmd.OutOrStdout(), bot.Verb.OutputMode, result)
}
```

### Flow 3: `go-go-goja bots help <verb>`

```text
user runs help command
    |
    v
resolve selector to one VerbSpec
    |
    v
build single-verb CommandDescription
    |
    v
convert description to one Cobra command
    |
    v
render usage/help to stdout
```

#### Pseudocode

```go
func runHelp(cmd *cobra.Command, selector string) error {
    bot := ResolveBot(selector, scanAll(cmd))
    desc := bot.Registry.CommandDescriptionForVerb(bot.Verb)
    verbCmd := glazedcli.NewCobraCommandFromCommandDescription(desc)
    parser := glazedcli.NewCobraParserFromSections(desc.Schema, cfg)
    parser.AddToCobraCommand(verbCmd)
    verbCmd.SetOut(cmd.OutOrStdout())
    return verbCmd.Help()
}
```

## Runtime composition details

### Required engine options

The runtime should be composed from two sources.

#### 1. Scanned-source overlay loader

Use `registry.RequireLoader()` so `require()` can load the exact scanned sources through the overlay mechanism (`pkg/jsverbs/runtime.go:38-42`, `pkg/jsverbs/runtime.go:167-213`).

#### 2. Module root/global folder resolution

For filesystem repositories, add folders that let local `require()` calls work from the repository root and nearby `node_modules`, similar to `loupedeck`’s `runtimeOptions()` helper (`cmd/loupedeck/cmds/verbs/bootstrap.go:364-375`).

### Recommended builder pattern

```go
factory, err := engine.NewBuilder(
    engine.WithRequireOptions(require.WithLoader(registry.RequireLoader())),
    engine.WithRequireOptions(require.WithGlobalFolders(rootDir, filepath.Join(rootDir, "node_modules"))),
).
    WithModules(engine.DefaultRegistryModules()).
    Build()
```

## Output handling

A bot verb can produce either:

- structured Glaze-style output (`OutputModeGlaze`), or
- plain text writer output (`OutputModeText`) (`pkg/jsverbs/model.go:154-157`).

The simplest v1 output printer should copy the behavior from `loupedeck`’s `printRuntimeCommandResult(...)` (`cmd/loupedeck/cmds/verbs/command.go:247-282`):

- text mode writes strings/bytes directly,
- glaze mode JSON-encodes the returned value if the wrapper is not using the full Glazed runner machinery.

If the future CLI wants first-class Glazed table rendering, that can be added later, but JSON is a reasonable v1 fallback for a stable operator-facing surface.

## Help and usage design

An intern should understand an important subtlety here: `bots help <verb>` is not generic help text. It must display the specific flag schema for that one verb. The easiest correct approach is to create an ephemeral Cobra command from the selected `CommandDescription` and ask Cobra to render its help.

That gives four benefits:

- field help text comes directly from `jsverbs` metadata,
- required/optional arguments stay accurate,
- section grouping survives,
- later help styling improvements automatically apply.

## Relationship to sandbox bots

The sandbox system should be documented carefully so future readers do not conflate it with the bot CLI.

### What sandbox is good at

- runtime-scoped capabilities,
- host-managed bot handles,
- event and command dispatch,
- in-memory runtime state (`modules/sandbox/runtime.go:34-104`, `pkg/sandbox/registrar.go:35-64`).

### What sandbox does not currently provide

- static scanner metadata,
- top-level `__verb__` registration,
- automatic CLI schema generation.

### Recommended coexistence rule

For v1, write CLI bots as `jsverbs`. If a CLI verb needs sandbox capabilities, create a thin JavaScript wrapper that imports `sandbox`, defines or loads the bot, and calls into it from a scanner-visible top-level function.

## Alternatives considered

### Alternative A: expose verbs directly as `go-go-goja bots <verb>`

This matches `jsverbs-example` and `loupedeck verbs`, but it does not match the requested UX. It is also more complex to document because the visible command tree changes based on scanned inputs.

**Rejected for v1** because the user explicitly wants `list`, `run`, and `help`.

### Alternative B: scan sandbox `defineBot(...)` objects directly

This sounds attractive because it would unify the bot runtime and bot CLI stories. In practice it is a much bigger project. The current scanner is syntax-driven around top-level functions and sentinel metadata (`pkg/jsverbs/scan.go`, `pkg/jsverbs/model.go`). Teaching it to understand sandbox bot registration would require either new static analysis rules or partial execution.

**Rejected for v1** because it explodes scope and mixes two different abstractions.

### Alternative C: hard-code one bot per file and skip `jsverbs`

This would reduce the amount of metadata, but it would also throw away existing schema generation, help, output mode handling, and runtime invocation logic.

**Rejected** because it duplicates a solved problem.

### Alternative D: copy all of `loupedeck` bootstrap complexity immediately

This is viable, but it is more surface than we need on day one.

**Deferred** because a repeatable `--bot-repository` flag is enough to unlock the requested workflow.

## File-by-file implementation plan

### Phase 1: create the canonical CLI entrypoint

**Goal:** decide where `go-go-goja bots ...` lives.

Recommended file:

- `cmd/go-go-goja/main.go`

Why:

- The repo currently has several small binaries but no canonical root CLI.
- This feature deserves a stable home instead of another demo binary.

If the project is not ready for that yet, use a transitional binary such as `cmd/bots-example/main.go`, but keep the internal package names stable so promotion later is easy.

### Phase 2: add `pkg/botcli/bootstrap.go`

**Goal:** translate repeatable `--bot-repository` flags into scanned repositories.

Responsibilities:

- read flag values,
- normalize paths,
- ensure paths exist and are directories,
- scan with `jsverbs.ScanDir(...)`,
- disable `IncludePublicFunctions` if the project wants explicit verbs only,
- deduplicate by `VerbSpec.FullPath()`.

### Phase 3: add `bots list`

**Goal:** provide discovery and debugging.

Responsibilities:

- scan repos,
- collect all verbs,
- sort by `FullPath()`,
- print verb path plus source file.

This is the easiest command and should land first.

### Phase 4: add `bots run`

**Goal:** resolve one selector and run it with verb-specific parsing.

Responsibilities:

- consume `<verb>` as the first positional argument,
- build description from `CommandDescriptionForVerb(...)`,
- parse remaining args using a single-verb parser,
- create runtime and invoke the verb,
- print the result in the correct mode.

This is the highest-value command.

### Phase 5: add `bots help`

**Goal:** print accurate, verb-specific help.

Responsibilities:

- resolve `<verb>`,
- create ephemeral Cobra command from the description,
- attach parser-generated flags,
- render help/usage.

### Phase 6: tests

**Goal:** prove the command surface works end to end.

Test categories:

1. bootstrap/path normalization tests,
2. duplicate selector tests,
3. `list` output tests,
4. `run` invocation tests,
5. `help` rendering tests,
6. Promise-returning verb tests,
7. relative `require()` tests.

## Testing and validation strategy

### Unit tests

- `pkg/botcli/bootstrap_test.go`
  - flag parsing
  - empty/missing directory handling
  - duplicate repository de-duplication if supported
- `pkg/botcli/resolve_test.go`
  - exact match
  - suffix match if supported
  - ambiguous selector error
  - missing selector error

### Integration tests

- `cmd/go-go-goja/main_test.go` or `pkg/botcli/command_test.go`
  - run `bots list` against a fixture tree
  - run `bots run greet Manuel --excited`
  - run `bots help greet`
  - verify output mode differences

### Suggested fixture layout

```text
testdata/bots/
  basics.js
  async.js
  nested/
    with-helper.js
    helper.js
```

You can reuse or copy patterns from `testdata/jsverbs` because that fixture set already exercises the expected authoring style.

### Manual validation checklist

```bash
go run ./cmd/go-go-goja bots list --bot-repository ./testdata/jsverbs
go run ./cmd/go-go-goja bots run greet Manuel --bot-repository ./testdata/jsverbs --excited
go run ./cmd/go-go-goja bots help greet --bot-repository ./testdata/jsverbs
```

## Risks and review points

### Risk 1: confusing bot selectors across multiple repositories

If two repositories export the same full verb path, the CLI must fail loudly and show both sources. Silently choosing one would be dangerous.

### Risk 2: help parsing drift

If `bots help` constructs help text differently from `bots run`, users will see invalid flag guidance. This is why both commands must share `CommandDescriptionForVerb(...)`.

### Risk 3: runtime module resolution surprises

If the command layer forgets to add the registry overlay or filesystem module roots, `require()` calls will fail even though the scanner succeeded.

### Risk 4: overreaching into sandbox too early

Trying to merge sandbox discovery with jsverbs scanning will slow delivery and muddy ownership.

## Open questions

1. Should v1 disable `IncludePublicFunctions` and require explicit `__verb__` metadata, or should it keep the current permissive default for convenience?
2. Should `bots run` accept suffix selectors like `ping` when only one full path ends with that name, similar to `scriptmeta.FindVerb(...)` (`loupedeck/pkg/scriptmeta/scriptmeta.go:117-160`)?
3. Does the project want a canonical `cmd/go-go-goja` root binary now, or should the first delivery be a focused example binary promoted later?
4. Should output mode `glaze` stay JSON-first in v1, or should the command be integrated deeper with the full Glazed runner stack for tables and alternate renderers?

## Recommendation summary

The cleanest implementation is:

1. create a `botcli` orchestration package,
2. add a root `bots` command with `list`, `run`, and `help`,
3. reuse `pkg/jsverbs` for scan/description/invocation,
4. borrow repository-scan and runtime-wrapper ideas from `loupedeck`,
5. keep sandbox integration out of scope for v1.

That design matches the user-requested UX, fits the repository’s current architecture, and gives a new intern a tractable, file-oriented implementation path.

## References

### go-go-goja core

- `README.md`
- `engine/factory.go:35-46`
- `engine/factory.go:101-154`
- `engine/factory.go:162-230`
- `engine/runtime_modules.go:12-27`
- `cmd/jsverbs-example/main.go:23-95`
- `pkg/jsverbs/model.go:74-152`
- `pkg/jsverbs/scan.go:17-149`
- `pkg/jsverbs/command.go:41-209`
- `pkg/jsverbs/runtime.go:44-107`
- `pkg/jsverbs/runtime.go:167-258`

### sandbox references

- `modules/sandbox/runtime.go:34-104`
- `modules/sandbox/runtime.go:123-141`
- `pkg/sandbox/registrar.go:35-64`

### loupedeck references

- `cmd/loupedeck/main.go:17-42`
- `cmd/loupedeck/cmds/verbs/bootstrap.go:64-118`
- `cmd/loupedeck/cmds/verbs/bootstrap.go:229-351`
- `cmd/loupedeck/cmds/verbs/bootstrap.go:364-375`
- `cmd/loupedeck/cmds/verbs/command.go:57-245`
- `cmd/loupedeck/cmds/verbs/command.go:247-282`
- `pkg/scriptmeta/scriptmeta.go:31-180`
- `cmd/loupedeck/cmds/run/session.go:151-165`
