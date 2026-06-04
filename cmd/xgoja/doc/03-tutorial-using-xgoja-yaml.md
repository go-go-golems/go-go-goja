---
Title: "Tutorial: using xgoja.yaml"
Slug: tutorial-using-xgoja-yaml
Short: "Build and run a generated xgoja binary with provider modules and JavaScript verbs."
Topics:
- xgoja
- tutorial
- goja
- jsverbs
- providers
Commands:
- xgoja build
- xgoja doctor
- xgoja list-modules
Flags:
- --file
- --keep-work
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial shows the normal xgoja workflow: write a build spec, validate it, inspect selected modules, build a generated binary, and run JavaScript through the generated runtime.

The examples assume a provider package already exists and exposes a module named `hello` under provider package ID `fixture`. Replace the import path and module names with your own provider package.

## 1. Write xgoja.yaml

Create `xgoja.yaml`:

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
  eval:
    enabled: true
    runtime: repl
    name: eval
  run:
    enabled: true
    runtime: repl
    name: run
  repl:
    enabled: true
    runtime: repl
    name: repl
  jsverbs:
    enabled: true
    runtime: repl
    name: verbs
```

The `packages` section controls which Go provider package is compiled into the generated binary. The `runtimes.repl.modules` section controls which modules are available through `require()` for this runtime profile.

## 2. Validate the spec

Run:

```bash
xgoja doctor -f xgoja.yaml
```

Fix all reported errors before building. The most common errors are unknown package IDs, missing target imports for target modes, missing embedded verb paths, and duplicate runtime aliases.

## 3. Inspect selected modules

Run:

```bash
xgoja list-modules -f xgoja.yaml
```

This reports modules selected by the spec. Use it to confirm that the runtime profile exposes the expected `require()` names.

## 4. Build the binary

Run from an installed xgoja binary:

```bash
xgoja build -f xgoja.yaml --keep-work
```

When testing from a local checkout before a release, point generated builds at that checkout:

```bash
xgoja build -f xgoja.yaml --keep-work --xgoja-replace /path/to/go-go-goja
```

`--keep-work` leaves the generated workspace on disk. Use it while learning or debugging because it lets you inspect generated `go.mod`, generated `main.go`, copied embedded jsverb files, and the embedded spec JSON.

The output binary is written to `target.output`, for example:

```bash
./dist/fixture
```

## 5. Evaluate JavaScript

Run a simple expression against the generated runtime:

```bash
./dist/fixture eval 'require("hello").greet("intern")'
```

Execute a JavaScript file with the generated `run` command:

```bash
cat > script.js <<'EOF'
const hello = require("hello")
console.log(hello.greet("file"))
EOF
./dist/fixture run script.js
```

The generated binary creates a fresh goja runtime, registers the modules selected by the command's runtime profile, evaluates the source, and prints a non-null result. The `run` command also adds script-local module roots so `require("./helper")` resolves relative to the script file.

For an interactive terminal session, run:

```bash
./dist/fixture repl
```

The REPL command starts a Bubble Tea terminal UI backed by the same runtime-profile module policy.

## 6. Optional: enable env vars and config files for command fields

Generated Glazed commands can read field values from config files and environment variables when the buildspec opts in.

Add app identity, an environment prefix, and config layers to `xgoja.yaml`:

```yaml
name: fixture
appName: fixture
envPrefix: FIXTURE
config:
  enabled: true
  layers:
    - cwd
    - explicit
  fileName: config.yaml
```

Create `config.yaml` in the working directory where you run the generated binary. Config files use Glazed's section-map format:

```yaml
fixture:
  value: from-config-file
```

Run with the current-directory config file:

```bash
./dist/fixture eval 'fixtureValue'
```

Override the same field with an environment variable:

```bash
FIXTURE_FIXTURE_VALUE=from-env ./dist/fixture eval 'fixtureValue'
```

Override both config and environment with a CLI flag:

```bash
FIXTURE_FIXTURE_VALUE=from-env ./dist/fixture eval --fixture-value from-flag 'fixtureValue'
```

Load an explicit config path only when `explicit` is present in `config.layers`:

```bash
./dist/fixture eval --config-file config.yaml 'fixtureValue'
```

The effective precedence is: field defaults, then config files, then environment variables, then positional args and CLI flags. For a runnable example, see `examples/xgoja/11-config-env`.

## 7. Add a runtime filesystem jsverb source

Create `verbs/tools.js`:

```js
__package__({ name: "tools" })
__verb__("greet", {
  name: "greet",
  output: "text",
  fields: {
    name: { type: "string", required: true }
  }
})
function greet(name) {
  const hello = require("hello")
  return hello.greet(name)
}
```

Add this to `xgoja.yaml`:

```yaml
jsverbs:
  - id: local-dev
    path: ./verbs
    embed: false
```

Build again and run:

```bash
./dist/fixture verbs tools greet --name intern
```

This mode scans `./verbs` from disk when the generated binary starts. Runtime filesystem paths are stored as written in the generated spec, so prefer an absolute path when the binary may be launched from a different directory. The validator still checks the path relative to the `xgoja.yaml` file during `xgoja doctor` and `xgoja build`.

## 8. Embed local jsverbs into the generated binary

Change the source to:

```yaml
jsverbs:
  - id: local
    path: ./verbs
    embed: true
```

Build again:

```bash
xgoja build -f xgoja.yaml --keep-work
```

xgoja resolves `./verbs` relative to the `xgoja.yaml` file, copies it into the generated workspace, rewrites the embedded spec path to `xgoja_embed/jsverbs/<id>`, and generated `main.go` embeds it with `go:embed`. The final binary no longer needs the original `./verbs` directory at runtime.

Embedded and runtime filesystem verbs are mounted below the configured jsverbs command by default. The default command is `verbs`, so the example is `./dist/fixture verbs tools greet --name intern`.

For a self-contained helper binary, you can mount discovered JavaScript verb packages directly under the generated root command:

```yaml
commands:
  jsverbs:
    enabled: true
    runtime: main
    mount: root
```

With that option, the same verb becomes `./dist/fixture tools greet --name intern`. Use root mounting when the JavaScript verbs are the primary user-facing command surface. Keep the default `verbs` container when you want to avoid collisions with built-in commands.

## 9. Embed assets and read them through fs aliases

Create an asset file:

```bash
mkdir -p assets/config
printf '{"name":"fixture","ok":true}\n' > assets/config/default.json
```

Select the guarded host provider and register the fs module twice: once for read-only embedded assets and once for explicit host filesystem access.

```yaml
packages:
  - id: go-go-goja-host
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/host
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

Read the embedded file from JavaScript:

```bash
./dist/fixture eval 'require("fs:assets").readFileSync("/app/config/default.json", "utf8")'
```

`as` is the actual `require()` name. The example registers `fs:assets` and `fs:host`; it does not register plain `fs` unless you add another module entry with `as: fs` or omit `as`.

For web/static asset trees, include dot directories such as `.well-known` directly in `./assets`; generated asset embeds use Go's `all:` pattern so those files are preserved. Do not use `include` or `exclude` fields in `assets:` yet. They are rejected until the generator enforces filtering.

## 10. Serve embedded assets with the HTTP provider

Add the HTTP provider package and select its `express` module in the same runtime profile as `fs:assets`:

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

Create a setup script:

```js
const express = require("express")
const assets = require("fs:assets")

const app = express.app()
app.staticFromAssetsModule("/static", assets, "/app/public")
app.get("/", (_req, res) => res.redirect("/static/"))
```

Run it with `--keep-alive` so the generated runtime stays alive after registering routes:

```bash
./dist/fixture run scripts/server.js --http-listen 127.0.0.1:8787 --keep-alive
```

Open `http://127.0.0.1:8787/static/`. For a complete runnable project, see `examples/xgoja/10-embedded-assets-fs` or `xgoja help tutorial-static-assets-http-server`.

## 11. Use provider-shipped jsverbs

Provider packages can ship JS verbs next to their native Go modules.

Provider side:

```go
//go:embed verbs/*.js
var verbsFS embed.FS

func Register(registry *providerapi.ProviderRegistry) error {
    return registry.Package("fixture",
        providerapi.Module{Name: "hello", NewModuleFactory: newHelloModule},
        providerapi.VerbSource{Name: "verbs", FS: verbsFS, Root: "verbs"},
    )
}
```

Spec side:

```yaml
jsverbs:
  - id: provider-defaults
    package: fixture
    source: verbs
```

This scans the provider's embedded filesystem, not the local project filesystem.

## 12. Debug generated builds

Use:

```bash
xgoja build -f xgoja.yaml --keep-work
```

Then inspect the generated workspace:

```text
go.mod
main.go
xgoja.gen.json
xgoja_embed/jsverbs/...
xgoja_embed/assets/...
```

If the generated build fails, the generated source usually shows whether the problem is an import path, module version, replace path, target function name, or embedded source path.

## Troubleshooting

| Problem | What to check |
| --- | --- |
| `doctor` reports unknown package | The `package` field must match a `packages[].id`. |
| `require("name")` fails | The runtime profile must include a module with `as: name` or `name: name`. |
| `require("fs")` fails after adding `fs:assets` | `as: fs:assets` registers only `fs:assets`; use that name or add a separate `as: fs` entry. |
| embedded asset write fails with `EROFS` | Embedded assets are read-only; use a separate host alias such as `fs:host` for writes. |
| jsverb command is missing | Confirm `commands.jsverbs.enabled: true` and that the source scans without diagnostics. |
| embedded jsverb missing | Build with `--keep-work` and inspect `xgoja_embed/jsverbs/<id>/`; remember the source `path` was resolved relative to the spec file directory. |
| provider source missing | Confirm the provider registers `VerbSource{Name, FS, Root}` and the spec uses the same package ID and source name. |
| want jsverbs at root | Set `commands.jsverbs.mount: root`; use a Go `commandProviders` entry instead when commands need non-JavaScript behavior or custom Go services. |

## See also

- `overview` for the architecture.
- `buildspec` for the complete spec reference.
