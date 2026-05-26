---
Title: Generated xgoja command-provider smoke tests
Ticket: XGOJA-014
Status: active
Topics:
  - xgoja
  - command-registration
  - testing
  - jsverbs
DocType: design-doc
Intent: implementation
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Design for real generated-binary smoke tests for the go-minitrace, loupedeck, and css-visual-diff command providers."
LastUpdated: 2026-05-25T21:10:00-04:00
WhatFor: "Define end-to-end xgoja examples that prove command providers mount and execute through generated binaries."
WhenToUse: "Use while adding real smoke examples for XGOJA-014."
---

# Generated xgoja command-provider smoke tests

## Why this follow-up exists

The first XGOJA-014 pass validated provider registration, command-set construction, and package-local behavior, but it did not run generated xgoja binaries with mounted command providers. The missing proof is:

1. `xgoja build` reads an `xgoja.yaml` with `commandProviders`.
2. The generated binary resolves the provider and mounts its Glazed commands.
3. A command from that mounted provider actually runs.
4. The command demonstrates useful cross-module behavior rather than only printing help.

## go-minitrace smoke: jsverb writes a Markdown report with `fs`

Create `go-minitrace/examples/xgoja/minitrace-command-provider`.

Buildspec:

- Package: `go-minitrace/pkg/minitracejs/provider`.
- Runtime: include `minitrace` plus host/default modules as appropriate.
- Command provider: `go-minitrace.queries`, mounted as `traces`, configured with a local `queries/` repository.

Interesting command:

- `queries/reports/markdown-summary.js`
- A jsverb queries the loaded minitrace archive, requires `fs`, creates an output directory, and writes `minitrace-summary.md`.
- The command returns a row containing the output path, session count, and report title.

Smoke command:

```bash
./dist/minitrace-command-provider traces reports markdown-summary \
  --archive-glob ./data/*.minitrace.json \
  --out ./dist/report \
  --output json

test -f ./dist/report/minitrace-summary.md
grep -q "# Minitrace Smoke Report" ./dist/report/minitrace-summary.md
```

This proves the command provider path and the JS command runtime path, including `require("minitrace")` and `require("fs")`.

## loupedeck smoke: xgoja-powered scene verb with Express

The existing `loupedeck.scenes` provider initially reused the live hardware scene invoker. For a generated smoke we need a non-hardware command-provider execution path. Add xgoja-aware verb execution to the provider:

- When `CommandSetContext.RuntimeFactory` is available, build annotated verb commands with an invoker that calls `ctx.RuntimeFactory.NewRuntime(ctx, ctx.RuntimeProfile, require.WithLoader(registry.RequireLoader()))`.
- Collect module-provided Glazed sections from `ctx.SelectedModules` using `providerutil.CollectConfigSections` and append them to command descriptions.
- Before invoking the verb, call `providerutil.InitRuntimeFromSections` so selected package capabilities such as the xgoja HTTP provider start correctly.
- Keep the `run` command as the hardware path; configure smoke with `includeRun: false`.

Create `loupedeck/examples/xgoja/loupedeck-command-provider`.

Buildspec:

- Packages:
  - `loupedeck/pkg/xgoja/provider` for safe loupedeck modules and `scenes` command provider.
  - `go-go-goja/pkg/xgoja/providers/http` for `express`.
  - optionally `go-go-goja/pkg/xgoja/providers/host` for `fs` and `timer` if needed.
- Runtime `scene`: modules `gfx`, `easing`, `express`, `fs`, and `timer`.
- Command provider `loupedeck.scenes`, mounted as `loupe`, runtime profile `scene`, config `includeRun: false`, repository `./verbs`.

Interesting command:

- `verbs/web-scene-switcher.js`
- The verb starts an Express route:
  - `GET /` renders the current scene name and a link/form to deal the page.
  - `POST /deal` switches scene state from `waiting` to `dealt`, writes a marker/report file, and allows the command to exit.
- It uses `gfx`/`easing` to generate a lightweight scene model without touching hardware.
- The smoke Makefile starts the command in the foreground while a background `curl` hits `/deal`, then asserts the marker file exists.

This proves command-provider verbs can participate in xgoja runtime-profile composition and module section initialization.

## css-visual-diff smoke: generated provider runs a visual workflow verb

Create `css-visual-diff/examples/xgoja/css-visual-diff-command-provider`.

Buildspec:

- Package: `css-visual-diff/pkg/xgoja/provider`.
- Runtime: modules `css-visual-diff`, `diff`, `report`, and host modules as needed.
- Command provider: `css-visual-diff.verbs`, mounted as `css`, runtime profile `browser`, configured with local `verbs/` repository.

Interesting command:

- Local jsverb reads two local HTML fixture files or uses data URLs, compares a region, and writes artifacts to `dist/artifacts`.
- Smoke asserts an artifact JSON/Markdown file exists and contains expected text.

This proves the public css-visual-diff provider can be consumed by xgoja and execute command-provider verbs through `RuntimeFactory`.

## Task sequencing

1. Add the go-minitrace generated example and smoke it.
2. Upgrade the loupedeck command provider to xgoja-aware verb execution and add its generated web-scene smoke.
3. Add the css-visual-diff generated example and smoke it.
4. Run focused validations and update the diary after each package.
