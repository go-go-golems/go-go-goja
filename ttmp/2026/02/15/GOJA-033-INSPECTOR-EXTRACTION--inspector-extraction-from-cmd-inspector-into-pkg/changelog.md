# Changelog

## 2026-02-15

- Initialized GOJA-033 workspace with plan/tasks/diary scaffolding.
- Uploaded implementation plan and tasks bundle to reMarkable (`/ai/2026/02/15/GOJA-033`).
- Extracted reusable tree row shaping into `pkg/inspector/tree` with tests (`b49af1f`).
- Extracted reusable source/tree sync helpers into `pkg/inspector/navigation` with tests (`b49af1f`).
- Rewired `cmd/inspector/app` to consume extracted pkg helpers and added regression coverage (`b49af1f`).
- Validated extraction with:
  - `go test ./pkg/inspector/... -count=1`
  - `go test ./cmd/inspector/... -count=1`
  - `go test ./... -count=1`
