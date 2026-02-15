# Tasks

## TODO

- [x] Add/expand analysis session API in `pkg/inspector/analysis` for globals, members, and source-jump metadata
- [x] Add unit tests for new analysis-session methods (sorting, membership, declaration line lookup)
- [x] Introduce `analysisSession` field in smalltalk-inspector model and initialize on file load
- [x] Migrate `buildGlobals` to consume analysis-session API (remove direct root-scope traversal from UI)
- [x] Migrate class/function member builders to analysis-session API
- [x] Migrate source jump helpers (`jumpToBinding`, `jumpToMember`) to analysis-session API
- [x] Keep runtime-only inspection paths unchanged and verify mixed static/runtime behavior
- [x] Remove obsolete direct `jsparse` graph reads from smalltalk-inspector static-analysis paths
- [x] Run regression suite: `pkg/inspector/analysis`, `cmd/smalltalk-inspector`, `cmd/inspector`, `go test ./...`
- [x] Update GOJA-032 changelog + diary with milestone notes and verification outputs
- [x] Add GOJA-028 cross-reference update noting GOJA-032 execution handoff/completion

## Done Criteria

- [x] Smalltalk inspector static-analysis behavior is served via `pkg/inspector/analysis` session methods
- [x] No direct UI dependency on `Resolution.Scopes` / `Index.Nodes` for migrated paths
- [x] Existing globals/members/jump behavior remains stable by tests
- [x] Full regression suite passes
