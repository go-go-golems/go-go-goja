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
    - Path: css-visual-diff/internal/cssvisualdiff/verbcli/command.go
      Note: CSS visual diff workflow verb command tree
    - Path: discord-bot/pkg/botcli/command_root.go
      Note: Discord bot repository command-tree generation
    - Path: go-go-goja/pkg/xgoja/app/host.go
      Note: Current fixed xgoja command attachment boundary
    - Path: go-go-goja/pkg/xgoja/app/root.go
      Note: Current eval and generic jsverbs command implementation
    - Path: go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go
      Note: Minitrace command-local JS runtime and jsverb invocation
    - Path: loupedeck/cmd/loupedeck/cmds/verbs/command.go
      Note: Loupedeck package-owned dynamic scene verb command tree
ExternalSources: []
Summary: Design guide for extending generated xgoja binaries with custom CLI command trees supplied by third-party Goja sandbox packages.
LastUpdated: 2026-05-24T23:55:00-04:00
WhatFor: Use this guide when designing or implementing xgoja support for package-provided custom CLI verbs beyond built-in eval run repl and generic jsverbs.
WhenToUse: Before changing xgoja buildspecs provider APIs or generated command attachment for third-party sandbox command trees.
---


# Custom xgoja CLI verbs for third-party JavaScript sandboxes

## Executive summary

Generated xgoja binaries currently know how to attach a fixed set of command families:

- `eval`: evaluate a JavaScript string in a selected runtime profile.
- `run`: run a JavaScript file in a selected runtime profile.
- `repl`: open an interactive JavaScript REPL for a selected runtime profile.
- `modules`: list compiled provider modules.
- `jsverbs`: scan configured JavaScript verb sources and mount those verbs under one command tree.

That is enough for generic JavaScript execution, but it is not enough for packages that already own richer JavaScript sandboxes. Packages such as `loupedeck`, `discord-bot`, `css-visual-diff`, and `go-minitrace` have package-specific command trees, discovery rules, host services, runtime factories, long-running sessions, and safety policies. The user wants xgoja to generate binaries that can expose those package-provided commands instead of only generic xgoja commands.

The recommended design is to add a **command provider** layer next to the existing module-provider layer:

```text
provider package
  ├─ providerapi.Module entries       -> require(...) modules selected by runtime profiles
  ├─ providerapi.VerbSource entries   -> generic JavaScript verb files scanned by xgoja
  └─ providerapi.CommandProvider      -> package-owned Cobra/Glazed command trees
```

A command provider should be able to contribute custom generated CLI commands such as:

```text
my-generated-binary
  run <file>                         # generic xgoja run, if enabled
  repl                               # generic xgoja repl, if enabled
  loupe scenes <scene> run ...       # loupedeck-owned scene runner
  bots <bot> run --bot-token ...     # discord-bot-owned bot runner
  cvd verbs compare-page ...         # css-visual-diff workflow verbs
  minitrace query <command> ...      # go-minitrace query command catalog
```

The key idea is that xgoja should not try to understand every third-party sandbox. It should provide a stable generated-binary host contract and let packages adapt their own command trees into it.

## Problem statement

The xgoja provider work in XGOJA-007 made packages importable as module providers. That solved this problem:

> A generated xgoja binary can compile Go-backed modules and make them available to JavaScript through `require(...)`.

It did not solve this different problem:

> A generated xgoja binary can expose package-specific CLI commands that create the correct sandbox, attach host services, discover JavaScript scripts, parse package-specific flags, and execute the JavaScript in the package's intended context.

The difference matters. `require("loupedeck/gfx")` is a module. `loupedeck run ./scene.js --duration 10s --send-interval 0ms` is an application command. `require("discord")` is a module. `discord-bot bots support run --bot-token ...` is an application command that opens a Discord session and stays alive until cancelled.

If xgoja only supports generic `run` and `jsverbs`, package authors will keep re-implementing command mounting outside xgoja. The generated binary will not be a useful composition target for real package-specific JS sandboxes.

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

- xgoja should not flatten Loupedeck to generic `run`. The package needs to contribute a command provider that can mount its scene runner and verb repository scanner.
- The Loupedeck command provider needs hardware/session config flags and lifecycle cancellation.

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

- Discord needs a command provider that can contribute a `bots` subtree, not just one command.
- It needs host config for tokens, guilds, sync-on-start behavior, and runtime module customization.
- It already has the right adapter idea: a command provider can expose `NewBotsCommand(bootstrap, opts...)` and let xgoja mount it.

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
- A public adapter package is required, such as `pkg/cssvisualdiff/xgoja` or `pkg/xgoja/provider`, which wraps `verbcli.NewLazyCommand` or exposes command-provider entries.

### go-minitrace

`go-minitrace` has command-local jsverb execution for query commands and now has a reusable `minitracejs` module provider from XGOJA-007.

Evidence:

- `go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go:24` runs a JS command into a Glazed processor.
- `go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go:42` scans a command source root with `jsverbs.ScanFS`.
- `go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go:56` builds a runtime with `go-go-goja/engine`.
- `go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go:63` registers the `minitrace` module with an SQL connection and runtime settings.
- `go-minitrace/cmd/go-minitrace/cmds/query/js_runtime.go:80` invokes the discovered JS verb in that runtime.

Design implication:

- Minitrace custom CLI verbs are not just arbitrary JS files. They are catalog-defined query commands that need a prepared DuckDB connection and structured output processor.
- xgoja needs a way for a command provider to own runtime setup and result emission.

## Design goals

1. Allow generated xgoja binaries to mount package-provided command trees.
2. Keep xgoja generic: it should not know Loupedeck, Discord, CSS visual diff, or minitrace semantics.
3. Reuse existing Glazed/Cobra command structures.
4. Support lazy discovery for expensive or environment-dependent command trees.
5. Support host services and command-provider config explicitly.
6. Keep provider modules, generic jsverbs, and custom command providers composable but separate.
7. Preserve generated-binary reproducibility: imports and command providers must be declared in `xgoja.yaml`.

## Proposed architecture

### New provider API concepts

Add a command-provider entry type to `pkg/xgoja/providerapi`.

```go
package providerapi

import "github.com/spf13/cobra"

type CommandProviderContext struct {
    Context context.Context
    PackageID string
    Name string
    Config json.RawMessage
    Host HostServices
    Providers *Registry
    RuntimeFactory RuntimeFactoryLike
    EmbeddedFS fs.FS
}

type CommandProvider struct {
    Name string
    DefaultMount string
    Description string
    ConfigSchema json.RawMessage
    New func(CommandProviderContext) (*cobra.Command, error)
}
```

`Registry.Package` would accept `CommandProvider` entries alongside `Module` and `VerbSource` entries:

```go
func Register(reg *providerapi.Registry) error {
    return reg.Package("loupedeck",
        providerapi.Module{...},
        providerapi.CommandProvider{
            Name: "scene-verbs",
            DefaultMount: "loupe",
            New: func(ctx providerapi.CommandProviderContext) (*cobra.Command, error) {
                bootstrap, err := loupedeeckcmd.BootstrapFromConfig(ctx.Config)
                if err != nil { return nil, err }
                return loupedeckverbs.NewCommand(bootstrap)
            },
        },
    )
}
```

### New buildspec section

Add a top-level `customCommands` or `commandProviders` section. Prefer `commandProviders` because it says where commands come from.

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

  - id: cvd-workflows
    package: css-visual-diff
    name: workflows
    mount: cvd
    config:
      repositories:
        - path: ./visual-tests

  - id: trace-query
    package: go-minitrace
    name: query-catalog
    mount: trace
    config:
      catalog: ./queries
      dbPath: :memory:
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
  | commandProviders[] select provider command trees
  v
generated main.go
  |
  | provider.Register(registry)
  v
providerapi.Registry
  |
  | ResolveCommandProvider(package, name)
  v
xgoja app.Host.AttachCommandProviders(root)
  |
  | provider.New(CommandProviderContext{...})
  v
*cobra.Command subtree supplied by package
  |
  | package-specific runtime factory / host services
  v
Goja sandbox executes user JS in package context
```

### Why not only extend generic jsverbs?

Generic jsverbs are still valuable for scripts that fit this shape:

```text
scan JS metadata -> build Glazed command -> xgoja RuntimeFactory -> InvokeInRuntime
```

Third-party sandboxes often need this shape instead:

```text
package discovery -> package-specific command description -> package-specific runtime/session/host -> package-specific result handling
```

For example:

- Loupedeck live scene verbs need hardware session options and render loops.
- Discord bot run commands need a Discord session and long-running cancellation.
- CSS visual diff verbs need browser lifecycle and artifact paths.
- Minitrace query commands need a prepared database connection and structured row processor.

Trying to force these through generic jsverbs would create hidden host-service hacks. A command-provider API makes the boundary explicit.

## Design patterns

### Pattern A: Static command tree provider

Use when a package has fixed commands and static config.

```go
providerapi.CommandProvider{
    Name: "tools",
    DefaultMount: "tools",
    New: func(ctx providerapi.CommandProviderContext) (*cobra.Command, error) {
        return mypkg.NewToolsCommand(mypkg.OptionsFromConfig(ctx.Config))
    },
}
```

Good for simple package-owned command groups.

### Pattern B: Lazy discovery command provider

Use when command discovery depends on filesystem repositories, config files, environment, or CLI flags. This is the closest match for Loupedeck and CSS visual diff.

```go
func NewLazyProviderCommand(ctx providerapi.CommandProviderContext) (*cobra.Command, error) {
    return &cobra.Command{
        Use: "verbs",
        DisableFlagParsing: true,
        RunE: func(cmd *cobra.Command, args []string) error {
            bootstrap, remaining, err := DiscoverBootstrapFromArgsAndConfig(args, ctx.Config)
            if err != nil { return err }
            resolved, err := NewCommand(bootstrap)
            if err != nil { return err }
            resolved.SetArgs(remaining)
            return resolved.ExecuteContext(cmd.Context())
        },
    }, nil
}
```

Benefits:

- Fast generated binary startup.
- Discovery can respect runtime working directory and CLI flags.
- Expensive scanners run only when the command is used.

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
cmdProvider.New = func(ctx providerapi.CommandProviderContext) (*cobra.Command, error) {
    return NewVerbsCommand(Options{
        RuntimeFactory: func(runCtx context.Context, source Source) (*engine.Runtime, error) {
            return ctx.RuntimeFactory.NewRuntime(runCtx, "main", require.WithLoader(source.RequireLoader()))
        },
    })
}
```

This pattern bridges generic xgoja runtime profiles with package-specific script discovery.

### Pattern E: Catalog command provider

Use when command metadata is not only in JS source annotations but in a package catalog. This matches `go-minitrace`.

```go
providerapi.CommandProvider{
    Name: "query-catalog",
    DefaultMount: "query",
    New: func(ctx providerapi.CommandProviderContext) (*cobra.Command, error) {
        cfg := DecodeConfig(ctx.Config)
        catalog, err := LoadCatalog(cfg.CatalogPath)
        if err != nil { return nil, err }
        return minitracequery.NewCatalogCommand(catalog, cfg.RuntimeSettings)
    },
}
```

## Implementation plan

### Phase 1: Add command-provider API to providerapi

Files:

- `go-go-goja/pkg/xgoja/providerapi/registry.go`
- `go-go-goja/pkg/xgoja/providerapi/module.go`
- new `go-go-goja/pkg/xgoja/providerapi/commands.go`

Tasks:

1. Add `CommandProvider` entry type.
2. Extend `Package` with `CommandProviders map[string]CommandProvider`.
3. Add `ResolveCommandProvider(packageID, name string)`.
4. Add tests for duplicate names, missing factory, and package cloning.

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
- possibly new `go-go-goja/pkg/xgoja/app/command_providers.go`

Tasks:

1. Add `Host.AttachCommandProviders(root)` after built-in commands.
2. For each selected command provider:
   - resolve provider entry;
   - marshal config;
   - construct `CommandProviderContext`;
   - call provider factory;
   - apply mount name if needed;
   - add subtree to root.
3. Detect collisions with built-ins and previous custom providers.
4. Add generated app tests with a fixture command provider.

Pseudocode:

```go
func (h *Host) AttachCommandProviders(root *cobra.Command) {
    for _, inst := range h.Spec.CommandProviders {
        provider, ok := h.Providers.ResolveCommandProvider(inst.Package, inst.Name)
        if !ok { addErrorStub(...); continue }
        data := marshalConfig(inst.Config)
        cmd, err := provider.New(providerapi.CommandProviderContext{
            Context: context.Background(),
            PackageID: inst.Package,
            Name: inst.Name,
            Config: data,
            Providers: h.Providers,
            RuntimeFactory: h.Factory,
        })
        if err != nil { addErrorStub(...); continue }
        if inst.Mount != "" { cmd.Use = inst.Mount }
        root.AddCommand(cmd)
    }
}
```

### Phase 4: Add fixture providers and examples

Create a first-party fixture command provider under `go-go-goja/pkg/xgoja/testprovider`.

Example provider:

```go
providerapi.CommandProvider{
    Name: "echo-tools",
    DefaultMount: "tools",
    New: func(ctx providerapi.CommandProviderContext) (*cobra.Command, error) {
        return &cobra.Command{
            Use: "tools",
            Short: "Fixture tools",
            RunE: ...,
        }, nil
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
./dist/custom-command-provider tools echo --message hello
```

### Phase 5: Adapt real packages

Recommended order:

1. `loupedeck`: expose `pkg/xgoja/commands` wrapping `cmd/loupedeck/cmds/verbs.NewLazyCommand` and optionally the scene `run` command.
2. `css-visual-diff`: move or wrap internal `verbcli` through a public package.
3. `discord-bot`: expose `pkg/xgoja/commands` wrapping `pkg/botcli.NewBotsCommand`.
4. `go-minitrace`: expose query catalog command provider after catalog/DB config is public and safe.

## API sketches for real package adapters

### Loupedeck adapter

```go
func Register(reg *providerapi.Registry) error {
    return reg.Package("loupedeck",
        providerapi.CommandProvider{
            Name: "scene-verbs",
            DefaultMount: "loupe",
            New: func(ctx providerapi.CommandProviderContext) (*cobra.Command, error) {
                cfg := DecodeConfig(ctx.Config)
                bootstrap := verbs.BootstrapFromConfig(cfg.Repositories)
                return verbs.NewCommand(bootstrap)
            },
        },
    )
}
```

### Discord adapter

```go
providerapi.CommandProvider{
    Name: "bots",
    DefaultMount: "bots",
    New: func(ctx providerapi.CommandProviderContext) (*cobra.Command, error) {
        cfg := DecodeConfig(ctx.Config)
        bootstrap := botcli.Bootstrap{Repositories: cfg.Repositories}
        return botcli.NewBotsCommand(bootstrap,
            botcli.WithAppName("discord"),
            botcli.WithRuntimeFactory(newXGojaAwareRuntimeFactory(ctx)),
        )
    },
}
```

### CSS visual diff adapter

```go
providerapi.CommandProvider{
    Name: "workflows",
    DefaultMount: "cvd",
    New: func(ctx providerapi.CommandProviderContext) (*cobra.Command, error) {
        bootstrap := verbcli.BootstrapFromConfig(ctx.Config)
        return verbcli.NewCommand(bootstrap)
    },
}
```

This requires moving the adapter out of `internal` or making generated binaries target the same module.

### Minitrace adapter

```go
providerapi.CommandProvider{
    Name: "query-catalog",
    DefaultMount: "query",
    New: func(ctx providerapi.CommandProviderContext) (*cobra.Command, error) {
        cfg := DecodeConfig(ctx.Config)
        return minitracecmd.NewQueryCatalogCommand(cfg.CatalogPath, cfg.RuntimeSettings)
    },
}
```

## Safety and lifecycle rules

1. Command providers must declare config schemas for dangerous capabilities.
2. Long-running commands must honor `cmd.Context()` and OS cancellation.
3. Commands that open devices, browsers, Discord sessions, or databases must close them with `defer` or command lifecycle hooks.
4. Generated xgoja should not pass global mutable state implicitly.
5. Provider command factories should fail early when required host services are missing.
6. Lazy commands should preserve help/output behavior when resolving into real command trees.
7. Mount collisions should be doctor errors, not runtime surprises.

## Testing strategy

### Unit tests

- provider registry tests for command provider entries;
- buildspec validation tests;
- generated main rendering tests;
- app host command attachment tests with fixture provider.

### Integration tests

- generated fixture command provider binary;
- generated provider with generic xgoja commands plus custom provider commands;
- mount collision negative test;
- config validation negative test.

### Real package smoke tests

- Loupedeck command provider with fake/no-hardware mode first.
- Discord `bots list` and `bots <name> help` without opening a network session.
- CSS visual diff `verbs --help` or a no-browser metadata command.
- Minitrace query command against an in-memory test database.

## Open questions

1. Should command providers be entries in `providerapi.Package`, or should they live in a separate registry?
2. Should generated xgoja pass `RuntimeFactory` into command providers directly, or only a narrower interface?
3. Should command providers be allowed to mutate root persistent flags?
4. How should generated xgoja represent command-provider host services in pure standalone binaries?
5. Should custom command provider config support embedding filesystem sources like `jsverbs.embed`?
6. Should command providers be able to opt into lazy construction to support dynamic help?

## Recommended first implementation slice

The smallest useful slice is:

1. Add command-provider registry support.
2. Add `commandProviders` buildspec list.
3. Add `Host.AttachCommandProviders`.
4. Add a fixture command provider and generated example.
5. Add docs.

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
