---
Title: Research Logbook
Ticket: XGOJA-017
Status: active
Topics:
    - xgoja
    - glazed
    - configuration
    - middleware
    - design
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/xgoja/internal/buildspec/spec.go
      Note: YAML spec struct; 34 resources catalogued in logbook
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Research Logbook

## Goal

Keep a running record of every resource consulted during the XGOJA-017 research phase so that future readers (including the original researcher returning weeks later) can:

- Quickly re-locate a source that proved useful.
- Know which sources were dead-ends or misleading.
- Spot stale documentation before trusting it.
- Understand the chain of reasoning that led to each design decision.

This logbook covers the research done on **2026-06-02** for the design of env-prefix / app-name / glazed source middleware support in xgoja generated binaries.

---

## How this logbook is organized

Each entry follows this template:

- **What I was researching** — the high-level question.
- **What I was looking for in this document** — the specific answer.
- **Why I chose it** — why this source was a promising place to look.
- **How I found the resource** — grep / ls / prior knowledge / cross-reference.
- **What I found useful** — concrete facts, APIs, line numbers.
- **What I didn't find useful** — irrelevant or redundant content.
- **What is out of date / wrong** — stale docs, misleading comments, dead code.
- **What would need updating** — specific edits or follow-up work.

---

## xgoja Internal Resources

### 1. `cmd/xgoja/internal/buildspec/spec.go`

- **What I was researching:** What fields exist in `xgoja.yaml` today and where new fields would fit.
- **What I was looking for:** The Go struct tags, field names, and any existing "settings" or "metadata" section.
- **Why I chose it:** This is the canonical schema for the build spec. Any new YAML field must map to a field in `Spec`.
- **How I found it:** `find go-go-goja/cmd/xgoja/internal/buildspec -name "*.go"` after locating the package from the build command.
- **What I found useful:** The `Spec` struct is flat and well-organized: `Name`, `Go`, `Target`, `Packages`, `Runtimes`, `Commands`, `CommandProviders`, `JSVerbs`, `Help`, `Assets`. Each field has clear JSON/YAML tags. The absence of any `appName`, `envPrefix`, `config`, or `profiles` fields confirmed this is a green-field addition.
- **What I didn't find useful:** Nothing here is irrelevant; every field is necessary for understanding the current shape.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** This file is the **first implementation target** in Phase 1. Add `AppName`, `EnvPrefix`, `Config`, `Profiles`, `Middlewares` fields.

### 2. `cmd/xgoja/cmd_build.go`

- **What I was researching:** How the `xgoja build` CLI command orchestrates code generation.
- **What I was looking for:** Whether the build command itself needs to change (e.g., new flags) or whether it simply passes the spec through.
- **Why I chose it:** The build command is the entry point; understanding its control flow tells us whether we need CLI changes.
- **How I found it:** Listed `cmd/xgoja/` directory.
- **What I found useful:** The build command is thin: it calls `buildspec.LoadFile`, `generate.WriteAll`, `buildexec.GoModTidy`, and `buildexec.GoBuild`. It does not inspect the spec contents beyond `spec.Go.Tags` and `spec.Go.LDFlags`. This means **no CLI changes are required** for the new fields; the generator just needs to read them from the spec.
- **What I didn't find useful:** The `defaultXGojaModuleVersion` function and `debug.ReadBuildInfo` logic are unrelated to this feature.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Likely nothing, unless we decide to add `--config` or `--profile` flags to `xgoja build` itself (not planned).

### 3. `cmd/xgoja/root.go`

- **What I was researching:** How the xgoja CLI itself is wired (not the generated binary).
- **What I was looking for:** Whether xgoja's own CLI uses env/config middlewares, which would serve as a live example.
- **Why I chose it:** If the xgoja tool already uses Glazed's env/config path, we can copy its pattern.
- **How I found it:** Listed `cmd/xgoja/` directory.
- **What I found useful:** The xgoja CLI uses `cli.BuildCobraCommand` with `CobraCommandDefaultMiddlewares`. It does **not** use `AppName` or `ConfigPlanBuilder`. This is expected because `xgoja build` is a build tool, not a long-running application that needs config files. However, it means there is **no internal example** of advanced middleware use within xgoja itself.
- **What I didn't find useful:** The `helpSystem` and `logging` wiring are standard Glazed boilerplate.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Not applicable.

### 4. `cmd/xgoja/internal/generate/templates/main.go.tmpl`

- **What I was researching:** What code is emitted for the generated binary's `main.go`.
- **What I was looking for:** Where `app.NewRootCommand` is called, how `Host` is constructed, and where middleware configuration would be injected.
- **Why I chose it:** The template is the **code generation surface**. Any new feature that changes runtime behavior must emit different Go code here.
- **How I found it:** `find cmd/xgoja/internal/generate -name "*.tmpl"`.
- **What I found useful:**
  - Three target branches: `adapter`, `cobra`, `xgoja` (default).
  - For `xgoja`, it calls `app.NewRootCommand(app.Options{...})`.
  - For `cobra`, it calls `app.NewHostWithOptions` then `host.AttachDefaultCommands(root)`.
  - The `HostConstruction` and `RootConstruction` strings are pre-built in Go code and injected into the template.
  - `embeddedSpecJSON` is embedded as a raw string constant.
- **What I didn't find useful:** The `embed` directives and `decodeSpec` helper are standard and not relevant to middleware configuration.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** This is the **primary template change** in Phase 3. Must conditionally emit a `middlewaresFunc` closure and pass it into `app.NewRootCommand` / `app.NewHostWithOptions`.

### 5. `cmd/xgoja/internal/generate/generate.go`

- **What I was researching:** How the generator assembles the build workspace.
- **What I was looking for:** Where `main.go` is rendered and whether the generator needs new imports for generated code.
- **Why I chose it:** To understand the full generation pipeline before proposing template changes.
- **How I found it:** Same directory as `templates.go`.
- **What I found useful:** `WriteAll` creates `go.mod`, `main.go`, `xgoja.gen.json`, and copies embedded assets. `main.go` is produced by `RenderMain(spec)`. The generator does not need to know about middlewares at this level; it just delegates to the template.
- **What I didn't find useful:** The `copyEmbeddedJSVerbs`, `copyEmbeddedHelpSources`, and `copyEmbeddedAssets` helpers are unrelated.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** If the template requires new imports (e.g., `glazed/pkg/config`), `RenderMain` or the template data builder must ensure those imports are present in `main.go.tmpl`.

### 6. `cmd/xgoja/internal/generate/templates.go`

- **What I was researching:** How template data is constructed from the spec.
- **What I was looking for:** The `mainTemplateData` struct and `mainTemplateDataFromSpec` function to see where new fields would be added.
- **Why I chose it:** The template is dumb; all logic lives in the Go code that builds `mainTemplateData`. Understanding this function is essential for proposing template changes.
- **How I found it:** Same directory.
- **What I found useful:**
  - `mainTemplateData` contains `SpecJSON`, `HasEmbedded*`, `TargetKind`, `ProviderImports`, `HostConstruction`, `RootConstruction`.
  - `mainTemplateDataFromSpec` computes `hasEmbedded`, `aliases`, `providers`, and pre-builds `HostConstruction` / `RootConstruction` strings.
- **What I didn't find useful:** Nothing.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Add `HasMiddlewares`, `AppName`, `EnvPrefix`, `ConfigPlanBuilder` fields to `mainTemplateData`. Modify `mainTemplateDataFromSpec` to populate them from the spec.

### 7. `cmd/xgoja/doc/03-tutorial-using-xgoja-yaml.md`

- **What I was researching:** Current user-facing documentation for `xgoja.yaml`.
- **What I was looking for:** The YAML schema as presented to users, common usage patterns, and any mention of env/config.
- **Why I chose it:** User docs reveal the intended mental model. New features should feel natural in this context.
- **How I found it:** `find cmd/xgoja/doc -name "*.md" | sort`.
- **What I found useful:**
  - Tutorial walks through `name`, `target`, `packages`, `runtimes`, `commands`, `jsverbs`, `assets`, `help`.
  - The `modules` section uses `package`, `name`, `as`, `config` — shows that nested config objects are already idiomatic in xgoja.
  - No mention of env, config files, or profiles. Confirms this is entirely new territory.
- **What I didn't find useful:** The troubleshooting table at the end is helpful for users but not for design.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** A new tutorial document (e.g., `10-tutorial-config-env-profiles.md`) should be added. This existing doc may need a cross-reference once the feature ships.

### 8. `cmd/xgoja/doc/06-buildspec-reference.md`

- **What I was researching:** The authoritative reference for all `xgoja.yaml` fields.
- **What I was looking for:** Whether there are hidden or advanced fields not obvious from the Go struct.
- **Why I chose it:** This is the user-facing spec reference. Any schema change must be documented here.
- **How I found it:** Same directory listing.
- **What I found useful:**
  - Confirms the common shape: `name`, `go`, `target`, `packages`, `runtimes`, `commands`, `assets`, `help`.
  - Mentions that `include` and `exclude` fields in `assets:` are "rejected until the generator enforces filtering." This shows the maintainers are cautious about adding schema fields without generator enforcement. We should follow the same pattern: add validation before adding behavior.
- **What I didn't find useful:** The static HTTP serving example is a cool feature but not relevant to middlewares.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Add new top-level fields (`appName`, `envPrefix`, `config`, `profiles`, `middlewares`) with descriptions, defaults, and examples.

### 9. `pkg/xgoja/app/host.go`

- **What I was researching:** How the generated binary's runtime host is constructed and how commands are attached.
- **What I was looking for:** The `Host` struct definition, `AttachDefaultCommands`, and the call sites for `buildGlazedCobraCommand`.
- **Why I chose it:** The `Host` is the runtime coordinator. If middleware configuration is to be propagated to all commands, the `Host` must carry it.
- **How I found it:** `find pkg/xgoja/app -name "*.go" | sort`.
- **What I found useful:**
  - `Host` struct contains `Providers`, `Spec`, `Factory`, `Embedded*`, `Services`, `Out`.
  - `AttachDefaultCommands` is the central mount point: it calls `AttachEval`, `AttachRun`, `AttachRepl`, `AttachModules`, `AttachVerbs`, `AttachCommandProviders`.
  - Every `Attach*` method calls `buildGlazedCobraCommand` (or `newVerbsCommand` for JS verbs, which does not use Glazed command building).
- **What I didn't find useful:** `HostServices` and `AssetStore` are unrelated.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Add `MiddlewaresFunc` to `Host` and `HostOptions`. Update every `Attach*` method to pass `h.MiddlewaresFunc` to the command builder.

### 10. `pkg/xgoja/app/root.go`

- **What I was researching:** How `NewRootCommand` decodes the embedded spec and constructs the `Host`.
- **What I was looking for:** The `Options` struct and whether it already has room for middleware configuration.
- **Why I chose it:** `NewRootCommand` is called from generated `main.go`. It's the runtime entry point.
- **How I found it:** Same directory.
- **What I found useful:**
  - `Options` struct: `Providers`, `SpecJSON`, `Out`, `EmbeddedJSVerbs`, `EmbeddedHelp`, `EmbeddedAssets`.
  - `NewRootCommand` unmarshals JSON spec, creates `Host`, creates root `cobra.Command`, calls `host.AttachDefaultCommands`.
  - `buildVerbCommands` (used by `newVerbsCommand`) also calls `cli.BuildCobraCommand` with a hardcoded `CobraCommandDefaultMiddlewares` for mounted verb subcommands.
- **What I didn't find useful:** The `evalCommand`, `modulesCommand`, and verb scanning logic are implementation details not relevant to middleware wiring.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Add `MiddlewaresFunc` to `Options`. Pass it into `NewHostWithOptions` and into `buildVerbCommands`.

### 11. `pkg/xgoja/app/spec.go`

- **What I was researching:** The JSON runtime spec struct.
- **What I was looking for:** Whether it already contains fields that are missing from the YAML spec (or vice versa) and how new fields would be added.
- **Why I chose it:** The spec is embedded as JSON in the generated binary. New YAML fields must have JSON equivalents.
- **How I found it:** Same directory.
- **What I found useful:** The JSON struct mirrors the YAML struct but omits build-time-only fields (`Go`, `BaseDir`, `PackageSpec.Import`, etc.). It includes `Name`, `Target`, `Packages`, `Runtimes`, `Commands`, `CommandProviders`, `JSVerbs`, `Help`, `Assets`. The runtime spec does not need `Go` or `BaseDir`.
- **What I didn't find useful:** Nothing.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Add `AppName`, `EnvPrefix`, `Config`, `Profiles`, `Middlewares` fields, keeping runtime-relevant fields only.

### 12. `pkg/xgoja/app/glazed.go`

- **What I was researching:** The exact chokepoint where Glazed commands are converted to Cobra commands.
- **What I was looking for:** The `buildGlazedCobraCommand` function and whether it could be parameterized.
- **Why I chose it:** This is the **single most important file** for middleware support. Every Glazed command in the generated binary passes through here.
- **How I found it:** Same directory.
- **What I found useful:**
  - `buildGlazedCobraCommand` is a zero-parameter function that hardcodes:
    ```go
    cli.WithParserConfig(cli.CobraParserConfig{
        ShortHelpSections: []string{schema.DefaultSlug},
        MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
    })
    ```
  - This confirms the feature gap: there is no mechanism to inject a custom middleware factory.
- **What I didn't find useful:** `commandErrorStub` is a helper for error handling.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Replace `buildGlazedCobraCommand` with `buildGlazedCobraCommandWithMiddlewares(command cmds.Command, middlewaresFunc cli.CobraMiddlewaresFunc)`. Fall back to `CobraCommandDefaultMiddlewares` when nil.

### 13. `pkg/xgoja/app/framework.go`

- **What I was researching:** How the root command's framework (logging, help) is installed.
- **What I was looking for:** Whether `appName` is already configurable and how it's derived.
- **Why I chose it:** The framework sets up `logging.AddLoggingSectionToRootCommand(root, appName)`, which uses the app name for log configuration. If we add `appName` to the spec, this is where it would be consumed.
- **How I found it:** Same directory.
- **What I found useful:**
  - `installRootFramework` derives `appName` from `spec.Name` if `spec != nil`.
  - If we add `spec.AppName`, the logic should prefer it over `spec.Name`.
- **What I didn't find useful:** Help source loading (`loadConfiguredHelpSources`) is detailed but not relevant to middlewares.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Change `appName := "xgoja"` fallback logic to:
  ```go
  appName := "xgoja"
  if spec != nil {
      if spec.AppName != "" {
          appName = spec.AppName
      } else if spec.Name != "" {
          appName = spec.Name
      }
  }
  ```

### 14. `examples/xgoja/01-core-provider/xgoja.yaml`

- **What I was researching:** A minimal real-world `xgoja.yaml` for reference.
- **What I was looking for:** Common field patterns and where new fields would naturally fit.
- **Why I chose it:** It's the simplest example; new features should not complicate this.
- **How I found it:** `find examples/xgoja -name "xgoja.yaml" | head -10`.
- **What I found useful:**
  - Shows `name`, `target`, `packages`, `runtimes`, `commands` at top level.
  - Confirms that adding `appName` and `envPrefix` at the same level feels natural.
- **What I didn't find useful:** Nothing.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Not applicable (this is a stable example). A new example demonstrating env/config should be created separately.

### 15. `examples/xgoja/05-command-provider/xgoja.yaml`

- **What I was researching:** A more complex spec that uses `commandProviders`.
- **What I was looking for:** How provider commands are mounted and whether they would also need middleware support.
- **Why I chose it:** Command-provider commands are built from Go code, not JavaScript. They might have different middleware needs.
- **How I found it:** Same directory listing.
- **What I found useful:**
  - `commandProviders` list references provider packages by ID.
  - The actual command building for provider commands happens inside `AttachCommandProviders` in `host.go`, which also calls `buildGlazedCobraCommand`.
  - This means **all command types** (eval, run, repl, provider commands, verb commands) pass through the same chokepoint. A single middleware factory change covers everything.
- **What I didn't find useful:** Nothing.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Not applicable.

---

## Glazed Internal Resources

### 16. `glazed/pkg/cli/cobra-parser.go`

- **What I was researching:** How Glazed's Cobra parser builds middleware chains and how `AppName` / `ConfigPlanBuilder` are used.
- **What I was looking for:** The exact code path that constructs `FromEnv` and `FromConfigPlanBuilder` middlewares, and the precedence order.
- **Why I chose it:** This is the core Glazed file for CLI parsing. Understanding it is essential for designing the generated binary's behavior.
- **How I found it:** `grep -r "AppName" glazed/pkg/cli/ --include="*.go"` led here.
- **What I found useful:**
  - `CobraParserConfig` struct contains `MiddlewaresFunc`, `ShortHelpSections`, `SkipCommandSettingsSection`, `EnableProfileSettingsSection`, `EnableCreateCommandSettingsSection`, `AppName`, `ConfigPlanBuilder`.
  - Built-in parser path (lines ~123-150): if `cfgCopy.AppName != ""`, it appends `FromEnv(strings.ToUpper(cfgCopy.AppName), ...)`. If `cfgCopy.ConfigPlanBuilder != nil`, it appends `FromConfigPlanBuilder(...)`.
  - `ParseCommandSettingsSection` is called **before** `middlewaresFunc`, so `--config` and `--profile` flags are already parsed when the config plan builder runs.
  - `ExecuteWithSchema` reverses middlewares before wrapping, so the **last appended** middleware (Cobra flags) has **highest** precedence.
- **What I didn't find useful:** The `ParseGlazedCommandSection` and `shouldValidateRequiredFields` logic are standard.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Not applicable (this is library code). The design doc references specific line ranges that may shift in future Glazed versions.

### 17. `glazed/pkg/config/plan_sources.go`

- **What I was researching:** What config discovery functions exist in Glazed and how they map to layer names.
- **What I was looking for:** The exact function signatures for `SystemAppConfig`, `XDGAppConfig`, `HomeAppConfig`, `GitRootFile`, `WorkingDirFile`, `ExplicitFile`.
- **Why I chose it:** The design proposes a declarative `config.layers` list in `xgoja.yaml`. Each layer name must map to a Glazed function.
- **How I found it:** `grep -r "SystemAppConfig" glazed/ --include="*.go"`.
- **What I found useful:**
  - `SystemAppConfig(appName)` -> `/etc/<appName>/config.yaml`
  - `XDGAppConfig(appName)` -> `$XDG_CONFIG_HOME/<appName>/config.yaml`
  - `HomeAppConfig(appName)` -> `~/.<appName>/config.yaml`
  - `GitRootFile(name)` -> `<git-root>/<name>`
  - `WorkingDirFile(name)` -> `./<name>`
  - `ExplicitFile(path)` -> exact path
  - All are `Optional: true` except `ExplicitFile`.
  - `LayerSystem`, `LayerUser`, `LayerRepo`, `LayerCWD`, `LayerExplicit` constants exist.
- **What I didn't find useful:** `gitRoot` implementation detail (scrubbing Git env vars) is interesting but not relevant.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Not applicable.

### 18. `glazed/pkg/cmds/sources/profiles.go`

- **What I was researching:** How Glazed's profile loading middleware works.
- **What I was looking for:** `GatherFlagsFromCustomProfiles`, `WithProfileAppName`, and the file resolution logic.
- **Why I chose it:** The design proposes optional profile support via `profiles.enabled`.
- **How I found it:** `grep -r "GatherFlagsFromCustomProfiles" glazed/ --include="*.go"`.
- **What I found useful:**
  - `GatherFlagsFromCustomProfiles(profileName, options...)` returns a `Middleware`.
  - `WithProfileAppName(appName)` resolves to `~/.config/<appName>/profiles.yaml`.
  - `WithProfileFile(path)` overrides the default location.
  - The profile file format is:
    ```yaml
    profileName:
      sectionSlug:
        fieldName: value
    ```
  - Profile values are merged into parsed values with configurable precedence depending on where the middleware is placed in the chain.
- **What I didn't find useful:** `loadProfileFromFile` and `resolveProfileFilePath` are implementation details.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Not applicable.

### 19. `glazed/pkg/cmds/sources/middlewares.go`

- **What I was researching:** The middleware execution model: how `Chain`, `Execute`, and `ExecuteWithSchema` work.
- **What I was looking for:** The exact reversal behavior and how middlewares interact with schema cloning.
- **Why I chose it:** Understanding precedence is critical for explaining why env beats config and flags beat env.
- **How I found it:** `grep -r "ExecuteWithSchema" glazed/pkg/cmds/sources/ --include="*.go"`.
- **What I found useful:**
  - `ExecuteWithSchema` clones the schema, reverses the middleware slice, then wraps each middleware around the handler.
  - The comment on lines ~44-56 is an excellent explanation of the reversal behavior:
    > "Middlewares basically get executed in the reverse order they are provided... [f1, f2, f3] will be executed as f1(f2(f3(handler)))(schema_, parsedValues)."
  - This means appending `FromDefaults`, then `FromConfigPlanBuilder`, then `FromEnv`, then `FromCobra` produces the expected precedence.
- **What I didn't find useful:** The `Identity` helper is trivial.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Not applicable.

### 20. `glazed/pkg/cmds/sources/load-fields-from-config.go`

- **What I was researching:** How config files are loaded via a plan builder and how `ConfigFileMapper` works.
- **What I was looking for:** `FromConfigPlanBuilder`, `FromResolvedFiles`, and the `ConfigPlanResolver` signature.
- **Why I chose it:** The design proposes generating a `ConfigPlanBuilder` closure in the template.
- **How I found it:** `grep -r "FromConfigPlanBuilder" glazed/pkg/cmds/sources/ --include="*.go"`.
- **What I found useful:**
  - `ConfigPlanResolver` signature: `func(ctx context.Context, parsedValues *values.Values) (*glazedconfig.Plan, error)`.
  - `FromConfigPlanBuilder` calls `resolver(ctx, parsedValues)`, then `plan.Resolve(ctx)`, then `FromResolvedFiles(files, ...)`.
  - `ConfigFileMapper` transforms raw config file structure into `map[sectionSlug]map[fieldName]value`.
  - Default mapper expects standard section map format.
- **What I didn't find useful:** `FromFile` and `FromFiles` (non-plan variants) are simpler but not what we need.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Not applicable.

### 21. `glazed/pkg/cmds/sources/update.go`

- **What I was researching:** How `FromEnv` works under the hood.
- **What I was looking for:** The `updateFromEnv` function and how environment variable names are constructed.
- **Why I chose it:** To confirm that `MYAPP_OPENAI_API_KEY` is the correct naming convention.
- **How I found it:** `grep -n "func FromEnv" glazed/pkg/cmds/sources/*.go`.
- **What I found useful:**
  - `FromEnv(prefix, options...)` calls `updateFromEnv(schema_, parsedValues, prefix, options...)`.
  - Env var names are derived from section prefix + field name, uppercased, with hyphens replaced by underscores.
  - The prefix is prepended as-is (e.g., `MYAPP_` + `OPENAI_API_KEY`).
- **What I didn't find useful:** `updateFromStringList` is for positional args, not env vars.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Not applicable.

### 22. `glazed/pkg/cmds/sources/vault.go`

- **What I was researching:** Whether Glazed has a standard pattern for sensitive config sources.
- **What I was looking for:** How Vault integration works and whether it should be considered for xgoja.
- **Why I chose it:** The user mentioned "potentially glazed source middleware support"; Vault is a Glazed source middleware.
- **How I found it:** `grep -r "vault" glazed/pkg/cmds/sources/ --include="*.go"`.
- **What I found useful:**
  - `FromVaultSettings` is a middleware that reads sensitive fields from HashiCorp Vault.
  - It only applies to fields marked `Type.IsSensitive()`.
  - This is an advanced feature that could be added to xgoja later via the explicit `middlewares:` list.
- **What I didn't find useful:** The full Vault client implementation (`apiVaultClient`, token resolution, KV v1/v2) is irrelevant for the current design.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Not applicable for this ticket, but the design doc mentions Vault as a future `middlewares[].source` option.

### 23. `glazed/cmd/examples/middlewares-config-env/main.go`

- **What I was researching:** A minimal, self-contained example of `AppName` + `ConfigPlanBuilder` in a Glazed application.
- **What I was looking for:** The exact boilerplate needed to enable env and config support in a simple command.
- **Why I chose it:** This is the simplest working example of the feature we want to add. It's the "hello world" of Glazed middleware configuration.
- **How I found it:** `find glazed/cmd/examples -name "*.go" | xargs grep -l "ConfigPlanBuilder"`.
- **What I found useful:**
  - Demonstrates `cli.CobraParserConfig{AppName: "glazed-mw-demo", ConfigPlanBuilder: func(...) (*glazedconfig.Plan, error) { ... }}`.
  - Shows the minimal config plan: `glazedconfig.NewPlan(glazedconfig.WithLayerOrder(glazedconfig.LayerExplicit)).Add(glazedconfig.ExplicitFile(...))`.
  - Confirms that no custom `MiddlewaresFunc` is needed for simple cases; the built-in parser path handles `AppName` and `ConfigPlanBuilder` automatically.
- **What I didn't find useful:** The `DemoCommand` itself (printing a row) is just a vehicle for the middleware demo.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Not applicable. This example is stable and should be referenced in the design doc's "References" section.

---

## Pinocchio Internal Resources

### 24. `pinocchio/pkg/cmds/profilebootstrap/profile_selection.go`

- **What I was researching:** How a mature Glazed application (Pinocchio) bootstraps its full configuration stack: app name, env prefix, config plan builder, config file mapper, and profile resolution.
- **What I was looking for:** The `AppBootstrapConfig` struct and how it wires env, config, and profiles together.
- **Why I chose it:** Pinocchio is the most complex Glazed app in the workspace. It uses every feature we want to add.
- **How I found it:** `grep -r "BootstrapConfig" pinocchio/ --include="*.go"`.
- **What I found useful:**
  - `pinocchioBootstrapConfig()` returns `bootstrap.AppBootstrapConfig` with:
    - `AppName: "pinocchio"`
    - `EnvPrefix: "PINOCCHIO"`
    - `ConfigFileMapper: configFileMapper`
    - `ConfigPlanBuilder: pinocchioConfigPlanBuilder`
    - `NewProfileSection` and `BuildBaseSections` (Geppetto-specific)
  - `pinocchioConfigPlanBuilder` constructs a full layered plan:
    - System app config (`/etc/pinocchio/config.yaml`)
    - Home app config (`~/.pinocchio/config.yaml`)
    - XDG app config (`~/.config/pinocchio/config.yaml`)
    - Git root files (`configdoc.LocalOverrideFileName`)
    - Working dir files
    - Explicit file from `--config`
  - `configFileMapper` remaps the `profile` block in config files to `profile-settings` section values.
- **What I didn't find useful:**
  - `ResolveCLIProfileRuntime` and `resolveConfigRuntime` are Geppetto-specific (AI engine profile resolution). They involve `gepprofiles.Registry`, `EngineProfileSlug`, and profile registry chains. These are overkill for xgoja.
  - `normalizeProfileRegistryEntries` is a utility.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Not applicable. However, we should **not** copy Pinocchio's full profile bootstrap for xgoja. The design doc correctly proposes a simpler model: `GatherFlagsFromCustomProfiles` as a middleware.

### 25. `pinocchio/pkg/cmds/cobra.go`

- **What I was researching:** How Pinocchio's custom `MiddlewaresFunc` is wired into its commands.
- **What I was looking for:** The exact middleware chain that Pinocchio uses.
- **Why I chose it:** This is the reference implementation for a custom middleware chain in a Glazed app.
- **How I found it:** `grep -r "GetPinocchioCommandMiddlewares" pinocchio/ --include="*.go"`.
- **What I found useful:**
  - `GetPinocchioCommandMiddlewares` builds the chain:
    1. `FromCobra(cmd, ...)`
    2. `FromArgs(args, ...)`
    3. `FromEnv(cfg.EnvPrefix, ...)`
    4. `FromConfigPlanBuilder(resolver, WithConfigFileMapper, WithParseOptions)`
    5. `FromDefaults(...)`
  - This is the **exact chain we want to generate** for xgoja binaries.
  - `BuildCobraCommandWithGeppettoMiddlewares` wraps `cli.BuildCobraCommand` with `cli.WithParserConfig(config)`.
- **What I didn't find useful:** Nothing.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Not applicable. This file should be referenced as the canonical example in implementation guides.

---

## Geppetto Internal Resources

### 26. `geppetto/pkg/cli/bootstrap/config.go`

- **What I was researching:** The `AppBootstrapConfig` struct definition to understand the contract Pinocchio implements.
- **What I was looking for:** Required fields, validation rules, and the `ConfigPlanBuilder` type alias.
- **Why I chose it:** Pinocchio's bootstrap config is an instance of this struct. Understanding the generic contract helps design xgoja's equivalent.
- **How I found it:** `find geppetto/pkg/cli/bootstrap -name "*.go" | xargs grep -l "AppBootstrapConfig"`.
- **What I found useful:**
  - `AppBootstrapConfig` fields: `AppName`, `EnvPrefix`, `ConfigFileMapper`, `NewProfileSection`, `BuildBaseSections`, `ConfigPlanBuilder`.
  - `Validate()` enforces that all fields are non-empty.
  - `ConfigPlanBuilder` type: `func(parsed *values.Values) (*glazedconfig.Plan, error)`.
- **What I didn't find useful:** `normalizedEnvPrefix()` is a trivial trim helper.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Not applicable. However, xgoja does not need the full `AppBootstrapConfig` abstraction at runtime. The design doc correctly proposes embedding equivalent information in `xgoja.yaml` and generating code.

---

## Skills / Process Resources

### 27. `ticket-research-docmgr-remarkable/references/writing-style.md`

- **What I was researching:** The expected structure and quality bar for design documents in this project.
- **What I was looking for:** Section ordering, decision record format, evidence rules, and detail level.
- **Why I chose it:** The user asked for a "detailed analysis / design / implementation guide that is for a new intern." This reference defines the project's writing standards.
- **How I found it:** Read via the pinned `ticket-research-docmgr-remarkable` skill.
- **What I found useful:**
  - Recommended section order: Executive summary, Problem statement, Current-state analysis, Gap analysis, Proposed architecture, Decision records, Pseudocode, Implementation phases, Test strategy, Risks, References.
  - Decision record format with Context, Options, Decision, Rationale, Consequences, Status.
  - Evidence rules: anchor claims to files, use line references, distinguish observed from inferred.
  - Clarity patterns: numbered lists, explicit naming ("Phase 1"), define terms before reuse.
- **What I didn't find useful:** Nothing.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Not applicable.

### 28. `ticket-research-docmgr-remarkable/references/deliverable-checklist.md`

- **What I was researching:** The checklist of deliverables expected for a ticket research task.
- **What I was looking for:** Validation steps (doctor, vocabulary, upload) and final handoff requirements.
- **Why I chose it:** Ensures the output meets the project's quality gate.
- **How I found it:** Same skill as above.
- **What I found useful:**
  - Ticket setup: design doc, diary, index/tasks/changelog.
  - Bookkeeping: relate files, changelog, tasks.
  - Validation: `docmgr doctor`, vocabulary warnings.
  - reMarkable delivery: dry-run, real upload, remote listing.
  - Final response must include ticket path, doc paths, validation status, upload destination.
- **What I didn't find useful:** Nothing.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Not applicable.

### 29. `diary/SKILL.md`

- **What I was researching:** The expected format for diary entries.
- **What I was looking for:** Section template, writing rules, and the working loop.
- **Why I chose it:** The user explicitly asked to "keep a diary as you work."
- **How I found it:** Pinned skill.
- **What I found useful:**
  - Strict step format: Goal, Prompt Context, What I did, Why, What worked, What didn't work, What I learned, What was tricky, What warrants review, Future work, Code review instructions, Technical details.
  - Working loop: implement -> commit -> docmgr task check -> diary update -> relate files -> changelog update -> commit docs.
  - Verbatim user prompt must be preserved the first time it appears.
- **What I didn't find useful:** Nothing.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Not applicable.

### 30. `docmgr/SKILL.md`

- **What I was researching:** How to create tickets, add documents, relate files, and validate.
- **What I was looking for:** Exact command syntax and conventions.
- **Why I chose it:** Needed to set up the XGOJA-017 ticket workspace.
- **How I found it:** Pinned skill.
- **What I found useful:**
  - `docmgr ticket create-ticket`, `docmgr doc add`, `docmgr doc relate`.
  - `--file-note "path:reason"` format.
  - Absolute paths preferred.
  - `docmgr doctor --ticket --stale-after 30`.
  - `docmgr vocab add` for unknown topics.
- **What I didn't find useful:** Nothing.
- **What is out of date / wrong:** Nothing observed.
- **What would need updating:** Not applicable.

---

## External Resources

### 31. Glazed external documentation (not consulted)

- **What I was researching:** Whether there is official Glazed documentation beyond the source code.
- **What I was looking for:** Public docs on middleware chains, config plans, or `CobraParserConfig`.
- **Why I chose it:** The user mentioned "reading glazed documentation."
- **How I found it:** Not searched. The workspace contains the full Glazed source code, and the relevant APIs are well-documented in Go doc comments.
- **What I found useful:** N/A.
- **What I didn't find useful:** N/A.
- **What is out of date / wrong:** N/A.
- **What would need updating:** If Glazed has public docs (e.g., on GitHub Pages or in a `docs/` directory), they should be checked for examples. However, the source code in this workspace is authoritative and up-to-date.

### 32. reMarkable upload process (`remarquee` CLI)

- **What I was researching:** How to upload the final document bundle.
- **What I was looking for:** Bundle upload syntax, dry-run behavior, and remote directory conventions.
- **Why I chose it:** The deliverable checklist requires reMarkable upload.
- **How I found it:** Pinned `remarkable-upload` skill.
- **What I found useful:**
  - `remarquee upload bundle --dry-run <files> --name <name> --remote-dir <dir> --toc-depth 2`.
  - Dry-run is mandatory before real upload.
  - Remote directory convention: `/ai/YYYY/MM/DD/<TICKET-ID>`.
  - Bundle name should not contain spaces (caused a pandoc failure; fixed by using `XGOJA-017-Design`).
- **What I didn't find useful:** Nothing.
- **What is out of date / wrong:** The skill does not mention the space-in-name pandoc failure. This is a runtime quirk of `remarquee`/`pandoc`.
- **What would need updating:** The skill or `remarquee` docs could warn against spaces in bundle names.

---

## Summary of Findings

| Category | Resources Read | Useful | Out of Date | Needs Updating |
|---|---|---|---|---|
| xgoja build spec | 2 (spec.go, cmd_build.go) | Yes | No | spec.go (Phase 1) |
| xgoja CLI | 1 (root.go) | Limited | No | No |
| xgoja generator | 3 (generate.go, templates.go, main.go.tmpl) | Yes | No | templates.go, main.go.tmpl (Phase 3) |
| xgoja docs | 2 (tutorial, buildspec-ref) | Yes | No | Add new tutorial (Phase 4) |
| xgoja runtime | 5 (host.go, root.go, spec.go, glazed.go, framework.go) | Yes | No | All five files (Phase 2) |
| xgoja examples | 2 (01-core, 05-command) | Yes | No | Add new example (Phase 4) |
| Glazed parser | 1 (cobra-parser.go) | Yes | No | No |
| Glazed config | 1 (plan_sources.go) | Yes | No | No |
| Glazed middlewares | 4 (middlewares.go, load-fields, update.go, profiles.go) | Yes | No | No |
| Glazed vault | 1 (vault.go) | Partial | No | No |
| Glazed examples | 1 (middlewares-config-env) | Yes | No | No |
| Pinocchio | 2 (profile_selection.go, cobra.go) | Yes | No | No |
| Geppetto | 1 (config.go) | Yes | No | No |
| Process/skills | 4 (writing-style, checklist, diary, docmgr) | Yes | No | No |

**Total resources read:** 30 internal files + 4 process skills = 34 resources.

**Critical files for implementation (in dependency order):**
1. `pkg/xgoja/app/spec.go` — add runtime fields.
2. `cmd/xgoja/internal/buildspec/spec.go` — add YAML fields.
3. `cmd/xgoja/internal/buildspec/validate.go` — add validation rules.
4. `pkg/xgoja/app/glazed.go` — parameterize command builder.
5. `pkg/xgoja/app/host.go` — propagate middleware factory.
6. `pkg/xgoja/app/root.go` — pass middleware factory through Options.
7. `pkg/xgoja/app/framework.go` — use `spec.AppName`.
8. `cmd/xgoja/internal/generate/templates.go` — build template data.
9. `cmd/xgoja/internal/generate/templates/main.go.tmpl` — emit middleware wiring.
10. `cmd/xgoja/internal/generate/generate_test.go` — add template tests.

---

## Open Research Questions

1. **Template complexity vs. helper package:** Should the generator emit a complex `ConfigPlanBuilder` closure, or should we create `pkg/xgoja/appconfig` to reduce template complexity? This was not resolved during research because it is an architectural tradeoff, not a factual question answerable by reading existing code.
2. **Profile section vs. middleware:** Should profiles be a Glazed section (like Geppetto's `profile-settings`) or a source middleware? Geppetto uses both. Reading `geppetto/pkg/cli/bootstrap/config.go` showed the complexity of full section support. The simpler middleware approach is likely sufficient for xgoja.
3. **Config file mapper necessity:** Does xgoja need a custom `ConfigFileMapper`? Pinocchio's mapper remaps the `profile` block. Xgoja's config files will likely use the standard section map format, so a mapper may be unnecessary in Phase 1.
4. **Existing xgoja validation coverage:** How comprehensive is `validate_test.go`? Not read in detail. This should be checked before Phase 1 implementation to ensure new validation rules have adequate test coverage.
5. **Glazed version pinning:** The generated `go.mod` pins a Glazed version. If we add new imports (e.g., `glazed/pkg/config`) to generated code, we must ensure the pinned version supports them. This is a build-time concern, not a research question.
