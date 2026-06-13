# xgoja examples

These examples are both runnable smoke tests and a numbered learning path for generated xgoja binaries.

Each example directory has its own `README.md`, `Makefile`, and native `schema: xgoja/v2` `xgoja.yaml` when it is an xgoja build fixture. Start with the first provider example and continue down the list; later examples assume the module-set and provider vocabulary from earlier examples.

## Learning path

1. `01-core-provider/` — safe first-party modules such as `path`, `yaml`, and `crypto`.
2. `02-host-provider/` — guarded host-capability modules such as `fs`, `exec`, and `database` with explicit config.
3. `03-single-runtime-modules/` — one generated binary with one shared module set used by eval, run, repl, and jsverbs.
4. `04-module-sections/` — provider Glazed config sections and runtime initializers on built-in `eval`, `run`, and `jsverbs` commands.
5. `05-command-provider/` — provider-owned Glazed command sets mounted into the generated root command.
6. `06-runtime-filesystem/` — JS verbs stay on disk and are scanned by the generated binary at runtime.
7. `07-embedded-jsverbs/` — local JS verbs are copied into the generated workspace and embedded into the binary.
8. `08-provider-shipped-jsverbs/` — JS verbs live inside a Go provider package and are selected by `package`/`source`.
9. `09-provider-shipped-help-docs/` — Glazed help pages live inside a Go provider package and are selected by a `kind: help` provider source.
10. `10-embedded-assets-fs/` — local files are embedded into the generated binary and read through `require("fs:assets")`, while host writes use `require("fs:host")`.
11. `11-config-env/` — generated binaries read Glazed config files and environment variables according to `appName`, `envPrefix`, and `config` settings.
12. `12-geppetto-host-services/` — generated Geppetto jsverbs use profile flags, a SQLite turn store, contributed Go tools, contributed Go middleware, and a JSONL event sink.
13. `13-http-serve-jsverbs/` — the HTTP provider contributes a `serve` command that keeps a jsverb-registered Express site alive.
14. `14-generated-runtime-package/` — `xgoja generate` writes an importable runtime package that a host application uses directly.
15. `15-protobuf-builder-provider/` — a provider exposes a generated protobuf builder module, generated from a local `.proto`, and tests JavaScript-to-Go protobuf extraction without JSON/protojson conversion.
16. `16-typescript-jsverbs/` — TypeScript-authored jsverbs use esbuild-backed compilation, `xgoja run` TypeScript entry support, generated declarations, and HTTP hot reload.
17. `17-express-planned-auth/` — JavaScript route-authoring sketch for planned public/auth/resource routes.
18. `18-express-auth-host/` — runnable Go-owned dev-auth host that wires login/logout, session cookies, CSRF, resources, authorization, audit, and strict raw-route rejection for planned Express auth routes.
19. `19-express-keycloak-auth-host/` — production-oriented Keycloak/OIDC host skeleton with Docker Compose Keycloak realm, app sessions, app-owned authorization, and planned Express routes.

The Discord bot xgoja example lives in the sibling `discord-bot` repository because it demonstrates inserting xgoja into an existing host-owned runner rather than a standalone generated binary.

## JSVerb source filters

Generated xgoja binaries can mount JavaScript verbs from three source origins:

- runtime filesystem directories (`sources[].from.dir` without an embedding artifact source dependency),
- local directories copied and embedded into the generated binary (`sources[].from.dir` listed under `artifacts[].sources`),
- provider-shipped sources (`sources[].from.provider`).

Each source can optionally declare `include`, `exclude`, and `extensions` filters. Filters match slash-separated paths relative to that source root and are applied before a file is read or parsed.

```yaml
sources:
  - id: site
    kind: jsverbs
    from:
      dir: .
    include:
      - site.js
      - jsverbs/**/*.js
    exclude:
      - assets/**
      - dist/**
      - webapp/**
    extensions:
      - .js
      - .cjs
```

Use filters when the source root also contains bundled browser assets, generated files, or other JavaScript that should not declare CLI verbs. Prefer a narrow source directory such as `./verbs` when possible; filters are most useful for application roots or provider sources that intentionally contain multiple kinds of JavaScript.

## Run all examples

```bash
for dir in \
  01-core-provider \
  02-host-provider \
  03-single-runtime-modules \
  04-module-sections \
  05-command-provider \
  06-runtime-filesystem \
  07-embedded-jsverbs \
  08-provider-shipped-jsverbs \
  09-provider-shipped-help-docs \
  10-embedded-assets-fs \
  13-http-serve-jsverbs \
  14-generated-runtime-package \
  15-protobuf-builder-provider \
  16-typescript-jsverbs \
  18-express-auth-host; do
  make -C examples/xgoja/$dir smoke
done
```

`11-config-env/` and `12-geppetto-host-services/` are intentionally omitted from the bulk loop. `11-config-env/` is a command/config fixture rather than a Makefile-based smoke, and `12-geppetto-host-services/` depends on a sibling checkout of `geppetto` plus optional live profile credentials for the full inference smoke. `17-express-planned-auth/` is a route-authoring sketch, and `19-express-keycloak-auth-host/` has its own Docker Compose Keycloak smoke.

The Makefiles use:

```bash
GOWORK=off go run ./cmd/xgoja ... --xgoja-replace <repo-root>
```

`GOWORK=off` avoids the local workspace `goja` override while this repository's workspace dependency mismatch is unresolved. `--xgoja-replace` makes generated binaries use this checkout of `go-go-goja` instead of a released module version.
