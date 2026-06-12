---
Title: "xgoja user guide and v2 spec reference"
Slug: user-guide
Short: "Reference for native xgoja/v2 providers, runtime modules, sources, commands, artifacts, and generated outputs."
Topics:
- xgoja
- xgoja-v2
- providers
- jsverbs
- help-system
- goja
Commands:
- xgoja build
- xgoja doctor
- xgoja list-modules
- xgoja generate
- xgoja gen-dts
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

`xgoja.yaml` is now a native `schema: xgoja/v2` document. The v2 schema is an intent-level runtime compiler input: it selects Go provider packages, chooses runtime modules, describes source sets, exposes command surfaces, and declares generated artifacts.

Legacy v1 specs are accepted only by `xgoja migrate-spec`. Normal commands such as `doctor`, `build`, `generate`, `gen-dts`, and `list-modules` expect `schema: xgoja/v2`.

## Minimal v2 spec

```yaml
schema: xgoja/v2
name: fixture

go:
  version: "1.26"
  module: xgoja.generated/fixture

providers:
  - id: fixture
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/testprovider
    register: Register

runtime:
  modules:
    - provider: fixture
      name: hello
      as: hello

commands:
  - id: eval
    type: builtin.eval
    name: eval
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
xgoja build -f xgoja.yaml
```

## Top-level fields

| Field | Purpose |
| --- | --- |
| `schema` | Must be `xgoja/v2`. |
| `name` | Logical generated application name. |
| `app` | Optional generated app identity, env prefix, and config-file settings. |
| `go` | Generated module metadata, build tags, ldflags, env, and extra imports. |
| `workspace` | Go workspace/module resolution behavior. |
| `providers` | Go packages that register runtime modules, command sets, and provider-owned sources. |
| `runtime.modules` | Runtime modules exposed to goja via `require()`. |
| `sources` | Local, workspace, or provider source sets for jsverbs, scripts, help, and assets. |
| `commands` | Built-in or provider command surfaces exposed by the generated CLI. |
| `artifacts` | Generated outputs such as binaries, runtime packages, declarations, and embedded assets. |

For the complete field-level reference, use `xgoja help xgoja-v2-reference`.

## Providers and runtime modules

Providers are Go packages imported by generated code. A provider entry identifies the package and registration function:

```yaml
providers:
  - id: http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
    register: Register
```

Runtime modules select modules registered by those providers:

```yaml
runtime:
  modules:
    - provider: http
      name: express
      as: express
```

The `as` value is the JavaScript `require()` alias. If omitted, xgoja uses the module name.

## Source sets

Sources describe files that xgoja scans, embeds, or exposes at runtime.

```yaml
sources:
  - id: verbs
    kind: jsverbs
    from:
      dir: ./verbs
    language: typescript
    compile:
      mode: runtime
      bundle: true
```

Common `kind` values:

- `jsverbs`: JavaScript/TypeScript files scanned for `__package__` and `__verb__` declarations.
- `help`: Glazed Markdown help pages.
- `assets`: static files embedded into the generated binary.
- `script`: script inputs for runtime/script-oriented planning.

## Commands

Built-in command surfaces are list entries:

```yaml
commands:
  - id: eval
    type: builtin.eval
    name: eval
  - id: run
    type: builtin.run
    name: run
  - id: repl
    type: builtin.repl
    name: repl
  - id: verbs
    type: builtin.jsverbs
    name: verbs
    sources: [verbs]
```

Provider command sets use `provider.command-set`:

```yaml
commands:
  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources: [verbs]
```

## Artifacts

A binary artifact declares the default output for `xgoja build`:

```yaml
artifacts:
  - id: binary
    type: binary
    output: dist/my-app
    sources: [verbs, docs]
```

When a binary/runtime-package/source/template-style artifact lists jsverb or help source IDs in `sources`, xgoja copies those local source sets into the generated embedded filesystem.

Static assets use a separate artifact:

```yaml
artifacts:
  - id: web-assets
    type: embedded-assets
    sources: [web-dist]
```

TypeScript declarations can be declared with a `dts` artifact:

```yaml
artifacts:
  - id: types
    type: dts
    output: js/types/xgoja-modules.d.ts
    strict: true
```

## TypeScript jsverbs

TypeScript source sets are compiled with xgoja's Go/esbuild pipeline. Runtime module aliases are treated as externals so code can import provider-backed modules:

```ts
import express from "express"

__package__({ name: "sites" })
__verb__("health", { name: "health", output: "text" })
export function health() {
  const app = express.app()
  app.get("/healthz", (_req, res) => res.send("ok"))
}
```

See `examples/xgoja/15-typescript-jsverbs` for a runnable example.

## Migration

Convert a legacy v1 spec with:

```bash
xgoja migrate-spec -f xgoja.yaml --out xgoja.v2.yaml
xgoja doctor -f xgoja.v2.yaml
```

After migration, normal commands should use the v2 file. Legacy v1 loading is intentionally quarantined to `xgoja migrate-spec`.
