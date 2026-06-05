---
Title: Research Logbook
Ticket: GOJA-064
Status: active
Topics:
    - goja
    - xgoja
    - http
    - verbs
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/2026-05-03--goja-hosting-site/pkg/app/multi_server.go
      Note: External goja-site multi-site reference evaluated in the logbook
    - Path: ../../../../../../../../../../code/wesen/2026-05-03--goja-hosting-site/pkg/app/server.go
      Note: External goja-site single-site reference evaluated in the logbook
    - Path: modules/express/express.go
      Note: Express module API evaluated in the logbook
    - Path: pkg/xgoja/app/command_providers.go
      Note: Command provider mounting and context construction evaluated in the logbook
    - Path: pkg/xgoja/app/root.go
      Note: Key xgoja jsverbs command flow evaluated in the logbook
    - Path: pkg/xgoja/providers/http/http.go
      Note: HTTP provider lifecycle and serve command provider target evaluated in the logbook
ExternalSources: []
Summary: Logbook of resources consulted for GOJA-064, with usefulness, stale areas, and update needs.
LastUpdated: 2026-06-04T23:37:00-04:00
WhatFor: Use to track which GOJA-064 research resources were useful, stale, wrong, or need follow-up updates.
WhenToUse: Read before continuing implementation or documentation work on xgoja HTTP serve support.
---


# Research Logbook

## Goal

This logbook records the resources consulted while researching GOJA-064: HTTP serve support for xgoja generated JavaScript verbs. It is meant to help future implementers understand which files and documents were useful, which were stale or incomplete, and what should be updated as the implementation proceeds.

## Context

The investigation focused on one question: how should generated xgoja binaries serve HTTP sites defined as JavaScript verbs that use the `express` package? The main candidate design is an HTTP provider command provider, rather than only a JavaScript-level `express.serve()` method.

Resource status terms used below:

- **Useful:** The resource directly informed the design or implementation guide.
- **Partially useful:** The resource provided context but not a complete answer.
- **Not useful / missing:** The resource path did not exist or did not contain the expected content.
- **Needs update:** The resource should be revised after the GOJA-064 implementation lands.

## Quick index

| Resource | Status | Main value |
| --- | --- | --- |
| `pkg/xgoja/app/root.go` | Useful | Existing generated root and jsverbs command flow. |
| `pkg/xgoja/app/run.go` | Useful | Keep-alive lifecycle model for long-running HTTP setup. |
| `pkg/xgoja/app/command_providers.go` | Useful | Provider-owned command mounting. |
| `pkg/xgoja/providerapi/commands.go` | Useful | Command provider API gap. |
| `pkg/xgoja/providers/http/http.go` | Useful | HTTP provider, express module registration, server lifecycle. |
| `modules/express/express.go` | Useful | JavaScript Express API surface. |
| `pkg/gojahttp/host.go` | Useful | HTTP request dispatch into Goja handlers. |
| `pkg/jsverbs/runtime.go` | Useful | Verb invocation in existing runtimes. |
| `examples/xgoja/10-embedded-assets-fs/*` | Useful | Existing generated HTTP/static-assets smoke pattern. |
| `cmd/xgoja/doc/09-tutorial-static-assets-http-server.md` | Useful, needs update later | Current documented `run --keep-alive` path. |
| `goja-site/pkg/app/server.go` | Useful | External reference for single-site runtime/server ownership. |
| `goja-site/pkg/app/multi_server.go` | Useful | External reference for future multi-site dispatch. |
| Missing `pkg/app/modules.go` | Missing | Search assumption was wrong; module wiring lives elsewhere. |
| Missing `scripts/server.js` | Missing | Example script was named differently. |

## Detailed resource entries

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/root.go`

- **What I was researching:** How generated xgoja roots attach commands and how JavaScript verbs become generated CLI commands.
- **What I was looking for in this document:** The implementation of `commands.jsverbs`, especially where runtimes are created, invoked, and closed.
- **Why I chose it:** Search results showed `newVerbsCommand`, `buildVerbCommands`, `scanVerbSource`, and `NewRootCommand` in this file.
- **How I found the resource:** `rg -n "CommandProvider|Provider|jsverbs|__verb__|NewLazyCommand" pkg/xgoja cmd/xgoja examples/xgoja pkg/jsverbs pkg/jsverbscli modules/express pkg/gojahttp pkg/engine -S`.
- **What I found useful:** `NewRootCommand` decodes the runtime spec and delegates command attachment. `buildVerbCommands` scans configured verb sources and builds one Glazed command per discovered verb. The current invoker creates a runtime with `factory.NewRuntimeFromSections`, invokes the verb, and defers `rt.Close(...)`.
- **What I didn't find useful:** It is intentionally oriented toward short-lived CLI verbs, so it does not provide a long-lived serve model by itself.
- **What is out of date / what was wrong:** Nothing appears wrong. It is current for ordinary jsverb command execution, but insufficient for HTTP serving.
- **What would need updating:** If command providers get reusable jsverb source access, some scanning logic should be factored so `root.go` and the HTTP serve command do not duplicate source resolution.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/host.go`

- **What I was researching:** How xgoja generated commands are attached to the Cobra root.
- **What I was looking for in this document:** The attachment order of built-in commands and provider command sets.
- **Why I chose it:** `root.go` constructs an `app.Host`; this file defines the host object and its `AttachDefaultCommands` method.
- **How I found the resource:** Followed `NewHostWithOptions(...)` from `pkg/xgoja/app/root.go`.
- **What I found useful:** `AttachDefaultCommands` installs `eval`, `run`, `repl`, `modules`, optionally `verbs`, and then provider command sets. This proves an HTTP provider can add a `serve` command without a new generated root mode.
- **What I didn't find useful:** It does not contain provider-specific command behavior.
- **What is out of date / what was wrong:** No obvious stale content.
- **What would need updating:** Probably no direct update unless the new serve feature needs a different command attachment order or root-level collision handling.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/run.go`

- **What I was researching:** Existing long-lived runtime behavior in generated xgoja commands.
- **What I was looking for in this document:** Signal handling, runtime creation, module initialization, script execution, and `--keep-alive` semantics.
- **Why I chose it:** The existing static-assets HTTP tutorial says long-running Express servers can use `run --keep-alive`.
- **How I found the resource:** Search results and direct read of xgoja app command files.
- **What I found useful:** `runScriptFileWithInitializers` creates a runtime, runs a script as a module, and waits for Ctrl-C/SIGTERM when `keepAlive` is true. This is the model for serving a jsverb-backed site.
- **What I didn't find useful:** It starts from a script file path, not from a scanned jsverb registry.
- **What is out of date / what was wrong:** Nothing wrong. It is the correct script-file workaround today.
- **What would need updating:** Documentation should cross-reference the new `serve` command once GOJA-064 is implemented, while keeping `run --keep-alive` documented for script setup files.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/command_providers.go`

- **What I was researching:** Whether the Express/HTTP provider can register a command provider that exposes `serve`.
- **What I was looking for in this document:** Command provider lookup, context construction, mount behavior, selected module filtering, and Glazed/Cobra attachment.
- **Why I chose it:** The user specifically asked whether the Express Goja module can register a command provider.
- **How I found the resource:** `rg` search for `CommandSetProvider` and `commandProviders`.
- **What I found useful:** `AttachCommandProviders` resolves configured providers and mounts returned commands. `newCommandSet` passes runtime profile, config, providers, runtime factory, and selected modules into `providerapi.CommandSetContext`.
- **What I didn't find useful:** The context currently lacks jsverb source access, which is the main missing feature for a provider-supplied `serve` command that mirrors configured verbs.
- **What is out of date / what was wrong:** No stale behavior; the API is just not yet broad enough for this use case.
- **What would need updating:** Add jsverb source accessor(s) to the command-provider context or another supported package-level API, then test provider command sets can scan configured sources.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/factory.go`

- **What I was researching:** How command providers create runtimes from selected xgoja runtime profiles.
- **What I was looking for in this document:** `RuntimeFactory.NewRuntimeFromSections`, module registration, host service contributions, and config merging.
- **Why I chose it:** A `serve` command provider must create the same kind of runtime as built-in commands.
- **How I found the resource:** Followed `RuntimeFactory` from `CommandSetContext` and `root.go`.
- **What I found useful:** It resolves selected modules, gathers host services, applies provider config sections, registers module loaders, and creates an `engine.Runtime` with startup/lifetime contexts.
- **What I didn't find useful:** It does not manage post-start command lifetime or HTTP server ownership directly.
- **What is out of date / what was wrong:** No obvious stale content.
- **What would need updating:** Probably no direct update unless HTTP serving requires richer host service exposure from runtime creation.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/module_sections.go`

- **What I was researching:** How provider module sections are collected and runtime initializers are called.
- **What I was looking for in this document:** Reusable helpers for appending HTTP provider sections to the proposed `serve` commands.
- **Why I chose it:** Serve commands need the same `--http-listen` section as `run` and `verbs`.
- **How I found the resource:** Followed calls to `sectionsForRuntimeProfile` and `initRuntimeFromSections`.
- **What I found useful:** `sectionsForRuntimeProfile` collects `GlazedConfigSectionCapability` sections, and `initRuntimeFromSections` runs runtime initializers such as the HTTP capability.
- **What I didn't find useful:** The helpers are on the app-side runtime factory and not automatically available to external provider code.
- **What is out of date / what was wrong:** No obvious stale content.
- **What would need updating:** If HTTP serve lives in a provider package, add exported provider utility or context access so provider command sets can collect the same sections without copying internals.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/providerapi/commands.go`

- **What I was researching:** The stable provider-facing API for custom command sets.
- **What I was looking for in this document:** What a command provider receives and whether it can access runtime profiles, modules, and JavaScript verb sources.
- **Why I chose it:** This is the API the HTTP provider would implement for a `serve` command.
- **How I found the resource:** Search results for `CommandSetProvider`.
- **What I found useful:** `CommandSetContext` already includes runtime profile, static config, host services, providers, runtime factory, and selected modules.
- **What I didn't find useful:** It does not expose configured jsverb sources or scanning helpers.
- **What is out of date / what was wrong:** Not wrong; incomplete for GOJA-064's desired provider-driven verb serve command.
- **What would need updating:** Add a jsverb source access abstraction. Decide whether this API should directly mention `pkg/jsverbs` or expose a narrower provider-safe interface.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/providerapi/module.go`

- **What I was researching:** What module setup context gives native modules at runtime.
- **What I was looking for in this document:** Host services, runtime owner, config, and closer registration available to modules such as `express`.
- **Why I chose it:** HTTP serving crosses provider module setup, runtime owner access, and runtime-scoped cleanup.
- **How I found the resource:** Followed provider module registration from `factory.go` and `providers/http/http.go`.
- **What I found useful:** `ModuleSetupContext` includes `RuntimeOwner` and `AddCloser`, which are important for runtime-safe HTTP request dispatch and server shutdown.
- **What I didn't find useful:** It is module setup API, not command provider API.
- **What is out of date / what was wrong:** No obvious stale content.
- **What would need updating:** No required update for v1 unless the HTTP provider chooses to expose route hosts as host services for manual server ownership.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/providerapi/capabilities.go`

- **What I was researching:** Provider capabilities that expose config sections and runtime initialization hooks.
- **What I was looking for in this document:** Interfaces used by the HTTP provider to add `--http-listen` and initialize per-runtime server settings.
- **Why I chose it:** The proposed serve command must reuse existing HTTP module sections and runtime initializer behavior.
- **How I found the resource:** Followed `WithPackageCapability` and `InitRuntimeFromSections` from the HTTP provider and app module sections.
- **What I found useful:** `GlazedConfigSectionCapability`, `RuntimeInitializerCapability`, and `HostServiceContributionCapability` define the extension points needed for HTTP configuration and future server ownership.
- **What I didn't find useful:** It does not directly help command providers scan jsverbs.
- **What is out of date / what was wrong:** No obvious stale content.
- **What would need updating:** If future multi-site serving uses manual server ownership, add or document a provider-defined host service contract for HTTP route hosts.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/providerapi/provider_registry.go`

- **What I was researching:** How provider packages register modules, capabilities, command sets, help, and verb sources.
- **What I was looking for in this document:** Whether command set providers are registered per package and resolved later by generated apps.
- **Why I chose it:** The first attempted path `providerapi/registry.go` was wrong; this is the actual registry file.
- **How I found the resource:** Listed `pkg/xgoja/providerapi` files after the failed read and opened `provider_registry.go`.
- **What I found useful:** `ProviderRegistry.Package` collects entries; `ResolveCommandSetProvider` and `ResolveVerbSource` are existing lookup methods relevant to GOJA-064.
- **What I didn't find useful:** It does not describe higher-level semantics; it is just the registry implementation.
- **What is out of date / what was wrong:** The path I initially guessed, `registry.go`, was wrong. The implementation itself is current.
- **What would need updating:** Probably no update unless adding convenience lookup methods for jsverb source contexts.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/jsverbs/runtime.go`

- **What I was researching:** How a scanned JavaScript verb is invoked inside a Goja runtime.
- **What I was looking for in this document:** Whether `InvokeInRuntime` can be reused by a long-lived serve command.
- **Why I chose it:** A provider `serve` command should reuse existing verb binding and invocation logic instead of inventing a new route setup function API.
- **How I found the resource:** Search results for `InvokeInRuntime`, `RequireLoader`, and `__verb__`.
- **What I found useful:** `RequireLoader` exposes the scanned-source loader, and `InvokeInRuntime` invokes a verb inside a caller-owned runtime without closing it. This is exactly the primitive the serve command needs.
- **What I didn't find useful:** Default `Registry.invoke` creates and closes its own runtime, so the serve command must avoid that path.
- **What is out of date / what was wrong:** The promise waiting comment says polling is a simple v1 prototype. That caveat remains important for HTTP setup verbs that return promises.
- **What would need updating:** If serve needs better readiness semantics for async setup verbs, promise handling may need stronger event-loop integration later.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/jsverbs/model.go`

- **What I was researching:** The metadata model for discovered JavaScript verbs.
- **What I was looking for in this document:** Fields such as tags, parents, fields, sections, and full command path.
- **Why I chose it:** The design considered whether `serve` should expose all verbs or filter by tags such as `http` and `site`.
- **How I found the resource:** Followed `VerbSpec` from `jsverbs/runtime.go` and command generation.
- **What I found useful:** `VerbSpec` includes `Tags`, `Parents`, `Fields`, `UseSections`, and `FullPath`, so a future command provider can filter or mirror command paths without changing the verb declaration format.
- **What I didn't find useful:** It does not decide policy; it only exposes metadata.
- **What is out of date / what was wrong:** No obvious stale content.
- **What would need updating:** If the implementation chooses tag filtering, docs should specify the recommended tags and tests should cover filtering.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/modules/express/express.go`

- **What I was researching:** What the JavaScript `express` module currently exposes and whether `express.serve()` already exists.
- **What I was looking for in this document:** Route registration methods, static serving helpers, runtime owner wiring, and server/listen APIs.
- **Why I chose it:** The user explicitly mentioned the Express package and asked whether one solution is just an `express serve()` method.
- **How I found the resource:** Search results for `modules/express` and `express.app`.
- **What I found useful:** The module exposes `app()`, route methods, `static`, and `staticFromAssetsModule`. It registers Goja callbacks with a `gojahttp.Host` and uses runtimebridge ownership when possible.
- **What I didn't find useful:** There is no explicit `serve()` or `listen()` JavaScript method today.
- **What is out of date / what was wrong:** Nothing wrong. The API is route-registration only.
- **What would need updating:** After GOJA-064, TypeScript docs and module docs may need to explain the preferred generated `serve` command. Add `express.listen()` only if a separate script convenience is still desired.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/modules/express/typescript.go`

- **What I was researching:** Whether the TypeScript declaration surface reflects current Express APIs.
- **What I was looking for in this document:** Express module declaration shape and whether serve/listen exists.
- **Why I chose it:** Search results pointed to it while inspecting the Express module.
- **How I found the resource:** `rg -n "express" modules/express`.
- **What I found useful:** It confirms the documented module surface is `app`, route methods, and static helpers.
- **What I didn't find useful:** It was not central to command-provider design.
- **What is out of date / what was wrong:** It will become out of date if any JavaScript-level `express.serve()` or `listen()` method is added later.
- **What would need updating:** Update declarations only if the JavaScript Express API changes. A pure generated command provider may not require this file to change.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/modules/express/express_integration_test.go`

- **What I was researching:** Existing integration coverage for Express/gojahttp behavior.
- **What I was looking for in this document:** Tests for route registration, static embedded assets, body handling, async handlers, and HEAD fallback.
- **Why I chose it:** It validates the behavior a serve command would depend on.
- **How I found the resource:** `rg --files modules/express` and search output for Express tests.
- **What I found useful:** The file is a good model for package-level HTTP tests, especially static asset and async handler behavior.
- **What I didn't find useful:** It does not exercise generated xgoja command providers or long-lived generated verb commands.
- **What is out of date / what was wrong:** No obvious stale content.
- **What would need updating:** Add or complement tests for the new generated `serve` command in xgoja provider/generate tests, not necessarily only here.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/gojahttp/host.go`

- **What I was researching:** How registered Express routes handle real HTTP requests.
- **What I was looking for in this document:** Runtime owner calls, request/response DTO creation, promise handling, static mount precedence, and error handling.
- **Why I chose it:** Serve support must keep a runtime alive so `gojahttp.Host` can call route callbacks safely.
- **How I found the resource:** Followed `gojahttp.Host` from `modules/express` and the HTTP provider.
- **What I found useful:** `ServeHTTP` calls route handlers through `h.owner.Call(...)`, proving runtime ownership is central and the runtime must not be closed after setup.
- **What I didn't find useful:** It does not own listener lifecycle; it is an `http.Handler`.
- **What is out of date / what was wrong:** No obvious stale content. Promise handling is polling-based, which is noted elsewhere as a v1 approach.
- **What would need updating:** If manual server ownership or multi-site mode is added, this file may remain unchanged; outer dispatch can wrap multiple `gojahttp.Host` values.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/providers/http/http.go`

- **What I was researching:** Current xgoja HTTP provider behavior and whether it is the right place for `serve`.
- **What I was looking for in this document:** Provider registration, HTTP config flags, runtime initialization, Express loader wiring, server start, and shutdown.
- **Why I chose it:** This provider owns the generated `express` module and HTTP server lifecycle.
- **How I found the resource:** `rg -n "go-go-goja-http|express|http" pkg/xgoja/providers modules/express -S`.
- **What I found useful:** The provider already exposes `--http-listen`, stores runtime-specific settings, starts `net/http.Server`, and registers cleanup with the runtime.
- **What I didn't find useful:** It does not yet register a command provider. Server startup is asynchronous and prints bind failures, which is not ideal for a CLI `serve` command.
- **What is out of date / what was wrong:** The behavior is current, but the async `ListenAndServe` start path is too weak for robust serve-command startup because port conflicts may surface only in a goroutine.
- **What would need updating:** Register `CommandSetProvider{Name: "serve"}`. Consider changing startup to bind synchronously with `net.Listen` before serving. Later add manual server ownership mode for multi-site.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/providers/http/http_test.go`

- **What I was researching:** Current tests for HTTP provider registration and config behavior.
- **What I was looking for in this document:** What existing behavior is tested and where new tests should be added.
- **Why I chose it:** The proposed command provider will live in the same package.
- **How I found the resource:** Listed `pkg/xgoja/providers/http` files after reading `http.go`.
- **What I found useful:** Tests already verify registration, config section prefix, nil runtime handling, default enablement when values are present, and explicit disable.
- **What I didn't find useful:** No command provider tests exist yet.
- **What is out of date / what was wrong:** Not out of date, but incomplete for GOJA-064.
- **What would need updating:** Add tests that `Register` exposes both `express` and `serve`, that the serve command set mirrors jsverb commands, and that runtime lifetime persists until cancellation.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/internal/buildspec/build_spec.go`

- **What I was researching:** The YAML schema for generated xgoja binaries.
- **What I was looking for in this document:** Whether command providers are already configurable in `xgoja.yaml`.
- **Why I chose it:** The design needed a concrete buildspec fragment for users.
- **How I found the resource:** Search for `type BuildSpec` after an attempted read of a non-existent `spec.go`.
- **What I found useful:** `BuildSpec` already has `CommandProviders`, and `CommandProviderInstanceSpec` includes all fields needed to opt into an HTTP `serve` command provider.
- **What I didn't find useful:** The schema does not describe behavior; it only carries data.
- **What is out of date / what was wrong:** I initially guessed `cmd/xgoja/internal/buildspec/spec.go`; the actual file is `build_spec.go`.
- **What would need updating:** If the implementation adds provider-specific `config` fields for filtering serve verbs, document them in user docs rather than changing this generic schema.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/internal/buildspec/validate.go`

- **What I was researching:** Whether a proposed `commandProviders` fragment would pass validation.
- **What I was looking for in this document:** Validation rules for provider IDs, packages, names, runtime profiles, and jsverb command mounts.
- **Why I chose it:** The design includes a YAML snippet for `commandProviders: go-go-goja-http.serve`.
- **How I found the resource:** Search results for `validateCommandProviders` and `commands.jsverbs.mount`.
- **What I found useful:** Existing validation already supports command providers and checks runtime profile references.
- **What I didn't find useful:** It cannot validate whether a provider actually registers a named command set until runtime/generated command construction.
- **What is out of date / what was wrong:** No obvious stale content.
- **What would need updating:** Optional: add static validation if xgoja doctor gains provider introspection for provider command set names.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/examples/xgoja/10-embedded-assets-fs/xgoja.yaml`

- **What I was researching:** A concrete generated xgoja app that uses HTTP and embedded assets.
- **What I was looking for in this document:** How users configure host `fs`, HTTP `express`, embedded assets, and generated commands.
- **Why I chose it:** The static-assets tutorial and search results indicated this was the canonical existing HTTP xgoja example.
- **How I found the resource:** Search results for `staticFromAssetsModule`, `--http-listen`, and docs references to `examples/xgoja/10-embedded-assets-fs`.
- **What I found useful:** It shows the provider package IDs and runtime module config needed for `require("fs:assets")` and `require("express")`.
- **What I didn't find useful:** It does not use jsverbs or command providers.
- **What is out of date / what was wrong:** Not out of date. It is just script-oriented rather than verb-oriented.
- **What would need updating:** Add a sibling example for GOJA-064, probably `examples/xgoja/13-http-serve-jsverbs`, rather than changing this one heavily.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/examples/xgoja/10-embedded-assets-fs/scripts/serve-static-assets.js`

- **What I was researching:** Existing JavaScript setup script style for Express serving in generated xgoja.
- **What I was looking for in this document:** How a script registers routes and static assets using `express.app()`.
- **Why I chose it:** It is the live example behind `make serve-smoke` for embedded static assets.
- **How I found the resource:** After the guessed `scripts/server.js` path failed, listed the example directory and found `serve-static-assets.js`.
- **What I found useful:** It is a minimal route-registration script that can be converted almost directly into a site setup verb.
- **What I didn't find useful:** It is a script, not a `__verb__` file, so it does not demonstrate generated verb command metadata.
- **What is out of date / what was wrong:** The resource is current. The guessed filename `server.js` was wrong.
- **What would need updating:** A new verb-based example should reuse this pattern and wrap it in `__verb__("static", ...)`.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/examples/xgoja/10-embedded-assets-fs/README.md`

- **What I was researching:** How the repository currently explains generated HTTP serving with embedded assets.
- **What I was looking for in this document:** User-facing explanation and smoke-test commands.
- **Why I chose it:** The example is likely what users would find before a new serve-verb example exists.
- **How I found the resource:** Directory listing of `examples/xgoja/10-embedded-assets-fs`.
- **What I found useful:** The README explains `run scripts/serve-static-assets.js --http-listen ... --keep-alive` and why the generated runtime must stay alive.
- **What I didn't find useful:** It does not mention jsverbs or command providers.
- **What is out of date / what was wrong:** It is current for script serving. It will become incomplete once generated verb serving exists.
- **What would need updating:** Add a "See also" link to the new jsverb serve example after implementation.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/examples/xgoja/10-embedded-assets-fs/Makefile`

- **What I was researching:** How generated HTTP examples are smoke tested.
- **What I was looking for in this document:** Build, run, and serve smoke command patterns.
- **Why I chose it:** The proposed implementation guide needed a generated example smoke-test recipe.
- **How I found the resource:** Directory listing of the embedded-assets example.
- **What I found useful:** `serve-smoke` starts the generated binary in the background, probes with `curl`, checks responses, and cleans up with traps.
- **What I didn't find useful:** It tests `run --keep-alive`, not a generated `serve` command.
- **What is out of date / what was wrong:** Current for its example.
- **What would need updating:** Copy the test pattern into the new GOJA-064 example and adapt the command from `run ... --keep-alive` to `serve ...`.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/doc/09-tutorial-static-assets-http-server.md`

- **What I was researching:** Current official documentation for generated xgoja HTTP serving.
- **What I was looking for in this document:** The public recommended way to serve static assets and configure `--http-listen`.
- **Why I chose it:** Search results showed this tutorial directly discusses generated binaries serving HTTP through Express.
- **How I found the resource:** `rg -n "Long-running scripts|static asset|http server|--keep-alive" cmd/xgoja/doc examples/xgoja`.
- **What I found useful:** It clearly explains embedded assets, `fs:assets`, `express.app()`, `staticFromAssetsModule`, `--http-listen`, and why `--keep-alive` matters.
- **What I didn't find useful:** It only covers script-file serving, not jsverb-backed serving.
- **What is out of date / what was wrong:** Current today. It will be incomplete after GOJA-064 implementation because it will not mention the generated `serve` command.
- **What would need updating:** Add a section comparing `run --keep-alive` for setup scripts with `serve <verb>` for generated verb-backed sites. Link to the new example.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/examples/xgoja/05-command-provider/README.md`

- **What I was researching:** Existing provider-shipped command set example.
- **What I was looking for in this document:** How command providers are explained to users and how generated commands are mounted.
- **Why I chose it:** The proposed design uses `CommandSetProvider`.
- **How I found the resource:** Search output from `examples/xgoja/README.md` and `commandProviders` references.
- **What I found useful:** It confirms the generated command tree model for provider-owned commands and provides a concise example of mounted commands.
- **What I didn't find useful:** The example is not HTTP-related and does not scan jsverbs.
- **What is out of date / what was wrong:** Current for generic command provider behavior.
- **What would need updating:** Add a cross-reference from any future HTTP serve example back to generic command-provider docs if helpful.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/examples/xgoja/05-command-provider/xgoja.yaml`

- **What I was researching:** Concrete YAML syntax for command provider mounting.
- **What I was looking for in this document:** `commandProviders` shape, including `id`, `package`, `name`, `mount`, and `runtimeProfile`.
- **Why I chose it:** The design doc needed an accurate YAML fragment.
- **How I found the resource:** Read alongside the command-provider example README.
- **What I found useful:** It provides a minimal working pattern for mounting a provider command set.
- **What I didn't find useful:** It does not include jsverbs or HTTP modules.
- **What is out of date / what was wrong:** No obvious stale content.
- **What would need updating:** No update required; the future HTTP serve example should follow this pattern.

### `/home/manuel/code/wesen/2026-05-03--goja-hosting-site/cmd/goja-site/serve.go`

- **What I was researching:** External reference for a real `serve` command that hosts JavaScript Express-style sites.
- **What I was looking for in this document:** CLI flags, signal handling, server construction, diagnostics setup, and run lifecycle.
- **Why I chose it:** The user specifically asked to inspect `/home/manuel/code/wesen/2026-05-03--goja-hosting-site`.
- **How I found the resource:** User-provided path plus `rg -n "express|serve|verb|CommandProvider|require\(|goja"` in that repository.
- **What I found useful:** `serve` creates a signal context, starts diagnostics, constructs `app.NewServer`, defers cleanup, prints status, and calls `srv.Run(ctx)`.
- **What I didn't find useful:** It is a standalone app command, not an xgoja provider command provider.
- **What is out of date / what was wrong:** No obvious stale content for goja-site.
- **What would need updating:** No direct update required. It remains an external reference model.

### `/home/manuel/code/wesen/2026-05-03--goja-hosting-site/pkg/app/server.go`

- **What I was researching:** How goja-site creates a runtime, loads scripts, and serves routes through Express.
- **What I was looking for in this document:** Single-site ownership of database, runtime, route host, and HTTP server.
- **Why I chose it:** It is the closest working implementation of the desired server shape.
- **How I found the resource:** Followed `app.NewServer` from `cmd/goja-site/serve.go`.
- **What I found useful:** It creates `gojahttp.Host`, registers `express`, creates one runtime, sets the host runtime, loads scripts, serves through `net/http.Server`, and closes runtime/database/server together.
- **What I didn't find useful:** It loads sorted script files rather than invoking jsverbs.
- **What is out of date / what was wrong:** No obvious stale content.
- **What would need updating:** No direct update. The xgoja implementation should borrow the lifecycle, not the app-specific database and observability layers.

### `/home/manuel/code/wesen/2026-05-03--goja-hosting-site/pkg/app/scripts.go`

- **What I was researching:** How goja-site finds and orders JavaScript setup files.
- **What I was looking for in this document:** Script discovery semantics and startup determinism.
- **Why I chose it:** The user mentioned how goja-site loads and serves verbs/scripts using Express.
- **How I found the resource:** Followed `LoadScripts` from `pkg/app/server.go`.
- **What I found useful:** It walks configured directories, deduplicates paths, sorts `.js` files, and fails if no scripts are found.
- **What I didn't find useful:** It is not directly applicable to jsverb source scanning because xgoja already has `jsverbs.ScanDir`/`ScanFS`.
- **What is out of date / what was wrong:** No obvious stale content.
- **What would need updating:** No direct update. It reinforces that startup resources should fail fast if missing.

### `/home/manuel/code/wesen/2026-05-03--goja-hosting-site/pkg/app/multi_server.go`

- **What I was researching:** Future multi-site serve architecture.
- **What I was looking for in this document:** How one outer HTTP listener dispatches to many isolated site runtimes.
- **Why I chose it:** The user asked whether a serve verb could serve different sites, and the external repo has a multi-site server.
- **How I found the resource:** Search results and direct read of goja-site app files.
- **What I found useful:** `MultiServer` creates one `Server` per site and dispatches by normalized Host header. This is the right future model for `serve-multi`.
- **What I didn't find useful:** It assumes each site is a goja-site `Server` with database config, not a generic xgoja runtime/verb pair.
- **What is out of date / what was wrong:** No obvious stale content.
- **What would need updating:** No direct update. The future xgoja multi-site implementation should adapt the pattern after HTTP provider server ownership is explicit.

### `/home/manuel/code/wesen/2026-05-03--goja-hosting-site/examples/kanban/scripts/app.js`

- **What I was researching:** A real site script using `require("express")`.
- **What I was looking for in this document:** How JavaScript code registers routes, static assets, UI DSL rendering, and application behavior.
- **Why I chose it:** Search results showed it as a representative goja-site example using Express.
- **How I found the resource:** `rg -n "const express = require\(\"express\"\)|app = express.app"` in the goja-site repository.
- **What I found useful:** It shows idiomatic site setup: require modules, create `app`, mount static assets, migrate/seed state, and define routes later in the file.
- **What I didn't find useful:** It is long and app-specific; most business logic is not relevant to generic xgoja serving.
- **What is out of date / what was wrong:** No obvious stale content.
- **What would need updating:** No direct update. A future xgoja example should be much smaller.

### `/home/manuel/code/wesen/2026-05-03--goja-hosting-site/pkg/doc/developer-guide/developer-guide.md`

- **What I was researching:** High-level explanation of goja-site internals.
- **What I was looking for in this document:** Architecture narrative that could inform the intern-facing GOJA-064 design style.
- **Why I chose it:** It appeared in goja-site repository search results as a developer-facing architecture guide.
- **How I found the resource:** `rg --files` and direct read of the goja-site doc path.
- **What I found useful:** It explains the CLI-to-runtime-to-route-host flow and names the boundaries a new contributor should understand.
- **What I didn't find useful:** It contains some package names that differ from current upstream go-go-goja package layout and is goja-site-specific.
- **What is out of date / what was wrong:** The guide references `pkg/web` in places, while this GOJA-064 investigation focused on upstream `pkg/gojahttp` and `modules/express`. That difference is expected because goja-site has its own historical docs and app structure.
- **What would need updating:** If goja-site is maintained as a current reference, update package references to match the active code organization and add notes distinguishing goja-site app code from upstream go-go-goja modules.

### `/home/manuel/code/wesen/2026-05-03--goja-hosting-site/pkg/app/modules.go`

- **What I was researching:** Expected module wiring file in goja-site.
- **What I was looking for in this document:** Central module registration for Express, UI DSL, database, and other site modules.
- **Why I chose it:** I inferred the path while looking for app module wiring.
- **How I found the resource:** Attempted direct read after inspecting `pkg/app` files.
- **What I found useful:** Nothing; the file does not exist.
- **What I didn't find useful:** The path was wrong.
- **What is out of date / what was wrong:** My assumption was wrong. Module wiring is in `pkg/app/server.go` and database-specific helper files.
- **What would need updating:** No repository update needed. Future researchers should not look for this path.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/examples/xgoja/10-embedded-assets-fs/scripts/server.js`

- **What I was researching:** Expected static HTTP setup script in the xgoja embedded-assets example.
- **What I was looking for in this document:** Minimal Express setup script.
- **Why I chose it:** The tutorial uses the generic name `scripts/server.js`; I tried that path in the concrete example.
- **How I found the resource:** Attempted direct read based on tutorial naming.
- **What I found useful:** Nothing; the file does not exist.
- **What I didn't find useful:** The example's actual file is `scripts/serve-static-assets.js`.
- **What is out of date / what was wrong:** The mismatch is between tutorial prose and example filename, not necessarily a broken example.
- **What would need updating:** Consider aligning tutorial and example names, or explicitly mention that the repository example uses `serve-static-assets.js`.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/internal/buildspec/spec.go`

- **What I was researching:** Expected buildspec type definitions.
- **What I was looking for in this document:** `BuildSpec` and command-provider schema.
- **Why I chose it:** I guessed a common filename.
- **How I found the resource:** Attempted direct read, then searched for `type BuildSpec`.
- **What I found useful:** Nothing from this path; it does not exist.
- **What I didn't find useful:** The filename was wrong.
- **What is out of date / what was wrong:** The actual file is `cmd/xgoja/internal/buildspec/build_spec.go`.
- **What would need updating:** No repository update needed. This is a research-path correction.

## Process references consulted

These were not product/source references, but they shaped the deliverable workflow.

### `/home/manuel/.pi/agent/skills/ticket-research-docmgr-remarkable/references/writing-style.md`

- **What I was researching:** Required style for the design deliverable.
- **What I was looking for in this document:** Structure, evidence rules, decision record format, and detail level.
- **Why I chose it:** The active ticket-research skill instructs loading this reference before writing.
- **How I found the resource:** It is referenced by the pinned `ticket-research-docmgr-remarkable` skill.
- **What I found useful:** It set the design doc structure and emphasized evidence-backed claims.
- **What I didn't find useful:** It is generic process guidance, not GOJA-specific technical content.
- **What is out of date / what was wrong:** No issue observed.
- **What would need updating:** N/A for GOJA-064.

### `/home/manuel/.pi/agent/skills/ticket-research-docmgr-remarkable/references/deliverable-checklist.md`

- **What I was researching:** Required completion steps for ticket delivery and reMarkable upload.
- **What I was looking for in this document:** Validation and upload checklist.
- **Why I chose it:** The active ticket-research skill instructs loading this reference before delivery.
- **How I found the resource:** It is referenced by the pinned `ticket-research-docmgr-remarkable` skill.
- **What I found useful:** It reminded me to run `docmgr doctor`, perform a dry-run upload, upload the bundle, and verify the remote listing.
- **What I didn't find useful:** It is not technical architecture content.
- **What is out of date / what was wrong:** No issue observed.
- **What would need updating:** N/A for GOJA-064.

## Ticket resources consulted

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-064--http-serve-support-for-xgoja-generated-verbs/design-doc/01-http-serve-support-for-xgoja-generated-verbs.md`

- **What I was researching:** The already-written GOJA-064 design package before creating this logbook.
- **What I was looking for in this document:** The resource list and claims to turn into a separate research logbook.
- **Why I chose it:** The user asked for a research logbook after the design doc had already been written.
- **How I found the resource:** `docmgr doc list --ticket GOJA-064` and direct ticket path.
- **What I found useful:** It contained the file reference map, architecture findings, and decisions that identify the resources used.
- **What I didn't find useful:** It was not structured as a resource-by-resource logbook.
- **What is out of date / what was wrong:** No known issue.
- **What would need updating:** If implementation decisions change, update both the design doc and this logbook.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-064--http-serve-support-for-xgoja-generated-verbs/reference/01-diary.md`

- **What I was researching:** The chronological investigation record.
- **What I was looking for in this document:** Commands run, failures encountered, and resources inspected.
- **Why I chose it:** The diary captured details not all repeated in the design doc.
- **How I found the resource:** `docmgr doc list --ticket GOJA-064` and direct ticket path.
- **What I found useful:** It recorded failed reads, vocabulary validation issues, upload retries, and the investigation sequence.
- **What I didn't find useful:** It is chronological rather than resource-indexed.
- **What is out of date / what was wrong:** No known issue.
- **What would need updating:** Add future implementation steps if the ticket continues beyond research.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-064--http-serve-support-for-xgoja-generated-verbs/tasks.md`

- **What I was researching:** Ticket task completion state.
- **What I was looking for in this document:** Whether the original research/design/upload tasks were complete and whether to add a logbook task.
- **Why I chose it:** The user asked for an additional deliverable in the same ticket.
- **How I found the resource:** `docmgr task list --ticket GOJA-064` and direct read.
- **What I found useful:** It showed prior tasks were checked and allowed tracking the new logbook/upload task.
- **What I didn't find useful:** It does not contain technical architecture content.
- **What is out of date / what was wrong:** No issue observed.
- **What would need updating:** Check the new logbook task after this document is validated and uploaded.

### `/home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-064--http-serve-support-for-xgoja-generated-verbs/changelog.md`

- **What I was researching:** Ticket history and delivery evidence.
- **What I was looking for in this document:** Existing entries to keep the logbook addition consistent with prior bookkeeping.
- **Why I chose it:** The logbook addition should be reflected in changelog updates.
- **How I found the resource:** Direct ticket path and docmgr workflow.
- **What I found useful:** It recorded initial ticket creation, design completion, validation, and upload.
- **What I didn't find useful:** It does not provide detailed resource evaluation.
- **What is out of date / what was wrong:** No issue observed.
- **What would need updating:** Add a changelog entry for this research logbook and reMarkable re-upload.

## Resources that need follow-up updates after implementation

1. **`cmd/xgoja/doc/09-tutorial-static-assets-http-server.md`**
   - Add `serve <verb>` as the preferred verb-backed site path.
   - Keep `run --keep-alive` for script-backed setup files.

2. **`examples/xgoja/10-embedded-assets-fs/README.md`**
   - Add a link to the new jsverb serve example after it exists.

3. **New example `examples/xgoja/13-http-serve-jsverbs`**
   - Add a minimal generated app with embedded assets, a `sites static` verb, and a `serve-smoke` target.

4. **`pkg/xgoja/providers/http/http_test.go`**
   - Add command-provider registration and serve-command behavior tests.

5. **`pkg/xgoja/app` command-provider tests**
   - Add tests for exposing configured jsverb sources to command providers.

6. **`modules/express/typescript.go`**
   - Update only if the JavaScript API itself gains `listen()` or `serve()`.

7. **goja-site developer guide**
   - Optional: refresh package references if goja-site docs are intended to remain a current upstream architecture reference.

## Suggested use during implementation

Before implementing GOJA-064, read this logbook in this order:

1. `pkg/xgoja/app/root.go`, `run.go`, and `command_providers.go` entries.
2. `providerapi/commands.go` and `providerapi/capabilities.go` entries.
3. `pkg/xgoja/providers/http/http.go` and `modules/express/express.go` entries.
4. `examples/xgoja/10-embedded-assets-fs` entries.
5. goja-site `server.go` and `multi_server.go` entries.

This sequence explains the current generated command path, the missing API seam, the HTTP provider lifecycle, and the future multi-site direction.
