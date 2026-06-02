---
Title: Env Prefix / App Name / Glazed Source Middleware Support for xgoja Generated Binaries
Ticket: XGOJA-017
Status: active
Topics:
    - xgoja
    - glazed
    - configuration
    - middleware
    - design
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../glazed/pkg/cli/cobra-parser.go
      Note: Glazed parser config and built-in middleware path
    - Path: ../../../../../../../pinocchio/pkg/cmds/cobra.go
      Note: Reference custom MiddlewaresFunc implementation
    - Path: cmd/xgoja/internal/buildspec/spec.go
      Note: Current YAML spec struct; needs new fields for appName
    - Path: cmd/xgoja/internal/generate/templates/main.go.tmpl
      Note: Generated main template; must emit middleware wiring conditionally
    - Path: pkg/xgoja/app/glazed.go
      Note: Critical chokepoint; buildGlazedCobraCommand hardcodes default middlewares
    - Path: pkg/xgoja/app/host.go
      Note: Host construction and AttachDefaultCommands; must propagate middleware factory
    - Path: pkg/xgoja/app/spec.go
      Note: Runtime JSON spec struct; needs new fields
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---








# Env Prefix / App Name / Glazed Source Middleware Support for xgoja Generated Binaries

## Executive Summary

Today, every binary produced by `xgoja build` parses command-line flags and arguments using Glazed's default middleware chain (`CobraCommandDefaultMiddlewares`). That chain is minimal: it reads Cobra flags, positional arguments, and field defaults. It does **not** read environment variables, config files, or profiles. This means a generated binary cannot be configured via `MYAPP_API_KEY=secret` or `/etc/myapp/config.yaml`. The only way to inject values is through CLI flags at invocation time.

This design document proposes extending `xgoja.yaml` so that a build spec can declare:

1. **`appName`** — a stable application identity used for env-prefix generation, config-file discovery, and logging labels.
2. **`envPrefix`** — an explicit environment-variable prefix (e.g. `MYAPP`). When set, every flag in every generated command becomes overridable via `MYAPP_<SECTION>_<FLAG>`.
3. **`config`** — a declarative config-plan that tells the generated binary where to look for layered config files (system, XDG, home, git-root, CWD, explicit).
4. **`profiles`** — optional profile settings that enable the generated binary to load named parameter sets from `profiles.yaml`.
5. **`middlewares`** — an optional explicit list of Glazed source middlewares that the generated binary should wire into every command's parser chain.

The goal is to make `xgoja` binaries behave like first-class Glazed applications (e.g. `pinocchio`, `glaze`) without forcing the user to write Go code. All wiring should be expressed in YAML and baked into the generated `main.go` at build time.

---

## Problem Statement and Scope

### What is broken / missing today

When you run `xgoja build -f xgoja.yaml`, the generator emits a `main.go` that creates commands through `app.NewRootCommand` or a target adapter. Each subcommand (`eval`, `run`, `repl`, `verbs`, provider commands) is converted to a Cobra command via `cli.BuildCobraCommand` with the following parser config:

```go
cli.WithParserConfig(cli.CobraParserConfig{
    ShortHelpSections: []string{schema.DefaultSlug},
    MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
})
```

Source: `pkg/xgoja/app/glazed.go:12-16`

`CobraCommandDefaultMiddlewares` is defined in `glazed/pkg/cli/cobra-parser.go:37-56` and returns only three middlewares:

1. `cmd_sources.FromCobra(cmd, ...)` — reads `--flag value` from Cobra's parsed flags.
2. `cmd_sources.FromArgs(args, ...)` — reads positional arguments.
3. `cmd_sources.FromDefaults(...)` — fills in field definition defaults.

There is **no** `FromEnv`, **no** `FromConfigPlanBuilder`, and **no** profile loading. If a provider module exposes a section `openai` with a field `api-key`, the user must pass `--openai-api-key` on every invocation. There is no mechanism for:

- `XGOJA_OPENAI_API_KEY=sk-... ./dist/myapp eval '...'`
- A project-local `config.yaml` that sets `openai.api-key`.
- A user-wide `~/.config/myapp/config.yaml` that sets defaults.
- A `profiles.yaml` with a `production` profile that overrides endpoint URLs.

### Scope of this ticket

This ticket is **design-only**; no code changes are being made yet. We will:

1. Map exactly how Glazed applications (especially `pinocchio`) achieve env/config/profile support.
2. Identify every file and data structure in xgoja that must change.
3. Propose a backward-compatible `xgoja.yaml` schema extension.
4. Specify how the Go template in `cmd/xgoja/internal/generate/templates/main.go.tmpl` must emit different code based on the new spec fields.
5. Specify how `pkg/xgoja/app` must accept runtime middleware configuration.
6. Provide a phased implementation plan, test strategy, and risk analysis.

Out of scope: Vault integration, dynamic middleware registration at runtime (plugins loading their own sources), and changing the xgoja CLI itself (the `xgoja build` command only needs to pass new fields through; it does not need env/config support).

---

## Current-State Architecture

To understand where the wires must go, we need to trace five subsystems: the build spec, the code generator, the generated binary's host layer, the command-wiring layer, and Glazed's middleware chain itself.

### 1. The Build Spec (`buildspec.Spec`)

File: `cmd/xgoja/internal/buildspec/spec.go`

The YAML spec is unmarshaled into:

```go
type Spec struct {
    Name             string                    `yaml:"name"`
    Go               GoSpec                    `yaml:"go"`
    Target           TargetSpec                `yaml:"target"`
    Packages         []PackageSpec             `yaml:"packages"`
    Runtimes         map[string]Runtime        `yaml:"runtimes"`
    Commands         CommandsSpec              `yaml:"commands"`
    CommandProviders []CommandProviderInstance `yaml:"commandProviders"`
    JSVerbs          []JSVerbSourceSpec        `yaml:"jsverbs"`
    Help             HelpSpec                  `yaml:"help"`
    Assets           []AssetSourceSpec         `yaml:"assets"`
    BaseDir          string                    `yaml:"-"`
}
```

There is **no** field for app identity, env prefix, config layers, profiles, or middleware overrides. The spec is purely about "what Go packages to import" and "what runtime modules to expose."

### 2. The Code Generator

File: `cmd/xgoja/internal/generate/generate.go` and `templates.go`

`WriteAll` produces a temporary workspace containing:

- `go.mod` — derived from `GoSpec` and provider imports.
- `main.go` — rendered from `templates/main.go.tmpl` via `mainTemplateDataFromSpec`.
- `xgoja.gen.json` — the spec serialized as JSON and embedded as a string constant.
- `xgoja_embed/...` — copied jsverb, help, and asset trees.

The template `main.go.tmpl` (file: `cmd/xgoja/internal/generate/templates/main.go.tmpl`) has two primary branches based on `TargetKind`:

- `xgoja` (default): calls `app.NewRootCommand(app.Options{...})`
- `cobra`: calls `target.NewRootCommand()` then `host.AttachDefaultCommands(root)`
- `adapter`: calls `target.Build(context.Background(), host)`

In all three cases, the generated binary receives its operational configuration through `embeddedSpecJSON`. It does not receive any parser configuration.

### 3. The Generated Host Layer

File: `pkg/xgoja/app/host.go` and `root.go`

`NewRootCommand` constructs a `Host` from the decoded `Spec` and attaches subcommands:

```go
func NewRootCommand(opts Options) (*cobra.Command, error) {
    // ... decode spec ...
    host := NewHostWithOptions(opts.Providers, spec, ...)
    root := &cobra.Command{Use: spec.Name, Short: "Generated xgoja binary"}
    host.AttachDefaultCommands(root)
    return root, nil
}
```

`AttachDefaultCommands` (file: `pkg/xgoja/app/host.go:35-57`) mounts `eval`, `run`, `repl`, `modules`, `verbs`, and command-provider commands. Every one of these commands is converted to Cobra via `buildGlazedCobraCommand` in `pkg/xgoja/app/glazed.go`.

### 4. The Command-Wiring Layer

File: `pkg/xgoja/app/glazed.go`

```go
func buildGlazedCobraCommand(command cmds.Command) (*cobra.Command, error) {
    return cli.BuildCobraCommand(command,
        cli.WithParserConfig(cli.CobraParserConfig{
            ShortHelpSections: []string{schema.DefaultSlug},
            MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
        }),
    )
}
```

This is the **single chokepoint** where env/config/profile support must be injected. Today it is a package-internal function with no parameters. To support configurable middlewares, it must become parameterized or the `Host` must carry a middleware factory that it passes down to every command builder.

### 5. Glazed's Middleware Chain

File: `glazed/pkg/cli/cobra-parser.go`

Glazed commands are parsed through a chain of `sources.Middleware` functions. The built-in chain (when `AppName` or `ConfigPlanBuilder` is set) looks like this:

```go
middlewares_ := []cmd_sources.Middleware{
    cmd_sources.FromCobra(cmd, fields.WithSource("cobra")),
    cmd_sources.FromArgs(args, fields.WithSource("arguments")),
}

if cfgCopy.AppName != "" {
    envPrefix := strings.ToUpper(cfgCopy.AppName)
    middlewares_ = append(middlewares_,
        cmd_sources.FromEnv(envPrefix, fields.WithSource("env")),
    )
}

if cfgCopy.ConfigPlanBuilder != nil {
    middlewares_ = append(middlewares_,
        cmd_sources.FromConfigPlanBuilder(
            func(_ context.Context, _ *values.Values) (*glazedconfig.Plan, error) {
                return cfgCopy.ConfigPlanBuilder(parsedCommandSections, cmd, args)
            },
            cmd_sources.WithParseOptions(fields.WithSource("config")),
        ),
    )
}

middlewares_ = append(middlewares_,
    cmd_sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
)
```

Source: `glazed/pkg/cli/cobra-parser.go:123-150`

The **precedence order** is critical. Because `ExecuteWithSchema` reverses the middleware list before wrapping, the **last** middleware appended has the **highest** precedence. The actual evaluation order at runtime is:

1. Defaults (lowest precedence)
2. Config files
3. Environment variables
4. CLI arguments / Cobra flags (highest precedence)

This means a flag `--openai-api-key` beats `MYAPP_OPENAI_API_KEY`, which beats a value in `config.yaml`, which beats the field's `Default` value.

### 6. How Pinocchio Does It (Reference Implementation)

File: `pinocchio/pkg/cmds/cobra.go`

Pinocchio does not use the built-in `CobraParserConfig` env/config path. Instead, it supplies a **custom `MiddlewaresFunc`**:

```go
func GetPinocchioCommandMiddlewares(parsedCommandSections *values.Values, cmd *cobra.Command, args []string) ([]sources.Middleware, error) {
    cfg := profilebootstrap.BootstrapConfig()
    return []sources.Middleware{
        sources.FromCobra(cmd, fields.WithSource("cobra")),
        sources.FromArgs(args, fields.WithSource("arguments")),
        sources.FromEnv(cfg.EnvPrefix, fields.WithSource("env")),
        sources.FromConfigPlanBuilder(
            func(_ context.Context, _ *values.Values) (*glazedconfig.Plan, error) {
                return cfg.ConfigPlanBuilder(parsedCommandSections)
            },
            sources.WithConfigFileMapper(cfg.ConfigFileMapper),
            sources.WithParseOptions(fields.WithSource("config")),
        ),
        sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
    }, nil
}
```

Pinocchio also wires profile loading as a **separate bootstrap layer** above the middleware chain (see `pinocchio/pkg/cmds/profilebootstrap/profile_selection.go`). Profile settings are parsed early, before command execution, and the effective profile influences which parameters are injected into the prompt engine. For xgoja, we can adopt a simpler model: profiles are just another source middleware that loads a `profiles.yaml` section map.

---

## Gap Analysis

| Gap | Impact | Files to Touch |
|---|---|---|
| `buildspec.Spec` has no `appName`, `envPrefix`, `config`, `profiles`, or `middlewares` fields | User cannot declare env/config behavior in YAML | `cmd/xgoja/internal/buildspec/spec.go` |
| `buildspec.Spec` JSON struct (`pkg/xgoja/app/spec.go`) mirrors the YAML struct but is also missing the new fields | Generated binary cannot decode the new fields from embedded JSON | `pkg/xgoja/app/spec.go` |
| `main.go.tmpl` hardcodes `app.NewRootCommand` with no middleware configuration | Generated binary has no hook to inject `FromEnv` / `FromConfigPlanBuilder` | `cmd/xgoja/internal/generate/templates/main.go.tmpl` |
| `mainTemplateDataFromSpec` does not populate middleware-related template variables | Template cannot conditionally emit middleware wiring | `cmd/xgoja/internal/generate/templates.go` |
| `buildGlazedCobraCommand` is a zero-parameter helper that always uses `CobraCommandDefaultMiddlewares` | Every command in the generated binary lacks env/config parsing | `pkg/xgoja/app/glazed.go` |
| `Host.AttachDefaultCommands` and `Host.AttachEval/AttachRun/AttachRepl` do not pass middleware configuration down to command builders | Even if `buildGlazedCobraCommand` gains a parameter, the Host must carry the factory | `pkg/xgoja/app/host.go` |
| `Host` struct has no field for middleware factory or parser config | No place to store runtime parser policy | `pkg/xgoja/app/host.go` |
| `NewRootCommand` and `NewHostWithOptions` do not accept middleware configuration | Entry points from generated `main.go` cannot inject policy | `pkg/xgoja/app/root.go`, `pkg/xgoja/app/host.go` |
| `pkg/xgoja/app/framework.go` sets `logging.AddLoggingSectionToRootCommand(root, appName)` but `appName` is derived only from `spec.Name` | No explicit app name override; no env prefix derivation | `pkg/xgoja/app/framework.go` |
| Validation (`doctor`) does not check new fields | Invalid configs could produce uncompilable or misbehaving binaries | `cmd/xgoja/internal/buildspec/validate.go` |

---

## Proposed Architecture and APIs

### Overview

We introduce a new top-level section in `xgoja.yaml` called `settings` (or we add fields directly to the spec root; see Decision Records). This section contains everything a generated binary needs to behave like a native Glazed application. The generator then:

1. Embeds the settings into `xgoja.gen.json`.
2. Emits a `main.go` that constructs a `Host` with a middleware factory derived from those settings.
3. Emits code so that every command built by the Host uses that middleware factory instead of the hardcoded default.

### New `xgoja.yaml` Schema (Proposed)

```yaml
name: my-app

# NEW: Explicit application identity
description: "My custom goja runtime"
appName: my-app          # default: same as `name`
envPrefix: MY_APP        # default: UPPER(appName)

# NEW: Declarative config layers
config:
  enabled: true
  layers:
    - system
    - xdg
    - home
    - git-root
    - cwd
  fileName: config.yaml   # discovered file name for system/xdg/home/git-root/cwd
  explicitFlag: config    # if non-empty, adds --config flag to root command

# NEW: Profile loading
profiles:
  enabled: true
  file: profiles.yaml     # default; resolves via XDG if relative
  appName: my-app         # default: same as top-level appName
  defaultProfile: default

# NEW: Explicit middleware list (advanced)
middlewares:
  - source: env
    prefix: MY_APP        # optional override
  - source: config
    # uses the config plan defined above
  - source: profiles
    # uses the profiles defined above
```

**Backward compatibility:** If `config.enabled` is false or absent, and `middlewares` is absent, the generated binary behaves exactly as it does today. No breaking changes.

### New `buildspec` Go Types

File: `cmd/xgoja/internal/buildspec/spec.go`

```go
type Spec struct {
    Name             string                    `yaml:"name"`
    Description      string                    `yaml:"description,omitempty"`
    AppName          string                    `yaml:"appName,omitempty"`
    EnvPrefix        string                    `yaml:"envPrefix,omitempty"`
    Config           *ConfigSpec               `yaml:"config,omitempty"`
    Profiles         *ProfileSpec              `yaml:"profiles,omitempty"`
    Middlewares      []MiddlewareSpec          `yaml:"middlewares,omitempty"`
    Go               GoSpec                    `yaml:"go"`
    Target           TargetSpec                `yaml:"target"`
    Packages         []PackageSpec             `yaml:"packages"`
    Runtimes         map[string]Runtime        `yaml:"runtimes"`
    Commands         CommandsSpec              `yaml:"commands"`
    CommandProviders []CommandProviderInstance `yaml:"commandProviders"`
    JSVerbs          []JSVerbSourceSpec        `yaml:"jsverbs"`
    Help             HelpSpec                  `yaml:"help"`
    Assets           []AssetSourceSpec         `yaml:"assets"`
    BaseDir          string                    `yaml:"-"`
}

type ConfigSpec struct {
    Enabled      bool     `yaml:"enabled"`
    Layers       []string `yaml:"layers,omitempty"`
    FileName     string   `yaml:"fileName,omitempty"`
    ExplicitFlag string   `yaml:"explicitFlag,omitempty"`
}

type ProfileSpec struct {
    Enabled        bool   `yaml:"enabled"`
    File           string `yaml:"file,omitempty"`
    AppName        string `yaml:"appName,omitempty"`
    DefaultProfile string `yaml:"defaultProfile,omitempty"`
}

type MiddlewareSpec struct {
    Source string            `yaml:"source"`
    Prefix string            `yaml:"prefix,omitempty"`
    Config map[string]interface{} `yaml:"config,omitempty"`
}
```

The JSON equivalent in `pkg/xgoja/app/spec.go` mirrors these fields so the embedded spec can be decoded at runtime.

### Runtime Types in `pkg/xgoja/app`

File: `pkg/xgoja/app/host.go`

```go
type Host struct {
    Providers       *providerapi.Registry
    Spec            *Spec
    Factory         *RuntimeFactory
    EmbeddedJSVerbs fs.FS
    EmbeddedHelp    fs.FS
    EmbeddedAssets  fs.FS
    Services        HostServices
    Out             io.Writer

    // NEW: parser configuration propagated to every command
    MiddlewaresFunc cli.CobraMiddlewaresFunc
}

type HostOptions struct {
    EmbeddedJSVerbs fs.FS
    EmbeddedHelp    fs.FS
    EmbeddedAssets  fs.FS
    Out             io.Writer

    // NEW
    MiddlewaresFunc cli.CobraMiddlewaresFunc
}
```

If `MiddlewaresFunc` is nil, the Host falls back to `cli.CobraCommandDefaultMiddlewares` for backward compatibility.

### Changes to `buildGlazedCobraCommand`

File: `pkg/xgoja/app/glazed.go`

```go
func buildGlazedCobraCommandWithMiddlewares(command cmds.Command, middlewaresFunc cli.CobraMiddlewaresFunc) (*cobra.Command, error) {
    if middlewaresFunc == nil {
        middlewaresFunc = cli.CobraCommandDefaultMiddlewares
    }
    return cli.BuildCobraCommand(command,
        cli.WithParserConfig(cli.CobraParserConfig{
            ShortHelpSections: []string{schema.DefaultSlug},
            MiddlewaresFunc:   middlewaresFunc,
        }),
    )
}
```

All call sites inside `host.go` (`AttachEval`, `AttachRun`, `AttachRepl`, `AttachModules`, `AttachCommandProviders`) switch from `buildGlazedCobraCommand` to `buildGlazedCobraCommandWithMiddlewares(..., h.MiddlewaresFunc)`.

### Generated `main.go` Changes

The template must emit different construction code based on whether middlewares are configured. For the common case (`config.enabled: true`), the generator should produce a `ConfigPlanBuilder` closure and pass it to `app.NewRootCommand` inside a custom middleware function.

**Pseudocode for the generator's template data additions:**

```go
type mainTemplateData struct {
    // ... existing fields ...

    // NEW
    HasMiddlewares    bool
    AppName           string
    EnvPrefix         string
    ConfigPlanBuilder string // Go source snippet for the closure
}
```

**Pseudocode for template branch in `main.go.tmpl`:**

```gotemplate
{{- if .HasMiddlewares }}
    middlewaresFunc := func(parsedCommandSections *values.Values, cmd *cobra.Command, args []string) ([]cmd_sources.Middleware, error) {
        middlewares_ := []cmd_sources.Middleware{
            cmd_sources.FromCobra(cmd, fields.WithSource("cobra")),
            cmd_sources.FromArgs(args, fields.WithSource("arguments")),
        }
        {{- if .EnvPrefix }}
        middlewares_ = append(middlewares_, cmd_sources.FromEnv("{{ .EnvPrefix }}", fields.WithSource("env")))
        {{- end }}
        {{- if .ConfigPlanBuilder }}
        middlewares_ = append(middlewares_, cmd_sources.FromConfigPlanBuilder(
            func(_ context.Context, _ *values.Values) (*glazedconfig.Plan, error) {
                return {{ .ConfigPlanBuilder }}
            },
            cmd_sources.WithParseOptions(fields.WithSource("config")),
        ))
        {{- end }}
        middlewares_ = append(middlewares_, cmd_sources.FromDefaults(fields.WithSource(fields.SourceDefaults)))
        return middlewares_, nil
    }
{{- end }}
```

For the `xgoja` target kind, the generated `main` then becomes:

```go
root, err := app.NewRootCommand(app.Options{
    Providers:       registry,
    SpecJSON:        embeddedSpecJSON,
    EmbeddedJSVerbs: embeddedJSVerbs,
    EmbeddedHelp:    embeddedHelp,
    EmbeddedAssets:  embeddedAssets,
    MiddlewaresFunc: middlewaresFunc, // NEW
})
```

For the `cobra` target kind, the `Host` construction gains the same field:

```go
host := app.NewHostWithOptions(registry, spec, app.HostOptions{
    EmbeddedJSVerbs: embeddedJSVerbs,
    EmbeddedHelp:    embeddedHelp,
    EmbeddedAssets:  embeddedAssets,
    MiddlewaresFunc: middlewaresFunc, // NEW
})
```

### Config Plan Builder Generation

The generator must turn the declarative `config` block into a Go closure that returns a `glazedconfig.Plan`. This is the most complex part of the template because it involves translating layer names into `glazedconfig.SourceSpec` function calls.

**Layer mapping:**

| Layer name | Generated call |
|---|---|
| `system` | `glazedconfig.SystemAppConfig(appName)` |
| `xdg` | `glazedconfig.XDGAppConfig(appName)` |
| `home` | `glazedconfig.HomeAppConfig(appName)` |
| `git-root` | `glazedconfig.GitRootFile(fileName)` |
| `cwd` | `glazedconfig.WorkingDirFile(fileName)` |
| `explicit` | `glazedconfig.ExplicitFile(explicitPath)` (resolved from `--config` flag) |

**Precedence order:** The generated plan must add layers in the order listed above (system lowest, explicit highest). The `glazedconfig.NewPlan` constructor should use `WithLayerOrder(...)` to ensure correct deduplication behavior.

**Pseudocode for the generated closure:**

```go
func buildConfigPlan(parsed *values.Values, cmd *cobra.Command, args []string) (*glazedconfig.Plan, error) {
    appName := "{{ .AppName }}"
    fileName := "{{ .ConfigFileName }}"
    plan := glazedconfig.NewPlan(
        glazedconfig.WithLayerOrder(
            glazedconfig.LayerSystem,
            glazedconfig.LayerUser,
            glazedconfig.LayerRepo,
            glazedconfig.LayerCWD,
            glazedconfig.LayerExplicit,
        ),
        glazedconfig.WithDedupePaths(),
    )
    {{- range .ConfigLayers }}
    {{- if eq . "system" }}
    plan.Add(glazedconfig.SystemAppConfig(appName).Named("system-app-config").Kind("app-config"))
    {{- else if eq . "xdg" }}
    plan.Add(glazedconfig.XDGAppConfig(appName).Named("xdg-app-config").Kind("app-config"))
    {{- else if eq . "home" }}
    plan.Add(glazedconfig.HomeAppConfig(appName).Named("home-app-config").Kind("app-config"))
    {{- else if eq . "git-root" }}
    plan.Add(glazedconfig.GitRootFile(fileName).Named("git-root-config").Kind("local-file"))
    {{- else if eq . "cwd" }}
    plan.Add(glazedconfig.WorkingDirFile(fileName).Named("cwd-config").Kind("local-file"))
    {{- end }}
    {{- end }}
    // explicit layer is added only if the user passed --config
    return plan, nil
}
```

Note: The `explicit` layer requires reading a `--config` flag value. In the built-in Glazed parser path, `ConfigPlanBuilder` receives `parsedCommandSections`, `cmd`, and `args`. The generated closure can look up the flag from `cmd.Flags().Lookup("config")` or from the parsed command settings section. We will need to decide whether xgoja generates a `--config` flag on the root command or expects the user to use Glazed's built-in `--config` command setting. See Decision Records.

### Profile Middleware Generation

If `profiles.enabled: true`, the generator emits an additional middleware in the chain:

```go
middlewares_ = append(middlewares_,
    sources.GatherFlagsFromCustomProfiles(
        "{{ .DefaultProfile }}",
        sources.WithProfileAppName("{{ .ProfileAppName }}"),
        sources.WithProfileParseOptions(fields.WithSource("profiles")),
    ),
)
```

This loads `~/.config/<appName>/profiles.yaml` (or the explicit file if specified) and injects the selected profile's section maps into the parsed values. The profile name itself can come from a `--profile` flag or an env variable.

---

## Decision Records

### Decision: Where to place the new fields in `xgoja.yaml`

- **Context:** We could add `appName`, `envPrefix`, `config`, and `profiles` as top-level fields, or nest them under a `settings:` or `glazed:` block.
- **Options considered:**
  1. **Top-level** — Flat, easy to read, consistent with `name`, `target`, `packages`.
  2. **`settings:` block** — Groups all runtime behavioral settings in one place.
  3. **`glazed:` block** — Makes it obvious these are Glazed-specific features.
- **Decision:** Use **top-level fields** for `appName`, `envPrefix`, `config`, `profiles`, and a separate `middlewares:` list for advanced overrides. This keeps the file readable and matches the flat style of existing fields (`help:`, `assets:`).
- **Rationale:** xgoja is a Glazed-native tool; prefixing everything with `glazed:` is redundant. A `settings:` block adds nesting without adding clarity. Top-level fields are discoverable.
- **Consequences:** The spec struct gains several new fields, but they are all optional and self-documenting.
- **Status:** proposed

### Decision: Should the generator emit a custom `MiddlewaresFunc` or use Glazed's built-in parser path?

- **Context:** Glazed's `CobraParserConfig` has two ways to get env/config support: (a) set `AppName` and `ConfigPlanBuilder` and let the built-in parser path construct the middlewares, or (b) supply a custom `MiddlewaresFunc` that builds the chain manually.
- **Options considered:**
  1. **Built-in parser path** — Set `AppName` and `ConfigPlanBuilder` on `CobraParserConfig`. Less generated code; relies on Glazed's internal wiring.
  2. **Custom `MiddlewaresFunc`** — The generator emits a Go function literal in `main.go` that returns the exact middleware slice. More explicit; easier to debug; allows profile middleware insertion.
- **Decision:** Use **custom `MiddlewaresFunc`** for advanced configurations (profiles, explicit middleware list), and fall back to the **built-in path** for simple `appName` + `envPrefix` only.
- **Rationale:** The built-in path does not support profile middlewares (`GatherFlagsFromCustomProfiles`) or arbitrary middleware ordering. A generated function literal is only ~20 lines of Go and is fully inspectable by the user when they run `--keep-work`. Explicit is better than implicit for generated code.
- **Consequences:** The template becomes slightly more complex, but the generated binary's behavior is fully visible in `main.go`. We must ensure the generated function compiles against the exact Glazed version in `go.mod`.
- **Status:** proposed

### Decision: How should `--config` explicit file override work?

- **Context:** Glazed commands already have a `--config` flag in the `CommandSettings` section (see `cli.CommandSettings`). However, `CobraParserConfig.ConfigPlanBuilder` receives `cmd` and `args`, so it can read that flag.
- **Options considered:**
  1. **Reuse Glazed's `--config`** — The `CommandSettings` section is parsed early; the `ConfigPlanBuilder` can extract `commandSettings.ConfigFile` and feed it into `glazedconfig.ExplicitFile(...)`.
  2. **Generate a root-level `--config` flag** — Add a persistent flag directly to the generated root Cobra command. Simpler but duplicates Glazed's existing mechanism.
- **Decision:** **Reuse Glazed's `--config`** through `CommandSettings.ConfigFile`. The generated `ConfigPlanBuilder` closure extracts the explicit config path from `parsedCommandSections` (which already contains `CommandSettings` after early parsing).
- **Rationale:** No duplication, no flag collision, consistent with `pinocchio` and other Glazed apps.
- **Consequences:** The generated closure needs to know how to decode `CommandSettings`. We can import `github.com/go-go-golems/glazed/pkg/cli` in generated code and call `parsed.DecodeSectionInto(cli.CommandSettingsSlug, &settings)`.
- **Status:** proposed

### Decision: Should `envPrefix` default to `UPPER(appName)` or remain empty?

- **Context:** Glazed's built-in parser path does `envPrefix := strings.ToUpper(cfgCopy.AppName)`. This means setting `AppName: "my-app"` automatically enables `MY_APP_*` env vars. Some users may want env support disabled.
- **Options considered:**
  1. **Auto-enable from `appName`** — If `appName` is set, env is always on. Simple but removes control.
  2. **Require explicit `envPrefix`** — Env support is opt-in. Safer but more verbose.
  3. **Default from `appName`, allow empty override** — `envPrefix` defaults to `UPPER(appName)`; set to `""` to disable.
- **Decision:** **Option 3** — `envPrefix` defaults to `UPPER(appName)` when `appName` is set; explicit `""` disables it.
- **Rationale:** This matches Glazed's convention and user expectations. A user who bothers to set `appName` usually wants the full Glazed application behavior.
- **Consequences:** We must document that `envPrefix: ""` is the escape hatch.
- **Status:** proposed

---

## Pseudocode and Key Flows

### Flow 1: Build-time path (xgoja build)

```
User writes xgoja.yaml with config.enabled: true
       |
       v
xgoja build reads buildspec.Spec (new fields decoded)
       |
       v
generate.WriteAll renders main.go.tmpl
       |
       +---> If config.enabled:
       |         template emits ConfigPlanBuilder closure
       |         template emits custom MiddlewaresFunc
       |
       +---> If profiles.enabled:
       |         template emits GatherFlagsFromCustomProfiles middleware
       |
       v
generated main.go contains full middleware wiring
       |
       v
go build produces binary
```

### Flow 2: Runtime path (generated binary execution)

```
./dist/myapp eval 'require("hello").greet("world")' --openai-api-key sk-...
       |
       v
Cobra parses --openai-api-key into its flag set
       |
       v
cli.ParseCommandSettingsSection extracts --config, --profile, etc.
       |
       v
Custom MiddlewaresFunc builds the chain:
   1. FromDefaults          -> openai.api-key = "" (field default)
   2. FromConfigPlanBuilder -> openai.api-key = "from config.yaml"
   3. FromEnv(MY_APP_)      -> openai.api-key = "from env"
   4. FromCobra             -> openai.api-key = "sk-..." (CLI wins)
       |
       v
Parsed values decoded into command settings struct
       |
       v
Command runs with effective configuration
```

### Flow 3: Config file discovery at runtime

```
ConfigPlanBuilder closure executes
       |
       v
plan.Resolve(ctx) walks each layer:
   LayerSystem:  /etc/my-app/config.yaml        (optional)
   LayerUser:    ~/.config/my-app/config.yaml   (optional)
   LayerUser:    ~/.my-app/config.yaml          (optional)
   LayerRepo:    <git-root>/config.yaml         (optional)
   LayerCWD:     ./config.yaml                  (optional)
   LayerExplicit: --config path                 (if provided)
       |
       v
Files returned in precedence order (low -> high)
       |
       v
FromResolvedFiles middleware loads each file,
merging section maps into parsed values
```

---

## Phased Implementation Plan

### Phase 1: Schema and validation (no generated code changes)

**Goal:** Teach xgoja to read and validate the new YAML fields without changing the generator.

1. Add new types to `cmd/xgoja/internal/buildspec/spec.go`.
2. Add JSON mirror types to `pkg/xgoja/app/spec.go`.
3. Add validation rules to `cmd/xgoja/internal/buildspec/validate.go`:
   - `envPrefix` must be uppercase alphanumeric + underscore if provided.
   - `config.layers` entries must be from the allowed set.
   - `middlewares[].source` must be one of `env`, `config`, `profiles`, `vault` (future), `defaults`.
4. Add test cases in `validate_test.go` and `load_test.go`.
5. Update `cmd/xgoja/doc/06-buildspec-reference.md` with new fields.

**Files:**
- `cmd/xgoja/internal/buildspec/spec.go`
- `pkg/xgoja/app/spec.go`
- `cmd/xgoja/internal/buildspec/validate.go`
- `cmd/xgoja/internal/buildspec/validate_test.go`
- `cmd/xgoja/internal/buildspec/load_test.go`
- `cmd/xgoja/doc/06-buildspec-reference.md`

### Phase 2: Host and command wiring (no template changes)

**Goal:** Make `pkg/xgoja/app` capable of accepting and using a custom middleware factory.

1. Add `MiddlewaresFunc` to `Host` and `HostOptions`.
2. Add `MiddlewaresFunc` to `app.Options`.
3. Change `buildGlazedCobraCommand` to `buildGlazedCobraCommandWithMiddlewares`.
4. Update every `Attach*` method in `host.go` to pass `h.MiddlewaresFunc`.
5. Update `framework.go` to use `spec.AppName` if set, else `spec.Name`.
6. Add unit tests in `root_test.go` and `host_test.go` (new file) that verify a custom middleware is invoked.

**Files:**
- `pkg/xgoja/app/host.go`
- `pkg/xgoja/app/root.go`
- `pkg/xgoja/app/glazed.go`
- `pkg/xgoja/app/framework.go`
- `pkg/xgoja/app/root_test.go`

### Phase 3: Template and generator wiring

**Goal:** Teach the generator to emit middleware-aware `main.go`.

1. Extend `mainTemplateData` in `templates.go` with new boolean and string fields.
2. Implement template helper functions (or pre-computed strings) for the `ConfigPlanBuilder` Go source snippet.
3. Modify `main.go.tmpl` to emit the `middlewaresFunc` closure conditionally.
4. Modify `main.go.tmpl` to pass `middlewaresFunc` into `app.NewRootCommand` or `app.NewHostWithOptions`.
5. Add generator tests in `generate_test.go` that assert on rendered source code for various spec configurations.
6. Add an end-to-end example under `examples/xgoja/` that demonstrates env + config + profiles.

**Files:**
- `cmd/xgoja/internal/generate/templates.go`
- `cmd/xgoja/internal/generate/templates/main.go.tmpl`
- `cmd/xgoja/internal/generate/generate_test.go`
- `examples/xgoja/11-config-env-profiles/xgoja.yaml` (new)

### Phase 4: Documentation and integration tests

**Goal:** Ensure the feature is discoverable and works end-to-end.

1. Write tutorial doc: `cmd/xgoja/doc/10-tutorial-config-env-profiles.md`.
2. Add integration test script in `cmd/xgoja/scripts/` that:
   - Builds an example binary with config enabled.
   - Creates a `config.yaml` and verifies the binary reads it.
   - Exports an env var and verifies it overrides the config file.
   - Passes a CLI flag and verifies it overrides the env var.
3. Update `README.md` with a "Configuration" section.

**Files:**
- `cmd/xgoja/doc/10-tutorial-config-env-profiles.md`
- `cmd/xgoja/scripts/verify-config-env.sh`
- `cmd/xgoja/README.md`

---

## Testing and Validation Strategy

### Unit tests

| Component | What to test |
|---|---|
| `buildspec` | Decode all new YAML shapes; reject invalid layers, bad prefixes, unknown middleware sources. |
| `validate.go` | Ensure `doctor` catches missing `appName` when `config.enabled` is true. |
| `pkg/xgoja/app` | Mock `CobraMiddlewaresFunc` and assert it is called during `AttachEval`. |
| `generate` | Render template with new fields; verify emitted Go compiles (use `format.Source` as proxy). |

### Integration tests

1. **Env override:** Build binary with `envPrefix: TESTAPP`. Run `TESTAPP_EVAL_RUNTIME=foo ./dist/app eval '1'`. Assert the runtime profile is `foo`.
2. **Config file override:** Create `config.yaml` with `eval: {runtime: bar}`. Run `./dist/app eval '1'`. Assert runtime is `bar`.
3. **Precedence:** Set config to `bar`, env to `baz`, and CLI flag `--runtime qux`. Assert `qux` wins.
4. **Profile loading:** Create `profiles.yaml` with `prod: {eval: {runtime: prod}}`. Run `./dist/app --profile prod eval '1'`. Assert runtime is `prod`.

### Regression tests

- Build every existing example in `examples/xgoja/` and confirm output is byte-identical to before (or at least functionally identical). The new fields are all optional; no existing spec should change behavior.

---

## Risks, Alternatives, and Open Questions

### Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Template-generated Go code does not compile against future Glazed versions | Medium | High | Pin Glazed version in generated `go.mod`; add compile-time test in `generate_test.go` |
| `ConfigPlanBuilder` closure in template becomes unwieldy | Medium | Medium | Extract a helper package `pkg/xgoja/appconfig` that provides `NewPlanFromSpec(*app.ConfigSpec) *glazedconfig.Plan`; template just calls that |
| Profile middleware ordering conflicts with config middleware | Low | Medium | Document ordering; allow explicit `middlewares:` list for power users |
| Existing `spec.Name` used as logging label changes if we introduce `appName` | Low | Low | Default `appName` to `spec.Name`; behavior is identical unless user overrides |

### Alternatives

1. **Runtime middleware registration (no generator changes):** Instead of generating middleware wiring, teach `pkg/xgoja/app` to build a middleware factory from the embedded JSON spec at runtime. This keeps the template small but requires more runtime code and makes the binary's behavior less visible.
2. **Provider-supplied middlewares:** Allow each provider package to export a middleware factory function, similar to how they export `Register(registry)`. This is more flexible but overkill for the common case of "I just want env vars and a config file."

### Open Questions

1. Should `config.fileName` default to `config.yaml` or to `<appName>.yaml`?
2. Should we support `config.includeLayers` / `config.excludeLayers` instead of an explicit `layers` list?
3. How should the generated binary handle missing config files? Glazed's plan marks them `Optional: true`, so they are silently skipped. Is that acceptable, or should we add a `--strict-config` flag?
4. Should `profiles` be a Glazed section (like `profile-settings` in Geppetto) or a source middleware? Geppetto uses both: a section for CLI flags (`--profile`, `--profile-registry`) and a middleware for loading the file. Do we need the section too?

---

## References

### Key Files (xgoja)

- `cmd/xgoja/internal/buildspec/spec.go` — YAML spec struct.
- `cmd/xgoja/internal/buildspec/validate.go` — Validation logic.
- `cmd/xgoja/internal/generate/templates.go` — Template data builder.
- `cmd/xgoja/internal/generate/templates/main.go.tmpl` — Generated main template.
- `pkg/xgoja/app/spec.go` — Runtime JSON spec struct.
- `pkg/xgoja/app/host.go` — Host construction and command attachment.
- `pkg/xgoja/app/root.go` — `NewRootCommand` and command definitions.
- `pkg/xgoja/app/glazed.go` — `buildGlazedCobraCommand` chokepoint.
- `pkg/xgoja/app/framework.go` — Root framework installation (logging, help).

### Key Files (glazed)

- `glazed/pkg/cli/cobra-parser.go` — `CobraParserConfig`, `CobraCommandDefaultMiddlewares`, built-in parser path.
- `glazed/pkg/cmds/sources/middlewares.go` — `Middleware`, `Chain`, `ExecuteWithSchema`.
- `glazed/pkg/cmds/sources/load-fields-from-config.go` — `FromConfigPlanBuilder`, `FromResolvedFiles`.
- `glazed/pkg/cmds/sources/update.go` — `FromEnv`, `updateFromEnv`.
- `glazed/pkg/cmds/sources/profiles.go` — `GatherFlagsFromCustomProfiles`, `WithProfileAppName`.
- `glazed/pkg/config/plan_sources.go` — `SystemAppConfig`, `XDGAppConfig`, `HomeAppConfig`, `GitRootFile`, `WorkingDirFile`.

### Key Files (pinocchio — reference implementation)

- `pinocchio/pkg/cmds/cobra.go` — Custom `MiddlewaresFunc` wiring.
- `pinocchio/pkg/cmds/profilebootstrap/profile_selection.go` — `AppBootstrapConfig`, `ConfigPlanBuilder`, profile registry resolution.

### Documentation

- `cmd/xgoja/doc/03-tutorial-using-xgoja-yaml.md` — Existing tutorial.
- `cmd/xgoja/doc/06-buildspec-reference.md` — Existing buildspec reference.
- Glazed example: `glazed/cmd/examples/middlewares-config-env/main.go` — Minimal demo of `AppName` + `ConfigPlanBuilder`.
