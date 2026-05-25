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
Summary: Design guide for extending generated xgoja binaries with custom Glazed command sets and module-provided Glazed configuration sections for both custom and built-in commands.
LastUpdated: 2026-05-25T00:55:00-04:00
WhatFor: Use this guide when designing or implementing xgoja support for package-provided custom Glazed commands and composable module configuration sections across eval run repl jsverbs and custom command providers.
WhenToUse: Before changing xgoja buildspecs provider APIs or generated command attachment for third-party sandbox command sets.
---

# Custom xgoja CLI verbs for third-party JavaScript sandboxes

## Revision note

This document was revised after review to keep the extension point inside the Go-Go-Golems / Glazed command ecosystem. The original draft proposed that provider command factories return `*cobra.Command`. The revised design instead has providers return **Glazed commands** (`cmds.Command` values implemented as `cmds.BareCommand`, `cmds.WriterCommand`, or `cmds.GlazeCommand`, usually `BareCommand`) plus optional grouping metadata. xgoja remains responsible for converting those Glazed commands into Cobra commands with the same parser and middleware path already used by generated xgoja jsverbs.

A second revision adds **module-provided Glazed configuration sections**. This is important because xgoja cannot code-generate arbitrary semantic integrations such as "run a Discord bot controlled by a Loupedeck". Instead, xgoja compiles selected providers and passes their descriptors to a package-owned command provider. A command provider such as Loupedeck can inspect the selected modules, ask each module for extra Glazed sections, include those sections in its final command, and later initialize each module by decoding its own section with `values.Values.DecodeSectionInto`.

A third revision clarifies that module-provided sections are **not only for custom command providers**. They should also be available to built-in xgoja commands (`run`, `repl`, `jsverbs`, and eventually `eval`) based on the command's selected runtime profile. Built-ins aggregate sections from the modules in that profile, expose them as CLI flags, create the runtime, and then call module runtime initializers with the parsed `*values.Values` before executing the script, opening the REPL, or invoking a JS verb.

This is the better boundary because:

- command schemas, flags, arguments, parents, help text, and output behavior stay in Glazed;
- packages can return structured row output when appropriate (`GlazeCommand`), direct writer output when needed (`WriterCommand`), or normal side-effect/session commands (`BareCommand`);
- modules can expose reusable config sections without owning the final command tree;
- built-in xgoja commands can expose module-specific runtime options without hard-coding module semantics;
- command providers can compose selected modules without xgoja understanding domain semantics;
- xgoja can reuse `glazedcli.AddCommandsToRootCommand` and `glazedcli.BuildCobraCommandFromCommand`;
- command providers do not need to construct Cobra directly except for rare app-root integration code outside this provider API;
- runtime code should decode typed settings with `DecodeSectionInto`, not scattered `GetString`/`GetBool` accessors.

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
  ├─ providerapi.Module entries             -> require(...) modules selected by runtime profiles
  ├─ providerapi.ConfigSectionCapability    -> extra Glazed sections exposed by selected modules
  ├─ providerapi.RuntimeInitializerCapability -> initialize run/repl/jsverbs runtimes from parsed sections
  ├─ providerapi.ComponentInitializerCapability -> initialize package-level components for custom commands
  ├─ providerapi.VerbSource entries         -> generic JavaScript verb files scanned by xgoja
  └─ providerapi.CommandSetProvider         -> package-owned Glazed command sets that can compose selected modules
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

For cross-package compositions, the command provider owns the orchestration. xgoja does not synthesize a bespoke `DiscordLoupedeckBotCommand`. Instead, the generated binary compiles a Loupedeck command provider and a Discord module provider; the Loupedeck command provider can discover that the Discord module was selected, import its Glazed configuration section, and call its typed initializer at runtime.

For built-in execution commands, xgoja owns the orchestration but only at the generic runtime level. `run`, `repl`, and `jsverbs` can aggregate sections from the selected runtime profile and call runtime initializer capabilities. They still do not know what Discord, Loupedeck, CSS visual diff, or minitrace settings mean.

## Problem statement

The xgoja provider work in XGOJA-007 made packages importable as module providers. That solved this problem:

> A generated xgoja binary can compile Go-backed modules and make them available to JavaScript through `require(...)`.

It did not solve this different problem:

> A generated xgoja binary can expose package-specific CLI commands that create the correct sandbox, attach host services, discover JavaScript scripts, parse package-specific flags, and execute the JavaScript in the package's intended context.

The difference matters. `require("loupedeck/gfx")` is a module. A Loupedeck scene runner with hardware/session flags is an application command. `require("discord")` is a module. `discord-bot bots support run --bot-token ...` is an application command that opens a Discord session and stays alive until cancelled.

If xgoja only supports generic `run` and `jsverbs`, package authors will keep re-implementing command mounting outside xgoja. The generated binary will not be a useful composition target for real package-specific JS sandboxes.

There is a second problem: final commands often need configuration from more than one provider. A Loupedeck-controlled Discord bot needs Loupedeck hardware/profile/mapping settings and Discord token/guild/channel/script settings. xgoja cannot know how those domains interact, but it can let selected modules expose Glazed sections and typed initialization hooks so a package-owned command provider can compose them.

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

## Module-provided Glazed sections

Command providers are only half of the composition story. Modules should also be able to contribute Glazed sections that describe provider-specific runtime configuration. Those sections become part of the final command schema when a command provider chooses to include them.

This solves the Discord + Loupedeck case:

- the Discord module provider exposes a `discord` section with token, guild, channel, script, intents, and similar settings;
- the Loupedeck module or command provider exposes a `loupedeck` section with device, profile, page, and mapping settings;
- the Loupedeck command provider owns the final `loupe bot` command;
- while building that command, it iterates over selected xgoja modules, asks them for extra sections, and adds compatible sections to the command description;
- in `Run`, the Loupedeck command decodes its own section and asks each participating module to initialize itself from the same parsed `*values.Values`.

The final command does **not** call `vals.GetString("discord-token")` manually. Each provider owns a typed settings struct and decodes its own section:

```go
type DiscordSettings struct {
    TokenEnv  string `glazed:"token-env"`
    GuildID   string `glazed:"guild-id"`
    ChannelID string `glazed:"channel-id"`
    Script    string `glazed:"bot-script"`
}

func (m *DiscordModule) InitComponentFromSections(ctx context.Context, vals *values.Values) (providerapi.InitializedModule, error) {
    var cfg DiscordSettings
    if err := vals.DecodeSectionInto("discord", &cfg); err != nil {
        return nil, err
    }
    return NewDiscordBot(ctx, cfg)
}
```

This makes sections the contract and keeps initialization typed, local, and testable.

## Built-in command section aggregation

Module-provided sections should apply to both custom command providers and built-in xgoja commands. The rule should be:

> Any command that creates a runtime from a runtime profile may aggregate Glazed sections from the modules selected by that profile, expose those sections on the command, and call module runtime initializers after the runtime is created.

This covers:

- `run`: add module sections to the `run` command and initialize modules before executing the file.
- `repl`: add module sections to the REPL command and initialize modules before handing control to the user.
- `jsverbs`: add module sections to every generated verb command for the jsverbs runtime profile and initialize modules before invoking the verb.
- `eval`: should eventually be converted from raw Cobra to a Glazed `BareCommand` so it can follow the same path.

The built-in commands do not need to understand domain semantics. They only need generic helper functions:

```go
func SectionsForRuntimeProfile(ctx SectionContext, profile string) ([]schema.Section, []ModuleDescriptor, error)

func InitRuntimeFromSections(
    ctx context.Context,
    vals *values.Values,
    rt RuntimeHandle,
    modules []ModuleDescriptor,
) error
```

Then `run` can be shaped like this:

```go
func newRunCommand(factory *RuntimeFactory, spec *Spec) cmds.Command {
    profile := commandRuntime(spec.Commands.Run, firstRuntime(spec))
    moduleSections, selected, err := factory.SectionsForRuntimeProfile(profile)
    if err != nil { return errorCommand(err) }

    sections := append([]schema.Section{run.DefaultSection(profile)}, moduleSections...)

    return &runCommand{
        CommandDescription: cmds.NewCommandDescription(
            "run",
            cmds.WithSections(sections...),
        ),
        selectedModules: selected,
    }
}

func (c *runCommand) Run(ctx context.Context, vals *values.Values) error {
    var settings runSettings
    if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
        return err
    }

    rt, err := c.factory.NewRuntime(ctx, settings.Runtime)
    if err != nil { return err }
    defer rt.Close(ctx)

    if err := providerapi.InitRuntimeFromSections(ctx, vals, rt, c.selectedModules); err != nil {
        return err
    }

    return rt.RunFile(ctx, settings.File)
}
```

For `jsverbs`, the generated command's schema should be:

```text
verb-declared sections
+ xgoja command/default section
+ module sections from spec.Commands.JSVerbs.Runtime
```

and the invoker should initialize the runtime from parsed values before calling `registry.InvokeInRuntime`.

This gives a generated CLI like:

```text
xgoja run ./bot.js \
  --runtime main \
  --discord-token-env DISCORD_BOT_TOKEN \
  --discord-guild-id 123

xgoja repl \
  --runtime main \
  --discord-token-env DISCORD_BOT_TOKEN

xgoja verbs announce \
  --discord-token-env DISCORD_BOT_TOKEN \
  --discord-channel-id 456
```

The command-specific code remains generic. The module owns section shape and typed decoding; the built-in command only exposes and forwards parsed sections.

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

Add two related extension points to `pkg/xgoja/providerapi`:

1. **Command set providers**: package-owned factories that return Glazed commands.
2. **Module configuration capabilities**: module-owned descriptors that expose Glazed sections and typed initialization hooks.

```go
package providerapi

import (
    "context"
    "encoding/json"

    "github.com/dop251/goja"
    "github.com/go-go-golems/glazed/pkg/cli"
    "github.com/go-go-golems/glazed/pkg/cmds"
    "github.com/go-go-golems/glazed/pkg/cmds/schema"
    "github.com/go-go-golems/glazed/pkg/cmds/values"
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

    // SelectedModules are the modules selected by the buildspec/runtime profile
    // for the command provider's execution context. Command providers can ask
    // these modules for extra Glazed sections and initialization hooks.
    SelectedModules []ModuleDescriptor
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

type ModuleDescriptor struct {
    PackageID string
    ModuleID string
    Module Module
    Capabilities []ModuleCapability
}

type ModuleCapability interface {
    CapabilityID() string
}

type ConfigSectionCapability interface {
    ModuleCapability
    ConfigSections(SectionContext) ([]schema.Section, error)
}

// RuntimeInitializerCapability is used by built-in xgoja commands such as
// run/repl/jsverbs/eval. The runtime already exists; the module configures it
// from parsed Glazed sections.
type RuntimeInitializerCapability interface {
    ModuleCapability
    InitRuntimeFromSections(context.Context, *values.Values, RuntimeHandle) error
}

// ComponentInitializerCapability is used by package-owned command providers
// that need initialized domain objects, not only JS runtime mutation.
type ComponentInitializerCapability interface {
    ModuleCapability
    InitComponentFromSections(context.Context, *values.Values) (InitializedModule, error)
}

type RuntimeHandle interface {
    Runtime() *goja.Runtime
    Close(context.Context) error
}

type InitializedModule interface {
    ModuleID() string
    Close(context.Context) error
}
```

The important design point is that command providers and built-ins do not know all possible modules at compile time. They know only the capability interfaces they care about. For example, a Loupedeck command provider can ask selected modules for `ConfigSectionCapability`, include those sections, and at runtime ask compatible modules to initialize domain objects through `ComponentInitializerCapability`. Built-in `run`, `repl`, and `jsverbs` commands use the same sections but call `RuntimeInitializerCapability` after creating the runtime.

`Registry.Package` would accept `CommandSetProvider` entries alongside `Module`, `VerbSource`, and module capabilities:

```go
func Register(reg *providerapi.Registry) error {
    return reg.Package("loupedeck",
        providerapi.Module{...},
        providerapi.CommandSetProvider{
            Name: "bot-controller",
            DefaultMount: "loupe",
            New: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
                cmd, err := loupedeckcmd.NewBotControllerCommand(ctx)
                if err != nil { return nil, err }
                return &providerapi.CommandSet{Commands: []cmds.Command{cmd}}, nil
            },
        },
    )
}
```

A Discord provider would expose module capabilities, not a Loupedeck-specific command:

```go
func Register(reg *providerapi.Registry) error {
    return reg.Package("discord",
        providerapi.Module{...},
        discord.NewConfigSectionCapability(),
        discord.NewRuntimeInitializerCapability(),
        discord.NewComponentInitializerCapability(), // optional, for package-level orchestration
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
    RuntimeProfile string `yaml:"runtimeProfile" json:"runtimeProfile,omitempty"`
    ModuleSelector []string `yaml:"modules" json:"modules,omitempty"`
    Config map[string]any `yaml:"config" json:"config,omitempty"`
    Lazy bool `yaml:"lazy" json:"lazy,omitempty"`
}
```

`RuntimeProfile` or `ModuleSelector` tells xgoja which selected modules should be visible to the command provider. If omitted, the provider can receive the default profile's selected modules. This lets a command provider compose only the modules intentionally selected by the generated binary author.

Built-in commands do not need a separate selector if they already have a runtime profile. They aggregate module sections from their effective runtime:

```yaml
commands:
  run:
    enabled: true
    runtime: main
    moduleSections: true
  repl:
    enabled: true
    runtime: main
    moduleSections: true
  jsverbs:
    enabled: true
    runtime: main
    moduleSections: true
```

`moduleSections` can default to `true` for commands that create a runtime. A generated binary author can disable it for very small or safety-sensitive binaries.

### Generated runtime flow

```text
xgoja.yaml
  |
  | packages[] import provider packages
  | runtimeProfiles[] select modules
  | commandProviders[] select provider command sets and module profile/selector
  v
generated main.go
  |
  | provider.Register(registry)
  v
providerapi.Registry
  |
  | built-ins: ResolveSelectedModules(command.runtime)
  | custom: ResolveCommandSetProvider(package, name)
  | custom: ResolveSelectedModules(runtimeProfile/modules)
  v
xgoja app.Host.AttachDefaultCommands(root)
  |
  | run/repl/jsverbs aggregate module sections
  | run/repl/jsverbs call RuntimeInitializerCapability after runtime creation
  v
xgoja app.Host.AttachCommandProviders(root)
  |
  | provider.New(CommandSetContext{SelectedModules: ...})
  v
package-owned command provider
  |
  | asks selected modules for ConfigSections(...)
  | builds []cmds.Command with those sections
  v
glazedcli.AddCommandsToRootCommand(root, commands, ...)
  |
  v
Cobra commands generated by Glazed bridge
  |
  | at execution time, command Run(ctx, vals)
  | decodes own section with DecodeSectionInto
  | asks selected modules to InitComponentFromSections(ctx, vals)
  v
initialized module graph owned by package command
```

### Why not return Cobra?

Returning Cobra would work mechanically, but it would bypass the ecosystem conventions we already rely on:

- Glazed schemas for flags and arguments.
- Glazed output processors for row-producing commands.
- Consistent environment/config/default middlewares.
- Glazed help and command description serialization.
- Existing `jsverbs.Registry.CommandForVerbWithInvoker` return type (`cmds.Command`).

A command provider that needs long-running behavior can still implement `BareCommand`. It does not need Cobra to block, handle context cancellation, or manage side effects.

### Why not generate cross-product commands?

xgoja should not generate semantic glue such as `DiscordLoupedeckBotCommand`. That would require xgoja to know how Discord bots, Loupedeck controls, browser workflows, and minitrace query engines interact. The number of possible combinations grows as more packages add providers.

Instead, xgoja should provide a generic composition substrate:

- generated code imports and registers providers;
- the buildspec chooses command providers and modules;
- modules expose sections and initialization hooks;
- a package-owned command provider decides which capabilities it understands;
- the command provider wires initialized modules together at runtime.

This keeps xgoja simple and makes integration semantics live in the package that owns them.

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

### Pattern E: Module-provided configuration sections

Use when a module wants to be configurable by an ultimate command that it does not own. The module exports sections and typed initialization logic. The final command provider chooses whether to include and initialize it.

```go
type DiscordSettings struct {
    TokenEnv  string `glazed:"token-env"`
    GuildID   string `glazed:"guild-id"`
    ChannelID string `glazed:"channel-id"`
    Script    string `glazed:"bot-script"`
}

type DiscordCapability struct{}

func (DiscordCapability) CapabilityID() string { return "discord.bot" }

func (DiscordCapability) ConfigSections(ctx providerapi.SectionContext) ([]schema.Section, error) {
    section, err := schema.NewSection(
        "discord",
        "Discord",
        schema.WithPrefix("discord-"),
        schema.WithFields(
            fields.New("token-env", fields.TypeString, fields.WithDefault("DISCORD_BOT_TOKEN")),
            fields.New("guild-id", fields.TypeString),
            fields.New("channel-id", fields.TypeString),
            fields.New("bot-script", fields.TypeString),
        ),
    )
    if err != nil { return nil, err }
    return []schema.Section{section}, nil
}

func (DiscordCapability) InitRuntimeFromSections(ctx context.Context, vals *values.Values, rt providerapi.RuntimeHandle) error {
    var cfg DiscordSettings
    if err := vals.DecodeSectionInto("discord", &cfg); err != nil {
        return err
    }
    return discord.InstallRuntimeGlobals(ctx, rt.Runtime(), cfg)
}

func (DiscordCapability) InitComponentFromSections(ctx context.Context, vals *values.Values) (providerapi.InitializedModule, error) {
    var cfg DiscordSettings
    if err := vals.DecodeSectionInto("discord", &cfg); err != nil {
        return nil, err
    }
    return discord.NewBot(ctx, cfg)
}
```

This is preferred over having the final command read individual values with `vals.GetString`. The provider that owns the section also owns decoding and validation of the section.

### Pattern F: Built-in command composing runtime-profile module sections

Use when a built-in xgoja command creates a runtime from a profile. The command adds its own base section, appends sections from selected modules, and calls runtime initializers after creating the runtime.

```go
func newReplCommand(factory *RuntimeFactory, spec *Spec) cmds.Command {
    profile := commandRuntime(spec.Commands.Repl, firstRuntime(spec))
    moduleSections, selected, err := factory.SectionsForRuntimeProfile(profile)
    if err != nil { return errorCommand(err) }

    return &replCommand{
        CommandDescription: cmds.NewCommandDescription(
            "repl",
            cmds.WithSections(append([]schema.Section{repl.DefaultSection(profile)}, moduleSections...)...),
        ),
        selectedModules: selected,
    }
}

func (c *replCommand) Run(ctx context.Context, vals *values.Values) error {
    var cfg replSettings
    if err := vals.DecodeSectionInto(schema.DefaultSlug, &cfg); err != nil {
        return err
    }

    rt, err := c.factory.NewRuntime(ctx, cfg.Runtime)
    if err != nil { return err }
    defer rt.Close(ctx)

    if err := providerapi.InitRuntimeFromSections(ctx, vals, rt, c.selectedModules); err != nil {
        return err
    }

    return startREPL(ctx, rt)
}
```

`run` and `jsverbs` follow the same shape. `jsverbs` differs only because the base sections come from the JavaScript verb metadata before module sections are appended.

### Pattern G: Command provider composing selected module sections

Use when a command provider owns an orchestration surface and wants selected modules to participate. Example: a Loupedeck command provider exposes `loupe bot`, includes its own Loupedeck section, discovers the selected Discord module's section, then initializes Discord and binds it to the deck.

```go
type LoupedeckBotCommand struct {
    *cmds.CommandDescription
    modules []providerapi.ModuleDescriptor
}

func NewBotControllerCommand(ctx providerapi.CommandSetContext) (cmds.BareCommand, error) {
    sections := []schema.Section{loupedeck.ControlSection()}

    for _, mod := range ctx.SelectedModules {
        for _, cap := range mod.Capabilities {
            sectionCap, ok := cap.(providerapi.ConfigSectionCapability)
            if !ok { continue }
            moduleSections, err := sectionCap.ConfigSections(providerapi.SectionContext{
                CommandProviderID: ctx.Name,
                ModuleID: mod.ModuleID,
            })
            if err != nil { return nil, err }
            sections = append(sections, moduleSections...)
        }
    }

    return &LoupedeckBotCommand{
        CommandDescription: cmds.NewCommandDescription(
            "bot",
            cmds.WithParents("loupe"),
            cmds.WithShort("Run a Loupedeck-controlled bot"),
            cmds.WithSections(sections...),
        ),
        modules: ctx.SelectedModules,
    }, nil
}

func (c *LoupedeckBotCommand) Run(ctx context.Context, vals *values.Values) error {
    var loupeCfg loupedeck.ControlSettings
    if err := vals.DecodeSectionInto("loupedeck", &loupeCfg); err != nil {
        return err
    }

    deck, err := loupedeck.NewController(ctx, loupeCfg)
    if err != nil { return err }
    defer deck.Close(ctx)

    for _, mod := range c.modules {
        for _, cap := range mod.Capabilities {
            initCap, ok := cap.(providerapi.ComponentInitializerCapability)
            if !ok { continue }
            initialized, err := initCap.InitComponentFromSections(ctx, vals)
            if err != nil { return err }
            defer initialized.Close(ctx)

            if bindable, ok := initialized.(loupedeck.Bindable); ok {
                if err := bindable.BindToLoupedeck(ctx, deck); err != nil {
                    return err
                }
            }
        }
    }

    return deck.Run(ctx)
}
```

The generated CLI can then expose both section prefixes on one final command:

```text
xgoja loupe bot \
  --loupe-device auto \
  --loupe-profile streaming \
  --discord-token-env DISCORD_BOT_TOKEN \
  --discord-guild-id 123 \
  --discord-channel-id 456 \
  --discord-bot-script ./bot.js
```

xgoja did not generate Discord/Loupedeck glue. It only made the selected module descriptors available to the Loupedeck command provider.

### Pattern H: Catalog command provider

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

### Phase 1: Add command-set and module capability APIs to providerapi

Files:

- `go-go-goja/pkg/xgoja/providerapi/registry.go`
- `go-go-goja/pkg/xgoja/providerapi/module.go`
- new `go-go-goja/pkg/xgoja/providerapi/commands.go`

Tasks:

1. Add `CommandSetProvider` entry type.
2. Add `CommandSet` return type containing `[]cmds.Command` and optional parser config.
3. Add module capability interfaces for `ConfigSections`, `InitRuntimeFromSections`, and `InitComponentFromSections`.
4. Extend `Package` with `CommandSetProviders map[string]CommandSetProvider` and module capability metadata.
5. Add `ResolveCommandSetProvider(packageID, name string)` and selected-module descriptor resolution.
6. Add tests for duplicate names, missing factory, capability registration, and package cloning.

### Phase 2: Add buildspec support

Files:

- `go-go-goja/cmd/xgoja/internal/buildspec/spec.go`
- `go-go-goja/cmd/xgoja/internal/buildspec/validate.go`
- `go-go-goja/cmd/xgoja/internal/generate/main.go`
- `go-go-goja/pkg/xgoja/app/spec.go`

Tasks:

1. Add `CommandProviders []CommandProviderInstance` to buildspec and runtime app spec.
2. Add command-provider runtime profile or module selector fields.
3. Validate package ID, provider name, duplicate IDs, mount collisions, selected module IDs, and config shape.
4. Include command providers and selected module descriptors in embedded spec JSON.
5. Add docs in `cmd/xgoja/doc/02-buildspec.md`.

### Phase 3: Add built-in command section aggregation

Files:

- `go-go-goja/pkg/xgoja/app/run.go`
- `go-go-goja/pkg/xgoja/app/tui.go`
- `go-go-goja/pkg/xgoja/app/root.go`
- `go-go-goja/pkg/xgoja/app/factory.go`
- new `go-go-goja/pkg/xgoja/app/module_sections.go`

Tasks:

1. Add helpers that resolve selected module descriptors for a runtime profile.
2. Add helpers that collect `ConfigSectionCapability` sections for a runtime profile.
3. Add helpers that call `RuntimeInitializerCapability.InitRuntimeFromSections` after runtime creation.
4. Update `run` to append module sections and initialize runtime before executing the file.
5. Update `repl`/TUI to append module sections and initialize runtime before starting the session.
6. Update `jsverbs` so each generated verb command appends module sections for `spec.Commands.JSVerbs.Runtime` and initializes runtime before `InvokeInRuntime`.
7. Convert `eval` to a Glazed `BareCommand` or document why `eval` is excluded from module section aggregation in the first slice.

### Phase 4: Attach custom command providers in generated app

Files:

- `go-go-goja/pkg/xgoja/app/host.go`
- `go-go-goja/pkg/xgoja/app/root.go`
- new `go-go-goja/pkg/xgoja/app/command_providers.go`

Tasks:

1. Add `Host.AttachCommandProviders(root)` after built-in commands.
2. For each selected command provider:
   - resolve provider entry;
   - marshal config;
   - resolve selected module descriptors and their capabilities;
   - construct `CommandSetContext` with `SelectedModules`;
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
            SelectedModules: h.ResolveSelectedModules(inst),
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

### Phase 5: Add fixture providers and examples

Create a first-party fixture command provider under `go-go-goja/pkg/xgoja/testprovider`. Include both a command provider fixture and a module capability fixture so the first implementation proves section composition, not only command mounting.

Example provider:

```go
providerapi.CommandSetProvider{
    Name: "echo-tools",
    DefaultMount: "tools",
    New: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
        sections := []schema.Section{fixture.CommandSection()}
        for _, mod := range ctx.SelectedModules {
            for _, cap := range mod.Capabilities {
                if sc, ok := cap.(providerapi.ConfigSectionCapability); ok {
                    ss, err := sc.ConfigSections(providerapi.SectionContext{})
                    if err != nil { return nil, err }
                    sections = append(sections, ss...)
                }
            }
        }
        cmd := &echoCommand{CommandDescription: cmds.NewCommandDescription(
            "echo",
            cmds.WithParents("tools"),
            cmds.WithSections(sections...),
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

### Phase 6: Adapt real packages

Recommended order:

1. `discord-bot`: expose a Discord module capability with `ConfigSections`, `InitRuntimeFromSections`, and optionally `InitComponentFromSections` first; only then refactor existing bot commands to return `[]cmds.Command` where useful.
2. `loupedeck`: expose a Loupedeck control section and a command provider that can aggregate selected module sections; wrap live scene/controller invokers as `BareCommand`.
3. `css-visual-diff`: move or wrap internal `verbcli` through a public package returning `[]cmds.Command`, and expose browser/artifact config sections through module capabilities.
4. `go-minitrace`: expose query catalog commands as `GlazeCommand` values after catalog/DB config is public and safe; expose database/runtime settings as typed sections decoded with `DecodeSectionInto`.

## API sketches for real package adapters

### Loupedeck adapter

The Loupedeck package should provide a normal command set provider for its own commands. Some of those commands can compose selected module sections.

```go
func Register(reg *providerapi.Registry) error {
    return reg.Package("loupedeck",
        providerapi.Module{...},
        loupedeck.NewControlConfigCapability(),
        providerapi.CommandSetProvider{
            Name: "bot-controller",
            DefaultMount: "loupe",
            New: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
                cmd, err := loupedeckcmd.NewBotControllerCommand(ctx)
                if err != nil { return nil, err }
                return &providerapi.CommandSet{Commands: []cmds.Command{cmd}}, nil
            },
        },
    )
}
```

The `Run` method of `NewBotControllerCommand` decodes Loupedeck settings with `DecodeSectionInto("loupedeck", &settings)`, initializes a deck controller, then initializes any selected modules that implement `ComponentInitializerCapability`.

### Discord adapter

The Discord package should not need to know about Loupedeck. It exposes a module, a section, and an initializer. If the initialized Discord object wants to participate in Loupedeck control, it can also implement a small domain interface such as `loupedeck.Bindable`.

```go
func Register(reg *providerapi.Registry) error {
    return reg.Package("discord",
        providerapi.Module{...},
        providerapi.ModuleCapability(discord.BotCapability{}),
    )
}

type BotCapability struct{}

func (BotCapability) ConfigSections(ctx providerapi.SectionContext) ([]schema.Section, error) {
    return []schema.Section{discord.Section()}, nil
}

func (BotCapability) InitRuntimeFromSections(ctx context.Context, vals *values.Values, rt providerapi.RuntimeHandle) error {
    var cfg discord.Settings
    if err := vals.DecodeSectionInto("discord", &cfg); err != nil {
        return err
    }
    return discord.InstallRuntimeGlobals(ctx, rt.Runtime(), cfg)
}

func (BotCapability) InitComponentFromSections(ctx context.Context, vals *values.Values) (providerapi.InitializedModule, error) {
    var cfg discord.Settings
    if err := vals.DecodeSectionInto("discord", &cfg); err != nil {
        return nil, err
    }
    return discord.NewBot(ctx, cfg)
}
```

The generated buildspec selects both providers, but the Loupedeck command provider owns orchestration:

```yaml
runtimeProfiles:
  - id: main
    modules:
      - loupedeck
      - discord

commandProviders:
  - id: loupe-bot
    package: loupedeck
    name: bot-controller
    mount: loupe
    runtimeProfile: main
```

Resulting CLI:

```text
xgoja loupe bot \
  --loupe-device auto \
  --discord-token-env DISCORD_BOT_TOKEN \
  --discord-guild-id 123 \
  --discord-channel-id 456
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

1. Command-set providers and module capabilities must declare config schemas/sections for dangerous capabilities.
2. Section slugs and prefixes must be stable and collision-checked before mounting commands.
3. Built-in commands must apply the same section aggregation rules as custom command providers when they create runtimes.
4. Runtime initializers must run after runtime creation and before script execution, REPL startup, or JS verb invocation.
5. Runtime code must decode typed structs with `values.Values.DecodeSectionInto`, not scattered field accessors.
6. Long-running `BareCommand` implementations must honor `ctx` cancellation.
7. Commands that open devices, browsers, Discord sessions, or databases must close them with `defer` or command lifecycle hooks.
8. Commands that emit rows should be `GlazeCommand`, not `BareCommand` that prints JSON manually.
9. Commands that need textual output should be `WriterCommand` and use the supplied writer.
10. Generated xgoja should not pass global mutable state implicitly.
11. Provider command factories should fail early when required host services are missing.
12. Mount and section collisions should be doctor errors, not runtime surprises.

## Testing strategy

### Unit tests

- provider registry tests for command-set provider entries and module capabilities;
- buildspec validation tests, including selected module profile/selector tests;
- generated main rendering tests;
- app host command attachment tests with fixture provider;
- built-in `run`, `repl`, and `jsverbs` section aggregation tests;
- runtime initializer invocation tests for built-in commands;
- config section aggregation and section collision tests;
- `DecodeSectionInto` fixture initializer tests;
- `applyMountPrefix` tests for command parent rewriting.

### Integration tests

- generated fixture command-provider binary;
- generated provider with generic xgoja commands plus custom provider commands;
- generated fixture where built-in `run` exposes a selected module's section and calls its runtime initializer;
- generated fixture where `jsverbs` exposes selected module sections on discovered verb commands;
- generated fixture where a command provider imports selected module sections;
- mount collision and section collision negative tests;
- config validation negative test;
- `BareCommand`, `WriterCommand`, and `GlazeCommand` fixture commands;
- fixture module initializer that decodes with `DecodeSectionInto`.

### Real package smoke tests

- Discord `bots list` and `bots <name> help` without opening a network session.
- Loupedeck command provider with fake/no-hardware metadata command first.
- CSS visual diff `verbs --help` or a no-browser metadata command.
- Minitrace query command against an in-memory test database.

## Open questions

1. Should command-set providers and module capabilities be entries in `providerapi.Package`, or should capabilities live in a separate registry?
2. Should `CommandSetContext` expose the full xgoja `RuntimeFactory`, or a narrower interface?
3. How exactly should xgoja apply `mount` to Glazed command parents without mutating provider-owned command descriptions unexpectedly?
4. Should command-set providers be allowed to customize parser config, or should generated xgoja enforce one parser configuration?
5. How should generated xgoja represent command-provider host services in pure standalone binaries?
6. Should command providers support embedded filesystem sources like `jsverbs.embed`?
7. What is the exact buildspec syntax for selecting modules visible to a command provider: `runtimeProfile`, explicit `modules`, or both?
8. Should module initializers receive command-provider config, or only parsed Glazed section values?
9. How should initialized modules advertise optional domain interfaces like `loupedeck.Bindable` without creating import cycles?
10. Should `moduleSections` default to true for all runtime-creating built-ins, or only when explicitly enabled?
11. Should `eval` be converted to a Glazed command in the first implementation slice or left as a raw Cobra exception?
12. How should `jsverbs` merge verb-declared sections with module-provided sections if slugs collide?

## Recommended first implementation slice

The smallest useful slice is:

1. Add module capability registry support first: `ConfigSectionCapability` and `RuntimeInitializerCapability`.
2. Add helper functions for runtime-profile section aggregation and runtime initialization.
3. Update built-in `run` with a fixture module section and `DecodeSectionInto` runtime initializer.
4. Update built-in `jsverbs` with the same fixture module section path.
5. Add command-set provider registry support and `commandProviders` buildspec list.
6. Add `Host.AttachCommandProviders` using `glazedcli.AddCommandsToRootCommand`.
7. Add a fixture command provider returning:
   - one `BareCommand`,
   - one `WriterCommand`,
   - one `GlazeCommand`.
8. Add a fixture module that exposes a Glazed config section and initializes from `DecodeSectionInto` for both built-ins and custom command providers.
9. Add generated examples and docs.

Do not start with Loupedeck or Discord. Use fixtures first: one built-in `run`/`jsverbs` fixture and one custom command-provider fixture. Then adapt one real package after the generated-binary mechanics are stable.

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
