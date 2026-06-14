---
Title: "Migrating to xgoja/v2"
Slug: migrating-to-xgoja-v2
Short: "Convert legacy xgoja.yaml files to the native xgoja/v2 schema."
Topics:
- xgoja
- migration
- v2
Commands:
- xgoja migrate-spec
- xgoja doctor
- xgoja build
Flags:
- --out
- --in-place
- --backup
- --check
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

xgoja/v2 is the native configuration format for the hard cutover. Legacy v1
files are migration input only. Convert them with `xgoja migrate-spec` before
using v2-era build, doctor, generation, and declaration workflows.

Generated v2 outputs embed `app.RuntimePlan` JSON with schema
`xgoja/runtime/v2`. The old generated runtime bridge shape is not part of the
active runtime contract: providers, runtime modules, unified sources, commands,
and artifacts are the authoritative concepts.

## Core rule

xgoja only compiles or bundles code that runs inside goja. Browser applications,
frontend bundles, web workers, and other non-goja JavaScript outputs should be
built by their own tooling and then included in xgoja as assets.

That means v2 configuration describes intent:

- provider packages;
- selected Go-backed runtime modules;
- goja-executed source sets;
- command surfaces;
- generated artifacts;
- local Go workspace resolution.

It does not expose ordinary fields for browser bundler platform, output format,
JavaScript target, package-manager installation, CSS loaders, SVG loaders, or
polyfills.

## Convert a file

Write a converted file next to the old one:

```bash
xgoja migrate-spec -f xgoja.yaml --out xgoja.v2.yaml
```

Overwrite in place and keep a backup:

```bash
xgoja migrate-spec -f xgoja.yaml --in-place --backup
```

Check whether a file is already rendered as v2:

```bash
xgoja migrate-spec -f xgoja.yaml --check
```

Warnings are printed as stable lines:

```text
warning: jsverbs[0].typescript.external: runtime module alias "express" is derived automatically in v2
```

## Field mapping

| v1 | v2 |
| --- | --- |
| `packages[]` | `providers[]` |
| `packages[].replace` | `providers[].module.replace` with a workspace warning |
| `modules[]` | `runtime.modules[]` |
| `commands.eval/run/repl/jsverbs` | `commands[]` with `type: builtin.*` |
| `commandProviders[]` | `commands[]` with `type: provider.command-set` |
| `jsverbs[]` | `sources[]` with `kind: jsverbs` |
| `jsverbs[].typescript.enabled` | `language: typescript` |
| `jsverbs[].typescript.bundle` | `compile.bundle` |
| `jsverbs[].typescript.checkCommand` | `compile.check.command` |
| `jsverbs[].embed` | source ID listed under the generated artifact's `sources` |
| `help.sources[]` | `sources[]` with `kind: help` |
| `assets[]` | `sources[]` with `kind: assets` and `artifacts[]` when embedded |
| `target` | `artifacts[]` |

## TypeScript migration

A v1 TypeScript source such as:

```yaml
jsverbs:
  - id: local-sites
    path: ./verbs
    extensions: [".ts"]
    typescript:
      enabled: true
      bundle: true
      target: es2015
      format: cjs
      platform: neutral
      external:
        - express
```

becomes:

```yaml
sources:
  - id: local-sites
    kind: jsverbs
    from:
      dir: ./verbs
    extensions: [".ts"]
    language: typescript
    compile:
      mode: runtime
      bundle: true
```

Runtime module aliases such as `express` are derived from `runtime.modules` and
externalized by xgoja automatically. Low-level TypeScript profile fields are not
migrated because xgoja owns the goja runtime compiler profile.

## Runtime modules, provider command sets, and templates

When migrating HTTP applications, keep three concepts separate.

A runtime module is selected under `runtime.modules` and is visible to scripts
through `require(...)` or externalized TypeScript imports:

```yaml
runtime:
  modules:
    - provider: http
      name: express
      as: express
```

A provider command set is selected under `commands` and contributes CLI commands:

```yaml
commands:
  - id: serve
    type: provider.command-set
    provider: http
    name: serve
    mount: serve
    sources: [sites]
```

The `sources` list is command-scoped. Provider command code receives a
`SourceRegistry` limited to those source IDs. If an old application relied on a
provider command seeing every jsverb source, make that dependency explicit by
listing the needed source IDs on the command.

A `template` artifact is generated output. It is not a runtime module and it is
not how HTTP routes or WebSocket handlers are mounted. HTTP serving behavior
belongs in provider command sets, runtime modules, host services, and JavaScript
route setup code.

## External frontend bundles

Build frontend code outside xgoja:

```bash
cd web
pnpm install
pnpm build
```

Then include the output directory as assets:

```yaml
sources:
  - id: web-dist
    kind: assets
    from:
      dir: ./web/dist

artifacts:
  - id: web-assets
    type: embedded-assets
    sources: [web-dist]
```

## Review checklist after migration

1. Run `xgoja migrate-spec -f xgoja.yaml --out xgoja.v2.yaml`.
2. Read all `warning:` lines.
3. Check provider command-set source dependencies, especially commands that use
   jsverbs.
4. For HTTP applications, check both sides of the provider configuration:
   `runtime.modules` should select `express`, while `commands` should select the
   HTTP provider's `serve` command set.
5. Check embedded jsverb/help sources. In v2, generated artifacts list embedded
   executable source sets under `artifacts[].sources`.
6. Check that `template` artifacts are only used for generated output, not for
   runtime HTTP behavior.
7. Check local replacements and prefer `workspace.mode: auto` when a `go.work`
   file already covers the local module.
8. If you generate a runtime package, update host code/docs to use
   `EmbeddedRuntimePlanJSON` and `DecodeRuntimePlan` for direct metadata access;
   `NewBundle` and `Bundle.NewRuntime` remain the preferred APIs.
9. Run `xgoja doctor -f xgoja.v2.yaml` once the v2 planner is available.
10. Run the example or application smoke test.
