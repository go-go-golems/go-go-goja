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
Summary: Analysis of bun build failing to treat native modules as external.
LastUpdated: 2026-01-10T20:30:17-05:00
WhatFor: Identify why bun build cannot resolve native modules and define the fix.
WhenToUse: When adjusting bundling flags or debugging CommonJS build failures.
---


# Bun external flag failure analysis

## Goal
Identify why `bun build` fails to treat native module specifiers as external and define a fix that restores CommonJS bundling.

## Context
The CommonJS demo expects `require("exec")`, `require("fs")`, and `require("database")` to be resolved by go-go-goja at runtime. The bundler should treat these as external so it does not attempt to resolve them from npm.

## Failure summary
`make js-bundle` fails with:

```
cd js && bun build --target=browser --format=cjs --outfile=dist/bundle.cjs src/main.js --external:fs --external:exec --external:database
4 | var exec = require("exec");
                       ^
error: Could not resolve: "exec". Maybe you need to "bun install"?
    at /home/manuel/workspaces/2026-01-10/package-bun-goja-js/go-go-goja/js/src/main.js:4:20
```

## Hypothesis
The current bundling flags are using an esbuild-style syntax (`--external:exec`) that bun does not interpret. Bun expects `--external=<name>` or `-e <name>`.

## Proposed fix
- Update `Makefile` and `js/package.json` to use the bun-expected external flag format:
  - `--external=exec --external=fs --external=database` (or `-e exec -e fs -e database`).
- Keep the bundling format as CommonJS and `--target=browser`.

## Validation
- Run `make js-bundle` and ensure no "Could not resolve" errors occur.
- Run `make go-run-bun` and confirm the demo prints a result (non-empty) from `run()`.

## Related files
- `js/src/main.js`
- `js/package.json`
- `Makefile`
