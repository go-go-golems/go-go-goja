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
