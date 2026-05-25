---
Title: Custom xgoja CLI verbs for third-party JavaScript sandboxes
Ticket: XGOJA-008
Status: active
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
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../go-go-goja/pkg/xgoja/app/host.go
      Note: Current generated xgoja command attachment boundary
    - Path: ../../../../../../../../../../go-go-goja/pkg/xgoja/app/root.go
      Note: Current eval run modules and generic jsverbs command implementation
    - Path: ../../../../../../../../../../loupedeck/cmd/loupedeck/cmds/verbs/command.go
      Note: Loupedeck package-owned dynamic scene verb command tree built from Glazed commands
    - Path: ../../../../../../../../../../discord-bot/pkg/botcli/command_root.go
      Note: Discord bot repository command-tree generation using Glazed command descriptions
    - Path: ../../../../../../../../../../css-visual-diff/internal/cssvisualdiff/verbcli/command.go
      Note: CSS visual diff workflow verb command tree using Glazed commands
    - Path: ../../../../../../../../../../go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go
      Note: Minitrace command-local JS runtime and jsverb invocation
ExternalSources: []
Summary: Design guide for extending generated xgoja binaries with custom Glazed command sets supplied by third-party Goja sandbox packages.
LastUpdated: 2026-05-25T00:10:00-04:00
WhatFor: Use this guide when designing or implementing xgoja support for package-provided custom Glazed commands beyond built-in eval run repl and generic jsverbs.
WhenToUse: Before changing xgoja buildspecs provider APIs or generated command attachment for third-party sandbox command sets.
---

# Custom xgoja CLI verbs for third-party JavaScript sandboxes

## Revision note

This document was revised after review to keep the extension point inside the Go-Go-Golems / Glazed command ecosystem. The original draft proposed that provider command factories return `*cobra.Command`. The revised design instead has providers return **Glazed commands** (`cmds.Command` values implemented as `cmds.BareCommand`, `cmds.WriterCommand`, or `cmds.GlazeCommand`, usually `BareCommand`) plus optional grouping metadata. xgoja remains responsible for converting those Glazed commands into Cobra commands with the same parser and middleware path already used by generated xgoja jsverbs.

This is the better boundary because:

- command schemas, flags, arguments, parents, help text, and output behavior stay in Glazed;
- packages can return structured row output when appropriate (`GlazeCommand`), direct writer output when needed (`WriterCommand`), or normal side-effect/session commands (`BareCommand`);
- xgoja can reuse `glazedcli.AddCommandsToRootCommand` and `glazedcli.BuildCobraCommandFromCommand`;
- command providers do not need to construct Cobra directly except for rare app-root integration code outside this provider API.

## Executive summary

Generated xgoja binaries currently know how to attach a fixed set of command families:

- `eval`: evaluate a JavaScript string in a selected runtime profile.
- `run`: run a JavaScript file in a selected runtime profile.
- `repl`: open an interactive JavaScript REPL for a selected runtime profile.
- `modules`: list compiled provider modules.
- `jsverbs`: scan configured JavaScript verb sources and mount those verbs under one command tree.

That is enough for generic JavaScript execution, but it is not enough for packages that already own richer JavaScript sandboxes. Packages such as `loupedeck`, `discord-bot`, `css-visual-diff`, and `go-minitrace` have package-specific command sets, discovery rules, host services, runtime factories, long-running sessions, and safety policies. The user wants xgoja to generate binaries that can expose those package-provided commands instead of only generic xgoja commands.

The recommended design is to add a **Glazed command provider** layer next to the existing module-provider layer:

```text
provider package
  ├─ providerapi.Module entries       -> require(...) modules selected by runtime profiles
  ├─ providerapi.VerbSource entries   -> generic JavaScript verb files scanned by xgoja
  └─ providerapi.CommandSetProvider   -> package-owned Glazed command sets
```

A command set provider should be able to contribute custom generated CLI commands such as:

```text
my-generated-binary
  run <file>                         # generic xgoja run, if enabled
  repl                               # generic xgoja repl, if enabled
  loupe scenes <scene> run ...       # loupedeck-owned scene command set
  bots <bot> run --bot-token ...     # discord-bot-owned bot command set
  cvd verbs compare-page ...         # css-visual-diff workflow command set
  minitrace query <command> ...      # go-minitrace query command catalog
```

The key idea is that xgoja should not try to understand every third-party sandbox. It should provide a stable generated-binary host contract and let packages adapt their own command sets into Glazed commands.

## Problem statement

The xgoja provider work in XGOJA-007 made packages importable as module providers. That solved this problem:

> A generated xgoja binary can compile Go-backed modules and make them available to JavaScript through `require(...)`.

It did not solve this different problem:

> A generated xgoja binary can expose package-specific CLI commands that create the correct sandbox, attach host services, discover JavaScript scripts, parse package-specific flags, and execute the JavaScript in the package's intended context.

The difference matters. `require("loupedeck/gfx")` is a module. A Loupedeck scene runner with hardware/session flags is an application command. `require("discord")` is a module. `discord-bot bots support run --bot-token ...` is an application command that opens a Discord session and stays alive until cancelled.

If xgoja only supports generic `run` and `jsverbs`, package authors will keep re-implementing command mounting outside xgoja. The generated binary will not be a useful composition target for real package-specific JS sandboxes.

## Glazed command types: the target return value

The provider API should use the existing Glazed command interfaces from `glazed/pkg/cmds`:

```go
type Command interface {
    Description() *CommandDescription
    ToYAML(w io.Writer) error
}

type BareCommand interface {
    Command
    Run(ctx context.Context, parsedValues *values.Values) error
}

type WriterCommand interface {
    Command
    RunIntoWriter(ctx context.Context, parsedValues *values.Values, w io.Writer) error
}

type GlazeCommand interface {
    Command
    RunIntoGlazeProcessor(ctx context.Context, parsedValues *values.Values, gp middlewares.Processor) error
}
```

Use these as follows:

- **BareCommand**: default for side-effect commands, long-running sessions, device control, bot runners, browser workflows, and commands that print/log manually.
- **WriterCommand**: use when command output is textual or file-like and must write to `cmd.OutOrStdout()`.
- **GlazeCommand**: use when command output is structured rows and should support normal `--output`, `--fields`, processors, and table/json/yaml rendering.

xgoja should mount provider commands using Glazed's Cobra bridge:

```go
glazedcli.AddCommandsToRootCommand(root, commandSet.Commands, nil,
    glazedcli.WithParserConfig(glazedcli.CobraParserConfig{
        MiddlewaresFunc: glazedcli.CobraCommandDefaultMiddlewares,
    }),
)
```

This keeps xgoja command generation aligned with `jsverbs`, `modules`, and existing Go-Go-Golems CLI patterns.

## Current xgoja architecture

### Runtime and command attachment

`go-go-goja/pkg/xgoja/app/host.go` is the current command attachment boundary. `Host.AttachDefaultCommands` installs the fixed commands if they are enabled in the embedded spec:

- `AttachEval`
- `AttachRun`
- `AttachRepl`
- `AttachModules`
- `AttachVerbs`

Evidence:

- `go-go-goja/pkg/xgoja/app/host.go:30` defines `AttachDefaultCommands`.
- `go-go-goja/pkg/xgoja/app/host.go:37` checks `Spec.Commands.Eval.Enabled`.
- `go-go-goja/pkg/xgoja/app/host.go:40` checks `Spec.Commands.Run.Enabled`.
- `go-go-goja/pkg/xgoja/app/host.go:43` checks `Spec.Commands.Repl.Enabled`.
- `go-go-goja/pkg/xgoja/app/host.go:47` always attaches `modules`.
- `go-go-goja/pkg/xgoja/app/host.go:48` checks `Spec.Commands.JSVerbs.Enabled`.

`go-go-goja/pkg/xgoja/app/root.go` constructs the generated root command. `NewRootCommand` decodes embedded JSON spec, creates a `Host`, and calls `host.AttachDefaultCommands(root)`.

### Buildspec shape

`go-go-goja/cmd/xgoja/internal/buildspec/spec.go` currently has fixed command fields:

```go
type CommandsSpec struct {
    Eval    CommandSpec `yaml:"eval" json:"eval"`
    Run     CommandSpec `yaml:"run" json:"run"`
    Repl    CommandSpec `yaml:"repl" json:"repl"`
    JSVerbs CommandSpec `yaml:"jsverbs" json:"jsverbs"`
}
```

There is no general list of custom command providers. There is also no place to pass command-provider-specific config, host-service config, command mount path, or discovery sources.

### Generic jsverbs support

xgoja already has one dynamic command path: generic JavaScript verbs.

Evidence:

- `go-go-goja/pkg/xgoja/app/root.go:128` creates the generic verbs root command.
- `go-go-goja/pkg/xgoja/app/root.go:157` builds verb commands from configured sources.
- `go-go-goja/pkg/xgoja/app/root.go:187` scans provider-shipped, embedded, or filesystem verb sources.
- `go-go-goja/pkg/xgoja/providerapi/verbs.go` defines provider-shipped verb sources.

This is useful but limited. Generic xgoja jsverbs assume xgoja owns the runtime factory and the command source list. Third-party sandboxes often need to own both.

## Third-party package current-state analysis

### loupedeck

`loupedeck` already has two command modes relevant to this design:

1. A direct `run` command for raw Loupedeck scene scripts.
2. A dynamic `verbs` command tree that scans annotated scene verbs.

Evidence:

- `loupedeck/cmd/loupedeck/cmds/run/command.go:31` defines `NewCommand` for `run`.
- `loupedeck/cmd/loupedeck/cmds/run/command.go:54` decodes command values and calls `run(ctx, opts)`.
- `loupedeck/cmd/loupedeck/cmds/run/command.go:88` prepares raw script bootstrap from a JS file.
- `loupedeck/cmd/loupedeck/cmds/run/command.go:113` creates a scene session with `RunSceneSession`.
- `loupedeck/cmd/loupedeck/cmds/verbs/bootstrap.go:48` discovers repositories from CLI flags, config, env, and embedded examples.
- `loupedeck/cmd/loupedeck/cmds/verbs/command.go:53` defines a lazy `verbs` command.
- `loupedeck/cmd/loupedeck/cmds/verbs/command.go:93` scans repositories and builds discovered verb commands.
- `loupedeck/cmd/loupedeck/cmds/verbs/command.go:132` wraps jsverbs with a package-specific live scene invoker.

Design implication:

- xgoja should not flatten Loupedeck to generic `run`.
- Loupedeck should contribute `cmds.Command` values: probably `BareCommand` for live scene/session commands, with `cmds.WithParents("loupe")` or a buildspec mount prefix.
- Existing Loupedeck code currently builds a Cobra tree directly in places. The adapter should expose the same command semantics as Glazed commands before xgoja mounts them.

### discord-bot

`discord-bot` exposes JavaScript bot scripts with a Discord-specific module, discovery, bot metadata, synthetic run commands, and jsverbs coexistence.

Evidence:

- `discord-bot/pkg/botcli/command_root.go:24` builds the public `bots` command tree.
- `discord-bot/pkg/botcli/command_root.go:81` discovers bots and registers discovered bot commands.
- `discord-bot/pkg/botcli/command_root.go:99` scans bot repositories and builds commands from bot scripts.
- `discord-bot/pkg/botcli/command_root.go:131` turns non-run jsverbs into CLI commands.
- `discord-bot/pkg/botcli/command_root.go:143` adds synthetic `run` commands for discovered bots without explicit run verbs.
- `discord-bot/pkg/botcli/command_run.go:25` creates a live Discord bot from a JS script and waits on context cancellation.
- `discord-bot/internal/jsdiscord/runtime.go:25` defines a registrar for `require("discord")`.
- `discord-bot/internal/jsdiscord/runtime.go:63` installs `defineBot` and jsverbs metadata polyfills.
- `discord-bot/pkg/botcli/options.go:19` defines a `RuntimeFactory` extension hook for ordinary jsverb execution.

Design implication:

- Discord should expose `[]cmds.Command` for discovered bot commands rather than a pre-built `*cobra.Command`.
- `bots list` and `bots help` are good `GlazeCommand` candidates because they emit structured rows.
- `bots <bot> run` is a `BareCommand` because it opens a long-running session and blocks on context cancellation.

### css-visual-diff

`css-visual-diff` has internal JavaScript modules and a `verbcli` package for annotated workflow verbs.

Evidence:

- `css-visual-diff/internal/cssvisualdiff/jsapi/module.go:15` registers `require("css-visual-diff")`.
- `css-visual-diff/internal/cssvisualdiff/jsapi/module.go:82` uses engine runtime owner callbacks for promises.
- `css-visual-diff/internal/cssvisualdiff/verbcli/command.go:15` defines an `InvokerFactory` for discovered workflow verbs.
- `css-visual-diff/internal/cssvisualdiff/verbcli/command.go:17` defines a lazy `verbs` command.
- `css-visual-diff/internal/cssvisualdiff/verbcli/command.go:35` builds the command tree.
- `css-visual-diff/internal/cssvisualdiff/verbcli/command.go:82` creates a runtime via `newRuntimeFactory(repo)` and invokes the verb.
- `css-visual-diff/internal/cssvisualdiff/verbcli/runtime_factory.go:12` delegates runtime creation to the CSS visual diff DSL runtime factory.

Design implication:

- The current code is under `internal`, so xgoja cannot import it from a generated binary outside the module.
- A public adapter package is required, such as `pkg/cssvisualdiff/xgoja` or `pkg/xgoja/provider`, which returns `[]cmds.Command` rather than Cobra.
- Browser/artifact workflows are usually `BareCommand` or `WriterCommand`; metadata/listing commands can be `GlazeCommand`.

### go-minitrace

`go-minitrace` has command-local jsverb execution for query commands and now has a reusable `minitracejs` module provider from XGOJA-007.

Evidence:

- `go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go:24` runs a JS command into a Glazed processor.
- `go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go:42` scans a command source root with `jsverbs.ScanFS`.
- `go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go:56` builds a runtime with `go-go-goja/engine`.
- `go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go:63` registers the `minitrace` module with an SQL connection and runtime settings.
- `go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go:80` invokes the discovered JS verb in that runtime.

Design implication:

- Minitrace custom CLI verbs are catalog-defined query commands that need a prepared DuckDB connection and structured output processor.
- Minitrace command providers should primarily return `GlazeCommand` implementations because query results are rows.

## Design goals

1. Allow generated xgoja binaries to mount package-provided Glazed command sets.
2. Keep xgoja generic: it should not know Loupedeck, Discord, CSS visual diff, or minitrace semantics.
3. Reuse existing Glazed command structures and Cobra bridging.
4. Support lazy discovery for expensive or environment-dependent command sets.
5. Support host services and command-provider config explicitly.
6. Keep provider modules, generic jsverbs, and custom command providers composable but separate.
7. Preserve generated-binary reproducibility: imports and command providers must be declared in `xgoja.yaml`.

## Proposed architecture

### New provider API concepts

Add a command-set provider entry type to `pkg/xgoja/providerapi`.

```go
package providerapi

import (
    "context"
    "encoding/json"

    "github.com/go-go-golems/glazed/pkg/cli"
    "github.com/go-go-golems/glazed/pkg/cmds"
)

type CommandSetProviderFactory func(CommandSetContext) (*CommandSet, error)

type CommandSetContext struct {
    Context context.Context
    PackageID string
    Name string
    Mount string
    Config json.RawMessage
    Host HostServices
    Providers *Registry
    RuntimeFactory RuntimeFactoryLike
}

type CommandSetProvider struct {
    Name string
    DefaultMount string
    Description string
    ConfigSchema json.RawMessage
    New CommandSetProviderFactory
}

type CommandSet struct {
    Commands []cmds.Command
    ParserConfig *cli.CobraParserConfig
}
```

`Registry.Package` would accept `CommandSetProvider` entries alongside `Module` and `VerbSource` entries:

```go
func Register(reg *providerapi.Registry) error {
    return reg.Package("loupedeck",
        providerapi.Module{...},
        providerapi.CommandSetProvider{
            Name: "scene-verbs",
            DefaultMount: "loupe",
            New: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
                bootstrap, err := loupedeckcmd.BootstrapFromConfig(ctx.Config)
                if err != nil { return nil, err }
                commands, err := loupedeckcmd.BuildGlazedCommands(bootstrap)
                if err != nil { return nil, err }
                return &providerapi.CommandSet{Commands: commands}, nil
            },
        },
    )
}
```

### Parent/mount semantics

Because providers return Glazed commands, grouping should be expressed with command metadata, not nested Cobra objects.

Two options are acceptable:

1. Provider commands set parents themselves:

```go
cmds.NewCommandDescription("run", cmds.WithParents("loupe", "scenes"), ...)
```

2. xgoja applies the buildspec `mount` as a parent prefix:

```yaml
commandProviders:
  - id: loupe-scenes
    package: loupedeck
    name: scene-verbs
    mount: loupe
```

xgoja should prefer option 2 for generated-binary composition because it lets the buildspec decide where a provider command set lives. Implementation-wise, xgoja can clone command descriptions and prepend the mount parent to each command's parents before passing the commands to `AddCommandsToRootCommand`.

### New buildspec section

Add a top-level `commandProviders` section.

```yaml
commandProviders:
  - id: loupe-scenes
    package: loupedeck
    name: scene-verbs
    mount: loupe
    config:
      repositories:
        - path: ./examples/loupedeck-scenes
      hardware:
        device: auto

  - id: discord-bots
    package: discord-bot
    name: bots
    mount: bots
    config:
      repositories:
        - path: ./examples/discord-bots
      tokenEnv: DISCORD_BOT_TOKEN
```

Suggested Go struct:

```go
type CommandProviderInstance struct {
    ID string `yaml:"id" json:"id"`
    Package string `yaml:"package" json:"package"`
    Name string `yaml:"name" json:"name"`
    Mount string `yaml:"mount" json:"mount,omitempty"`
    Config map[string]any `yaml:"config" json:"config,omitempty"`
    Lazy bool `yaml:"lazy" json:"lazy,omitempty"`
}
```

### Generated runtime flow

```text
xgoja.yaml
  |
  | packages[] import provider packages
  | commandProviders[] select provider command sets
  v
generated main.go
  |
  | provider.Register(registry)
  v
providerapi.Registry
  |
  | ResolveCommandSetProvider(package, name)
  v
xgoja app.Host.AttachCommandProviders(root)
  |
  | provider.New(CommandSetContext{...})
  v
[]cmds.Command supplied by package
  |
  | optional mount parent prefix applied by xgoja
  v
glazedcli.AddCommandsToRootCommand(root, commands, ...)
  |
  v
Cobra commands generated by Glazed bridge
```

### Why not return Cobra?

Returning Cobra would work mechanically, but it would bypass the ecosystem conventions we already rely on:

- Glazed schemas for flags and arguments.
- Glazed output processors for row-producing commands.
- Consistent environment/config/default middlewares.
- Glazed help and command description serialization.
- Existing `jsverbs.Registry.CommandForVerbWithInvoker` return type (`cmds.Command`).

A command provider that needs long-running behavior can still implement `BareCommand`. It does not need Cobra to block, handle context cancellation, or manage side effects.

### Why not only extend generic jsverbs?

Generic jsverbs are still valuable for scripts that fit this shape:

```text
scan JS metadata -> build Glazed command -> xgoja RuntimeFactory -> InvokeInRuntime
```

Third-party sandboxes often need this shape instead:

```text
package discovery -> package-specific Glazed command -> package-specific runtime/session/host -> package-specific result handling
```

For example:

- Loupedeck live scene verbs need hardware session options and render loops.
- Discord bot run commands need a Discord session and long-running cancellation.
- CSS visual diff verbs need browser lifecycle and artifact paths.
- Minitrace query commands need a prepared database connection and structured row processor.

Trying to force these through generic jsverbs would create hidden host-service hacks. A Glazed command-set provider API makes the boundary explicit while staying inside the command ecosystem.

## Design patterns

### Pattern A: Static Glazed command set provider

Use when a package has fixed commands and static config.

```go
providerapi.CommandSetProvider{
    Name: "tools",
    DefaultMount: "tools",
    New: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
        cmd, err := mypkg.NewFooCommand(mypkg.OptionsFromConfig(ctx.Config)) // returns cmds.Command
        if err != nil { return nil, err }
        return &providerapi.CommandSet{Commands: []cmds.Command{cmd}}, nil
    },
}
```

Good for simple package-owned command groups.

### Pattern B: Lazy Glazed command provider

Use when command discovery depends on filesystem repositories, config files, environment, or CLI flags. The provider still returns a Glazed command. That command performs lazy discovery in `Run`.

```go
type lazyVerbCommand struct {
    *cmds.CommandDescription
    cfg Config
}

func (c *lazyVerbCommand) Run(ctx context.Context, vals *values.Values) error {
    bootstrap, remaining, err := DiscoverBootstrapFromValues(vals, c.cfg)
    if err != nil { return err }
    commands, err := BuildDiscoveredCommands(bootstrap)
    if err != nil { return err }
    return DispatchToDiscoveredCommand(ctx, commands, remaining)
}
```

Benefits:

- The generated binary still mounts a Glazed command.
- Discovery can happen lazily.
- The command can remain testable without constructing Cobra directly.

### Pattern C: Host-services command provider

Use when commands need live services from an embedding host. This is needed for generated binaries that are not pure standalone CLIs.

```go
type DiscordHostServices interface {
    Token() string
    Logger() zerolog.Logger
    RuntimeModules() []engine.RuntimeModuleSpec
}
```

The provider fails early if services are missing:

```go
host, ok := ctx.Host.(DiscordHostServices)
if !ok {
    return nil, fmt.Errorf("discord command provider requires DiscordHostServices")
}
```

### Pattern D: Runtime-factory command provider

Use when the provider needs xgoja-selected modules plus package-specific runtime behavior.

```go
cmdProvider.New = func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
    cmd := &runVerbCommand{
        runtimeFactory: func(runCtx context.Context, source Source) (*engine.Runtime, error) {
            return ctx.RuntimeFactory.NewRuntime(runCtx, "main", require.WithLoader(source.RequireLoader()))
        },
    }
    return &providerapi.CommandSet{Commands: []cmds.Command{cmd}}, nil
}
```

This pattern bridges generic xgoja runtime profiles with package-specific script discovery.

### Pattern E: Catalog command provider

Use when command metadata is not only in JS source annotations but in a package catalog. This matches `go-minitrace`.

```go
providerapi.CommandSetProvider{
    Name: "query-catalog",
    DefaultMount: "query",
    New: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
        cfg := DecodeConfig(ctx.Config)
        commands, err := minitracequery.BuildCatalogGlazedCommands(cfg.CatalogPath, cfg.RuntimeSettings)
        if err != nil { return nil, err }
        return &providerapi.CommandSet{Commands: commands}, nil
    },
}
```

## Implementation plan

### Phase 1: Add command-set provider API to providerapi

Files:

- `go-go-goja/pkg/xgoja/providerapi/registry.go`
- `go-go-goja/pkg/xgoja/providerapi/module.go`
- new `go-go-goja/pkg/xgoja/providerapi/commands.go`

Tasks:

1. Add `CommandSetProvider` entry type.
2. Add `CommandSet` return type containing `[]cmds.Command` and optional parser config.
3. Extend `Package` with `CommandSetProviders map[string]CommandSetProvider`.
4. Add `ResolveCommandSetProvider(packageID, name string)`.
5. Add tests for duplicate names, missing factory, and package cloning.

### Phase 2: Add buildspec support

Files:

- `go-go-goja/cmd/xgoja/internal/buildspec/spec.go`
- `go-go-goja/cmd/xgoja/internal/buildspec/validate.go`
- `go-go-goja/cmd/xgoja/internal/generate/main.go`
- `go-go-goja/pkg/xgoja/app/spec.go`

Tasks:

1. Add `CommandProviders []CommandProviderInstance` to buildspec and runtime app spec.
2. Validate package ID, provider name, duplicate IDs, mount collisions, and config shape.
3. Include command providers in embedded spec JSON.
4. Add docs in `cmd/xgoja/doc/02-buildspec.md`.

### Phase 3: Attach custom command providers in generated app

Files:

- `go-go-goja/pkg/xgoja/app/host.go`
- `go-go-goja/pkg/xgoja/app/root.go`
- new `go-go-goja/pkg/xgoja/app/command_providers.go`

Tasks:

1. Add `Host.AttachCommandProviders(root)` after built-in commands.
2. For each selected command provider:
   - resolve provider entry;
   - marshal config;
   - construct `CommandSetContext`;
   - call provider factory;
   - apply mount prefix to command parents if needed;
   - pass the resulting commands to `glazedcli.AddCommandsToRootCommand`.
3. Detect collisions with built-ins and previous custom providers.
4. Add generated app tests with a fixture command provider.

Pseudocode:

```go
func (h *Host) AttachCommandProviders(root *cobra.Command) {
    for _, inst := range h.Spec.CommandProviders {
        provider, ok := h.Providers.ResolveCommandSetProvider(inst.Package, inst.Name)
        if !ok { addErrorStub(...); continue }
        data := marshalConfig(inst.Config)
        set, err := provider.New(providerapi.CommandSetContext{
            Context: context.Background(),
            PackageID: inst.Package,
            Name: inst.Name,
            Mount: inst.Mount,
            Config: data,
            Providers: h.Providers,
            RuntimeFactory: h.Factory,
        })
        if err != nil { addErrorStub(...); continue }
        commands := applyMountPrefix(set.Commands, inst.Mount)
        parserConfig := defaultParserConfig()
        if set.ParserConfig != nil { parserConfig = *set.ParserConfig }
        if err := glazedcli.AddCommandsToRootCommand(root, commands, nil, glazedcli.WithParserConfig(parserConfig)); err != nil {
            root.AddCommand(commandErrorStub(inst.Mount, "Attach custom command provider", err))
        }
    }
}
```

### Phase 4: Add fixture providers and examples

Create a first-party fixture command provider under `go-go-goja/pkg/xgoja/testprovider`.

Example provider:

```go
providerapi.CommandSetProvider{
    Name: "echo-tools",
    DefaultMount: "tools",
    New: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
        cmd := &echoCommand{CommandDescription: cmds.NewCommandDescription(
            "echo",
            cmds.WithParents("tools"),
            cmds.WithArguments(fields.New("message", fields.TypeString, fields.WithIsArgument(true))),
        )}
        return &providerapi.CommandSet{Commands: []cmds.Command{cmd}}, nil
    },
}
```

Example spec:

```yaml
commandProviders:
  - id: fixture-tools
    package: fixture
    name: echo-tools
    mount: tools
```

Validation:

```bash
xgoja doctor -f examples/xgoja/custom-command-provider/xgoja.yaml
xgoja build -f examples/xgoja/custom-command-provider/xgoja.yaml --xgoja-replace .
./dist/custom-command-provider tools echo hello
```

### Phase 5: Adapt real packages

Recommended order:

1. `discord-bot`: it already has many Glazed command pieces (`list`, `help`, synthetic run commands, jsverb commands). Refactor `NewBotsCommand` internals to expose `BuildBotCommands(...) []cmds.Command` publicly.
2. `loupedeck`: expose `BuildSceneCommands(...) []cmds.Command` and wrap live scene invokers as `BareCommand`.
3. `css-visual-diff`: move or wrap internal `verbcli` through a public package returning `[]cmds.Command`.
4. `go-minitrace`: expose query catalog commands as `GlazeCommand` values after catalog/DB config is public and safe.

## API sketches for real package adapters

### Loupedeck adapter

```go
func Register(reg *providerapi.Registry) error {
    return reg.Package("loupedeck",
        providerapi.CommandSetProvider{
            Name: "scene-verbs",
            DefaultMount: "loupe",
            New: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
                cfg := DecodeConfig(ctx.Config)
                bootstrap := verbs.BootstrapFromConfig(cfg.Repositories)
                commands, err := verbs.BuildGlazedCommands(bootstrap)
                if err != nil { return nil, err }
                return &providerapi.CommandSet{Commands: commands}, nil
            },
        },
    )
}
```

### Discord adapter

```go
providerapi.CommandSetProvider{
    Name: "bots",
    DefaultMount: "bots",
    New: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
        cfg := DecodeConfig(ctx.Config)
        bootstrap := botcli.Bootstrap{Repositories: cfg.Repositories}
        commands, err := botcli.BuildBotCommands(bootstrap,
            botcli.WithAppName("discord"),
            botcli.WithRuntimeFactory(newXGojaAwareRuntimeFactory(ctx)),
        )
        if err != nil { return nil, err }
        return &providerapi.CommandSet{Commands: commands}, nil
    },
}
```

### CSS visual diff adapter

```go
providerapi.CommandSetProvider{
    Name: "workflows",
    DefaultMount: "cvd",
    New: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
        bootstrap := verbcli.BootstrapFromConfig(ctx.Config)
        commands, err := verbcli.BuildGlazedCommands(bootstrap)
        if err != nil { return nil, err }
        return &providerapi.CommandSet{Commands: commands}, nil
    },
}
```

This requires moving the adapter out of `internal` or making generated binaries target the same module.

### Minitrace adapter

```go
providerapi.CommandSetProvider{
    Name: "query-catalog",
    DefaultMount: "query",
    New: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
        cfg := DecodeConfig(ctx.Config)
        commands, err := minitracecmd.BuildQueryCatalogCommands(cfg.CatalogPath, cfg.RuntimeSettings)
        if err != nil { return nil, err }
        return &providerapi.CommandSet{Commands: commands}, nil
    },
}
```

## Safety and lifecycle rules

1. Command-set providers must declare config schemas for dangerous capabilities.
2. Long-running `BareCommand` implementations must honor `ctx` cancellation.
3. Commands that open devices, browsers, Discord sessions, or databases must close them with `defer` or command lifecycle hooks.
4. Commands that emit rows should be `GlazeCommand`, not `BareCommand` that prints JSON manually.
5. Commands that need textual output should be `WriterCommand` and use the supplied writer.
6. Generated xgoja should not pass global mutable state implicitly.
7. Provider command factories should fail early when required host services are missing.
8. Mount collisions should be doctor errors, not runtime surprises.

## Testing strategy

### Unit tests

- provider registry tests for command-set provider entries;
- buildspec validation tests;
- generated main rendering tests;
- app host command attachment tests with fixture provider;
- `applyMountPrefix` tests for command parent rewriting.

### Integration tests

- generated fixture command-provider binary;
- generated provider with generic xgoja commands plus custom provider commands;
- mount collision negative test;
- config validation negative test;
- `BareCommand`, `WriterCommand`, and `GlazeCommand` fixture commands.

### Real package smoke tests

- Discord `bots list` and `bots <name> help` without opening a network session.
- Loupedeck command provider with fake/no-hardware metadata command first.
- CSS visual diff `verbs --help` or a no-browser metadata command.
- Minitrace query command against an in-memory test database.

## Open questions

1. Should command-set providers be entries in `providerapi.Package`, or should they live in a separate registry?
2. Should `CommandSetContext` expose the full xgoja `RuntimeFactory`, or a narrower interface?
3. How exactly should xgoja apply `mount` to Glazed command parents without mutating provider-owned command descriptions unexpectedly?
4. Should command-set providers be allowed to customize parser config, or should generated xgoja enforce one parser configuration?
5. How should generated xgoja represent command-provider host services in pure standalone binaries?
6. Should command providers support embedded filesystem sources like `jsverbs.embed`?

## Recommended first implementation slice

The smallest useful slice is:

1. Add command-set provider registry support.
2. Add `commandProviders` buildspec list.
3. Add `Host.AttachCommandProviders` using `glazedcli.AddCommandsToRootCommand`.
4. Add a fixture provider returning:
   - one `BareCommand`,
   - one `WriterCommand`,
   - one `GlazeCommand`.
5. Add generated example and docs.

Do not start with Loupedeck or Discord. Use a fixture first, then adapt one real package after the generated-binary mechanics are stable.

## References

### xgoja core

- `go-go-goja/pkg/xgoja/app/host.go`
- `go-go-goja/pkg/xgoja/app/root.go`
- `go-go-goja/pkg/xgoja/app/spec.go`
- `go-go-goja/pkg/xgoja/providerapi/registry.go`
- `go-go-goja/pkg/xgoja/providerapi/module.go`
- `go-go-goja/pkg/xgoja/providerapi/verbs.go`
- `go-go-goja/cmd/xgoja/internal/buildspec/spec.go`
- `go-go-goja/cmd/xgoja/internal/buildspec/validate.go`
- `go-go-goja/cmd/xgoja/internal/generate/main.go`

### Glazed command APIs

- `glazed/pkg/cmds/cmds.go`
- `glazed/pkg/cli/cobra.go`

### Existing sandbox command implementations

- `loupedeck/cmd/loupedeck/cmds/run/command.go`
- `loupedeck/cmd/loupedeck/cmds/verbs/bootstrap.go`
- `loupedeck/cmd/loupedeck/cmds/verbs/command.go`
- `discord-bot/pkg/botcli/command_root.go`
- `discord-bot/pkg/botcli/command_run.go`
- `discord-bot/pkg/botcli/runtime_factory.go`
- `discord-bot/pkg/botcli/options.go`
- `discord-bot/internal/jsdiscord/runtime.go`
- `css-visual-diff/internal/cssvisualdiff/jsapi/module.go`
- `css-visual-diff/internal/cssvisualdiff/verbcli/command.go`
- `css-visual-diff/internal/cssvisualdiff/verbcli/runtime_factory.go`
- `go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go`
- `go-minitrace/pkg/minitracejs/module.go`
- `go-minitrace/pkg/minitracejs/provider/provider.go`

### Ticket-local evidence

- `ttmp/2026/05/24/XGOJA-008--design-custom-xgoja-cli-verbs-for-third-party-javascript-sandboxes/scripts/01-inventory-custom-sandbox-commands.sh`
- `ttmp/2026/05/24/XGOJA-008--design-custom-xgoja-cli-verbs-for-third-party-javascript-sandboxes/sources/01-inventory-custom-sandbox-commands.txt`
