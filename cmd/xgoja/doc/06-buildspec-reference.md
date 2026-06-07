---
Title: "xgoja buildspec quick reference"
Slug: buildspec-reference
Short: "Quick pointers for xgoja.yaml fields; see user-guide for the full reference."
Topics:
- xgoja
- buildspec
- yaml
Commands:
- xgoja
- xgoja build
- xgoja doctor
- help
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

Use `xgoja help user-guide` for the full buildspec reference.

The most common `xgoja.yaml` shape is:

```yaml
name: my-app
appName: my-app      # optional: generated application identity for logging/config conventions
envPrefix: MY_APP    # optional: enables MY_APP_* environment variables for command fields
target:
  kind: xgoja
  output: dist/my-app
packages:
  - id: go-go-goja-core
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/core
modules:
  - package: go-go-goja-core
    name: path
    as: path
commands:
  eval:
    enabled: true
  run:
    enabled: true
assets:
  - id: app-assets
    path: ./assets
    embed: true
help:
  sources:
    - id: project-docs
      path: ./docs/help
      embed: true
```

Help sources add Glazed Markdown pages to the generated binary's `help` command. Use provider-shipped docs for package API references:

```yaml
help:
  sources:
    - id: loupedeck-runtime-api
      package: loupedeck
      source: runtime-api
```

Use embedded local docs for project-specific tutorials. The local directory is resolved relative to the `xgoja.yaml` file, copied into `xgoja_embed/help/<id>/` during generation, and does not need to exist when the generated binary runs. For a runnable provider-shipped help example, see `examples/xgoja/09-provider-shipped-help-docs`.

Path resolution rules to remember:

- `packages[].replace`, embedded `jsverbs[].path`, embedded `help.sources[].path`, and embedded `assets[].path` are resolved relative to the directory containing the `xgoja.yaml` file.
- Runtime filesystem `jsverbs[].path` is checked relative to the spec directory during validation, but is stored as written in the generated binary; use an absolute path if runtime launch directories vary.
- `target.output` is resolved by `xgoja build` from the shell's current working directory, unless it is absolute or overridden with `--output`.
- `commands.jsverbs.name` defaults to `verbs`. JavaScript verbs are mounted below that command by default.
- Set `commands.jsverbs.mount: root` (also accepts `/` or `.`) to mount discovered JavaScript verb packages directly under the generated root command.

Use embedded assets for templates, static data, static web files, and default configuration that JavaScript should read from the final binary:

```yaml
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

Then JavaScript can use `require("fs:assets")` for read-only embedded files and `require("fs:host")` for explicitly allowed host filesystem access.

Use `go.env` when `go build` needs additional environment variables. This is primarily for CGO-enabled packages whose linker flags are too specific to bake into `ldflags`. For example, a Bleve vector-search binary that links FAISS can declare both build tags and CGO linker flags:

```yaml
go:
  tags:
    - vectors
  ldflags:
    - -r
    - /usr/local/lib
  env:
    CGO_LDFLAGS: "-L/usr/local/lib -lfaiss_c -lfaiss -lstdc++ -lm"
```

Use `go.imports` when generated code must compile an additional Go package even though the generated source does not reference a Go identifier from it. The most common case is a SQL driver package that registers itself through `init()` and is later selected by the database module's runtime `driverName` config:

```yaml
go:
  version: "1.26.1"
  module: xgoja.generated/db-app
  imports:
    - import: github.com/lib/pq
      alias: _
      version: v1.10.9
packages:
  - id: go-go-goja-host
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/host
modules:
  - package: go-go-goja-host
    name: db
    as: db
    config:
      driverName: postgres
      dataSourceName: ${DATABASE_URL}
```

`go.imports[].alias` is optional. Use `_` for side-effect imports such as SQL drivers, `.` only when a package intentionally needs a dot import, and a normal Go identifier for named imports used by custom templates. `go.imports[].version` adds the module to the generated `go.mod`; if the module path differs from the import path, set `go.imports[].module` explicitly:

```yaml
go:
  imports:
    - import: example.com/acme/hooks/register
      alias: _
      module: example.com/acme/hooks
      version: v0.2.0
```

Keep the distinction clear: `go.imports` is compile-time linking, while `modules[].config.driverName` is runtime database configuration.

`go.module` names the generated host module. It defaults to `xgoja.generated/<name>`. For checked-in generated hosts, set it explicitly so the nested module looks intentional:

```yaml
go:
  module: github.com/acme/project/cmd/my-app
```

When packaging that checked-in generated host with GoReleaser, build from the nested module directory:

```yaml
builds:
  - id: my-app-linux
    dir: cmd/my-app
    main: .
    binary: my-app
```

Use `main: ./cmd/my-app` only when `cmd/my-app` is a package inside the parent module and does not contain its own `go.mod`.

Asset entries currently support only `id`, `path`, `embed`, and `description`. `include` and `exclude` filters are intentionally rejected until the generator applies them; otherwise excluded secrets or build artifacts could still be bundled silently.

For importable runtime package generation, use `target.kind: package`:

```yaml
target:
  kind: package
  output: internal/xgojaruntime
  package: xgojaruntime
```

Generate it with `xgoja generate -f xgoja.yaml`. Package generation writes `xgoja_runtime.gen.go` and embedded `xgoja_embed/...` resources into the output directory, but it does not create a temporary module or compile a binary. See `examples/xgoja/14-generated-runtime-package` and `xgoja help tutorial-generated-runtime-package`.

For static HTTP serving, add the HTTP provider and its `express` module:

```yaml
packages:
  - id: go-go-goja-host
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/host
  - id: go-go-goja-http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
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
  - package: go-go-goja-http
    name: express
```

JavaScript can then serve embedded files directly:

```js
const express = require("express")
const assets = require("fs:assets")
const app = express.app()
app.staticFromAssetsModule("/static", assets, "/app/public")
```

Use `run --keep-alive` for server setup scripts so the runtime stays alive after route registration. See `examples/xgoja/10-embedded-assets-fs` and `xgoja help tutorial-static-assets-http-server`.

To expose HTTP setup functions as generated JavaScript verb commands, add the HTTP provider's `serve` command provider and configure jsverb sources:

```yaml
command_providers:
  - id: http-serve
    package: go-go-goja-http
    name: serve
    mount: serve
jsverbs:
  - id: local
    path: verbs
```

The resulting generated command shape is `./dist/app serve <package> <verb> --http-listen 127.0.0.1:8787`. The selected verb registers Express routes and the provider-backed command keeps the runtime alive. See `examples/xgoja/13-http-serve-jsverbs` and `xgoja help tutorial-http-serve-jsverbs`.

Generated binaries can opt into Glazed-style environment variable parsing with `appName` or `envPrefix`:

```yaml
name: my-app
appName: my-app
# envPrefix defaults to MY_APP when omitted; set it explicitly when you need a stable namespace.
envPrefix: MY_APP
```

`appName` is the human/application identity used by the generated root framework. `envPrefix` is the shell-safe environment namespace. If `envPrefix` is omitted and `appName` is set, xgoja derives a shell-safe prefix by uppercasing and converting separators such as `-`, `.`, `_`, and spaces to underscores. Existing specs with neither field keep the historical behavior: only CLI flags, positional arguments, and field defaults are parsed.

Environment variables use Glazed's section-prefix plus field-name convention. For example, the fixture provider's `fixture` section has prefix `fixture-` and field `value`, so this YAML:

```yaml
appName: env-fixture
```

allows:

```bash
ENV_FIXTURE_FIXTURE_VALUE=from-env ./dist/env-fixture eval 'fixtureValue'
```

CLI flags still have higher precedence than environment variables.

Generated binaries can also opt into Glazed-style config file loading:

```yaml
name: my-app
appName: my-app
config:
  enabled: true
  layers:
    - cwd
    - explicit
  fileName: config.yaml
```

Config files use Glazed's standard section map shape:

```yaml
section-slug:
  field-name: value
```

For example, if a provider contributes a section with slug `fixture`, prefix `fixture-`, and field `value`, a config file can set it with:

```yaml
fixture:
  value: from-config-file
```

The equivalent environment variable for `envPrefix: DEMO` is:

```bash
DEMO_FIXTURE_VALUE=from-env
```

The equivalent CLI flag is:

```bash
./dist/my-app eval --fixture-value from-flag 'fixtureValue'
```

Config precedence is:

```text
field defaults < config files < environment variables < positional args / CLI flags
```

Supported `config.layers` values:

| Layer | Discovery |
| --- | --- |
| `system` | `/etc/<appName>/config.yaml` |
| `xdg` | `$XDG_CONFIG_HOME/<appName>/config.yaml` (usually `~/.config/<appName>/config.yaml`) |
| `home` | `~/.<appName>/config.yaml` |
| `git-root` | `<git-root>/<fileName>` |
| `cwd` | `<current-working-directory>/<fileName>` |
| `explicit` | Path supplied with the Glazed `--config-file` flag |

The `system`, `xdg`, and `home` layers require `appName` because they are app-scoped locations. The `cwd`, `git-root`, and `explicit` layers can be used without `appName`. Explicit config files are only loaded when `explicit` is listed in `config.layers`; passing `--config-file` has no effect unless that layer is enabled.

For a runnable config/env example, see `examples/xgoja/11-config-env`.

Validate and build with:

```bash
xgoja doctor -f xgoja.yaml
xgoja build -f xgoja.yaml --xgoja-replace /path/to/go-go-goja
```
