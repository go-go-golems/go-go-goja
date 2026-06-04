---
Title: xgoja Codegen and Script Execution Runthrough
Ticket: GOJA-053
Status: active
Topics:
    - xgoja
    - code-generation
    - lifecycle
    - provider
    - glazed
    - architecture
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/xgoja/cmd_build.go
      Note: |-
        xgoja build command entrypoint and build workspace orchestration.
        xgoja build command entrypoint
    - Path: go-go-goja/cmd/xgoja/internal/buildspec/load.go
      Note: |-
        YAML load, defaults, and validation entrypoint for xgoja.yaml.
        xgoja.yaml load/default/validate flow
    - Path: go-go-goja/cmd/xgoja/internal/buildspec/spec.go
      Note: Build-time xgoja.yaml schema, including runtimes/modules/config.
    - Path: go-go-goja/cmd/xgoja/internal/generate/generate.go
      Note: Generated workspace file emission and embedded asset copying.
    - Path: go-go-goja/cmd/xgoja/internal/generate/main.go
      Note: Embedded runtime spec rendering.
    - Path: go-go-goja/cmd/xgoja/internal/generate/templates.go
      Note: Generated main.go template data and provider import assembly.
    - Path: go-go-goja/cmd/xgoja/internal/generate/templates/main.go.tmpl
      Note: |-
        Generated target program main function template.
        Generated target main function template
    - Path: go-go-goja/engine/factory.go
      Note: |-
        Engine runtime construction, RuntimeModuleContext creation, require registry, runtimebridge.Store.
        Engine runtime construction and RuntimeModuleContext creation
    - Path: go-go-goja/engine/runtime.go
      Note: Runtime lifecycle, closers, runtime context, and shutdown behavior.
    - Path: go-go-goja/engine/runtime_modules.go
      Note: RuntimeModuleSpec and RuntimeModuleContext definitions.
    - Path: go-go-goja/pkg/runtimebridge/runtimebridge.go
      Note: |-
        RuntimeServices, VM service lookup, lifetime context, and current owner-call context.
        RuntimeServices and VM-based runtime bridge
    - Path: go-go-goja/pkg/runtimeowner/runner.go
      Note: Owner-thread Call/Post scheduling and runtimebridge call-context propagation.
    - Path: go-go-goja/pkg/xgoja/app/factory.go
      Note: |-
        xgoja RuntimeFactory and provider module registration adapter.
        xgoja RuntimeFactory and provider module adapter
    - Path: go-go-goja/pkg/xgoja/app/host.go
      Note: Host object that wires spec, registry, runtime factory, services, assets, and commands.
    - Path: go-go-goja/pkg/xgoja/app/middlewares.go
      Note: Generated command parsing sources and precedence for flags/env/config/defaults.
    - Path: go-go-goja/pkg/xgoja/app/module_sections.go
      Note: Selected module descriptors, public command sections, and post-runtime initializers.
    - Path: go-go-goja/pkg/xgoja/app/root.go
      Note: Generated program root command, eval command, jsverbs runtime flow.
    - Path: go-go-goja/pkg/xgoja/app/run.go
      Note: |-
        Generated program run command and script execution flow.
        Generated run command script execution path
    - Path: go-go-goja/pkg/xgoja/providerapi/capabilities.go
      Note: Provider capabilities for public sections and runtime initializers.
    - Path: go-go-goja/pkg/xgoja/providerapi/module.go
      Note: providerapi.Module, ModuleFactory, and ModuleContext.
    - Path: go-go-goja/pkg/xgoja/providerapi/registry.go
      Note: Provider package registry, modules, capabilities, command sets, and help/verb sources.
ExternalSources: []
Summary: A full intern-friendly walkthrough of xgoja build/codegen and generated-program script execution, clarifying factories, registries, contexts, bridges, module specs, and provider APIs.
LastUpdated: 2026-06-04T00:00:00Z
WhatFor: Use to understand how xgoja.yaml becomes a generated binary and how that binary creates a runtime, registers provider modules, parses flags, initializes runtime features, and executes JavaScript.
WhenToUse: Before modifying ModuleContext, RuntimeModuleContext, runtimebridge, provider capabilities, generated command flags, or GOJA-053 pre-runtime module config flow.
---


# xgoja Codegen and Script Execution Runthrough

## Executive summary

xgoja has two distinct phases that are easy to blur together:

1. **Build/codegen time**: `xgoja build -f xgoja.yaml` reads a YAML build spec, validates it, writes a temporary Go module (`go.mod`, `main.go`, `xgoja.gen.json`, embedded asset trees), then runs `go mod tidy` and `go build`.
2. **Generated-program runtime**: the compiled target binary starts, registers provider packages into a provider registry, decodes the embedded runtime spec, builds Cobra/Glazed commands, parses user flags/config/env, creates an engine runtime, registers provider modules into the Goja `require` registry, runs post-runtime provider initializers, then evaluates or requires JavaScript.

The confusing names mostly come from different layers owning different abstractions:

- `providerapi.Registry`: xgoja-level catalog of provider packages, modules, capabilities, command providers, help sources, and jsverb sources.
- `app.RuntimeFactory`: xgoja adapter that turns a named runtime profile from the embedded spec into an `engine.Runtime`.
- `engine.FactoryBuilder` / `engine.Factory`: low-level engine composition plan for one kind of runtime.
- `engine.RuntimeModuleSpec`: low-level instruction: “register this native module into this concrete runtime’s `require.Registry`.”
- `providerapi.Module`: provider-facing module description; its `New ModuleFactory` returns a `require.ModuleLoader`.
- `providerapi.ModuleContext`: provider-facing setup/config context passed to `providerapi.Module.New`.
- `engine.RuntimeModuleContext`: engine-facing runtime setup context passed to each `engine.RuntimeModuleSpec` during runtime construction.
- `runtimebridge.RuntimeServices`: VM-indexed services stored after a Goja VM exists, so native module code can later find the runtime lifetime context and owner-thread scheduler.
- `runtimeowner.RuntimeOwner`: serialized owner-thread gateway for safely executing code on the Goja/event-loop thread.

The most important GOJA-053 insight is timing: **public command sections are parsed before `runCommand.Run`, but today xgoja only runs `RuntimeInitializerCapability` after `Module.New` has already happened**. That is why CLI/config/env values can configure runtime initializers today, but cannot yet patch `ModuleContext.Config` before `providerapi.Module.New`. The proposed GOJA-053 change inserts parsed Glazed `SectionValues` into the `app.RuntimeFactory.NewRuntime(...)` path before provider module specs call `module.New(...)`.

---

## 1. Glossary: the objects that sound similar

### `buildspec.Spec`: build-time YAML model

Defined in `go-go-goja/cmd/xgoja/internal/buildspec/spec.go`.

This is the direct Go model for `xgoja.yaml` at build time. It includes:

- `Packages []PackageSpec`: provider Go packages to import and register.
- `Runtimes map[string]Runtime`: named runtime profiles.
- `Runtime.Modules []ModuleInstance`: selected provider modules per profile.
- `ModuleInstance.Config map[string]any`: static module config from `xgoja.yaml`.
- `Commands`: which generated commands to attach.
- `Config`: generated command config-file support.
- `EnvPrefix`: generated command env-var support.
- embedded `JSVerbs`, `Help`, and `Assets` sources.

### `app.Spec`: generated-program runtime spec

Defined in `go-go-goja/pkg/xgoja/app/spec.go`.

This is the runtime-side shape embedded into `xgoja.gen.json` and decoded by the generated binary. It intentionally omits build-only fields such as Go module version details and local replacement paths. Runtime code only needs the selected packages, runtime profiles, commands, config/env behavior, embedded sources, help, and assets.

### `providerapi.Registry`: provider catalog

Defined in `go-go-goja/pkg/xgoja/providerapi/registry.go`.

The generated binary creates one provider registry at startup:

```go
registry := providerapi.NewRegistry()
must(provider1.Register(registry))
must(provider2.Register(registry))
```

The registry stores provider packages and entries:

- `providerapi.Module` values, resolved by `ResolveModule(packageID, moduleName)`.
- package capabilities, resolved by `ResolvePackageCapabilities(packageID)`.
- command set providers, help sources, and jsverb sources.

This is not the Goja `require` registry. It is xgoja’s provider metadata registry.

### `require.Registry`: Goja module registry

Created in `go-go-goja/engine/factory.go` with:

```go
reg := require.NewRegistry(f.settings.requireOptions...)
```

This registry is per Goja runtime instance. `RuntimeModuleSpec.RegisterRuntimeModule(...)` registers `require.ModuleLoader` functions into it. After all native modules are registered, `reg.Enable(vm)` installs `require(...)` into the VM.

### `providerapi.Module`: provider-facing module definition

Defined in `go-go-goja/pkg/xgoja/providerapi/module.go`:

```go
type Module struct {
    Name         string
    DefaultAs    string
    Description  string
    ConfigSchema json.RawMessage
    New          ModuleFactory
}

type ModuleFactory func(ModuleContext) (require.ModuleLoader, error)
```

`Module.New` is a field, not a method. Providers populate it during registration. For example, Geppetto defines `New: func(ctx providerapi.ModuleContext) ...` in `geppetto/pkg/js/modules/geppetto/provider/provider.go`.

### `providerapi.ModuleContext`: provider-facing module setup context

Defined in `go-go-goja/pkg/xgoja/providerapi/module.go`.

Fields:

- `Context context.Context`: currently the runtime startup/setup context. This should probably be renamed to `StartupContext` or `SetupContext` for clarity.
- `Name`: selected module name from the runtime profile.
- `As`: require alias, usually `as` or module name.
- `Config json.RawMessage`: marshaled static `ModuleInstance.Config` today; GOJA-053 wants this to become merged static+flags config before `Module.New`.
- `Host`: provider host services such as asset resolution.
- `RuntimeOwner`: owner-thread scheduler for the created runtime.

`ModuleContext` is useful because it is a provider-facing adapter. It deliberately hides most of `engine.RuntimeModuleContext`.

### `engine.RuntimeModuleContext`: engine-facing runtime setup context

Defined in `go-go-goja/engine/runtime_modules.go`.

Fields:

- `Context`: startup/setup context.
- `VM`: concrete `*goja.Runtime`.
- `Loop`: concrete `*eventloop.EventLoop`.
- `Owner`: concrete runtime owner scheduler.
- `AddCloser`: runtime cleanup hook registration.
- `Values`: runtime-scoped mutable value bag.

Engine modules receive this during runtime construction, before `require(...)` is enabled.

### `runtimebridge.RuntimeServices`: lookup bridge from native module code back to runtime services

Defined in `go-go-goja/pkg/runtimebridge/runtimebridge.go`.

The engine stores services by VM:

```go
runtimebridge.Store(vm, runtimebridge.RuntimeServices{
    LifetimeContext: runtimeCtx,
    Loop:            loop,
    Owner:           runtimebridgeOwner{owner: owner},
})
```

Native module code that later receives only a `*goja.Runtime` can call `runtimebridge.Lookup(vm)` or `runtimebridge.CurrentOwnerContext(vm)` to find:

- runtime lifetime context,
- event loop,
- owner-thread `Call`/`Post` scheduler,
- current owner-call context.

This is a runtime lookup table, not a configuration system.

### `runtimeowner.RuntimeOwner`: serialized Goja owner-thread gateway

Defined in `go-go-goja/pkg/runtimeowner/runner.go`.

`RuntimeOwner.Call(ctx, op, fn)` schedules `fn` on the Goja/event-loop owner thread and waits for a result. While the function runs, it wraps it in `runtimebridge.WithCallContext(...)`, so native module functions can inherit the current command/request context.

---

## 2. Phase A: build/codegen runthrough

This is what happens when a user runs:

```bash
xgoja build -f xgoja.yaml --output ./dist/my-app
```

### 2.1 `xgoja build` command parses its own flags

Entrypoint: `go-go-goja/cmd/xgoja/cmd_build.go`.

`newBuildCommand(...)` defines Glazed fields for:

- `file`, default `xgoja.yaml`, short `-f`;
- `output`;
- `work-dir`;
- `keep-work`;
- `dry-run`;
- `xgoja-version`;
- `xgoja-replace`.

Then `buildCommand.Run(ctx, vals)` decodes those parsed values into `buildSettings`.

Pseudocode:

```go
settings := buildSettings{}
vals.DecodeSectionInto(schema.DefaultSlug, &settings)
spec, report, err := buildspec.LoadFile(settings.File)
workDir := settings.WorkDir or os.MkdirTemp(...)
generate.WriteAll(workDir, spec, generate.Options{...})
buildexec.GoModTidy(ctx, workDir)
buildexec.GoBuild(ctx, workDir, outputPath, spec.Go.Tags, spec.Go.LDFlags)
```

### 2.2 `buildspec.LoadFile` reads and validates `xgoja.yaml`

Entrypoint: `go-go-goja/cmd/xgoja/internal/buildspec/load.go`.

Steps:

1. Resolve the file path, defaulting blank input to `xgoja.yaml`.
2. `os.ReadFile(abs)`.
3. Parse unsupported asset-field warnings/errors from the raw YAML node tree.
4. `yaml.Unmarshal(data, spec)` into `buildspec.Spec`.
5. Set `spec.BaseDir` to the directory containing `xgoja.yaml`.
6. `applyDefaults(spec)`.
7. `Validate(spec)`.
8. Return `spec`, `report`, and possibly `ValidationError`.

Important: `ModuleInstance.Config` is parsed here only as `map[string]any`. No provider-specific typed schema is applied at build time today.

### 2.3 `applyDefaults` normalizes buildspec defaults

Source: `go-go-goja/cmd/xgoja/internal/buildspec/load.go`.

Examples:

- empty spec name becomes `xgoja-app`;
- empty Go version becomes `1.26`;
- empty Go module becomes `example.com/generated/<name>`;
- empty target kind becomes `xgoja`;
- empty target output becomes `dist/<name>`;
- empty provider register function becomes `Register`;
- enabled commands get default names like `eval`, `run`, `repl`, and `verbs`;
- enabled config with no filename gets `config.yaml`.

### 2.4 `generate.WriteAll` writes the temporary Go module

Entrypoint: `go-go-goja/cmd/xgoja/internal/generate/generate.go`.

It writes:

```text
<workDir>/go.mod
<workDir>/main.go
<workDir>/xgoja.gen.json
<workDir>/xgoja_embed/jsverbs/...   # optional
<workDir>/xgoja_embed/help/...      # optional
<workDir>/xgoja_embed/assets/...    # optional
```

The important functions are:

- `RenderGoMod(...)` in `cmd/xgoja/internal/generate/gomod.go`.
- `RenderMain(...)` in `cmd/xgoja/internal/generate/main.go`.
- `RenderEmbeddedSpec(...)` in `cmd/xgoja/internal/generate/main.go`.
- `mainTemplateDataFromSpec(...)` in `cmd/xgoja/internal/generate/templates.go`.
- `templates/main.go.tmpl` for generated `main.go`.

### 2.5 `RenderGoMod` decides generated dependencies

Source: `go-go-goja/cmd/xgoja/internal/generate/gomod.go`.

The generated `go.mod` always requires `github.com/go-go-golems/go-go-goja`. It also requires provider modules when versions are specified and target modules for `adapter`/`cobra` targets. `replace` directives come from:

- `--xgoja-replace`, for local go-go-goja development;
- `packages[].replace`, resolved relative to `xgoja.yaml`.

### 2.6 `RenderEmbeddedSpec` creates runtime JSON

Source: `go-go-goja/cmd/xgoja/internal/generate/main.go`.

`RenderEmbeddedSpec` converts the build-time spec into the runtime `app.Spec` JSON. It strips or rewrites build-only details and rewrites embedded source paths to generated embed roots.

Build-time local path:

```yaml
jsverbs:
  - id: local
    path: ./verbs
    embed: true
```

Runtime embedded path may become:

```json
{
  "jsverbs": [
    {"id": "local", "path": "xgoja_embed/jsverbs/local", "embed": true}
  ]
}
```

### 2.7 Generated `main.go` imports providers and registers them

Source: `go-go-goja/cmd/xgoja/internal/generate/templates/main.go.tmpl`.

For the normal `target.kind: xgoja` case, generated code looks conceptually like:

```go
func main() {
    registry := providerapi.NewRegistry()
    must(providerA.Register(registry))
    must(providerB.Register(registry))

    root, err := app.NewRootCommand(app.Options{
        Providers: registry,
        SpecJSON: embeddedSpecJSON,
        EmbeddedJSVerbs: embeddedJSVerbs,
        EmbeddedHelp: embeddedHelp,
        EmbeddedAssets: embeddedAssets,
    })
    must(err)

    if err := root.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

The generated binary compiles provider packages into the target. That means provider registration happens at target startup, not at `xgoja build` time.

### 2.8 Build execution compiles the generated target

Back in `cmd/xgoja/cmd_build.go`:

1. `buildexec.GoModTidy(ctx, workDir)`.
2. `buildexec.GoBuild(ctx, workDir, outputPath, spec.Go.Tags, spec.Go.LDFlags)`.
3. Print `xgoja build ok: <outputPath>`.

At this point the target binary contains:

- provider imports,
- an embedded runtime JSON spec,
- optional embedded fs trees,
- xgoja app/runtime code.

---

## 3. Phase B: generated target program startup

This is what happens when the user runs the generated binary, before a script command actually creates a Goja runtime.

```bash
./dist/my-app run ./script.js --runtime main --http-listen :8787
```

### 3.1 Generated `main` registers provider packages

Source: `cmd/xgoja/internal/generate/templates/main.go.tmpl`.

The generated program creates a provider registry and calls every package’s configured registration function:

```go
registry := providerapi.NewRegistry()
must(provider.Register(registry))
```

Provider registration stores metadata. It does not create a Goja runtime yet.

Example provider registration pattern:

```go
return registry.Package(PackageID,
    providerapi.Module{Name: "...", New: func(ctx providerapi.ModuleContext) (...) { ... }},
    providerapi.WithPackageCapability(capability),
)
```

Registry mechanics are in `go-go-goja/pkg/xgoja/providerapi/registry.go`.

### 3.2 `app.NewRootCommand` decodes embedded spec and constructs a Host

Source: `go-go-goja/pkg/xgoja/app/root.go`.

`NewRootCommand(opts)`:

1. Requires `opts.Providers`.
2. Decodes `opts.SpecJSON` into `app.Spec`.
3. Calls `NewHostWithOptions(...)`.
4. Creates a Cobra root command.
5. Calls `host.AttachDefaultCommands(root)`.

### 3.3 `Host` is the generated program’s wiring object

Source: `go-go-goja/pkg/xgoja/app/host.go`.

`Host` holds:

- `Providers`: the provider registry.
- `Spec`: decoded runtime spec.
- `Factory`: app-level `RuntimeFactory`.
- embedded filesystems.
- `Services`: host services exposed to providers.
- `MiddlewaresFunc`: Glazed parser middleware builder.

Construction:

```go
services := HostServices{Assets: NewAssetStore(opts.EmbeddedAssets, spec)}
factory := NewRuntimeFactory(providers, spec, services)
middlewaresFunc := MiddlewaresFromSpec(spec)
```

This is why provider `ModuleContext.Host` can later expose asset services.

### 3.4 `Host.AttachDefaultCommands` adds commands

Source: `go-go-goja/pkg/xgoja/app/host.go`.

Depending on `spec.Commands`, it attaches:

- eval command,
- run command,
- repl/TUI command,
- modules command,
- jsverbs command,
- package-owned command providers.

Each Glazed command is converted to a Cobra command using `buildGlazedCobraCommand(...)` and the host’s `MiddlewaresFunc`.

### 3.5 Command construction gathers provider public sections

Sources:

- `go-go-goja/pkg/xgoja/app/run.go`
- `go-go-goja/pkg/xgoja/app/root.go`
- `go-go-goja/pkg/xgoja/app/module_sections.go`
- `go-go-goja/pkg/xgoja/providerutil/sections.go`

For `run`, `newRunCommand(factory, spec)` chooses the default runtime profile for the command, then calls:

```go
moduleSections, _, sectionErr := factory.sectionsForRuntimeProfile("run", profile)
```

`sectionsForRuntimeProfile` does:

1. `selectedModuleDescriptors(profile)`.
2. `providerutil.CollectConfigSections(...)`.

Current type name:

```go
type ConfigSectionCapability interface {
    PackageCapability
    ConfigSections(SectionContext) ([]schema.Section, error)
}
```

For clarity, GOJA-053 should rename this to something like:

```go
type CommandLineFlagsSectionCapability interface { ... }
```

with a comment that it provides public command-line/config/env sections, not internal module config sections.

These sections are added to the command description, so they become parseable Glazed fields. They are public user-facing sections.

### 3.6 Generated command parser source precedence

Source: `go-go-goja/pkg/xgoja/app/middlewares.go`.

`MiddlewaresFromSpec(spec)` returns the Glazed source chain. If `envPrefix` or `config.enabled` is set, effective precedence is:

```text
defaults < config files < env < positional args < cobra flags
```

The code comment explains why the returned slice is highest-to-lowest: Glazed source middlewares call `next` before applying themselves.

This is where user input becomes `*values.Values` passed into command `Run(ctx, vals)`.

---

## 4. Phase C: executing a script with `run`

This is the detailed flow for:

```bash
./dist/my-app run ./script.js --runtime main --some-provider-flag value
```

### 4.1 Cobra/Glazed parses command values

Before `runCommand.Run` is called, Glazed has parsed:

- default command section fields: `file`, `runtime`, `keep-alive`;
- provider public sections collected from selected runtime modules;
- config file values if enabled;
- env values if enabled;
- Cobra flags;
- positional args.

The parsed result is `*values.Values`.

### 4.2 `runCommand.Run` decodes built-in run settings

Source: `go-go-goja/pkg/xgoja/app/run.go`.

```go
settings := runSettings{}
vals.DecodeSectionInto(schema.DefaultSlug, &settings)
selectedModules := c.factory.selectedModuleDescriptors(settings.Runtime)
runScriptFileWithInitializers(ctx, c.factory, settings.Runtime, settings.File, vals, selectedModules, settings.KeepAlive)
```

At this point xgoja has both:

- the target runtime profile name;
- all parsed public command section values.

### 4.3 `runScriptFileWithInitializers` prepares module resolution

Source: `go-go-goja/pkg/xgoja/app/run.go`.

It resolves the script path and constructs a `require.Option` that adds the script’s directory to module resolution roots:

```go
requireOpt := engine.RequireOptionWithModuleRootsFromScript(scriptPath, engine.DefaultModuleRootsOptions())
rt, err := factory.NewRuntime(ctx, profile, requireOpt)
```

### 4.4 `app.RuntimeFactory.NewRuntime` selects modules for the profile

Source: `go-go-goja/pkg/xgoja/app/factory.go`.

`app.RuntimeFactory` is the xgoja layer that converts an `app.Spec` runtime profile into engine module specs.

For each `ModuleInstance` in `spec.Runtimes[profile].Modules`:

1. Resolve provider module from `providerapi.Registry`.
2. Create `providerRuntimeModuleSpec{instance, module, services}`.
3. Add it to the engine builder with `.WithModules(modules...)`.

Pseudocode:

```go
runtime := f.spec.Runtimes[profile]
for _, instance := range runtime.Modules {
    module := f.providers.ResolveModule(instance.Package, instance.Name)
    modules = append(modules, providerRuntimeModuleSpec{instance, module, services})
}
builder := engine.NewBuilder(
    engine.WithImplicitDefaultRegistryModules(false),
    engine.WithDataOnlyDefaultRegistryModules(false),
).WithModules(modules...)
return builder.Build().NewRuntime(
    engine.WithStartupContext(ctx),
    engine.WithLifetimeContext(ctx),
)
```

Important: today, `values.Values` from command parsing is **not** passed into `NewRuntime`. That is the main GOJA-053 gap.

### 4.5 Engine builder freezes the runtime composition

Source: `go-go-goja/engine/factory.go`.

`engine.FactoryBuilder.Build()` validates modules and runtime initializers, checks duplicate IDs, and returns an immutable `engine.Factory`.

This is engine-level, not xgoja provider-level. It knows only `RuntimeModuleSpec` entries and engine runtime initializers.

### 4.6 `engine.Factory.NewRuntime` creates the concrete VM/runtime

Source: `go-go-goja/engine/factory.go`.

Steps:

1. Normalize startup and lifetime contexts.
2. Create `vm := goja.New()`.
3. Create and start `eventloop.NewEventLoop()`.
4. Create `runtimeowner.NewRuntimeOwner(vm, loop, ...)`.
5. Derive a runtime lifetime context from `lifetimeCtx`.
6. Construct `engine.Runtime` with `VM`, `Loop`, `Owner`, `Values`, runtime context, and closers.
7. Store runtime services in `runtimebridge` by VM.
8. Create a Goja Node `require.Registry`.
9. Construct `engine.RuntimeModuleContext`.
10. Register data-only default modules if enabled.
11. Register each explicit xgoja provider runtime module spec.
12. Enable `require(...)` on the VM.
13. Enable console, buffer, url, performance globals, and console timers.
14. Run engine-level runtime initializers.
15. Return `*engine.Runtime`.

Sequence sketch:

```text
engine.Factory.NewRuntime
  ├─ goja.New()
  ├─ eventloop.NewEventLoop(); loop.Start()
  ├─ runtimeowner.NewRuntimeOwner(vm, loop)
  ├─ runtimeCtx := context.WithCancel(lifetimeCtx)
  ├─ rt := &engine.Runtime{VM, Loop, Owner, Values, runtimeCtx}
  ├─ runtimebridge.Store(vm, RuntimeServices{runtimeCtx, loop, owner})
  ├─ reg := require.NewRegistry(...)
  ├─ moduleCtx := &engine.RuntimeModuleContext{startupCtx, VM, Loop, Owner, AddCloser, Values}
  ├─ for mod in f.modules: mod.RegisterRuntimeModule(moduleCtx, reg)
  ├─ rt.Require = reg.Enable(vm)
  ├─ install built-in globals
  └─ return rt
```

### 4.7 `providerRuntimeModuleSpec.RegisterRuntimeModule` calls provider `Module.New`

Source: `go-go-goja/pkg/xgoja/app/factory.go`.

This is the bridge from engine module registration to provider modules.

Current code:

```go
config, err := json.Marshal(s.instance.Config)
loader, err := s.module.New(providerapi.ModuleContext{
    Context:      ctx.Context,
    Name:         s.instance.Name,
    As:           s.instance.Alias(),
    Config:       config,
    Host:         s.services,
    RuntimeOwner: ctx.Owner,
})
reg.RegisterNativeModule(s.instance.Alias(), loader)
```

So the order is:

```text
xgoja.yaml ModuleInstance.Config
  → json.Marshal
  → providerapi.ModuleContext.Config
  → providerapi.Module.New
  → require.ModuleLoader
  → require.Registry.RegisterNativeModule(alias, loader)
```

This is exactly where GOJA-053 wants to replace `json.Marshal(s.instance.Config)` with:

```text
parse static config through internal module config schema
merge command/env/config/flag SectionValues override
marshal final values to JSON
pass merged JSON to ModuleContext.Config
```

### 4.8 `Module.New` happens before script code runs

Provider `Module.New` returns a `require.ModuleLoader`. It is called during runtime construction, before `require(...)` is enabled and before the user script is required.

The returned loader is registered under the selected alias. The loader itself may run lazily when JavaScript first calls `require(alias)`, depending on goja_nodejs require behavior and the loader implementation.

The practical consequence: provider configuration needed to construct the loader must be available at `Module.New` time.

### 4.9 xgoja runtime initializer capabilities run after runtime construction

Source: `go-go-goja/pkg/xgoja/app/run.go` and `go-go-goja/pkg/xgoja/app/module_sections.go`.

After `factory.NewRuntime(...)` returns, run/eval/jsverbs do:

```go
if vals != nil && len(selectedModules) > 0 {
    initRuntimeFromSections(ctx, vals, rt, selectedModules)
}
```

`initRuntimeFromSections` wraps the runtime in a small `runtimeHandle` exposing:

- `Runtime() *goja.Runtime`,
- `Close(ctx) error`,
- `AddCloser(fn)`.

Then `providerutil.InitRuntimeFromSections(...)` calls every package capability implementing:

```go
type RuntimeInitializerCapability interface {
    PackageCapability
    InitRuntimeFromSections(context.Context, *values.Values, RuntimeHandle) error
}
```

This is useful for post-runtime setup such as HTTP server settings. It is too late for changing `ModuleContext.Config`, because `Module.New` already ran.

### 4.10 The script runs through `RuntimeOwner.Call`

Source: `go-go-goja/pkg/xgoja/app/run.go`.

```go
_, err = rt.Owner.Call(ctx, "xgoja.run", func(_ context.Context, vm *goja.Runtime) (any, error) {
    return rt.Require.Require(scriptPath)
})
```

`RuntimeOwner.Call` is defined in `go-go-goja/pkg/runtimeowner/runner.go`. It schedules work on the Goja/event-loop owner thread. While running, it installs the current context into runtimebridge:

```go
runtimebridge.WithCallContext(r.vm, ctx, func() (any, error) { ... })
```

This is how native module methods called by JavaScript can later recover the command/request context using `runtimebridge.CurrentOwnerContext(vm)`.

### 4.11 Runtime close unwinds resources

Source: `go-go-goja/engine/runtime.go`.

`runScriptFileWithInitializers` defers:

```go
rt.Close(ctx)
```

`Runtime.Close`:

1. Cancels the runtime lifetime context.
2. Waits for owner-thread work to become idle, interrupting if needed.
3. Runs registered closers in reverse order.
4. Deletes runtimebridge services for the VM.
5. Shuts down the runtime owner.
6. Stops the event loop.

This is why long-lived native module resources should register closers through `RuntimeCloserRegistry` or `Runtime.AddCloser`.

---

## 5. One-page sequence diagram

### Build/codegen

```text
User
  │
  ├─ xgoja build -f xgoja.yaml
  │
  ▼
buildCommand.Run                         cmd/xgoja/cmd_build.go
  ├─ Decode build flags from Glazed values
  ├─ buildspec.LoadFile                  cmd/xgoja/internal/buildspec/load.go
  │    ├─ os.ReadFile(xgoja.yaml)
  │    ├─ yaml.Unmarshal → buildspec.Spec
  │    ├─ applyDefaults
  │    └─ Validate
  ├─ generate.WriteAll                   cmd/xgoja/internal/generate/generate.go
  │    ├─ copy embedded dirs
  │    ├─ RenderGoMod                    generate/gomod.go
  │    ├─ RenderMain                     generate/main.go + templates/main.go.tmpl
  │    └─ RenderEmbeddedSpec             generate/main.go
  ├─ go mod tidy                         buildexec
  └─ go build → ./dist/my-app            buildexec
```

### Generated target startup and command construction

```text
./dist/my-app
  │
  ▼
generated main.go                        templates/main.go.tmpl
  ├─ registry := providerapi.NewRegistry()
  ├─ provider.Register(registry)         provider packages
  ├─ app.NewRootCommand                  pkg/xgoja/app/root.go
  │    ├─ decode embeddedSpecJSON → app.Spec
  │    ├─ app.NewHostWithOptions         pkg/xgoja/app/host.go
  │    │    ├─ HostServices
  │    │    ├─ RuntimeFactory
  │    │    └─ MiddlewaresFromSpec
  │    └─ Host.AttachDefaultCommands
  │         ├─ newRunCommand
  │         │    └─ sectionsForRuntimeProfile
  │         │         └─ CollectConfigSections
  │         └─ buildGlazedCobraCommand
  └─ root.Execute()
```

### Script execution

```text
./dist/my-app run ./script.js --flags...
  │
  ▼
Cobra + Glazed parser                    app/middlewares.go
  └─ values.Values                       defaults < config < env < args < flags
  │
  ▼
runCommand.Run                           app/run.go
  ├─ decode runSettings
  ├─ selectedModuleDescriptors
  └─ runScriptFileWithInitializers
       ├─ RequireOptionWithModuleRootsFromScript
       ├─ app.RuntimeFactory.NewRuntime  app/factory.go
       │    ├─ resolve provider modules from providerapi.Registry
       │    ├─ create providerRuntimeModuleSpec entries
       │    └─ engine.NewBuilder(...).WithModules(...).Build().NewRuntime(...)
       │         ├─ goja.New + eventloop + runtimeowner
       │         ├─ runtimebridge.Store(vm, RuntimeServices)
       │         ├─ require.NewRegistry
       │         ├─ RuntimeModuleContext
       │         ├─ providerRuntimeModuleSpec.RegisterRuntimeModule
       │         │    ├─ marshal ModuleInstance.Config
       │         │    ├─ providerapi.Module.New(ModuleContext)
       │         │    └─ require.Registry.RegisterNativeModule(alias, loader)
       │         ├─ reg.Enable(vm)
       │         └─ return engine.Runtime
       ├─ initRuntimeFromSections         post-runtime provider capabilities
       ├─ rt.Owner.Call("xgoja.run", ...)
       │    └─ rt.Require.Require(scriptPath)
       └─ rt.Close(ctx)
```

---

## 6. Contexts: what each one means

| Name | Defined in | Meaning | Lifetime |
|---|---|---|---|
| Command `ctx` | Glazed/Cobra command `Run(ctx, vals)` | User command/request context. Carries cancellation/deadlines/tracing for command execution. | One command invocation. |
| `engine.WithStartupContext(ctx)` | `engine/options.go` | Context used while constructing the runtime and running setup hooks. | Runtime construction phase. |
| `engine.WithLifetimeContext(ctx)` | `engine/options.go` | Parent for runtime-owned lifetime context. | Runtime lifetime; canceled on `Runtime.Close`. |
| `engine.RuntimeModuleContext.Context` | `engine/runtime_modules.go` | Startup/setup context passed to low-level module registration. | Runtime construction phase. |
| `providerapi.ModuleContext.Context` | `providerapi/module.go` | Provider-facing copy of startup/setup context. Should probably be renamed `StartupContext`. | `Module.New` setup phase. |
| `runtimebridge.RuntimeServices.LifetimeContext` | `runtimebridge/runtimebridge.go` | Runtime-owned lifetime context discoverable from a VM. | Until runtime close. |
| `runtimebridge.CurrentOwnerContext(vm)` | `runtimebridge/runtimebridge.go` | Current owner-call context while JS/native functions are executing on owner thread; falls back to runtime lifetime. | Dynamic stack during owner calls. |

Recommendation: rename `ModuleContext.Context` to `StartupContext` or `SetupContext`, or at least document it that way. It is not the long-lived runtime context.

---

## 7. Factories and registries: why there are several

### Build generation is not runtime construction

`generate.WriteAll` is a build-time factory for files. It does not create Goja runtimes.

### Provider registry is not require registry

- `providerapi.Registry`: generated app metadata registry; resolves provider modules and capabilities by package/module ID.
- `require.Registry`: per-Goja-runtime Node-style module registry; registers aliases and module loaders.

### xgoja runtime factory is not engine factory

- `app.RuntimeFactory`: knows about `app.Spec`, runtime profile names, selected provider modules, provider registry, and host services.
- `engine.Factory`: knows about concrete runtime construction: VM, loop, owner, require registry, runtimebridge, module specs, and engine-level initializers.

The adapter between them is `providerRuntimeModuleSpec` in `go-go-goja/pkg/xgoja/app/factory.go`.

---

## 8. How GOJA-053 fits into this flow

The current flow is:

```text
Glazed parses public command values
  ↓
runCommand.Run(ctx, vals)
  ↓
factory.NewRuntime(ctx, profile)             # vals not passed today
  ↓
engine.Factory.NewRuntime
  ↓
providerRuntimeModuleSpec.RegisterRuntimeModule
  ↓
json.Marshal(ModuleInstance.Config)
  ↓
providerapi.Module.New(ModuleContext{Config: static JSON})
  ↓
return runtime
  ↓
initRuntimeFromSections(ctx, vals, rt, selectedModules)
```

The timing bug for GOJA-053 is visible: `initRuntimeFromSections` sees parsed flags/config/env, but it runs after `Module.New`.

The proposed GOJA-053 flow is:

```text
Glazed parses public command values
  ↓
runCommand.Run(ctx, vals)
  ↓
factory.NewRuntimeFromSections(ctx, profile, vals)
  ↓
for each selected ModuleInstance:
    internalSection := provider.ModuleConfigSection(...)
    staticValues := parse ModuleInstance.Config through internalSection
    overrideValues := provider maps public command values → internal SectionValues
    finalValues := merge(staticValues, overrideValues)
    configJSON := json.Marshal(finalValues.Fields.ToInterfaceMap())
  ↓
providerapi.Module.New(ModuleContext{Config: configJSON})
  ↓
return runtime
  ↓
optional post-runtime initializers still run
```

This is why the design separates:

1. **Public command-line/config/env sections** — current `ConfigSectionCapability`, better named `CommandLineFlagsSectionCapability`.
2. **Internal module config section** — new `ModuleConfigSectionCapability` for parsing `xgoja.yaml` module config and validating pre-runtime merged config.
3. **Mapping hook/helper** — provider maps public parsed values into internal config `SectionValues`.

---

## 9. Review checklist for an intern

When debugging xgoja runtime behavior, ask these questions in order:

1. **Was the provider compiled into the target?**
   - Check `xgoja.yaml packages[]`.
   - Check generated `main.go` imports and calls provider `Register`.
2. **Was the provider module selected in the runtime profile?**
   - Check `xgoja.yaml runtimes.<profile>.modules[]`.
   - Check `app.RuntimeFactory.NewRuntime` resolving `ResolveModule`.
3. **Was the module alias correct?**
   - Check `ModuleInstance.As` / `Alias()`.
   - Check `reg.RegisterNativeModule(alias, loader)`.
4. **Were public flags added to the command?**
   - Check provider capability implementing current `ConfigSectionCapability`.
   - Check `sectionsForRuntimeProfile` and `CollectConfigSections`.
5. **Were command values parsed?**
   - Check `MiddlewaresFromSpec` and Glazed `values.Values`.
   - Check config/env layers if enabled.
6. **Did the setting need to affect `Module.New`?**
   - If yes, current runtime initializers are too late; this is GOJA-053 territory.
7. **Did runtime initialization run?**
   - Check `RuntimeInitializerCapability` and `initRuntimeFromSections`.
8. **Did JavaScript actually require the module alias?**
   - `Module.New` registers a loader, but JavaScript still needs `require(alias)` unless another module triggers it.
9. **Is async/native code using the correct context?**
   - Use `runtimebridge.CurrentOwnerContext(vm)` for current owner-call context.
   - Use `runtimebridge.LifetimeContext(vm)` for runtime-owned background work.
10. **Are resources cleaned up?**
   - Use `Runtime.AddCloser` / `RuntimeCloserRegistry` and verify `Runtime.Close` runs.

---

## 10. Source file map

### Codegen and buildspec

- `go-go-goja/cmd/xgoja/cmd_build.go` — `xgoja build` command flow.
- `go-go-goja/cmd/xgoja/internal/buildspec/spec.go` — YAML model.
- `go-go-goja/cmd/xgoja/internal/buildspec/load.go` — load/default/validate.
- `go-go-goja/cmd/xgoja/internal/generate/generate.go` — generated workspace writing.
- `go-go-goja/cmd/xgoja/internal/generate/gomod.go` — generated `go.mod`.
- `go-go-goja/cmd/xgoja/internal/generate/main.go` — embedded spec rendering.
- `go-go-goja/cmd/xgoja/internal/generate/templates.go` — generated main template data.
- `go-go-goja/cmd/xgoja/internal/generate/templates/main.go.tmpl` — generated target main.

### Generated app layer

- `go-go-goja/pkg/xgoja/app/spec.go` — runtime embedded spec model.
- `go-go-goja/pkg/xgoja/app/root.go` — root command, eval, jsverbs.
- `go-go-goja/pkg/xgoja/app/run.go` — run command and script execution.
- `go-go-goja/pkg/xgoja/app/host.go` — app host and command attachment.
- `go-go-goja/pkg/xgoja/app/factory.go` — xgoja runtime factory and provider module adapter.
- `go-go-goja/pkg/xgoja/app/module_sections.go` — selected modules, public sections, runtime init.
- `go-go-goja/pkg/xgoja/app/middlewares.go` — env/config/default/flag parser middleware.

### Provider API layer

- `go-go-goja/pkg/xgoja/providerapi/registry.go` — provider package registry.
- `go-go-goja/pkg/xgoja/providerapi/module.go` — module factory/context.
- `go-go-goja/pkg/xgoja/providerapi/capabilities.go` — section and runtime initializer capabilities.
- `go-go-goja/pkg/xgoja/providerapi/commands.go` — provider command set APIs.
- `go-go-goja/pkg/xgoja/providerutil/sections.go` — collect sections and run init capabilities.

### Engine/runtime layer

- `go-go-goja/engine/factory.go` — concrete runtime construction.
- `go-go-goja/engine/runtime.go` — runtime lifecycle and close.
- `go-go-goja/engine/runtime_modules.go` — runtime module specs and context.
- `go-go-goja/engine/options.go` — startup/lifetime contexts.
- `go-go-goja/pkg/runtimebridge/runtimebridge.go` — runtime services and context lookup.
- `go-go-goja/pkg/runtimeowner/runner.go` — owner-thread `Call`/`Post`.
