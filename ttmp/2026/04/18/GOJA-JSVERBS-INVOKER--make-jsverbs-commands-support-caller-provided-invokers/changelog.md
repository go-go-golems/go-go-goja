# Changelog

## 2026-04-18

- Created ticket `GOJA-JSVERBS-INVOKER` to track a minimal upstream `pkg/jsverbs` API extension for caller-provided execution.
- Added a design doc and implementation diary for the work.
- Implemented a pluggable invoker seam in `pkg/jsverbs/command.go` with:
  - `VerbInvoker`
  - `Registry.CommandsWithInvoker(...)`
  - `Registry.CommandForVerb(...)`
  - `Registry.CommandForVerbWithInvoker(...)`
- Kept `Registry.Commands()` backward compatible by delegating to the new API with a nil invoker.
- Added regression coverage for structured-output custom invokers, text-output custom invokers, and nil-invoker fallback behavior.
- Updated jsverbs developer docs and reference docs to describe the new host-owned execution hook.
- Validated the change with:
  - `go test ./pkg/jsverbs ./cmd/jsverbs-example`
