---
Title: Generated binary polish design and implementation guide
Ticket: XGOJA-005
Status: active
Topics:
    - xgoja
    - glazed
    - help-system
    - logging
    - templates
    - repl
    - runtime
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/goja-repl/cmd_run.go
      Note: |-
        Reference implementation for script-file execution command
        Reference run-file command
    - Path: cmd/goja-repl/tui.go
      Note: |-
        Reference implementation for Bubble Tea REPL command
        Reference TUI REPL command
    - Path: cmd/xgoja/internal/generate/main.go
      Note: |-
        Current generated main.go renderer; convert inline string construction to templates
        Current generated main.go renderer to convert to templates
    - Path: pkg/xgoja/app/host.go
      Note: |-
        Host attaches generated commands to xgoja, Cobra, and adapter targets
        Host command attachment for xgoja/cobra/adapter generated targets
    - Path: pkg/xgoja/app/root.go
      Note: |-
        Generated runtime command assembly lives here
        Generated runtime root and command construction
ExternalSources: []
Summary: Design and implementation guide for making generated xgoja binaries use Glazed logging/help/command plumbing, template-based source generation, a run-file command, and a TUI REPL command.
LastUpdated: 2026-05-24T14:18:00-04:00
WhatFor: Give a new intern enough context to implement XGOJA-005 safely and reviewably.
WhenToUse: Read before changing generated xgoja source rendering, generated command wiring, run-file behavior, or TUI REPL integration.
---


# Generated binary polish design and implementation guide

## Executive Summary

`xgoja` builds custom Goja-powered binaries from a build specification. The generated binaries already compile provider packages, create runtime profiles, expose an evaluation command, list modules, and mount JavaScript verbs. The next improvement is to make those generated binaries behave like first-class Glazed CLIs rather than minimal Cobra programs.

This ticket has five implementation goals:

1. Replace the current generated `main.go` inline string construction with Go templates.
2. Install Glazed logging and help-system plumbing in generated binaries.
3. Ensure generated JavaScript verb commands and support commands use Glazed command construction consistently.
4. Add a generated `run` command that executes a JavaScript file in a selected xgoja runtime profile.
5. Add a generated `tui` REPL command modeled on `cmd/goja-repl/tui.go`.

The key boundary is this: `cmd/xgoja` is the builder CLI, while `pkg/xgoja/app` is the generated-runtime support library. The generated `main.go` should stay thin. It imports provider packages, registers them, decodes the embedded runtime spec, constructs an `app.Host`, and delegates all reusable command behavior to `pkg/xgoja/app`.

## Problem Statement

Generated xgoja binaries currently work, but their runtime command surface is not yet aligned with the rest of the repository's CLI conventions.

Current limitations:

- `cmd/xgoja/internal/generate/main.go` builds Go source by hand with a long sequence of `strings.Builder` writes and `fmt.Fprintf` calls.
- Generated binaries do not install Glazed logging flags and logger initialization.
- Generated binaries do not install a bundled/generated help system.
- The eval and modules support commands are hand-written Cobra commands using `fmt.Fprintln` instead of Glazed command objects.
- The generated runtime has no `run` command for script files, even though `cmd/goja-repl run` already provides the expected behavior pattern.
- The generated runtime has no TUI REPL command, even though `cmd/goja-repl tui` already demonstrates the Bubble Tea / bobatea / replapi integration.

These are not correctness bugs in the existing xgoja builder. They are product-quality gaps: generated binaries should feel like normal Glazed CLIs and should expose the common runtime entrypoints users expect.

## System Overview

### Main packages

| Package or file | Role |
|---|---|
| `cmd/xgoja` | Builder CLI. Reads `xgoja.yaml`, validates it, generates source, runs `go mod tidy`, and builds the final binary. |
| `cmd/xgoja/internal/buildspec` | Builder-side YAML schema and validation. It is not imported by generated binaries. |
| `cmd/xgoja/internal/generate` | Produces generated `go.mod`, `main.go`, and embedded jsverb files. |
| `pkg/xgoja/providerapi` | Public provider API imported by generated binaries and provider packages. |
| `pkg/xgoja/app` | Public generated-runtime support package. Creates runtime factories, hosts, root commands, eval/modules/jsverb commands, and should receive run/TUI support. |
| `pkg/jsverbs` | Scans JavaScript verb files and converts them into Glazed commands. |
| `engine` | Owns Goja runtime lifecycle, event loop, runtime owner, require registry, runtime-aware modules, and module middleware. |
| `cmd/goja-repl/cmd_run.go` | Existing script-file execution behavior to adapt into generated xgoja runtimes. |
| `cmd/goja-repl/tui.go` | Existing TUI REPL behavior to study before adding generated-runtime TUI support. |

### Generated binary data flow

```text
xgoja.yaml
  |
  v
cmd/xgoja build
  |
  +--> buildspec.Load + buildspec.Validate
  |
  +--> generate.RenderMain + generate.RenderEmbeddedSpec + generate.RenderGoMod
  |
  +--> temporary Go module
         |
         +--> generated main.go
         |     - imports provider packages
         |     - registers providerapi packages
         |     - embeds normalized app.Spec JSON
         |     - optionally embeds local jsverb source files
         |     - creates app.Host
         |     - executes Cobra root
         |
         +--> go build
               |
               v
            generated binary
```

### Runtime command flow

```text
generated binary argv
  |
  v
Cobra root command
  |
  +--> Glazed logging PersistentPreRunE
  |
  +--> help command from Glazed help system
  |
  +--> app.Host.AttachDefaultCommands
        |
        +--> repl/eval command
        +--> modules command
        +--> verbs command
        |     |
        |     +--> jsverbs.Registry.CommandForVerbWithInvoker
        |           |
        |           +--> RuntimeFactory.NewRuntime(profile, require.WithLoader(...))
        |           +--> Registry.InvokeInRuntime(...)
        |
        +--> run command
        +--> tui command
```

### Runtime profile flow

```text
app.Spec.Runtimes[profile]
  |
  v
app.RuntimeFactory.NewRuntime(ctx, profile, require options...)
  |
  +--> providerapi.Registry.LookupModule(package, name)
  +--> adapt provider module to engine.RuntimeModuleSpec
  +--> engine.NewBuilder(...).WithModules(selected modules...)
  +--> engine.Factory.NewRuntime(ctx)
  |
  v
*engine.Runtime
  - VM
  - Require
  - Owner
  - Loop
  - lifecycle closers
```

## Current Implementation Notes

### Current generated `main.go` rendering

`cmd/xgoja/internal/generate/main.go` currently emits Go source directly:

```go
func RenderMain(spec *buildspec.Spec) string {
    aliases := importAliases(spec.Packages)
    hasEmbedded := hasEmbeddedJSVerbSources(spec)
    var b strings.Builder
    b.WriteString("// Code generated by xgoja; DO NOT EDIT.\n")
    ...
    fmt.Fprintf(&b, "\t%s \"%s\"\n", aliases[pkg.ID], pkg.Import)
    ...
    return b.String()
}
```

This works, but it is difficult to review because generated Go syntax is interleaved with renderer control flow. Adding logging, help, and more command paths will make the string-writer approach harder to maintain.

### Current generated command assembly

`pkg/xgoja/app.NewRootCommand` creates a simple Cobra root and delegates to `Host.AttachDefaultCommands`.

`Host.AttachDefaultCommands` currently attaches:

- eval/repl command when `commands.repl.enabled` is true,
- modules command always,
- verbs command when `commands.jsverbs.enabled` is true.

The verbs command already uses Glazed for the actual JavaScript verb commands:

```go
glazedcli.AddCommandsToRootCommand(root, mounted, nil, glazedcli.WithParserConfig(...))
```

The support commands are still plain Cobra commands.

## Target Design

### 1. Template-based generated `main.go`

Introduce a template-driven renderer in `cmd/xgoja/internal/generate`.

Recommended file layout:

```text
cmd/xgoja/internal/generate/
  main.go                  # public RenderMain plus data preparation helpers
  templates.go             # //go:embed templates/*.tmpl and execution helpers
  templates/
    main.go.tmpl           # generated main.go template
```

If the project prefers fewer files, `templates.go` can keep the template as a raw string constant. A real `.tmpl` file is easier for interns and reviewers because it looks like Go source.

Template data should be explicit:

```go
type mainTemplateData struct {
    SpecJSON       string
    HasEmbedded    bool
    TargetKind     string
    TargetImport   string
    TargetRoot     string
    PackageImports []providerImport
}

type providerImport struct {
    Alias    string
    Import   string
    Register string
}
```

Renderer pseudocode:

```text
RenderMain(spec):
  data := buildMainTemplateData(spec)
  parse embedded template main.go.tmpl
  execute template with data into bytes.Buffer
  go/format the result if possible
  return string(result)
```

The template should still generate a small `main` package, not move large behavior into generated code. Generated source should be boring:

```go
func main() {
    registry := providerapi.NewRegistry()
    must(provider.Register(registry))

    spec := decodeSpec()
    host := app.NewHostWithOptions(...)
    root, err := app.NewRootCommand(...)
    must(err)
    must(root.Execute())
}
```

For `adapter` and `cobra` target modes, keep the same integration points:

- `adapter`: call `target.Build(context.Background(), host)`.
- `cobra`: call `target.<rootFn>()`, then `host.AttachDefaultCommands(root)`.
- `xgoja`: call `app.NewRootCommand(...)`.

### 2. Glazed logging in generated binaries

Generated binaries should install Glazed logging exactly like `cmd/xgoja/root.go` and `cmd/goja-repl/root.go`.

Reference API:

```go
import "github.com/go-go-golems/glazed/pkg/cmds/logging"

root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
    return logging.InitLoggerFromCobra(cmd)
}
if err := logging.AddLoggingSectionToRootCommand(root, spec.Name); err != nil {
    return nil, err
}
```

Implementation recommendation:

- Put this in `pkg/xgoja/app`, not generated `main.go`.
- Add a helper such as `InstallRootCommandFramework(root *cobra.Command, opts RootFrameworkOptions) error`.
- Call it from `NewRootCommand` for `target.kind: xgoja`.
- Also call it from `Host.AttachDefaultCommands` or a new explicit host method for `adapter` and `cobra` target modes, so attached roots get the same logging/help behavior.

Be careful with existing target roots:

- A caller-provided root might already have a `PersistentPreRunE`.
- The implementation should preserve it by chaining:

```go
existing := root.PersistentPreRunE
root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
    if existing != nil {
        if err := existing(cmd, args); err != nil { return err }
    }
    return logging.InitLoggerFromCobra(cmd)
}
```

### 3. Help system in generated binaries

Generated binaries should have a help command and at least generated xgoja help topics.

Reference API from `cmd/xgoja/root.go`:

```go
helpSystem := help.NewHelpSystem()
if err := doc.AddDocToHelpSystem(helpSystem); err != nil { return nil, err }
help_cmd.SetupCobraRootCommand(helpSystem, root)
```

Options:

1. Reuse `cmd/xgoja/doc` in generated binaries.
2. Move xgoja runtime help docs to a public package such as `pkg/xgoja/doc` or `pkg/xgoja/appdoc`.

Use option 2 if possible. Generated binaries should avoid importing `cmd/xgoja/...` packages. A public doc package keeps the generated runtime dependency tree clear.

Suggested package:

```text
pkg/xgoja/doc/
  doc.go
  01-overview.md
  02-buildspec.md
  03-runtime.md
```

Generated runtime support can load this:

```go
helpSystem := help.NewHelpSystem()
_ = xgojadoc.AddDocToHelpSystem(helpSystem)
help_cmd.SetupCobraRootCommand(helpSystem, root)
```

If moving the existing help docs is too large for the first pass, keep builder docs where they are and add a small generated-runtime help system in `pkg/xgoja/app` with minimal topics:

- `runtime-overview`,
- `runtime-profiles`,
- `jsverbs`,
- `run-and-tui`.

### 4. Glazed command plumbing for support commands

The generated `verbs` command already converts JavaScript verbs into Glazed commands. This ticket should also make the support commands Glazed commands where practical.

Candidates:

- modules command: `cmds.GlazeCommand`, emits rows for package/module/alias/profile.
- eval command: `cmds.BareCommand`, evaluates source and writes output.
- run command: `cmds.BareCommand`, executes a file.
- tui command: `cmds.BareCommand`, starts Bubble Tea program.

Glazed command types:

```go
type evalCommand struct { *cmds.CommandDescription; host *Host }
type modulesCommand struct { *cmds.CommandDescription; host *Host }
type runCommand struct { *cmds.CommandDescription; host *Host }
type tuiCommand struct { *cmds.CommandDescription; host *Host }
```

Attach with:

```go
cobraCommand, err := glazedcli.BuildCobraCommand(command,
    glazedcli.WithParserConfig(glazedcli.CobraParserConfig{
        MiddlewaresFunc: glazedcli.CobraCommandDefaultMiddlewares,
    }),
)
root.AddCommand(cobraCommand)
```

For row-producing commands, use Glaze commands and rows rather than `fmt.Fprintf` loops. For commands that need terminal control or direct output, Bare commands are appropriate.

### 5. Generated `run` command

The generated `run` command should execute a JavaScript file using a selected runtime profile from `app.Spec`.

It is similar to `cmd/goja-repl run`, but it must use `app.RuntimeFactory`, not `engine.NewBuilder()` directly. The selected provider modules come from the xgoja buildspec runtime profile.

Proposed CLI:

```text
<generated> run [file] --runtime repl
```

Behavior:

- Resolve the file to an absolute path.
- Configure require module roots from the script path so sibling `require()` calls work.
- Create an xgoja runtime for the selected profile.
- Require the script path as a module.
- Close the runtime.

Pseudocode:

```text
runCommand.Run(ctx, values):
  file := required argument
  profile := --runtime or default first runtime
  scriptPath := abs(file)
  requireOpt := engine.RequireOptionWithModuleRootsFromScript(scriptPath, engine.DefaultModuleRootsOptions())
  rt := host.Factory.NewRuntime(ctx, profile, requireOpt)
  defer rt.Close(ctx)
  rt.Owner.Call(ctx, "xgoja.run", func(vm):
      rt.Require.Require(scriptPath)
  )
```

Important difference from `cmd/goja-repl run`:

- `goja-repl run` uses global engine module middleware and plugin discovery flags.
- xgoja `run` uses buildspec-selected provider modules from the generated binary. It should not implicitly enable modules that the runtime profile did not select.

### 6. Generated TUI REPL command

The generated `tui` command should reuse the existing bobatea REPL UI concepts but evaluate code against an xgoja runtime profile.

Reference file:

```text
cmd/goja-repl/tui.go
```

That implementation uses:

- `bobatea/pkg/repl` model,
- `pkg/repl/adapters/bobatea` adapter,
- `pkg/replapi` sessions,
- Watermill event bus,
- shared help system.

There are two possible implementation strategies.

#### Strategy A: Use replapi

Create a `replapi.App` configured with an engine builder that mirrors the xgoja runtime profile. This gives persistent/session semantics and uses the existing adapter.

Risk: replapi's current app construction is oriented around its own profiles and plugin/module setup. Mapping xgoja runtime profiles into replapi may require new extension points.

#### Strategy B: Implement a small xgoja bobatea adapter

Create an adapter that satisfies bobatea's REPL interface and evaluates each input through an `app.RuntimeFactory` runtime. This keeps xgoja runtime policy exact.

Risk: it duplicates some replapi behavior and may not get persistent session history immediately.

Recommended first implementation: Strategy B for xgoja generated binaries.

Reason: XGOJA-005 is about generated binaries honoring buildspec-selected runtime profiles. A small adapter can be implemented with a narrow contract:

```text
Evaluate(input):
  rt := factory.NewRuntime(ctx, profile)
  value := vm.RunString(input)
  return rendered output
```

Later, a separate ticket can unify this with `replapi` if persistent history is required.

Proposed CLI:

```text
<generated> tui --runtime repl --alt-screen=true
```

Pseudocode:

```text
tuiCommand.Run(ctx, values):
  profile := --runtime or default first runtime
  adapter := newXGojaREPLAdapter(host.Factory, profile)
  cfg := bobarepl.DefaultConfig()
  cfg.Title = fmt.Sprintf("%s TUI (%s runtime)", spec.Name, profile)
  bus := newQuietInMemoryBus()
  model := bobarepl.NewModel(adapter, cfg, bus.Publisher)
  program := tea.NewProgram(model, optional tea.WithAltScreen())
  run bus and program with errgroup
```

### 7. Buildspec command configuration

Current command configuration has:

```yaml
commands:
  repl:
    enabled: true
    runtime: repl
    name: repl
  jsverbs:
    enabled: true
    runtime: repl
    name: verbs
```

The new commands need buildspec controls. Recommended schema extension:

```yaml
commands:
  repl:
    enabled: true
    runtime: repl
    name: repl
  run:
    enabled: true
    runtime: repl
    name: run
  tui:
    enabled: true
    runtime: repl
    name: tui
  jsverbs:
    enabled: true
    runtime: repl
    name: verbs
```

Implementation notes:

- Add `Run CommandSpec` and `TUI CommandSpec` to both builder-side and runtime-side command specs:
  - `cmd/xgoja/internal/buildspec/spec.go`
  - `pkg/xgoja/app/spec.go`
- Update validation so enabled `run` and `tui` commands must reference an existing runtime profile.
- Decide defaults:
  - `run.enabled` should probably default to true when a runtime exists.
  - `tui.enabled` may default to false if dependencies or terminal behavior are considered heavy.
- The first implementation can make both explicit and update examples to enable them.

## Implementation Plan

### Commit 1: Ticket, analysis, and design guide

- Create XGOJA-005.
- Add this design guide.
- Add diary.
- Add tasks.
- Relate key files.
- Upload guide to reMarkable.
- Commit docs.

Validation:

```bash
docmgr doctor --ticket XGOJA-005 --stale-after 30
```

### Commit 2: Template renderer

- Add `cmd/xgoja/internal/generate/templates/main.go.tmpl`.
- Add renderer data structs and template execution helper.
- Keep generated output semantically equivalent.
- Update tests to compare expected generated snippets if needed.

Validation:

```bash
GOWORK=off go test ./cmd/xgoja/internal/generate ./cmd/xgoja -count=1
```

### Commit 3: Generated root framework

- Add Glazed logging setup to generated roots.
- Add generated runtime help system.
- Ensure xgoja, cobra, and adapter target modes all get the same framework installation.
- Preserve existing target root persistent pre-run behavior.

Validation:

```bash
GOWORK=off go test ./pkg/xgoja/app ./cmd/xgoja/internal/generate ./cmd/xgoja -count=1
```

### Commit 4: Glazed support commands

- Convert eval and modules support commands to Glazed command objects where practical.
- Keep jsverb mounting through `glazedcli.AddCommandsToRootCommand`.
- Add tests for help output and command output.

Validation:

```bash
GOWORK=off go test ./pkg/xgoja/app ./pkg/jsverbs ./cmd/xgoja/internal/generate -count=1
```

### Commit 5: Run command

- Extend command spec with `run`.
- Implement generated runtime `run` command.
- Add tests for:
  - running a file,
  - sibling module require from script directory,
  - runtime profile selection.

Validation:

```bash
GOWORK=off go test ./pkg/xgoja/app ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate -count=1
```

### Commit 6: TUI command

- Extend command spec with `tui`.
- Add command help test.
- Add minimal non-interactive construction tests; avoid trying to run a full terminal UI in normal unit tests.
- If possible, add adapter-level tests for evaluation without starting Bubble Tea.

Validation:

```bash
GOWORK=off go test ./pkg/xgoja/app ./cmd/xgoja/internal/generate -count=1
```

### Commit 7: Docs, examples, and closure

- Update xgoja bundled docs.
- Update `examples/xgoja/*` specs and smoke Makefiles if commands are enabled by default or examples should demonstrate `run`.
- Run full focused suite and examples.
- Update diary/changelog/tasks.
- `docmgr doctor`.
- Close ticket.

Validation:

```bash
GOWORK=off go test \
  ./engine \
  ./pkg/runtimebridge \
  ./pkg/jsverbs \
  ./pkg/xgoja/app \
  ./cmd/xgoja/internal/generate \
  ./cmd/xgoja \
  ./cmd/xgoja/internal/buildspec \
  ./pkg/xgoja/providerapi \
  ./pkg/xgoja/testprovider \
  ./pkg/xgoja/testcobra \
  ./pkg/xgoja/testadapter \
  ./modules/express \
  ./modules/uidsl \
  ./pkg/hashiplugin/host \
  ./pkg/repl/evaluators/javascript \
  ./pkg/docaccess/runtime \
  ./pkg/jsverbscli \
  ./pkg/gojahttp \
  ./pkg/doc \
  -count=1

for dir in runtime-filesystem embedded-jsverbs provider-shipped-jsverbs; do
  make -C examples/xgoja/$dir smoke
done
```

## Review Checklist

- Generated source is template-rendered and gofmt-compatible.
- Generated binaries install logging flags and initialize logging.
- Generated binaries install a help command.
- Existing xgoja target mode still works.
- Existing cobra target mode still attaches commands and preserves fixture command behavior.
- Existing adapter target mode still attaches commands and preserves fixture command behavior.
- JS verbs still mount through Glazed and can use runtime/provider/embedded sources.
- `run` command uses buildspec-selected runtime modules and script-local module roots.
- `tui` command does not bypass runtime profile policy.
- Tests do not require an interactive terminal.
- Examples remain runnable under `GOWORK=off` in this workspace.

## Open Questions

1. Should `run` be enabled by default in generated binaries?
   - Recommendation: yes, if a runtime profile exists.
2. Should `tui` be enabled by default?
   - Recommendation: no for the first implementation unless the user explicitly wants generated binaries to always include TUI dependencies.
3. Should generated help docs reuse builder docs or have runtime-only docs?
   - Recommendation: move/copy runtime-safe docs into a public package under `pkg/xgoja/doc`.
4. Should TUI use `replapi` immediately?
   - Recommendation: first implement a direct xgoja runtime adapter; later evaluate replapi integration if persistent history is required.

## API Reference

### `pkg/xgoja/app.NewRootCommand`

Creates a generated xgoja root command from embedded provider registry and JSON spec.

```go
func NewRootCommand(opts Options) (*cobra.Command, error)
```

Expected future behavior:

- decode embedded spec,
- create host,
- install Glazed logging/help framework,
- attach configured commands.

### `pkg/xgoja/app.Host`

Runtime command attachment point used by generated xgoja, adapter, and cobra target modes.

```go
type Host struct {
    Providers       *providerapi.Registry
    Spec            *Spec
    Factory         *RuntimeFactory
    EmbeddedJSVerbs fs.FS
}
```

### `pkg/xgoja/app.RuntimeFactory.NewRuntime`

Creates an `engine.Runtime` for a runtime profile and optional require options.

```go
func (f *RuntimeFactory) NewRuntime(ctx context.Context, profile string, opts ...require.Option) (*engine.Runtime, error)
```

Use this for `eval`, `run`, `jsverbs`, and direct xgoja TUI evaluation.

### `engine.RequireOptionWithModuleRootsFromScript`

Configures Goja `require()` module roots from a script file.

```go
requireOpt, err := engine.RequireOptionWithModuleRootsFromScript(scriptPath, engine.DefaultModuleRootsOptions())
```

Use this in the generated `run` command so scripts can `require()` sibling files.

### `glazedcli.BuildCobraCommand`

Converts Glazed command objects to Cobra commands.

```go
cobraCommand, err := glazedcli.BuildCobraCommand(command,
    glazedcli.WithParserConfig(glazedcli.CobraParserConfig{
        MiddlewaresFunc: glazedcli.CobraCommandDefaultMiddlewares,
    }),
)
```

Use this for generated support commands where practical.

### `logging.AddLoggingSectionToRootCommand`

Adds Glazed logging flags to a root Cobra command.

```go
err := logging.AddLoggingSectionToRootCommand(root, appName)
```

### `help_cmd.SetupCobraRootCommand`

Adds Glazed help commands to a Cobra root.

```go
helpSystem := help.NewHelpSystem()
help_cmd.SetupCobraRootCommand(helpSystem, root)
```

## File References

- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/cmd/xgoja/internal/generate/main.go`
- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/cmd/xgoja/internal/generate/generate_test.go`
- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/cmd/xgoja/internal/buildspec/spec.go`
- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/cmd/xgoja/internal/buildspec/validate.go`
- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/spec.go`
- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/root.go`
- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/host.go`
- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/factory.go`
- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/jsverbs/runtime.go`
- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/cmd/goja-repl/cmd_run.go`
- `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/cmd/goja-repl/tui.go`
