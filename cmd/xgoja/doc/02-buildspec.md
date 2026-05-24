---
Title: "xgoja buildspec reference"
Slug: buildspec
Short: "Reference for xgoja.yaml packages, runtimes, commands, targets, and jsverb source modes."
Topics:
- xgoja
- buildspec
- providers
- jsverbs
- goja
Commands:
- xgoja build
- xgoja doctor
- xgoja list-modules
Flags:
- --file
- --output
- --work-dir
- --keep-work
- --xgoja-version
- --xgoja-replace
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

An `xgoja.yaml` file describes the generated binary. It names the output program, selects provider packages, defines runtime profiles, enables commands, and configures JavaScript verb sources.

A minimal shape looks like this:

```yaml
name: fixture
go:
  version: "1.26"
  module: example.com/generated/fixture
target:
  kind: xgoja
  output: dist/fixture
packages:
  - id: fixture
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider
    register: Register
runtimes:
  repl:
    modules:
      - package: fixture
        name: hello
        as: hello
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

## Top-level fields

`name` is the generated application name. It defaults to `xgoja-app`.

`go` controls the generated module. `go.version` defaults to `1.26`, and `go.module` defaults to `example.com/generated/<name>`.

`target` controls the generated binary shape and output path.

`packages` selects Go provider packages that will be imported by generated source.

`runtimes` defines named runtime profiles.

`commands` enables generated command families and points them at runtime profiles.

`jsverbs` configures JavaScript verb sources that are mounted under the generated jsverbs command.

## Packages

Each package entry gives xgoja enough information to import and register a provider package.

```yaml
packages:
  - id: fixture
    import: github.com/example/fixture/xgoja
    version: v0.1.0
    register: Register
    replace: ../fixture
```

Fields:

- `id` is the local package ID referenced by runtimes and provider-shipped verb sources.
- `import` is the Go import path used by generated `main.go`.
- `version` is the module version added to generated `go.mod` when needed.
- `register` is the provider registration function. It defaults to `Register`.
- `replace` is a local development replacement path added to generated `go.mod`.

Package IDs must be unique.

## Runtimes

A runtime profile selects modules from registered provider packages.

```yaml
runtimes:
  main:
    modules:
      - package: fixture
        name: hello
        as: hello
        config:
          greeting: hello
```

Fields:

- `package` references a package ID from `packages`.
- `name` is the provider module name.
- `as` is the JavaScript `require()` alias. If omitted, it defaults to `name`.
- `config` is marshaled to JSON and passed to the provider module factory.

Aliases must be unique within one runtime profile.

## Commands

The `commands` section enables generated command families.

```yaml
commands:
  repl:
    enabled: true
    runtime: main
    name: repl
  run:
    enabled: true
    runtime: main
    name: run
  tui:
    enabled: true
    runtime: main
    name: tui
  jsverbs:
    enabled: true
    runtime: main
    name: verbs
```

`runtime` must reference an existing runtime profile when the command is enabled. `name` controls the command name exposed by the generated binary.

The `repl` command spec controls the JavaScript evaluation command. When enabled, `commands.repl.name` is the command name exposed by the generated binary and `commands.repl.runtime` selects the runtime profile used by that command.

The `run` command spec controls file execution. It creates a fresh runtime from the selected profile and executes the given JavaScript file with script-local module roots, so sibling `require("./helper")` calls resolve next to the script.

The `tui` command spec controls the interactive Bubble Tea REPL. It uses the selected runtime profile for `require()` visibility and is intended for terminal sessions; automated tests should validate command construction or help output rather than launching the interactive program.

## Target modes

Pure xgoja mode creates a standalone generated root command.

```yaml
target:
  kind: xgoja
  output: dist/myapp
```

Cobra mode imports an existing target package and attaches xgoja commands to its root command.

```yaml
target:
  kind: cobra
  import: example.com/myapp
  root: NewRootCommand
  output: dist/myapp
```

Adapter mode imports a target package that builds the root with access to an xgoja host.

```yaml
target:
  kind: adapter
  import: example.com/myapp/xgojaadapter
  output: dist/myapp
```

Adapter packages expose a function compatible with:

```go
Build(context.Context, *app.Host) (*cobra.Command, error)
```

## JavaScript verb sources

`jsverbs` supports three distinct source modes.

### Runtime filesystem source

Use this during development when the generated binary should scan files from disk each time it starts.

```yaml
jsverbs:
  - id: local-dev
    path: ./verbs
    embed: false
```

The files must exist at runtime.

### Embedded local source

Use this when local files should become part of the generated binary.

```yaml
jsverbs:
  - id: local
    path: ./verbs
    embed: true
```

At build time, xgoja copies the directory into the generated workspace under `xgoja_embed/jsverbs/<id>/`. Generated source embeds it with `go:embed`, and the runtime scans the embedded filesystem.

The original `path` must exist when `xgoja build` runs. It does not need to exist when the generated binary runs.

### Provider-shipped source

Use this when verbs live inside a provider package.

```yaml
jsverbs:
  - id: provider-defaults
    package: fixture
    source: verbs
```

The provider must register the source with an `fs.FS`:

```go
//go:embed verbs/*.js
var verbsFS embed.FS

func Register(registry *providerapi.Registry) error {
    return registry.Package("fixture",
        providerapi.VerbSource{Name: "verbs", FS: verbsFS, Root: "verbs"},
    )
}
```

Provider-shipped sources are selected by package ID and source name.

## Validation

Run validation before building:

```bash
xgoja doctor -f xgoja.yaml
```

The validator checks supported target kinds, package uniqueness, known runtime package IDs, duplicate runtime aliases, enabled command runtime references, jsverb source IDs, and local paths for embedded sources.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `unknown package id` | A runtime or jsverb source references a package not listed in `packages`. | Add the package entry or fix the ID. |
| `duplicate alias` | Two modules in one runtime resolve to the same `require()` name. | Set distinct `as` values. |
| embedded source path error | `embed: true` uses a path missing at build time. | Fix `path` relative to the spec file or use an absolute path. |
| provider verb source has no filesystem | The provider registered metadata but no `FS`. | Register `providerapi.VerbSource{FS: ..., Root: ...}`. |
| generated build cannot resolve `github.com/go-go-golems/go-go-goja v0.0.0` | You are running xgoja from source, so no released module version is recorded in the binary. | Pass `--xgoja-replace /path/to/go-go-goja` while developing locally, or build with a released xgoja binary. |
| generated build fails | The generated module cannot resolve imports or replacements. | Re-run with `--keep-work` and inspect generated `go.mod` and `main.go`. |

## See also

- `overview` for the system model.
- `tutorial` for a runnable workflow.
