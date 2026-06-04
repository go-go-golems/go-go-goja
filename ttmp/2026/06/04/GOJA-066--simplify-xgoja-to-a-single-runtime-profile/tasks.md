# Tasks

## TODO

- [ ] Phase 1: Replace buildspec `runtimes` map with top-level `modules` and remove command `runtime` / command-provider `runtimeProfile` fields.
- [ ] Phase 2: Update embedded runtime JSON generation to emit `modules` instead of `runtimes`.
- [ ] Phase 3: Update `pkg/xgoja/app` runtime DTOs and `RuntimeFactory` APIs to use the single module set.
- [ ] Phase 4: Remove generated command `--runtime` flags and profile-selection code from eval/run/repl/jsverbs.
- [ ] Phase 5: Simplify command-provider runtime context and preserve optional module filtering.
- [ ] Phase 6: Update xgoja docs and all examples from `runtimes.<name>.modules` to top-level `modules`.
- [ ] Phase 7: Run focused and full test suites plus representative example smokes.

## DONE

- [x] Create GOJA-066 ticket workspace.
- [x] Write single-runtime-profile analysis/design/implementation guide for intern onboarding.
- [x] Relate current multi-profile schema, validation, generation, runtime factory, commands, and provider API files to the design document.
- [x] Validate GOJA-066 with `docmgr doctor`.
- [x] Upload the design + diary bundle to reMarkable at `/ai/2026/06/04/GOJA-066`.
