---
Title: "xgoja user guide and buildspec reference"
Slug: user-guide
Short: "Reference for xgoja.yaml packages, runtimes, commands, targets, help docs, and jsverb source modes."
Topics:
- xgoja
- buildspec
- providers
- jsverbs
- help-system
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

An `xgoja.yaml` file describes the generated binary. It names the output program, selects provider packages, defines the top-level module set, enables commands, configures JavaScript verb sources, optionally embeds application assets, optionally bundles Glazed help entries, and can opt generated Glazed commands into environment-variable and config-file sources.

A minimal shape looks like this:

```yaml
name: fixture
go:
  version: "1.26"
  module: xgoja.generated/fixture
target:
  kind: xgoja
  output: dist/fixture
packages:
  - id: fixture
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider
    register: Register
modules:
  - package: fixture
    name: hello
    as: hello
commands:
  eval:
    enabled: true
    name: eval
  run:
    enabled: true
    name: run
  repl:
    enabled: true
    name: repl
  jsverbs:
    enabled: true
    name: verbs
```

`go.env` can pass build-time environment variables to `go build`. Use it for CGO-heavy builds that need linker environment in addition to `go.tags` and `go.ldflags`; for example, Bleve vector-search binaries can set `CGO_LDFLAGS` for FAISS. See `xgoja help buildspec-reference` for a complete example.

## Top-level fields

`name` is the generated application name. It defaults to `xgoja-app`.

`appName` is an optional application identity used by the generated root framework and app-scoped config discovery. When omitted, framework setup falls back to `name`.

`envPrefix` is an optional shell-safe environment namespace for generated command fields. If `envPrefix` is omitted and `appName` is set, xgoja derives one by uppercasing `appName`, converting separators such as `-`, `.`, `_`, and spaces to underscores, and prefixing leading digits with `APP_`.

`config` optionally enables Glazed config-file loading for generated commands.

`go` controls the generated module. `go.version` defaults to `1.26`, and `go.module` defaults to `xgoja.generated/<name>`. Set `go.module` explicitly when you check generated host files into a repository and want the nested module to carry a project-owned path.

`target` controls the generated binary shape and output path. `target.output` is resolved by `xgoja build` from the shell's current working directory, unless it is absolute or overridden with `xgoja build --output`. It is not resolved relative to the spec file and it is not written inside the temporary generated module.

`packages` selects Go provider packages that will be imported by generated source.

`modules` defines the generated runtime module set.

`commands` enables generated command families. Runtime-backed commands all use the top-level `modules` list.

`jsverbs` configures JavaScript verb sources. By default, discovered verbs are mounted under the generated jsverbs command. The command name defaults to `verbs` when `commands.jsverbs.enabled: true`; set `commands.jsverbs.name` to rename that container command. Set `commands.jsverbs.mount: root` when discovered verb packages should be registered directly under the generated binary root instead.

`assets` configures local file trees copied into and embedded by the generated binary.

`help.sources` configures additional Glazed help pages loaded into the generated binary's `help` command.

`commandProviders` mounts Glazed command sets supplied by provider packages.

## App identity, environment variables, and config files

Generated commands normally read values from CLI flags, positional arguments, and field defaults. Add `appName`, `envPrefix`, or `config` when generated Glazed commands should also read environment variables or config files.

```yaml
name: config-env-demo
appName: config-env-demo
envPrefix: DEMO
config:
  enabled: true
  layers:
    - cwd
    - explicit
  fileName: config.yaml
```

### Environment variables

If `envPrefix` is set, generated commands include Glazed's environment-variable source middleware. Environment variables use the generated prefix, the command section prefix, and the field name. For example, a provider field in section `fixture` with field `value` can be set as:

```bash
DEMO_FIXTURE_VALUE=from-env ./dist/config-env-demo eval 'fixtureValue'
```

If `envPrefix` is omitted but `appName: config-env-demo` is set, xgoja derives `CONFIG_ENV_DEMO`.

### Config files

Enable config files with `config.enabled: true`. Config files use Glazed's standard section-map shape:

```yaml
fixture:
  value: from-config-file
```

Supported layers are:

| Layer | Discovery |
| --- | --- |
| `system` | `/etc/<appName>/<fileName>` |
| `xdg` | `$XDG_CONFIG_HOME/<appName>/<fileName>` |
| `home` | `~/.<appName>/<fileName>` |
| `git-root` | `<git-root>/<fileName>` |
| `cwd` | `<current-working-directory>/<fileName>` |
| `explicit` | Path supplied with `--config-file` |

The app-scoped layers (`system`, `xdg`, `home`) require `appName`. Local layers (`cwd`, `git-root`, `explicit`) do not. The `--config-file` flag only loads a file when `explicit` appears in `config.layers`.

Effective precedence is:

```text
field defaults < config files < environment variables < positional args / CLI flags
```

For a runnable example, see `examples/xgoja/11-config-env`.

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

A top-level `modules` selects modules from registered provider packages.

```yaml
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

Aliases must be unique within the top-level module set. `as` is the actual `require()` name; it does not also register the original `name`. For example, `name: fs` with `as: fs:assets` registers `require("fs:assets")`, not `require("fs")`.

## Commands

The `commands` section enables generated command families.

```yaml
commands:
  eval:
    enabled: true
    name: eval
  run:
    enabled: true
    name: run
  repl:
    enabled: true
    name: repl
  jsverbs:
    enabled: true
    name: verbs
```

`name` controls the command name exposed by the generated binary. If a built-in command name is omitted, xgoja applies defaults: `eval`, `run`, `repl`, and `verbs` for `commands.jsverbs`. Runtime-backed commands all use the top-level `modules` list.

`commands.jsverbs.mount` controls where discovered JavaScript verb commands are attached. Omit it for the default container command (`./app verbs ...`). Set it to `root`, `/`, or `.` to attach discovered verb packages directly under the binary root (`./app tools greet ...`). Root mounting is convenient for self-contained helper binaries, but it increases the chance of command-name collisions with built-in commands such as `help`, `eval`, `run`, `repl`, and `modules`.

The `eval` command spec controls one-shot JavaScript string evaluation. When enabled, `commands.eval.name` is the command name exposed by the generated binary; the command uses the top-level generated module set.

The `run` command spec controls file execution. It creates a fresh runtime from the generated module set and executes the given JavaScript file with script-local module roots, so sibling `require("./helper")` calls resolve next to the script.

The `repl` command spec controls the interactive Bubble Tea REPL. It uses the generated module set for `require()` visibility and is intended for terminal sessions; automated tests should validate command construction or help output rather than launching the interactive program.

Provider modules may expose Glazed configuration sections for the generated module set. xgoja appends those sections to `run`, `repl`, and `jsverbs` commands and runs provider-owned runtime initializers before JavaScript executes. For example, a provider section with prefix `fixture-` can add flags such as `--fixture-value` to built-in commands.

## Command providers

Provider packages can also ship whole Glazed command sets. The `commandProviders` section selects those providers and mounts them into the generated root command.

```yaml
commandProviders:
  - id: fixture-tools
    package: fixture
    name: tools
    mount: fixture
```

Fields:

- `id` is a unique buildspec-local command provider entry ID.
- `package` references a package ID from `packages`.
- `name` is the command set provider name registered by that package.
- `mount` optionally overrides the provider's default mount path. A single segment such as `fixture` creates commands under `fixture ...`; a slash-delimited path can mount deeper.
- `modules` optionally filters the generated module descriptors passed to the provider-owned command set.

Command providers return Glazed command values, so provider authors can expose `BareCommand`, `WriterCommand`, or `GlazeCommand` commands. xgoja passes the generated module descriptors and a typed runtime factory to the provider so those commands can reuse module-provided sections and runtime initializers.

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

## Release packaging generated hosts

`xgoja build` creates a generated Go module in its build workspace, runs `go mod tidy`, and then runs `go build .` from that generated module root. The workspace is temporary unless you pass `--work-dir` or `--keep-work`.

If a project checks the generated host into a repository as a nested module, release tools must also build from that nested module root. For example, if the generated host lives in `cmd/my-app` and that directory contains its own `go.mod`, configure GoReleaser with `dir: cmd/my-app` and `main: .`:

```yaml
builds:
  - id: my-app-linux
    dir: cmd/my-app
    main: .
    binary: my-app
    goos:
      - linux
    goarch:
      - amd64
```

Do not use `main: ./cmd/my-app` for this nested-module shape. That form asks GoReleaser to build a package inside the parent module; it fails once `cmd/my-app/go.mod` makes the generated host a separate module. A failure such as `main module (...) does not contain package .../cmd/my-app` usually means the release config crossed this module boundary.

For checked-in generated hosts, set a meaningful module path explicitly:

```yaml
go:
  module: github.com/acme/project/cmd/my-app
```

This module path names the generated host module. Provider and runtime dependencies still come from the generated `require` and `replace` entries.

## Help document sources

Generated xgoja binaries always include generic xgoja runtime help, such as `runtime-overview`. The optional `help.sources` section adds package-specific or project-specific Glazed help pages to the same help system.

Use help sources for API references, tutorials, troubleshooting guides, and package-specific runtime documentation. For example, a Loupedeck-oriented generated binary can include the Loupedeck JavaScript API reference and expose it as `help loupedeck-js-api-reference`.

`help.sources` supports three source modes.

### Provider-shipped help source

Use this when documentation lives inside a provider package. This is the preferred mode for package-owned API references because the docs live with the code that owns the API. See `examples/xgoja/09-provider-shipped-help-docs` for a smoke-tested Loupedeck provider example.

```yaml
help:
  sources:
    - id: loupedeck-runtime-api
      package: loupedeck
      source: runtime-api
```

The provider must register the source with an `fs.FS`:

```go
import helpdoc "github.com/go-go-golems/loupedeck/docs/help"

func Register(registry *providerapi.ProviderRegistry) error {
    return registry.Package("loupedeck",
        providerapi.HelpSource{
            Name:        "runtime-api",
            Description: "Loupedeck JavaScript runtime API reference and tutorials",
            FS:          helpdoc.FS(),
            Root:        ".",
        },
    )
}
```

### Embedded local help source

Use this when project-local Markdown files should become part of the generated binary.

```yaml
help:
  sources:
    - id: project-docs
      path: ./docs/help
      embed: true
```

At build time, xgoja copies the directory into the generated workspace under `xgoja_embed/help/<id>/`. Generated source embeds it with `go:embed`, and the generated root loads the pages into the Glazed help system.

The original path must exist when `xgoja build` runs. It does not need to exist when the generated binary runs.

### Runtime filesystem help source

Use this during documentation development when the generated binary should read pages from disk at runtime.

```yaml
help:
  sources:
    - id: dev-docs
      path: ./docs/help
      embed: false
```

The files must exist when the generated binary runs. Prefer `embed: true` for distributed binaries.

Every Markdown file must be a Glazed help entry with frontmatter fields such as `Title`, `Slug`, `Short`, `Topics`, `Commands`, `Flags`, `IsTopLevel`, `IsTemplate`, `ShowPerDefault`, and `SectionType`. Slugs are global within one generated binary, so avoid collisions between built-in, provider, and local docs.

## Embedded assets

`assets` embeds project-local file trees into the generated binary. Use this for templates, fixtures, configuration defaults, static data, and other files that JavaScript should read without requiring the original source directory at runtime.

```yaml
packages:
  - id: go-go-goja-host
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/host
assets:
  - id: app-assets
    path: ./assets
    embed: true
modules:
  - package: go-go-goja-host
    name: fs
    as: fs:assets
    config:
      embedded:
        allow: true
        mounts:
          - asset: app-assets
            mount: /app
  - package: go-go-goja-host
    name: fs
    as: fs:host
    config:
      allow: true
```

At build time, xgoja copies each embedded asset directory into the generated workspace under `xgoja_embed/assets/<id>/`, rewrites the embedded runtime spec to that generated root, and emits `//go:embed all:xgoja_embed/assets/*` in generated `main.go`. The `all:` prefix is intentional: asset trees may contain web-standard dot directories such as `.well-known`.

Embedded assets are exposed through configured fs module instances. Prefer separate aliases:

```js
const assetsFS = require("fs:assets")
const hostFS = require("fs:host")

const config = JSON.parse(assetsFS.readFileSync("/app/config/default.json", "utf8"))
hostFS.writeFileSync("./out.json", JSON.stringify(config), "utf8")
```

`fs:assets` is read-only. Mutating operations under embedded mounts fail with `EROFS`. `fs:host` is separate and requires `config.allow: true`; it is not enabled by `embedded.allow: true`.

`assets[].embed` currently must be `true`. Runtime filesystem assets are intentionally not part of the first embedded-assets implementation. `assets[].include` and `assets[].exclude` are not supported yet and are rejected by validation so the buildspec cannot claim a narrower asset set than the generator embeds.

Long-running scripts, such as Express-style HTTP servers, can use the generated `run` command's `--keep-alive` flag. The script registers routes or static mounts, returns, and xgoja keeps the runtime open until Ctrl-C or SIGTERM:

```bash
./dist/my-app run scripts/server.js --http-listen 127.0.0.1:8787 --keep-alive
```

See `examples/xgoja/10-embedded-assets-fs` and `xgoja help tutorial-static-assets-http-server` for a static-site example that passes `require("fs:assets")` directly to `require("express").app().staticFromAssetsModule("/static", assets, "/app/public")`, so embedded assets are served without a host staging directory.

Generated binaries can also expose HTTP setup functions as JavaScript verbs by enabling the HTTP provider's `serve` command provider. In that mode, `./dist/my-app serve sites demo --http-listen 127.0.0.1:8787` invokes the selected verb once, then keeps the runtime alive for request handling. See `examples/xgoja/13-http-serve-jsverbs` and `xgoja help tutorial-http-serve-jsverbs`.

During development, add `--hot-reload` to the HTTP `serve` command to enable blue/green reloads with last-known-good fallback:

```bash
./dist/my-app serve sites demo \
  --http-listen 127.0.0.1:8787 \
  --hot-reload \
  --hot-reload-watch-root ./sites \
  --hot-reload-smoke-path /healthz
```

Hot reload starts one Go-owned listener, runs each reload in a fresh JavaScript runtime with a fresh route host, smoke-tests the candidate path when configured, and swaps only successful candidates live. Broken edits record an error on the status endpoint and keep serving the previous good runtime. The default status endpoint is `/__xgoja/status`; set `--hot-reload-status-path ""` to disable it.

## Generated runtime packages

Use `xgoja generate` with `target.kind: package` when an existing Go application should import xgoja-generated runtime wiring instead of executing a generated binary.

```yaml
target:
  kind: package
  output: internal/xgojaruntime
  package: xgojaruntime
```

Then run:

```bash
xgoja generate -f xgoja.yaml
```

The generated package exposes `NewBundle`, `NewRuntime`, `NewRuntimeFromSections`, `DecodeSpec`, `RegisterProviders`, and `AttachDefaultCommands`. `NewBundle` accepts `Options{ConfigureServices: func(*app.HostServices) { ... }}` so an embedding Go application can inject host-owned services before provider module setup. For example, an application that owns its `net/http.Server` can inject `httpprovider.ExternalHostService{Host: jsHost, OwnsListen: false}` and let JavaScript register Express routes into that host without the generated runtime binding its own listener.

Package generation writes source into the existing module and does not create `go.mod`, write `main.go`, run `go mod tidy`, or compile a binary. See `examples/xgoja/14-generated-runtime-package` and `xgoja help tutorial-generated-runtime-package`.

For hot reload in embedding applications, use `pkg/xgoja/hotreload` as a blue/green manager: its reload callback builds a fresh generated runtime bundle around a fresh `gojahttp.Host`, bootstraps JavaScript routes, smoke-tests the candidate, and swaps it into service only on success. Broken reloads keep serving the previous runtime.

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

The files must exist at runtime. For runtime filesystem sources, `path` is stored in the generated spec as written and is interpreted by the generated binary at startup. Use an absolute path when the binary may be launched from a different working directory.

### Embedded local source

Use this when local files should become part of the generated binary.

```yaml
jsverbs:
  - id: local
    path: ./verbs
    embed: true
```

At build time, xgoja copies the directory into the generated workspace under `xgoja_embed/jsverbs/<id>/`. Generated source embeds it with `go:embed`, rewrites the embedded spec to point at that generated root, and the runtime scans the embedded filesystem.

The original `path` is resolved relative to the directory containing the `xgoja.yaml` file, not relative to the shell's current working directory. It must exist when `xgoja build` runs. It does not need to exist when the generated binary runs.

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

func Register(registry *providerapi.ProviderRegistry) error {
    return registry.Package("fixture",
        providerapi.VerbSource{Name: "verbs", FS: verbsFS, Root: "verbs"},
    )
}
```

Provider-shipped sources are selected by package ID and source name. The provider source's `Root` is interpreted inside the provider's registered `fs.FS`.

By default, discovered JavaScript verb commands are added below the configured `commands.jsverbs.name` command. A verb package can still define nested command paths with `__package__({ name, parents })`, but those paths are nested inside the jsverbs container command. When `commands.jsverbs.mount: root` is set, the same package/parent path is attached directly to the generated root command instead.

## Validation

Run validation before building:

```bash
xgoja doctor -f xgoja.yaml
```

The validator checks supported target kinds, package uniqueness, known runtime package IDs, duplicate runtime aliases, enabled command runtime references, command provider package/runtime references, jsverb source IDs, help source IDs, asset source IDs, provider help source package references, and local paths for embedded sources. Local `replace`, embedded `jsverbs.path`, embedded `help.sources[].path`, and embedded `assets[].path` entries are checked relative to the spec file directory. Runtime filesystem jsverb paths are also checked relative to the spec file for validation, but the generated binary stores the path as written; use absolute paths for runtime filesystem sources when launch directories vary.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `unknown package id` | A runtime, jsverb source, command provider, or help source references a package not listed in `packages`. | Add the package entry or fix the ID. |
| `duplicate alias` | Two modules in one runtime resolve to the same `require()` name. | Set distinct `as` values. |
| embedded source path error | `embed: true` uses a path missing at build time. | Fix `path` relative to the spec file or use an absolute path. |
| `require("fs")` is missing | The runtime registered `name: fs` with `as: fs:assets` or `as: fs:host`, so only those names exist. | Use `require("fs:assets")` or register an explicit `as: fs` instance. |
| embedded asset write fails with `EROFS` | Embedded assets are read-only. | Write to a separate host fs alias such as `fs:host` with `config.allow: true`. |
| unknown embedded asset | A fs embedded mount references an asset ID not declared in `assets`. | Add the asset declaration or fix the mount's `asset` value. |
| asset `include` or `exclude` is rejected | Asset filtering is not implemented yet. | Remove the field or pre-build a filtered directory before running `xgoja build`. |
| provider verb source has no filesystem | The provider registered metadata but no `FS`. | Register `providerapi.VerbSource{FS: ..., Root: ...}`. |
| provider help source not found | `help.sources[].package` or `help.sources[].source` does not match a registered `providerapi.HelpSource`. | Verify the provider registration and the buildspec package/source names. |
| help topic not found | The docs were not selected, the slug is different, or the page failed to load due to a duplicate slug. | Run `help` to list visible topics and inspect the Markdown frontmatter. |
| command provider not mounted | `commandProviders[].package` or `commandProviders[].name` does not match a registered command set provider, or mounting failed during generated command construction. | Verify the provider registration and run the generated binary with `--help` to inspect the command tree. |
| generated build cannot resolve `github.com/go-go-golems/go-go-goja v0.0.0` | You are running xgoja from source, so no released module version is recorded in the binary. | Pass `--xgoja-replace /path/to/go-go-goja` while developing locally, or build with a released xgoja binary. |
| generated build fails | The generated module cannot resolve imports or replacements. | Re-run with `--keep-work` and inspect generated `go.mod` and `main.go`. |
| `main module ... does not contain package .../cmd/...` in GoReleaser | The generated host directory has its own `go.mod`, so it is a nested module rather than a package in the parent module. | Configure GoReleaser with `dir: <generated-module-dir>` and `main: .`. |

## See also

- `overview` for the system model.
- `tutorial` for a runnable workflow.
