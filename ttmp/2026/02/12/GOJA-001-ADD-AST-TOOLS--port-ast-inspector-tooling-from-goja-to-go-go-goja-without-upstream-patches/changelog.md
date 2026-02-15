# Changelog

## 2026-02-12

- Initial workspace created
- Added detailed migration diary with command-level trace and failure notes
- Added full porting analysis for moving inspector from `goja/internal/inspector` into `go-go-goja`
- Recorded baseline validation results:
  - `goja/internal/inspector` tests pass
  - `goja/cmd/goja-inspector` builds
  - `go-go-goja` full test run has pre-existing bun-demo embed setup failure (`assets-split/*` missing)
- Added external portability proof using `GOWORK=off` smoke module, including dependency pinning guidance for Charm stack
- Added vocabulary entries to clear ticket doctor warnings (topics/docTypes/intent/status)
- Uploaded bundle `GOJA-001 AST Tools Porting Analysis.pdf` to `/ai/2026/02/12/GOJA-001-ADD-AST-TOOLS` on reMarkable cloud
- Updated the implementation plan to require split architecture:
  - reusable framework layer `pkg/jsparse` (parsing/indexing/resolution/completion APIs)
  - inspector-specific layer was initially modeled as `pkg/inspector` + `cmd/goja-inspector` (later superseded)
- Re-uploaded the bundled analysis PDF to reMarkable after plan changes to keep the remote copy current
- Updated plan placement for inspector example code:
  - moved from proposed `pkg/inspector` to command-local `cmd/inspector` (`cmd/inspector/app` for UI internals)
  - kept `pkg/jsparse` as the reusable public API surface
- Implemented reusable core extraction in `pkg/jsparse` and stabilized lint/test hooks (commit `6ae8af2`)
- Implemented command-local inspector app under `cmd/inspector` wired to `pkg/jsparse` public APIs (commit `ca1879c`)
- Added reusable `pkg/jsparse` facade APIs (`Analyze` + completion/context helpers) for non-inspector consumers (commit `96ec0a2`)
- Added glazed help user guide entry `inspector-example-user-guide` for `cmd/inspector` as a `pkg/jsparse` consumer (commit `a1d2b42`)
- Added glazed help reference entry `jsparse-framework-reference` documenting reusable `pkg/jsparse` APIs and integration patterns (commit `e522c42`)
- Added REPL discoverability hints for both new help slugs and validated interactive/direct help flows (commit `92c2e65`)
- Added inspector UX validation tests and TTY smoke coverage for sync/go-to-def/usages/drawer-completion (commit `443946b`)
- Recorded final strategy decision to keep `cmd/inspector` standalone now and defer `repl` integration as follow-up (commit `c90c867`)
- Added `make test-inspector` and CI `inspector-validation` job so inspector migration checks are bun-demo independent (commit `5fd439d`)

## 2026-02-15

Ticket closed

