# Changelog

## 2026-06-09

- Initial workspace created
- Investigation complete: identified three architectural gaps (provider layer loses TypeScriptDeclarer, no xgoja gen-dts subcommand, no runtime exposure)
- Initial design doc written: four-phase implementation plan with decision records
- Related files: `modules/typing.go`, `pkg/xgoja/providerapi/module.go`, `pkg/xgoja/providers/core/core.go`, `pkg/xgoja/providers/host/host.go`, `cmd/gen-dts/main.go`
- Design review and corrected redesign appended to `design-doc/01-d-ts-export-architecture-and-implementation-plan.md`
- Corrected the implementation plan to account for xgoja's generated-code boundary: arbitrary provider packages cannot be imported dynamically by the compiled xgoja CLI, so `xgoja gen-dts` must use a generated sidecar or generated runtime code
- Updated `tasks.md` to match the corrected design: provider metadata + reusable `pkg/xgoja/dtsgen`, generated `types` command/package API, and sidecar-backed `xgoja gen-dts`
- Moved ticket from accidental nested root `go-go-goja/ttmp/...` under the repository to the intended repository `ttmp/...` root
- Implemented Phase 1: provider TypeScript metadata and reusable `pkg/xgoja/dtsgen` library (commit ffdae1b)
- Implemented Phase 2: generated runtime/package declaration exposure via `types` command and `Bundle.TypeScriptDeclarations()` APIs (commit a92940e)
- Implemented Phase 3: sidecar-backed `xgoja gen-dts` command for arbitrary provider imports
- Added embedded xgoja help tutorial for TypeScript declaration workflows and generated runtime `types` command (commit 41079a8)
- Ran `xgoja gen-dts` for `ClubMedMeetup/minitrace-viz`, generated `types/xgoja-modules.d.ts`, and added a root `jsconfig.json` for JetBrains indexing

## 2026-06-09

Added third-party provider TypeScript descriptors for go-minitrace, goja-text forwarding, and rag-widget-site Widget DSL; strict minitrace-viz xgoja declaration generation now passes (provider commits c0a0165, 0648b48, 1b44ea5; ClubMedMeetup commit 4835055; diary commit ce80071).

### Related Files

- /home/manuel/workspaces/2026-06-07/club-meetup-site/2026-05-27--rag-evaluation-system/pkg/widgetdsl/typescript.go — Widget DSL descriptors
- /home/manuel/workspaces/2026-06-07/club-meetup-site/ClubMedMeetup/minitrace-viz/types/xgoja-modules.d.ts — strict generated declarations
- /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracejs/typescript.go — go-minitrace descriptor
- /home/manuel/workspaces/2026-06-07/club-meetup-site/goja-text/pkg/xgoja/providers/text/text.go — descriptor forwarding

