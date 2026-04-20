# Tasks

## Phase 1 — Critical + User-Visible Bugs

### BUG-1: Empty/whitespace source panics in `jsparse.Resolve`

- [ ] Guard: skip analysis and return early in `replsession.Evaluate()` when `strings.TrimSpace(source) == ""`
  - File: `pkg/replsession/evaluate.go` — add check before `jsparse.Analyze()` call (~line 54)
  - Return a minimal `EvaluateResponse` with status `"empty-source"` and empty bindings/diffs
  - Also guard in `jsparse.Resolve()` itself: return an empty `Resolution` when `program.Body` is empty (defence in depth)
- [ ] Add test: `TestServiceEvaluateEmptySource` — verify empty, whitespace-only, newline-only all return gracefully
- [ ] Add test: `TestServiceEvaluateWhitespaceSource` — verify no panic, status is not crash
- [ ] Verify via `exp04_lowlevel.go` that T01/T02 now pass

### BUG-6: Thrown errors lose message in IIFE promise rejection

- [ ] Fix `waitPromise()` in `pkg/replsession/evaluate.go` to extract `.message` from Error rejection values
  - When `PromiseStateRejected`, check if the result is an Error object via `runtime.Owner.Call`
  - If it has a `.message` property, include that in the formatted error string
  - If it has a `.stack` property, optionally include first line
- [ ] Add test: `TestServiceThrowNewError` — verify `throw new Error('boom')` produces error containing `'boom'`
- [ ] Verify via `exp04_lowlevel.go` that T12 now passes
- [ ] Also verify `throw 'string'` and `throw {message:'x'}` produce reasonable output

## Phase 2 — Medium Bugs

### BUG-2: No panic recovery in HTTP handler

- [ ] Add a `recover()` middleware or per-handler defer in `pkg/replhttp/handler.go`
  - Wrap the evaluate handler (and ideally all handlers) in `defer func() { if r := recover(); r != nil { writeJSONError(w, 500, ...) } }()`
  - Log the panic with stack trace via zerolog before returning 500
- [ ] Add test: `TestHandlerPanicRecovery` — send empty source, verify 500 JSON response (not empty reply)
- [ ] Verify BUG-1 fix prevents the panic in the first place; BUG-2 is defence in depth

### BUG-3: Export API uses PascalCase (missing `json` struct tags)

- [ ] Add `json:"..."` tags to all exported structs in `pkg/repldb/types.go`:
  - `SessionRecord` → `session_id` → `sessionId`, `created_at` → `createdAt`, etc.
  - `EvaluationRecord` → `evaluation_id` → `evaluationId`, `session_id` → `sessionId`, etc.
  - `ConsoleEventRecord`, `BindingVersionRecord`, `BindingDocRecord` — all fields
  - `SessionExport` → `session`, `evaluations`
- [ ] Verify existing tests still pass (test JSON round-trips may need tag updates)
- [ ] Add test: `TestExportReturnsCamelCase` — POST create, eval, GET export, verify JSON keys are camelCase

### BUG-7: Top-level `await` not supported in instrumented mode

- [ ] In `pkg/replsession/evaluate.go`, add pre-wrap step before `jsparse.Analyze()`:
  - If `policy.UsesInstrumentedExecution() && policy.Eval.SupportTopLevelAwait`
  - And source starts with `await ` (same check as `wrapTopLevelAwaitExpression`)
  - Wrap source in `(async () => { return <source>; })()` before passing to parser
  - Store the pre-wrapped source so `buildRewrite` operates on the original
- [ ] Alternative: teach `jsparse.Analyze` to accept a parser mode that allows top-level await
- [ ] Add test: `TestServiceInstrumentedAwaitExpression` — verify `await Promise.resolve(99)` returns `99`
- [ ] Verify via `exp04_lowlevel.go` that T18 now passes

### BUG-8: Function source mapping always nil

- [ ] In `pkg/replsession/observe.go` (`refreshBindingRuntimeDetails`), adjust `MapFunctionToSource` call:
  - Pass the *rewritten* source AST (from `cellState.analysis` of the declaring cell) instead of the original
  - Or: skip `MapFunctionToSource` and build the mapping from the static analysis binding data directly (we already know the function name, parameters, and line from `upsertDeclaredBinding`)
  - Or: offset the source positions by the IIFE header length to account for the wrapper
- [ ] Add test: `TestServiceFunctionSourceMapping` — verify `FunctionMapping` is non-nil after declaring a function
- [ ] Verify via `exp04_lowlevel.go` that T09 now passes
