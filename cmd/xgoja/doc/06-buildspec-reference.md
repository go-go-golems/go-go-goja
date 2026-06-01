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

Validate and build with:

```bash
xgoja doctor -f xgoja.yaml
xgoja build -f xgoja.yaml --xgoja-replace /path/to/go-go-goja
```
