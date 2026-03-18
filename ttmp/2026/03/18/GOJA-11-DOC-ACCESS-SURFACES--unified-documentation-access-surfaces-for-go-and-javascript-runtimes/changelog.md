# Changelog

## 2026-03-18

- Initial workspace created

## 2026-03-18

Created the design ticket for unified documentation access across Glazed help, jsdoc, and plugin metadata. The main deliverable is an intern-facing architecture and implementation guide that recommends a runtime-scoped documentation hub with provider adapters and a shared JS-facing `docs` module rather than separate one-off wrappers.

## 2026-03-18

Validated GOJA-11 with `docmgr doctor`, related the main docs to the current code seams, and uploaded the bundle to reMarkable at `/ai/2026/03/18/GOJA-11-DOC-ACCESS-SURFACES`.

## 2026-03-18

Rescoped GOJA-11 from a pure design ticket into an active implementation ticket. The updated plan now treats plugin method docs as first-class contract data with no backward compatibility fields, fixes the target JavaScript module name at `docs`, and breaks the implementation into schema, provider, runtime-wiring, and validation phases.

## 2026-03-18

Implemented the schema slice in commit `af6c7e5`. Plugin method docs are now first-class manifest data, the SDK emits them, host validation/reification/reporting understands them, and runtime module registrars can exchange runtime-scoped values so the docs hub can consume retained plugin manifest snapshots without falling back to global state.

## 2026-03-18

Implemented the shared documentation core in commit `74b95a4`. The new `pkg/docaccess` package now provides a hub, common entry/query model, and provider adapters for Glazed help, jsdoc stores, and retained plugin manifests, all with focused package tests.

## 2026-03-18

Implemented the runtime surface in commit `1b8d2ef`. A runtime-scoped documentation registrar now builds a hub per runtime, registers `require("docs")`, integrates with the line REPL and Bobatea `js-repl`, and exposes help, jsdoc, and plugin metadata through one JS-facing module. Integration tests cover both generic docs access and plugin method docs.

## 2026-03-18

Added user-facing help and discoverability in commit `78bdf97`. The REPL help text now points users at the `docs` module, and `pkg/doc/15-docs-module-guide.md` documents the runtime API, common workflows, and troubleshooting guidance.

## 2026-03-18

Closed out GOJA-11 by relating the new implementation files back into the ticket docs, re-running `docmgr doctor --ticket GOJA-11-DOC-ACCESS-SURFACES --stale-after 30`, and uploading the refreshed bundle to reMarkable at `/ai/2026/03/18/GOJA-11-DOC-ACCESS-SURFACES`, where the listing shows `GOJA-11 Unified documentation access surfaces`.
