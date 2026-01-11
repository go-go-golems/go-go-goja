---
Title: Bun external flag failure analysis
Ticket: BUN-002
Status: active
Topics:
    - bun
    - bundling
    - build
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: Makefile
      Note: js-bundle target includes external flags
    - Path: js/package.json
      Note: Bun build script contains external flag syntax
    - Path: js/src/main.js
      Note: Native module requires trigger bundler failure
ExternalSources: []
Summary: Analysis of bun build failing to externalize native modules and the target fix.
LastUpdated: 2026-01-10T20:35:55-05:00
WhatFor: Identify why bun build cannot resolve native modules and define the fix.
WhenToUse: When adjusting bundling flags or debugging CommonJS build failures.
---


# Bun external flag failure analysis

## Goal
Identify why `bun build` fails to treat native module specifiers as external and define a fix that restores CommonJS bundling.

## Context
The CommonJS demo expects `require("exec")`, `require("fs")`, and `require("database")` to be resolved by go-go-goja at runtime. The bundler should treat these as external so it does not attempt to resolve them from npm.

## Failure summary
First failure (`--external:<name>` syntax ignored):

```
cd js && bun build --target=browser --format=cjs --outfile=dist/bundle.cjs src/main.js --external:fs --external:exec --external:database
4 | var exec = require("exec");
                       ^
error: Could not resolve: "exec". Maybe you need to "bun install"?
    at /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/js/src/main.js:4:20
```

Second failure (after fixing `--external` syntax but still using `--target=browser`):

```
run(): TypeError: Object has no member 'writeFileSync' at run (assets/bundle.cjs:5720:19(34))
```

Inspecting the bundle shows bun stubbed `fs` as an empty object under the browser target:

```
var fs = (() => ({}));
```

## Hypothesis
- The current bundling flags are using an esbuild-style syntax (`--external:exec`) that bun does not interpret. Bun expects `--external=<name>` or `-e <name>`.
- Using `--target=browser` causes bun to stub Node-style builtins (including `fs`), even when marked external.

## Proposed fix
- Update `Makefile` and `js/package.json` to use the bun-expected external flag format:
  - `--external=exec --external=fs --external=database` (or `-e exec -e fs -e database`).
- Switch bundling to `--target=node` so bun preserves `require("fs")` instead of stubbing it for browsers.
- Keep CommonJS format for goja_nodejs/require.

## Validation
- Run `make js-bundle` and ensure no "Could not resolve" errors occur.
- Run `make go-run-bun` and confirm the demo prints a result (non-empty) from `run()`.

## Related files
- `js/src/main.js`
- `js/package.json`
- `Makefile`
