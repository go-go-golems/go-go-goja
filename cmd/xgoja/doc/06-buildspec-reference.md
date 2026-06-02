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
runtimes:
  main:
    modules:
      - package: go-go-goja-core
        name: path
        as: path
commands:
  eval:
    enabled: true
    runtime: main
  run:
    enabled: true
    runtime: main
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

Use embedded local docs for project-specific tutorials. The local directory is copied into `xgoja_embed/help/<id>/` during generation and does not need to exist when the generated binary runs. For a runnable provider-shipped help example, see `examples/xgoja/09-provider-shipped-help-docs`.

Use embedded assets for templates, static data, static web files, and default configuration that JavaScript should read from the final binary:

```yaml
assets:
  - id: app-assets
    path: ./assets
    embed: true
runtimes:
  main:
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

Asset entries currently support only `id`, `path`, `embed`, and `description`. `include` and `exclude` filters are intentionally rejected until the generator applies them; otherwise excluded secrets or build artifacts could still be bundled silently.

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
runtimes:
  main:
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
