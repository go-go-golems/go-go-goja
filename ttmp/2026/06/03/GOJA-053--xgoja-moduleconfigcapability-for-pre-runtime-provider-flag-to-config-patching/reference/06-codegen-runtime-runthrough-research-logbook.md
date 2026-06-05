---
Title: Codegen Runtime Runthrough Research Logbook
Ticket: GOJA-053
Status: active
Topics:
    - xgoja
    - code-generation
    - lifecycle
    - research
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/xgoja/cmd_build.go
      Note: |-
        xgoja build command flow resource evaluated for codegen entrypoint.
        Build command resource evaluated in codegen runtime logbook
    - Path: go-go-goja/cmd/xgoja/internal/buildspec/load.go
      Note: xgoja.yaml load/default/validation resource evaluated for buildspec lifecycle.
    - Path: go-go-goja/cmd/xgoja/internal/buildspec/spec.go
      Note: |-
        Buildspec schema resource evaluated for xgoja.yaml data model.
        Buildspec schema resource evaluated in codegen runtime logbook
    - Path: go-go-goja/cmd/xgoja/internal/generate/generate.go
      Note: Generated workspace file emission resource evaluated for codegen outputs.
    - Path: go-go-goja/cmd/xgoja/internal/generate/main.go
      Note: Embedded runtime spec rendering resource evaluated for build-to-runtime handoff.
    - Path: go-go-goja/cmd/xgoja/internal/generate/templates.go
      Note: Generated main template data resource evaluated for provider imports and host construction.
    - Path: go-go-goja/cmd/xgoja/internal/generate/templates/main.go.tmpl
      Note: |-
        Generated target program template resource evaluated for startup order.
        Generated main template resource evaluated in codegen runtime logbook
    - Path: go-go-goja/engine/factory.go
      Note: |-
        Engine runtime construction resource evaluated for RuntimeModuleContext and runtimebridge setup.
        Engine runtime construction resource evaluated in codegen runtime logbook
    - Path: go-go-goja/pkg/runtimebridge/runtimebridge.go
      Note: |-
        RuntimeServices resource evaluated for VM-to-runtime service lookup and owner context propagation.
        RuntimeServices bridge resource evaluated in codegen runtime logbook
    - Path: go-go-goja/pkg/runtimeowner/runner.go
      Note: Owner scheduling resource evaluated in codegen runtime logbook
    - Path: go-go-goja/pkg/xgoja/app/factory.go
      Note: |-
        xgoja RuntimeFactory and provider module adapter resource evaluated for Module.New timing.
        RuntimeFactory and Module.New timing resource evaluated in codegen runtime logbook
    - Path: go-go-goja/pkg/xgoja/app/run.go
      Note: Generated run command resource evaluated in codegen runtime logbook
ExternalSources: []
Summary: Tracks the source files and docs used for the xgoja codegen/runtime execution runthrough, noting usefulness, gaps, stale comments, and update opportunities.
LastUpdated: 2026-06-04T00:00:00Z
WhatFor: Use when maintaining or extending the xgoja lifecycle runthrough and deciding which source files or docs need cleanup to reduce confusion around factories, registries, module contexts, and runtimebridge.
WhenToUse: Before revising design/05-xgoja-codegen-and-script-execution-runthrough.md, renaming provider capabilities, changing ModuleContext, or documenting generated xgoja runtime behavior.
---


# Codegen Runtime Runthrough Research Logbook

## Purpose

This logbook records the resources used specifically for `design/05-xgoja-codegen-and-script-execution-runthrough.md`. Its purpose is to make the evidence trail reviewable: which files were read, why each file was chosen, what it clarified, what was not useful, what looked stale or confusing, and what should be updated.

The scope is the xgoja build/codegen process and the generated target program's runtime execution flow. This logbook is not the Glazed SectionValues research logbook; that separate resource is `reference/05-glazed-config-research-logbook.md`.

---

## Resource 1: `go-go-goja/cmd/xgoja/cmd_build.go`

### What I was researching

The top-level `xgoja build` command flow: how the CLI reads `xgoja.yaml`, creates a generated workspace, and compiles the generated target program.

### What I was looking for in this document in particular

- The build command entrypoint.
- Which flags influence code generation.
- Where `buildspec.LoadFile` is called.
- Where generated files are written.
- Where `go mod tidy` and `go build` happen.

### Why I chose it

A full lifecycle runthrough must start at the user-visible command that turns `xgoja.yaml` into a binary. This file is the first concrete handoff from command-line values into the buildspec/codegen pipeline.

### How I found the resource itself

I searched for xgoja YAML and build references with repository search patterns such as:

```bash
rg -n "xgoja.yaml|Build reads|LoadFile|generate.WriteAll|GoBuild" go-go-goja/cmd/xgoja go-go-goja/pkg/xgoja -S
```

### What I found useful in the document

- `buildCommand.Run` gives a concise sequence: decode Glazed settings, load the spec, create or use a work dir, call `generate.WriteAll`, run `go mod tidy`, then run `go build`.
- The command flags document the developer-facing controls: `--file`, `--output`, `--work-dir`, `--keep-work`, `--dry-run`, `--xgoja-version`, and `--xgoja-replace`.
- The code shows that codegen happens before module dependencies are resolved, because generated `go.mod` is created before `buildexec.GoModTidy`.

### What I didn't find useful

- The file does not explain what generated `main.go` contains; that requires following `generate.WriteAll` into the generate package.
- It does not explain runtime behavior after the binary starts; it only gets to build output.

### What is out of date / what was wrong

- Nothing obviously wrong in the code.
- The long command description is accurate but high-level; it does not name the generated files, which is useful when debugging a kept work directory.

### What would need updating

- Add a short comment or help paragraph listing generated workspace files: `go.mod`, `main.go`, `xgoja.gen.json`, and optional `xgoja_embed/...` trees.
- In docs/help, point users who pass `--keep-work` to inspect generated `main.go` and `xgoja.gen.json` for runtime debugging.

---

## Resource 2: `go-go-goja/cmd/xgoja/internal/buildspec/load.go`

### What I was researching

How `xgoja.yaml` is loaded, parsed, defaulted, and validated before code generation.

### What I was looking for in this document in particular

- Whether `xgoja.yaml` is parsed as YAML or JSON.
- Where default values are applied.
- Whether provider-specific module config is type-checked at build time.
- How relative paths are rooted.

### Why I chose it

The runthrough needed to explain what happens to `xgoja.yaml` before the generated program exists. This file is the entrypoint after `cmd_build.go` delegates to `buildspec.LoadFile`.

### How I found the resource itself

From `cmd_build.go`, which calls:

```go
spec, report, err := buildspec.LoadFile(settings.File)
```

### What I found useful in the document

- `LoadFile` resolves the spec path to an absolute path and records `spec.BaseDir`.
- `yaml.Unmarshal(data, spec)` confirms `xgoja.yaml` is parsed into a Go struct.
- `applyDefaults(spec)` shows defaulting is centralized after parse.
- Validation is run after defaults, and validation errors are returned with a `Report`.
- `ConfigSpec.FileName` defaults to `config.yaml` only when config is enabled.

### What I didn't find useful

- The file does not show the full spec schema; it must be read with `spec.go`.
- It does not show provider-specific module config validation, because that does not exist today.

### What is out of date / what was wrong

- Nothing obviously wrong.
- For GOJA-053, the absence of provider-specific validation is an important gap, not a bug in this file.

### What would need updating

- If GOJA-053 adds build-time validation of module config sections, this file or a nearby validation layer would need hooks to ask providers for internal module config schemas.
- Add documentation warning that `runtimes.*.modules[].config` is currently an untyped map at buildspec-load time.

---

## Resource 3: `go-go-goja/cmd/xgoja/internal/buildspec/spec.go`

### What I was researching

The build-time shape of `xgoja.yaml` and the fields that survive into the generated runtime spec.

### What I was looking for in this document in particular

- The type of `Spec`.
- The shape of `Runtimes`, `Runtime`, and `ModuleInstance`.
- The type of `ModuleInstance.Config`.
- Build-only versus runtime-relevant fields.

### Why I chose it

The user asked for interplay between classes/data. `spec.go` is the canonical place to understand what data the build pipeline starts from.

### How I found the resource itself

From `buildspec.LoadFile`, which unmarshals into `&Spec{}`. I then read the file defining `Spec`.

### What I found useful in the document

- `ModuleInstance.Config map[string]any` confirms static module config is a raw untyped map today.
- `ModuleInstance.Alias()` and `Ref()` clarify module aliasing and package/module identity.
- `CommandProviderInstance.Config map[string]any` shows command-provider config has a similar raw-map shape.
- The YAML and JSON tags explain why the same model feeds both YAML input and generated JSON output.

### What I didn't find useful

- The file is structural only; it does not explain lifecycle or ordering.
- It does not document which fields are build-only versus runtime-facing in prose.

### What is out of date / what was wrong

- Nothing obviously wrong.
- The `Config map[string]any` field is under-documented relative to how important it is for provider initialization.

### What would need updating

- Add field comments for `ModuleInstance.Config`, especially that it is currently passed to `providerapi.Module.New` as JSON at runtime.
- If GOJA-053 introduces internal config schemas, document whether `Config` remains raw at YAML load time and is validated later.

---

## Resource 4: `go-go-goja/cmd/xgoja/internal/generate/generate.go`

### What I was researching

What files and embedded directories the code generator writes into the temporary build workspace.

### What I was looking for in this document in particular

- The list of generated top-level files.
- Whether embedded jsverbs/help/assets are copied before or after rendering the spec.
- How relative source paths are resolved.
- Which functions render `go.mod`, `main.go`, and embedded spec JSON.

### Why I chose it

The codegen runthrough needed to move from parsed buildspec to concrete generated files. This file is the codegen orchestration point.

### How I found the resource itself

From `cmd_build.go`, which calls:

```go
generate.WriteAll(workDir, spec, generate.Options{...})
```

### What I found useful in the document

- `WriteAll` explicitly writes `go.mod`, `main.go`, and `xgoja.gen.json`.
- Embedded jsverbs/help/assets are copied before generated files are written.
- `resolveSourcePath` resolves relative paths against `spec.BaseDir`, which explains why source paths are relative to the `xgoja.yaml` location rather than the shell CWD.
- Asset copying skips `node_modules`, which is useful operational detail.

### What I didn't find useful

- It does not explain how `RenderMain` chooses provider imports or host construction; that is in `templates.go`.
- It does not explain generated runtime behavior after `main.go` starts.

### What is out of date / what was wrong

- Nothing obviously wrong.

### What would need updating

- Add a comment in `WriteAll` describing the generated workspace contract and why `xgoja.gen.json` exists even though generated `main.go` embeds `embeddedSpecJSON`.

---

## Resource 5: `go-go-goja/cmd/xgoja/internal/generate/gomod.go`

### What I was researching

How the generated Go module decides dependencies and replacements.

### What I was looking for in this document in particular

- Whether generated `go.mod` always depends on go-go-goja.
- How provider package versions and replacement paths are handled.
- How target imports are handled for adapter/cobra targets.

### Why I chose it

The generated binary is a real Go module. Understanding dependency resolution is part of the codegen lifecycle, especially when local provider replacements or local go-go-goja development are involved.

### How I found the resource itself

From `generate.WriteAll`, which calls `RenderGoMod(spec, opts)`.

### What I found useful in the document

- `RenderGoMod` always requires `github.com/go-go-golems/go-go-goja`.
- Provider package versions are included when specified in `packages[].version`.
- `--xgoja-replace` adds a replacement for go-go-goja.
- `packages[].replace` paths are resolved relative to `spec.BaseDir`.

### What I didn't find useful

- It does not discuss generated runtime semantics, only module dependency rendering.

### What is out of date / what was wrong

- Nothing obviously wrong.
- `providerModulePath` is heuristic; it is useful but could be surprising for unusual import paths.

### What would need updating

- Add tests or docs for unusual provider import paths if providers start registering from nested packages outside the common `/pkg/`, `/cmd/`, `/internal/`, or `/xgoja` patterns.

---

## Resource 6: `go-go-goja/cmd/xgoja/internal/generate/main.go`

### What I was researching

How the build-time spec is converted into embedded runtime JSON and generated main source.

### What I was looking for in this document in particular

- How `RenderMain` delegates to template rendering.
- How `RenderEmbeddedSpec` constructs the runtime payload.
- Which fields are included in generated runtime JSON.
- How embedded paths are rewritten.

### Why I chose it

This file is the build-to-runtime handoff. The generated program later decodes `embeddedSpecJSON` into `app.Spec`, so this file determines what runtime data exists.

### How I found the resource itself

From `generate.WriteAll`, which writes:

```go
"main.go": RenderMain(spec)
"xgoja.gen.json": RenderEmbeddedSpec(spec)
```

### What I found useful in the document

- `RenderEmbeddedSpec` shows the exact runtime JSON shape.
- It includes runtime-relevant fields: `Name`, `AppName`, `EnvPrefix`, `Config`, `Target`, `Packages`, `Runtimes`, `Commands`, `CommandProviders`, `JSVerbs`, `Help`, and `Assets`.
- It rewrites embedded jsverb/help/asset paths to generated embed roots.
- It omits build-only details such as Go version/module/replacements.

### What I didn't find useful

- The file does not explain why both `embeddedSpecJSON` and `xgoja.gen.json` are emitted.
- The distinction between build-time `buildspec.Spec` and runtime `app.Spec` must be inferred.

### What is out of date / what was wrong

- Nothing obviously wrong.

### What would need updating

- Add comments explaining `runtimeSpec` as the boundary between buildspec and app runtime spec.
- Document the purpose of generated `xgoja.gen.json` if it is intended for debugging/introspection.

---

## Resource 7: `go-go-goja/cmd/xgoja/internal/generate/templates.go`

### What I was researching

How generated `main.go` receives provider imports, host construction, embedded filesystem variables, and target-kind-specific behavior.

### What I was looking for in this document in particular

- How provider imports are derived from `spec.Packages`.
- How target kinds (`xgoja`, `adapter`, `cobra`) affect generated main.
- How embedded filesystem variables are passed to app construction.
- Where `RootConstruction` and `HostConstruction` strings come from.

### Why I chose it

The generated `main.go` template has placeholders. `templates.go` fills them, so it explains why generated startup has different shapes for xgoja/adapter/cobra targets.

### How I found the resource itself

From `RenderMain`, which calls:

```go
renderMainTemplate(mainTemplateDataFromSpec(spec))
```

### What I found useful in the document

- `mainTemplateDataFromSpec` builds provider imports from `spec.Packages`.
- It decides when `context` and `embed` imports are needed.
- It builds `app.NewHostWithOptions(...)` when embedded filesystems exist.
- It builds `app.NewRootCommand(...)` for normal xgoja targets.

### What I didn't find useful

- It uses generated code strings for `HostConstruction` and `RootConstruction`, which are compact but harder to read than typed template branches.

### What is out of date / what was wrong

- Nothing obviously wrong.

### What would need updating

- Consider adding comments that distinguish normal `target.kind: xgoja` from `adapter` and `cobra`, since the generated startup flow diverges there.

---

## Resource 8: `go-go-goja/cmd/xgoja/internal/generate/templates/main.go.tmpl`

### What I was researching

The exact startup order of the generated target program.

### What I was looking for in this document in particular

- Whether provider registration happens before root command construction.
- Where `embeddedSpecJSON` is decoded.
- Whether the generated binary creates a provider registry or a runtime immediately.
- How generated target kinds differ.

### Why I chose it

The user wanted a full runthrough of the generated target program. This template is the source of generated `main.go`, so it is the most direct evidence for startup order.

### How I found the resource itself

From `templates.go`, which parses `templates/main.go.tmpl`.

### What I found useful in the document

- Generated startup always creates `providerapi.NewRegistry()` first.
- It calls each provider package’s configured `Register` function.
- For normal xgoja targets, it calls `app.NewRootCommand(app.Options{...})` and then `root.Execute()`.
- `decodeSpec()` decodes `embeddedSpecJSON` only for target kinds that need an explicit `Host`; normal xgoja target decoding happens inside `app.NewRootCommand`.

### What I didn't find useful

- The template itself does not explain what `app.NewRootCommand` or `Host` do; that requires following into app code.

### What is out of date / what was wrong

- Nothing obviously wrong.
- The generated target-kind branches are concise but not self-documenting for newcomers.

### What would need updating

- Add comments or public docs explaining the three generated target modes and where runtime spec decoding happens in each.

---

## Resource 9: `go-go-goja/pkg/xgoja/app/root.go`

### What I was researching

How the generated target program creates its Cobra root command and how eval/jsverbs commands create runtimes.

### What I was looking for in this document in particular

- `NewRootCommand` flow.
- How embedded spec JSON becomes `app.Spec`.
- How `Host` gets constructed.
- How eval/jsverbs use `RuntimeFactory` and runtime initializers.

### Why I chose it

Generated `main.go` delegates to `app.NewRootCommand` for normal xgoja targets. This file is therefore the runtime app entrypoint after generated startup.

### How I found the resource itself

From `templates/main.go.tmpl`, which calls `app.NewRootCommand(...)`.

### What I found useful in the document

- `NewRootCommand` decodes `embeddedSpecJSON`, constructs `Host`, and calls `host.AttachDefaultCommands(root)`.
- `newEvalCommand` gathers provider sections for the selected runtime profile.
- `evalSourceWithInitializers` creates the runtime first, then runs `initRuntimeFromSections`, then evaluates JavaScript through `rt.Owner.Call`.
- The jsverbs path has the same essential timing: create runtime, run section initializers, invoke JS verb in runtime.

### What I didn't find useful

- The file is large and mixes root setup, eval, modules listing, and jsverbs logic. It is not ideal as a single onboarding reference.
- The run command lives in a separate file, so the main script-execution path requires switching to `run.go`.

### What is out of date / what was wrong

- Nothing obviously wrong.
- The phrase “provider modules may add Glazed sections” in command long text is accurate, but the type name `ConfigSectionCapability` remains confusing because it sounds like static config rather than public command values.

### What would need updating

- If `ConfigSectionCapability` is renamed to `CommandLineFlagsSectionCapability`, update command prose/comments here to use the new terminology.
- Consider splitting onboarding docs by command path: root setup, eval, run, jsverbs.

---

## Resource 10: `go-go-goja/pkg/xgoja/app/run.go`

### What I was researching

The exact flow for executing a JavaScript file in a generated xgoja target.

### What I was looking for in this document in particular

- How run command settings are decoded from Glazed values.
- Where selected module descriptors are computed.
- Where the runtime is created.
- Where post-runtime section initializers run.
- How the script file is executed.

### Why I chose it

The user explicitly asked for a runthrough of executing a script with the generated target program. `run.go` is the primary source for that path.

### How I found the resource itself

From `Host.AttachRun` in `app/host.go`, and from file naming under `pkg/xgoja/app`.

### What I found useful in the document

- `runCommand.Run` decodes `file`, `runtime`, and `keep-alive` from `values.Values`.
- It passes the full parsed `values.Values` into `runScriptFileWithInitializers`.
- `runScriptFileWithInitializers` creates the runtime with script-root require options.
- It runs `initRuntimeFromSections` after `factory.NewRuntime` returns.
- It executes the script through `rt.Owner.Call(... rt.Require.Require(scriptPath) ...)`.
- This file makes the GOJA-053 ordering problem clear: parsed command values are available, but not used before `Module.New` today.

### What I didn't find useful

- It does not show what happens inside `factory.NewRuntime`; that requires `app/factory.go` and `engine/factory.go`.

### What is out of date / what was wrong

- Nothing obviously wrong.

### What would need updating

- After GOJA-053, update this flow to call a runtime creation method that receives parsed section values, such as `NewRuntimeFromSections(ctx, profile, vals, opts...)`.
- Update comments to distinguish pre-runtime config merging from post-runtime initializers.

---

## Resource 11: `go-go-goja/pkg/xgoja/app/host.go`

### What I was researching

The generated app wiring object that connects provider registry, spec, runtime factory, host services, embedded filesystems, parser middlewares, and Cobra commands.

### What I was looking for in this document in particular

- What `Host` stores.
- How `RuntimeFactory` is constructed.
- How commands are attached.
- How embedded assets and host services are passed along.

### Why I chose it

`Host` is a key intermediary in generated program startup. Without it, `app.NewRootCommand`, command attachment, and runtime factory construction look disconnected.

### How I found the resource itself

From `app.NewRootCommand`, which calls `NewHostWithOptions(...)`.

### What I found useful in the document

- `NewHostWithOptions` constructs `HostServices`, `RuntimeFactory`, and `MiddlewaresFromSpec` in one place.
- `AttachDefaultCommands` shows the generated command set: eval, run, repl, modules, jsverbs, and command providers.
- `AttachRun` and `AttachEval` show where Glazed command descriptions become Cobra commands.

### What I didn't find useful

- The file does not define `HostServices` behavior in detail; that requires asset-service files not needed for this runthrough.

### What is out of date / what was wrong

- Nothing obviously wrong.

### What would need updating

- Add comments explaining that `Host` is generated-app wiring, not a Goja runtime host.
- If GOJA-053 changes runtime factory APIs, `Host` may need to expose the new parser/value-aware factory path to command providers.

---

## Resource 12: `go-go-goja/pkg/xgoja/app/module_sections.go`

### What I was researching

How xgoja selects provider modules for a runtime profile, collects public command sections, and runs provider runtime initializers.

### What I was looking for in this document in particular

- How `ModuleDescriptor` is assembled.
- Where current `ConfigSectionCapability` is invoked.
- Where `RuntimeInitializerCapability` is invoked.
- How duplicate provider package capabilities are deduped.

### Why I chose it

The runthrough needed to explain how provider flags/config sections get attached to generated commands and why post-runtime initializers are too late for `Module.New` config.

### How I found the resource itself

From `newRunCommand` and `newEvalCommand`, which call `factory.sectionsForRuntimeProfile(...)`, and from `runScriptFileWithInitializers`, which calls `initRuntimeFromSections(...)`.

### What I found useful in the document

- `selectedModuleDescriptors` resolves modules and package capabilities for a runtime profile.
- `sectionsForRuntimeProfile` passes `SectionContext{CommandName, RuntimeProfile}` to `providerutil.CollectConfigSections`.
- `initRuntimeFromSections` wraps `engine.Runtime` in a small provider-facing handle and delegates to providerutil.
- `runtimeHandle` exposes only `Runtime`, `Close`, and `AddCloser`, which clarifies why runtime initializers see less than full `engine.Runtime`.

### What I didn't find useful

- The file uses the current confusing `ConfigSectionCapability` terminology indirectly through providerutil.

### What is out of date / what was wrong

- Nothing wrong, but names are confusing for GOJA-053: “config sections” here are public command parse sections, not internal `xgoja.yaml` module config sections.

### What would need updating

- Rename or alias `ConfigSectionCapability` to `CommandLineFlagsSectionCapability` and update comments.
- Add comments explaining that `initRuntimeFromSections` is post-`Module.New`.

---

## Resource 13: `go-go-goja/pkg/xgoja/app/middlewares.go`

### What I was researching

How generated commands parse defaults, config files, environment variables, positional args, and flags into `values.Values`.

### What I was looking for in this document in particular

- Whether command values exist before runtime creation.
- Effective precedence of defaults/config/env/args/flags.
- How spec-level config and env settings control parser sources.

### Why I chose it

GOJA-053 depends on parsed command values being available before runtime creation. This file explains where those values come from and their precedence.

### How I found the resource itself

From `Host.NewHostWithOptions`, which sets `MiddlewaresFunc = MiddlewaresFromSpec(spec)`, and from generated command construction through Glazed/Cobra.

### What I found useful in the document

- `MiddlewaresFromSpec` explicitly documents effective precedence: `defaults < config < env < args < cobra flags`.
- It shows `config.enabled` and `envPrefix/appName` control whether config/env sources are enabled.
- It labels parse sources with `fields.WithSource("cobra")`, `"arguments"`, `"env"`, `"config"`, and defaults.

### What I didn't find useful

- It does not explain the full Glazed middleware call convention; the comment gives enough for this runthrough, but implementation details live in Glazed.

### What is out of date / what was wrong

- Nothing obviously wrong.

### What would need updating

- If public command sections are renamed to command-line flag sections, update comments/docs to say these same sections also feed config/env parsing.
- If GOJA-053 adds pre-runtime config merging, document that this parsed `values.Values` must be passed into runtime creation.

---

## Resource 14: `go-go-goja/pkg/xgoja/app/factory.go`

### What I was researching

How xgoja turns a selected runtime profile into a concrete engine runtime and where provider `Module.New` is called.

### What I was looking for in this document in particular

- The definition of `app.RuntimeFactory`.
- How `ModuleInstance` becomes `providerRuntimeModuleSpec`.
- Where `ModuleInstance.Config` is marshaled.
- Where `providerapi.Module.New(ModuleContext)` is called.
- Where engine builder/runtime factory is invoked.

### Why I chose it

This is the central GOJA-053 insertion point. It bridges xgoja provider metadata and engine runtime construction.

### How I found the resource itself

From `run.go`, which calls `factory.NewRuntime(ctx, profile, requireOpt)`.

### What I found useful in the document

- `RuntimeFactory.NewRuntime` resolves each runtime profile module through `providerapi.Registry.ResolveModule`.
- It constructs `providerRuntimeModuleSpec` entries and passes them to `engine.NewBuilder(...).WithModules(...)`.
- `providerRuntimeModuleSpec.RegisterRuntimeModule` marshals static `s.instance.Config` and passes it to `s.module.New(providerapi.ModuleContext{...})`.
- It passes `ctx.Owner` from `engine.RuntimeModuleContext` as `ModuleContext.RuntimeOwner`.
- It registers the returned loader with `require.Registry.RegisterNativeModule(alias, loader)`.

### What I didn't find useful

- The file does not receive parsed command values today, so it cannot show the desired GOJA-053 behavior yet.

### What is out of date / what was wrong

- Nothing wrong in current behavior.
- The type name `RuntimeFactory` is generic and can be confused with `engine.Factory`; the lifecycle doc had to explain this carefully.

### What would need updating

- Add a new value-aware runtime creation path for GOJA-053.
- Consider naming/documentation updates that clarify `app.RuntimeFactory` is a profile-to-engine-runtime adapter.
- Replace raw `json.Marshal(s.instance.Config)` with internal-section parsing/merging once GOJA-053 is implemented.

---

## Resource 15: `go-go-goja/pkg/xgoja/providerapi/registry.go`

### What I was researching

The provider-level registry: what providers register and how xgoja resolves modules/capabilities at runtime.

### What I was looking for in this document in particular

- How packages are registered.
- How modules and capabilities are stored.
- How `ResolveModule` and `ResolvePackageCapabilities` work.
- Whether this registry is distinct from Goja `require.Registry`.

### Why I chose it

The user explicitly mentioned confusion around registries. This file establishes `providerapi.Registry` as metadata/catalog registry, not the per-runtime JavaScript require registry.

### How I found the resource itself

From generated `main.go`, which creates `providerapi.NewRegistry()` and calls provider registration functions, and from `app.RuntimeFactory.NewRuntime`, which calls `ResolveModule`.

### What I found useful in the document

- `Registry.Package` creates a package entry with modules, verb sources, help sources, package capabilities, and command set providers.
- `ResolveModule` looks up provider modules by package ID and module name.
- `ResolvePackageCapabilities` returns package capabilities used for public sections and runtime initializers.
- `addModule` requires `module.New != nil`, confirming provider modules must provide a module factory.

### What I didn't find useful

- It does not explain when provider packages are registered; generated `main.go` provides that context.

### What is out of date / what was wrong

- Nothing obviously wrong.

### What would need updating

- Add comments emphasizing this is not Goja's `require.Registry`.
- If GOJA-053 adds new capabilities, this file may not need structural changes, but comments/tests should include the new capability kind.

---

## Resource 16: `go-go-goja/pkg/xgoja/providerapi/module.go`

### What I was researching

The provider module factory API and `ModuleContext` passed to `Module.New`.

### What I was looking for in this document in particular

- Where `Module.New` is defined.
- What `ModuleContext` contains.
- Whether `ModuleContext.Context` is long-lived or setup-only.
- How static module config reaches providers.

### Why I chose it

The lifecycle runthrough needed to explain `module.New` and distinguish `ModuleContext` from `RuntimeModuleContext`.

### How I found the resource itself

By searching for `type Module interface`, `ModuleContext`, and `ModuleFactory`, then following `app/factory.go` where `s.module.New(...)` is called.

### What I found useful in the document

- `ModuleFactory` is `func(ModuleContext) (require.ModuleLoader, error)`.
- `Module.New` is a struct field, not a method.
- `ModuleContext.Config` is `json.RawMessage`, which explains why xgoja currently marshals `ModuleInstance.Config` before provider setup.
- `ModuleContext.RuntimeOwner` gives providers a narrow owner-thread scheduling handle without exposing full engine internals.

### What I didn't find useful

- The fields have no explanatory comments, so their intended lifetime and meaning must be inferred from call sites.

### What is out of date / what was wrong

- `Context` is too generic and misleading. In current flow it is the startup/setup context from `engine.RuntimeModuleContext.Context`, not necessarily the runtime lifetime context.

### What would need updating

- Rename `ModuleContext.Context` to `StartupContext` or `SetupContext`, or at least document it.
- Add comments for `Config`, `Host`, and `RuntimeOwner`.
- If GOJA-053 eventually exposes `ConfigValues *values.SectionValues`, document the relationship between typed config values and `Config json.RawMessage`.

---

## Resource 17: `go-go-goja/pkg/xgoja/providerapi/capabilities.go`

### What I was researching

Provider extension points: public command sections, runtime initializers, runtime handles, and module descriptors.

### What I was looking for in this document in particular

- `SectionContext` shape.
- `ModuleDescriptor` shape.
- Current `ConfigSectionCapability` name and semantics.
- `RuntimeInitializerCapability` timing and inputs.
- `RuntimeHandle` and `RuntimeCloserRegistry` purpose.

### Why I chose it

The runthrough needed to explain why public flags/config sections and post-runtime initializers are separate from `Module.New` config.

### How I found the resource itself

From `providerutil.CollectConfigSections` and `providerutil.InitRuntimeFromSections`, which type-assert capabilities from provider packages.

### What I found useful in the document

- `SectionContext` records command name, command provider ID, runtime profile, package ID, and module ID.
- `ModuleDescriptor` ties selected module info to package capabilities.
- `RuntimeInitializerCapability` receives parsed `*values.Values` and a provider-facing `RuntimeHandle`.
- `RuntimeCloserRegistry` lets provider initializers register cleanup hooks.

### What I didn't find useful

- The name `ConfigSectionCapability` is confusing in the context of GOJA-053 because it is public command parse schema, not internal module config schema.

### What is out of date / what was wrong

- The comment says “Glazed sections that can be attached to built-in commands or package-owned command providers,” which is accurate, but the type name still suggests generic config.

### What would need updating

- Rename `ConfigSectionCapability` to `CommandLineFlagsSectionCapability` or another public-command-section name.
- Add a new capability for internal module config schema, e.g. `ModuleConfigSectionCapability`.
- Document that runtime initializers run after `Module.New` in current app command flows.

---

## Resource 18: `go-go-goja/pkg/xgoja/providerapi/commands.go`

### What I was researching

Provider-owned command set APIs and how command providers can create runtimes from generated app context.

### What I was looking for in this document in particular

- The provider-facing `RuntimeFactory` interface.
- `CommandSetContext` fields.
- Whether command providers receive selected modules and parsed config.
- How provider command sets fit beside built-in run/eval/repl commands.

### Why I chose it

The generated app can attach provider-owned commands, and those commands can create runtimes too. The lifecycle document needed at least enough context to avoid implying built-in `run` is the only runtime creation path.

### How I found the resource itself

From `Host.AttachDefaultCommands`, which calls `AttachCommandProviders`, and from providerapi package files related to commands.

### What I found useful in the document

- `providerapi.RuntimeFactory` is a narrow interface exposing `NewRuntime(ctx, profile, opts...)`.
- `CommandSetContext` includes `RuntimeFactory`, `SelectedModules`, `Providers`, `Host`, and raw command-provider `Config`.
- This confirms provider command sets can create xgoja runtimes without directly depending on `app.RuntimeFactory` concrete type.

### What I didn't find useful

- It does not show how command providers are attached; that is in app command provider code not needed deeply for the main runthrough.

### What is out of date / what was wrong

- Nothing obviously wrong.
- If GOJA-053 introduces a value-aware runtime factory method, this interface may become insufficient for provider command sets that need the same pre-runtime config merge behavior.

### What would need updating

- Consider extending `providerapi.RuntimeFactory` with a parsed-values-aware method or a separate interface after GOJA-053.
- Document whether provider command sets are responsible for running runtime initializer capabilities or whether the host should provide a helper.

---

## Resource 19: `go-go-goja/pkg/xgoja/providerutil/sections.go`

### What I was researching

Shared helper logic for collecting provider sections and running runtime initializers from selected module descriptors.

### What I was looking for in this document in particular

- How public sections are deduped.
- How package-level capabilities avoid being applied multiple times when multiple modules from one package are selected.
- How runtime initializers are invoked.

### Why I chose it

The lifecycle runthrough needed to explain how provider capabilities are applied without duplicating per module selection.

### How I found the resource itself

From `app/module_sections.go`, which delegates to providerutil.

### What I found useful in the document

- `CollectConfigSections` enriches `SectionContext` with `PackageID` and `ModuleID` before invoking providers.
- It dedupes package capabilities by `packageID + capabilityID`.
- `AppendUniqueSections` rejects duplicate section slugs.
- `InitRuntimeFromSections` uses the same package/capability dedupe pattern.

### What I didn't find useful

- The helper names still use `ConfigSections`, reinforcing the confusing terminology.

### What is out of date / what was wrong

- Nothing functionally wrong.
- For GOJA-053 terminology, “config section” is too ambiguous.

### What would need updating

- Rename helper comments and possibly functions if the capability is renamed to `CommandLineFlagsSectionCapability`.
- Add separate helpers for internal module config sections and SectionValues mapping once GOJA-053 is implemented.

---

## Resource 20: `go-go-goja/engine/runtime_modules.go`

### What I was researching

The low-level runtime module registration interface and the `RuntimeModuleContext` type.

### What I was looking for in this document in particular

- What `RuntimeModuleSpec` is.
- What fields are in `RuntimeModuleContext`.
- Whether `RuntimeModuleContext` is the same as provider `ModuleContext`.

### Why I chose it

The user explicitly asked about confusion between module contexts and runtime module contexts. This file defines the engine-side concept.

### How I found the resource itself

From `app/factory.go`, where `providerRuntimeModuleSpec` implements `RegisterRuntimeModule(ctx *engine.RuntimeModuleContext, reg *require.Registry)`.

### What I found useful in the document

- `RuntimeModuleSpec` registers one or more require modules for a concrete runtime instance.
- `RuntimeModuleContext` includes concrete runtime objects: `VM`, `Loop`, `Owner`, `AddCloser`, and `Values`.
- The comment says module specs receive these before the require registry is enabled.

### What I didn't find useful

- It does not show how `RuntimeModuleContext` is constructed; that is in `engine/factory.go`.

### What is out of date / what was wrong

- Nothing obviously wrong.

### What would need updating

- Add an explicit comment that provider-facing `providerapi.ModuleContext` is a narrowed adapter derived from this engine context, not the same object.

---

## Resource 21: `go-go-goja/engine/options.go`

### What I was researching

Startup and lifetime contexts used during runtime construction.

### What I was looking for in this document in particular

- What `WithStartupContext` means.
- What `WithLifetimeContext` means.
- Default context behavior.
- Whether `ModuleContext.Context` should be treated as startup or lifetime context.

### Why I chose it

The user asked whether `ModuleContext.Context` is appropriate. This file defines the upstream contexts that eventually flow into `RuntimeModuleContext.Context` and runtime lifetime services.

### How I found the resource itself

From `app.RuntimeFactory.NewRuntime`, which calls:

```go
runtimeFactory.NewRuntime(engine.WithStartupContext(ctx), engine.WithLifetimeContext(ctx))
```

### What I found useful in the document

- `WithStartupContext` is explicitly for constructing a runtime and running runtime initializers.
- `WithLifetimeContext` is the parent for runtime-owned resources.
- Defaults are `context.Background()` for both.

### What I didn't find useful

- It does not mention provider-facing `ModuleContext.Context`, so the connection must be followed through `engine/factory.go` and `app/factory.go`.

### What is out of date / what was wrong

- Nothing wrong.

### What would need updating

- If `ModuleContext.Context` remains, document it as setup/startup context using the same terminology as this file.

---

## Resource 22: `go-go-goja/engine/factory.go`

### What I was researching

How the low-level engine constructs a concrete runtime and when runtimebridge, RuntimeModuleContext, require registry, and runtime initializers are set up.

### What I was looking for in this document in particular

- `engine.FactoryBuilder` and `engine.Factory` roles.
- `Factory.NewRuntime` sequence.
- Where `runtimebridge.Store` is called.
- Where `RuntimeModuleContext` is created.
- Where module specs register native modules.
- Where `require(...)` is enabled.

### Why I chose it

This is the most important file for understanding the interplay between engine factory, VM, event loop, owner, runtimebridge, require registry, and module specs.

### How I found the resource itself

From `app.RuntimeFactory.NewRuntime`, which calls `engine.NewBuilder(...).WithModules(...).Build().NewRuntime(...)`.

### What I found useful in the document

- `FactoryBuilder.Build` freezes modules/runtime initializers into an immutable `Factory`.
- `Factory.NewRuntime` creates `goja.Runtime`, `eventloop`, `RuntimeOwner`, runtime lifetime context, and `engine.Runtime`.
- It stores `RuntimeServices` in `runtimebridge` before module registration.
- It creates `require.NewRegistry` and `RuntimeModuleContext` before invoking each module spec.
- It enables `require` only after module specs have registered native module loaders.
- It distinguishes engine-level runtime initializers from xgoja provider runtime initializer capabilities.

### What I didn't find useful

- The function is necessarily dense and not easy for an intern to read without a sequence diagram.

### What is out of date / what was wrong

- Nothing obviously wrong.
- The name `Factory` is generic and easily confused with `app.RuntimeFactory`.

### What would need updating

- Add comments near `runtimebridge.Store` explaining why it happens before module registration.
- Add docs distinguishing `engine.Factory` from `app.RuntimeFactory`.
- Add a short sequence diagram to engine docs if available.

---

## Resource 23: `go-go-goja/engine/runtime.go`

### What I was researching

Runtime lifecycle, runtime-owned context, closers, runtimebridge cleanup, owner shutdown, and event loop shutdown.

### What I was looking for in this document in particular

- What `engine.Runtime` stores.
- What `Runtime.Context()` means.
- How closers are registered and executed.
- What `Runtime.Close` does.
- Where runtimebridge entries are deleted.

### Why I chose it

A full script execution runthrough must include shutdown, especially because runtime initializers can register resources that need cleanup.

### How I found the resource itself

From `run.go`, which defers `rt.Close(ctx)`, and from `engine/factory.go`, which constructs `engine.Runtime`.

### What I found useful in the document

- `Runtime.Context()` returns the runtime-owned lifecycle context, not the startup context.
- `AddCloser` registers cleanup hooks.
- `Close` cancels runtime context, waits for owner idle, runs closers in reverse order, deletes runtimebridge services, shuts down owner, and stops event loop.
- `waitOwnerIdleOrInterrupt` can interrupt active JavaScript during close.

### What I didn't find useful

- It does not explain how runtime resources are originally registered; that is in `engine/factory.go` and provider initializer code.

### What is out of date / what was wrong

- Nothing obviously wrong.

### What would need updating

- Add comments tying `Runtime.Context()` to `runtimebridge.RuntimeServices.LifetimeContext`.
- In xgoja docs, advise provider initializers to use `RuntimeCloserRegistry`/`AddCloser` for resources they start.

---

## Resource 24: `go-go-goja/pkg/runtimebridge/runtimebridge.go`

### What I was researching

The purpose of runtimebridge: VM-indexed runtime services, lifetime context lookup, owner scheduling helpers, and current owner-call context propagation.

### What I was looking for in this document in particular

- What `RuntimeServices` contains.
- How services are stored and looked up by `*goja.Runtime`.
- What `CurrentOwnerContext` means.
- How custom contexts are linked to runtime lifetime.
- Whether runtimebridge is a module registry or a service bridge.

### Why I chose it

The user named runtimebridge and runtimeservices as a source of confusion. This file defines both and clarifies they are not another module/config registry.

### How I found the resource itself

From `engine/factory.go`, which calls `runtimebridge.Store(vm, runtimebridge.RuntimeServices{...})`, and from runtimeowner code that calls `runtimebridge.WithCallContext`.

### What I found useful in the document

- `RuntimeServices` contains `LifetimeContext`, `Loop`, and `Owner`.
- `Store`, `Lookup`, and `Delete` maintain a VM-to-services map.
- `Lifetime()` falls back to `context.Background()`.
- `CurrentOwnerContext(vm)` returns the current owner-call context or falls back to runtime lifetime context.
- `CallWithCustomContext` and `PostWithCustomContext` link caller context cancellation with runtime lifetime cancellation.

### What I didn't find useful

- It does not show where services are stored; that is in `engine/factory.go`.
- It does not show how the owner-call context stack is populated; that is in `runtimeowner/runner.go`.

### What is out of date / what was wrong

- Nothing obviously wrong.

### What would need updating

- Add a short package-level comment: “runtimebridge is a VM service lookup and context propagation bridge, not a module registry.”
- Add source comments or docs linking `Store` to `engine.Factory.NewRuntime` and `Delete` to `Runtime.Close`.

---

## Resource 25: `go-go-goja/pkg/runtimeowner/runner.go`

### What I was researching

How runtime owner scheduling serializes access to the Goja VM and propagates contexts into runtimebridge.

### What I was looking for in this document in particular

- How `Call` schedules work on the event loop.
- How `Post` schedules asynchronous work.
- Where `runtimebridge.WithCallContext` is invoked.
- How cancellation and active-call tracking work.

### Why I chose it

The runthrough needed to explain why script execution uses `rt.Owner.Call` and how native module code can later recover the current command context.

### How I found the resource itself

From `run.go`, which calls `rt.Owner.Call(...)`, and from `engine/factory.go`, which creates `runtimeowner.NewRuntimeOwner(vm, loop, ...)`.

### What I found useful in the document

- `Call` normalizes context, schedules work on the event loop, waits for result, and handles cancellation.
- `Post` schedules fire-and-forget owner work.
- `invoke` and `invokePost` wrap execution in `runtimebridge.WithCallContext` / `WithCallContextVoid`.
- Active-call tracking supports `WaitIdle`, which matters during runtime close.

### What I didn't find useful

- The scheduling abstraction itself is not defined in this file excerpt; understanding event-loop details may require additional engine/runtimeowner files if debugging races.

### What is out of date / what was wrong

- Nothing obviously wrong.

### What would need updating

- Add a short high-level comment near `Call` explaining that it is the normal way to execute JS-touching code from outside the owner thread.
- In lifecycle docs, point users from `rt.Owner.Call` to `runtimebridge.CurrentOwnerContext` for native callbacks.

---

## Resource 26: `geppetto/pkg/js/modules/geppetto/provider/provider.go`

### What I was researching

A concrete provider implementation of `providerapi.Module.New` that uses `ModuleContext.Config`, `ModuleContext.Host`, and `ModuleContext.Context`.

### What I was looking for in this document in particular

- A real `providerapi.Module{New: ...}` example.
- How provider config is decoded.
- Whether `ModuleContext.Context` is used in production provider code.
- Why pre-runtime config matters for GOJA-053.

### Why I chose it

The runthrough and preceding questions needed a real provider example. Geppetto is the motivating provider for GOJA-053.

### How I found the resource itself

By searching for `providerapi.Module{` and `New: func(ctx providerapi.ModuleContext)` in provider packages.

### What I found useful in the document

- Geppetto decodes `ctx.Config` immediately in `Module.New`.
- It calls host services with `ctx.Context`, e.g. `host.GeppettoOptions(ctx.Context, cfg)`.
- It applies registry and storage options before returning `geppettomodule.NewLoader(opts)`.
- This confirms values needed by Geppetto provider setup must be available before `Module.New` returns.

### What I didn't find useful

- The current Geppetto config shape includes fields that later design feedback wants to simplify, so it is useful as motivation but not necessarily the final desired config surface.

### What is out of date / what was wrong

- For current GOJA-053 design, broad allow-gate and nested turn-storage config are likely stale and should be simplified toward explicit `turns-dsn` / `turns-db` style config.

### What would need updating

- Update Geppetto provider config once GOJA-053 implementation begins.
- If `ModuleContext.Context` is renamed to `StartupContext`, update this file.
- Add tests proving command/config/env-derived settings reach Geppetto before `Module.New` options are finalized.

---

## Resource 27: `go-go-goja/pkg/xgoja/doc/01-runtime-overview.md`

### What I was researching

Whether existing xgoja docs already explain the generated runtime lifecycle at a high level.

### What I was looking for in this document in particular

- A concise statement of what a generated xgoja binary is.
- Whether provider registration and runtime profile execution were already documented.
- Whether existing docs could replace the new runthrough.

### Why I chose it

Search results surfaced this document while looking for `xgoja.yaml` references. It looked like the closest existing runtime overview.

### How I found the resource itself

Repository search for `xgoja.yaml` and runtime overview references:

```bash
rg -n "xgoja.yaml|runtime profile|generated xgoja binary" go-go-goja/pkg/xgoja go-go-goja/cmd/xgoja/doc -S
```

### What I found useful in the document

- It states the high-level concept: a generated xgoja binary is built from `xgoja.yaml`, imports provider packages, registers native Goja modules, embeds a normalized runtime specification, and creates commands.
- It helped confirm the top-level framing for the runthrough.

### What I didn't find useful

- It was too high-level for the user's question.
- It does not detail the factory/context/registry/runtimebridge interplay.

### What is out of date / what was wrong

- Nothing necessarily wrong; it is just not deep enough for debugging GOJA-053.

### What would need updating

- Link to or incorporate a shortened version of the new lifecycle runthrough.
- Add terminology definitions for provider registry, require registry, app runtime factory, engine factory, and runtimebridge.

---

## Resource 28: `go-go-goja/cmd/xgoja/doc/02-user-guide.md`, `03-tutorial-using-xgoja-yaml.md`, and `06-buildspec-reference.md`

### What I was researching

Existing user-facing documentation around `xgoja.yaml`, generated commands, config files, and buildspec fields.

### What I was looking for in this document in particular

- Existing explanations of `xgoja.yaml` structure.
- Existing command config-file examples.
- Whether lifecycle details already existed in user docs.

### Why I chose it

Search results for `xgoja.yaml` showed these docs as existing user-facing references. They were relevant context for what the new internal runthrough should not duplicate too heavily.

### How I found the resource itself

Repository search for `xgoja.yaml`, `config.yaml`, and generated command documentation in `cmd/xgoja/doc`.

### What I found useful in the document

- They provide user-level examples of `xgoja.yaml` and generated command config files.
- They establish that config files use Glazed section-map format for generated command values.
- They are useful references for command-facing behavior.

### What I didn't find useful

- They do not explain internal runtime creation order.
- They do not explain `ModuleContext`, `RuntimeModuleContext`, `runtimebridge`, or owner scheduling.
- They do not make the GOJA-053 timing gap visible.

### What is out of date / what was wrong

- Not necessarily wrong; just scoped to user operation rather than internal architecture.
- If `ConfigSectionCapability` is renamed and GOJA-053 changes runtime config behavior, examples and terminology may need revision.

### What would need updating

- Add a concise “internals/lifecycle” link or appendix to point maintainers to the deeper runthrough.
- Update examples if command-derived config can patch module config before `Module.New` after GOJA-053.

---

## Overall findings

### Most useful resources

- `app/factory.go`: identifies the exact `Module.New` and config timing point.
- `engine/factory.go`: explains concrete runtime construction and `RuntimeModuleContext`.
- `app/run.go`: shows command values exist before runtime creation but only feed initializers after runtime creation today.
- `runtimebridge/runtimebridge.go` plus `runtimeowner/runner.go`: explain context propagation and owner-thread scheduling.
- `cmd/xgoja/internal/generate/templates/main.go.tmpl`: clarifies generated target startup order.

### Most confusing or update-worthy resources

- `providerapi/capabilities.go`: `ConfigSectionCapability` name is ambiguous and should become a public command/flags/config/env section capability.
- `providerapi/module.go`: `ModuleContext.Context` should be documented or renamed to `StartupContext` / `SetupContext`.
- `engine/factory.go`: dense but central; needs a small sequence comment or linked doc.
- Existing xgoja user docs: good user-facing material, but not enough for internal lifecycle debugging.

### Resources that were useful mainly as context, not primary evidence

- xgoja user docs and runtime overview.
- Geppetto provider implementation: essential as motivation, but its current config surface is likely not the final desired GOJA-053 shape.

### Suggested documentation updates

1. Add comments to `providerapi.ModuleContext` fields.
2. Rename or alias `ConfigSectionCapability` to a less ambiguous public-command-section name.
3. Add a short lifecycle diagram to xgoja docs or package docs.
4. Document that xgoja provider `RuntimeInitializerCapability` runs after `Module.New` today.
5. Add a debug-oriented note in `xgoja build --keep-work` docs: inspect generated `main.go` and `xgoja.gen.json`.
