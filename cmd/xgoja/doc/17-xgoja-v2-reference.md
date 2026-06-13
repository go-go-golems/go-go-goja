---
Title: "xgoja/v2 configuration reference"
Slug: xgoja-v2-reference
Short: "Reference for native xgoja/v2 providers, runtime modules, sources, commands, artifacts, and workspace planning."
Topics:
- xgoja
- v2
- providers
- jsverbs
- typescript
- workspace
Commands:
- xgoja doctor
- xgoja build
- xgoja gen-dts
- xgoja migrate-spec
Flags:
- --file
- --output
- --out
- --xgoja-replace
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

`xgoja/v2` is the native configuration schema for planner-backed xgoja builds.
It describes provider packages, selected Go-backed runtime modules,
goja-executed source sets, command surfaces, generated artifacts, and local Go
workspace behavior.

The central rule is simple and strict: xgoja compiles or bundles code that runs
inside goja. Browser applications, frontend bundles, workers, and other
non-goja JavaScript outputs should be built by their own tools and included as
asset directories.

## Minimal binary

```yaml
schema: xgoja/v2
name: fixture

app:
  name: fixture

go:
  module: xgoja.generated/fixture
  version: "1.26"

workspace:
  mode: auto

providers:
  - id: core
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/core

runtime:
  modules:
    - provider: core
      name: path
      as: path

commands:
  - id: run
    type: builtin.run
    name: run

artifacts:
  - id: binary
    type: binary
    output: dist/fixture
```

Validate and build it with:

```bash
xgoja doctor -f xgoja.yaml
xgoja build -f xgoja.yaml --output dist/fixture
```

## Top-level fields

| Field | Meaning |
| --- | --- |
| `schema` | Must be `xgoja/v2` for native v2 files. |
| `name` | Application/spec name. Defaults are derived from this name when possible. |
| `app` | Runtime application identity and config-file options. |
| `go` | Generated Go module settings and extra generated imports. |
| `workspace` | Local Go module resolution behavior. |
| `providers` | Go packages that contribute xgoja capabilities. |
| `runtime.modules` | Go-backed CommonJS modules selected into the runtime. |
| `sources` | Source sets for jsverbs, scripts, help, and assets. |
| `commands` | User-facing command surfaces. |
| `artifacts` | Generated outputs such as binaries and declaration files. |
| `profiles` | Optional profile overrides. Current support is intentionally small. |

## Application identity

```yaml
app:
  name: my-tool
  envPrefix: MY_TOOL
  configFile:
    enabled: true
    layers: [system, user, project]
    fileName: my-tool.yaml
```

`app.name` is the generated application identity. `app.envPrefix` is used for
environment-variable backed command fields. `app.configFile` enables the
existing Glazed config-file integration in generated commands.

## Go module settings

```yaml
go:
  module: xgoja.generated/my-tool
  version: "1.26"
  tags: [sqlite]
  ldflags: ["-s", "-w"]
  env:
    CGO_ENABLED: "1"
  imports:
    - import: github.com/acme/project/internal/xgoja/extra
      module: github.com/acme/project
      version: v0.3.0
```

`go.module` is the module path for generated build workspaces. `go.version`
defaults to `1.26`. `go.imports` adds extra Go imports required by generated
hosts. When `go.imports[].module` is omitted, xgoja infers the module root from
the import path.

## Workspace resolution

```yaml
workspace:
  mode: auto # auto | off | path
  file: ../../go.work
```

Workspace planning is build-time behavior. It does not enter the generated
runtime spec.

- `auto` searches upward from the spec directory for `go.work` and uses matching
  local modules.
- `off` ignores `go.work` and uses versions or explicit replacements only.
- `path` uses `workspace.file` explicitly.

Resolution precedence is:

1. explicit provider module replacement;
2. CLI replacement such as `--xgoja-replace`;
3. matching local module from `go.work`;
4. versioned module requirement.

`xgoja doctor` reports module-resolution rows so you can inspect the selected
module path, local directory, version, resolution kind, and source before build.

## Providers

```yaml
providers:
  - id: http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
    register: Register
    module:
      version: v0.1.0
      replace: ../go-go-goja
```

A provider is a Go package that registers xgoja capabilities. It can contribute
Go-backed runtime modules, command sets, jsverb sources, TypeScript descriptors,
help sources, assets, host services, and runtime initializers.

`id` is the local spec identifier used by `runtime.modules`, `sources`, and
`commands`. `import` is the Go package import path. `register` defaults to
`Register`. `module.version` and `module.replace` control generated `go.mod`
requirements when workspace resolution does not provide a better local module.

## Runtime modules

```yaml
runtime:
  modules:
    - provider: http
      name: express
      as: express
      config:
        debug: true
```

Runtime modules are Go-backed CommonJS modules. JavaScript or TypeScript source
imports them with `require("express")` or an equivalent compiled import.

The planner derives runtime module aliases from `runtime.modules`. TypeScript
source sets do not need to repeat those aliases under a separate `external`
field. During bundling, xgoja preserves those imports so the Go-backed module is
resolved by the goja runtime.

## Sources

A source set is a named group of files. It has a kind, an origin, optional
filters, language metadata, and optional compile intent.

```yaml
sources:
  - id: local-sites
    kind: jsverbs
    from:
      dir: ./verbs
    include: ["**/*.ts"]
    exclude: ["**/*.test.ts"]
    extensions: [.ts]
    language: typescript
    compile:
      mode: runtime
      bundle: true
```

### Source kinds

| Kind | Meaning |
| --- | --- |
| `jsverbs` | JavaScript or TypeScript files scanned for jsverb command metadata. |
| `script` | Goja-executed script source for run/runtime planning. |
| `help` | Help markdown source files. |
| `assets` | Static asset files. Build frontend/browser outputs outside xgoja, then include them as assets. |

### Source origins

Disk directory:

```yaml
from:
  dir: ./verbs
```

Provider-shipped source:

```yaml
from:
  provider:
    provider: docs
    source: bundled-help
```

Workspace module source:

```yaml
from:
  workspace:
    module: github.com/acme/project
    path: internal/xgoja/verbs
```

Workspace origins require the module to resolve to a local directory.

### TypeScript compile intent

```yaml
language: typescript
compile:
  mode: runtime
  bundle: true
  define:
    __DEV__: "false"
  check:
    command: ["npx", "tsc", "--noEmit"]
```

`mode: runtime` means the generated runtime compiles the source before goja
loads it. `bundle: true` lets TypeScript files import local helpers such as
`./helper`. Provider and embedded TypeScript sources are bundled from their
`fs.FS` root; they do not need to be copied to disk for local helper resolution.

xgoja owns the normal goja compiler profile. Do not put browser or Node bundler
settings such as platform, format, target, package-manager installation, CSS
loaders, or polyfills in v2 source config.

## Commands

Commands are explicit surfaces. Builtin commands and provider command sets use
the same list.

```yaml
commands:
  - id: run
    type: builtin.run
    name: run

  - id: verbs
    type: builtin.jsverbs
    name: verbs
    sources: [local-sites]

  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources: [local-sites]
```

Supported builtin command types are:

- `builtin.eval`
- `builtin.run`
- `builtin.repl`
- `builtin.jsverbs`

`provider.command-set` mounts a command set contributed by a selected provider.
Provider command sets commonly depend on source sets. For example, the HTTP
`serve` provider command uses jsverb sources to register Express routes.

Runtime modules and provider command sets are separate provider outputs. A
runtime module is selected under `runtime.modules` and is imported by JavaScript
code. A command set is selected under `commands` and contributes CLI commands.
The HTTP provider demonstrates the distinction: `express` is the runtime module,
while `serve` is a provider command set that runs a jsverb long enough for the
registered HTTP routes to serve traffic.

The `mount` field on a command controls where the command appears in the
generated CLI command tree. It does not mount an HTTP handler. HTTP handler
mounting happens at runtime through the Express module, for example
`app.mount("/ws", handlerObject)`, or through provider host-service integration.

## Artifacts

Artifacts describe generated outputs.

```yaml
artifacts:
  - id: binary
    type: binary
    output: dist/my-tool
    sources: [local-sites]

  - id: declarations
    type: dts
    output: js/types/xgoja-modules.d.ts
    strict: true

  - id: public-assets
    type: embedded-assets
    sources: [web-dist]
```

Common artifact types are:

| Type | Meaning |
| --- | --- |
| `binary` | Generated xgoja binary. When `sources` lists local jsverb/help source sets, those sources are copied into the generated embedded filesystem. |
| `dts` | TypeScript declaration output for selected runtime modules. |
| `embedded-assets` | Static assets embedded into the generated host. |
| `runtime-package` | Generated runtime package output. |
| `adapter`, `cobra`, `source`, `template` | Additional generated output shapes consumed through the v2 plan-backed generator. |

For binary/runtime-package style artifacts, `sources` marks local jsverb and
help source sets that should be copied into the generated embedded filesystem.
For assets, use a separate `embedded-assets` artifact with `sources` pointing at
asset source IDs.

A `template` artifact is a code-generation output shape. It should not be used
to model runtime behavior such as HTTP serving, WebSocket mounting, or provider
module setup. Runtime behavior belongs in provider packages, runtime modules,
command sets, and host services.

`xgoja gen-dts` uses the first `type: dts` artifact as its default output when `--out` is omitted; `strict: true` on the artifact enables strict declaration checks.

## TypeScript jsverbs example

```yaml
schema: xgoja/v2
name: typescript-jsverbs

providers:
  - id: http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http

runtime:
  modules:
    - provider: http
      name: express

sources:
  - id: sites
    kind: jsverbs
    from:
      dir: ./verbs
    language: typescript
    compile:
      mode: runtime
      bundle: true

commands:
  - id: verbs
    type: builtin.jsverbs
    sources: [sites]

  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources: [sites]

artifacts:
  - id: binary
    type: binary
    output: dist/typescript-jsverbs
    sources: [sites]
```

A verb in `./verbs/site.ts` can import a local helper and the selected runtime
module:

```ts
import { message } from "./message"
import express from "express"

__package__({ name: "sites" })
__verb__("demo", { name: "demo", output: "text" })
export function demo() {
  const app = express.app()
  app.get("/", (_req, res) => res.send(message()))
}
```

The local helper import is bundled. The `express` import is externalized because
it is a selected runtime module alias.

Go-backed modules can also expose mountable HTTP handlers. Express can mount
those handlers while JavaScript remains the composition layer:

```ts
import express from "express"
import sessionstream from "sessionstream"

__package__({ name: "sites" })
__verb__("site", { name: "site", output: "text" })
export function site() {
  const app = express.app()
  const hub = sessionstream.hub({ schemas })

  app.get("/healthz", (_req, res) => res.send("ok"))
  app.mount("/ws", sessionstream.webSocket.server(hub))
}
```

The mounted object must carry the shared `gojahttp` hidden `http.Handler` ref.
This is how a Go-backed transport such as a WebSocket server can be mounted
without reimplementing upgrade handling in JavaScript.

Express route patterns and mounted handlers use different matching semantics:

- `app.get("/users/:id", ...)` captures one segment as `req.params.id`.
- `app.get("/assets/*", ...)` matches the rest of the path but does not expose a splat capture today.
- `app.mount("/ws", handler)` uses prefix matching and gives the request to the Go handler.

## Current limits

The normal command path is v2-plan-native: `doctor`, `build`, `generate`, `gen-dts`, and `list-modules` load `schema: xgoja/v2` and consume `plan.Plan` directly.

Known limits:

- v2 doctor uses a synthetic provider registry for static validation. It cannot fully validate provider package implementation details unless a provider is linked into a generated sidecar or described by future provider manifests.
- Multiple artifacts are not fully orchestrated by `xgoja build`. The first binary-style artifact controls the current build target.
- Provider package import path and Go module path are inferred when a provider does not specify replacement/version metadata.

## Migration policy

Legacy v1 specs remain supported as migration input for `xgoja migrate-spec`.
New examples and docs should use `schema: xgoja/v2`. Normal command paths are
moving toward v2-only behavior.

Use:

```bash
xgoja migrate-spec -f xgoja.yaml --out xgoja.v2.yaml
```

Then validate:

```bash
xgoja doctor -f xgoja.v2.yaml
```
