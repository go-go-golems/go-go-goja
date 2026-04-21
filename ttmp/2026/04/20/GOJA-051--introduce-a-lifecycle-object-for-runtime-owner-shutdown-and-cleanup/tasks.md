# Tasks

## Done

- [x] Create ticket workspace GOJA-051
- [x] Gather evidence from current runtime, factory, runtime module, and runtime owner code
- [x] Write design and implementation guide for a lifecycle object refactor
- [x] Write investigation diary
- [x] Relate key files to the design doc and diary
- [x] Run `docmgr doctor --ticket GOJA-051 --stale-after 30`
- [x] Dry-run reMarkable bundle upload
- [x] Upload final bundle to reMarkable and verify remote listing

## Follow-up implementation work

- [ ] Implement `engine/lifecycle.go`
- [ ] Migrate `Runtime.Close()` to lifecycle-managed phases
- [ ] Expose phase-aware lifecycle registration through `RuntimeModuleContext`
- [ ] Add lifecycle-phase tests and migrate existing close-hook tests
