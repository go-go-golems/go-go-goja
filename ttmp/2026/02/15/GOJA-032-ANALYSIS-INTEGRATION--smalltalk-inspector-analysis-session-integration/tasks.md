# Tasks

## TODO

- [x] Add/expand analysis session API in `pkg/inspector/analysis` for globals, members, and source-jump metadata
- [x] Add unit tests for new analysis-session methods (sorting, membership, declaration line lookup)
- [ ] Introduce `analysisSession` field in smalltalk-inspector model and initialize on file load
- [ ] Migrate `buildGlobals` to consume analysis-session API (remove direct root-scope traversal from UI)
- [ ] Migrate class/function member builders to analysis-session API
- [ ] Migrate source jump helpers (`jumpToBinding`, `jumpToMember`) to analysis-session API
- [ ] Keep runtime-only inspection paths unchanged and verify mixed static/runtime behavior
- [ ] Remove obsolete direct `jsparse` graph reads from smalltalk-inspector static-analysis paths
- [ ] Run regression suite: `pkg/inspector/analysis`, `cmd/smalltalk-inspector`, `cmd/inspector`, `go test ./...`
- [ ] Update GOJA-032 changelog + diary with milestone notes and verification outputs
- [ ] Add GOJA-028 cross-reference update noting GOJA-032 execution handoff/completion

## Done Criteria

- [ ] Smalltalk inspector static-analysis behavior is served via `pkg/inspector/analysis` session methods
- [ ] No direct UI dependency on `Resolution.Scopes` / `Index.Nodes` for migrated paths
- [ ] Existing globals/members/jump behavior remains stable by tests
- [ ] Full regression suite passes
