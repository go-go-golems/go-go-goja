# Tasks

## TODO

- [ ] Add a focused Geppetto unit/regression test for building an agent from profile settings with `Chat.ApiType` set and `API` unset.
- [ ] Fix `ensureInferenceSettingsProviderDefaults` so it handles nil `InferenceSettings.API` safely.
- [ ] Re-run the exact generated xgoja no-inference session-construction repro and confirm it returns JSON rather than a panic.
- [ ] Restore the `examples/xgoja/12-geppetto-host-services` deterministic `profile-smoke` port to construct a session once the crash is fixed.
- [ ] Run the live generated xgoja Geppetto host-services smoke after the fix to confirm no regression.
- [ ] Update GOJA-053 docs/example notes if the final fix changes provider-author guidance.

## DONE

- [x] Create GOJA-063 ticket workspace.
- [x] Create analysis document and investigation diary.
- [x] Add a reproducible generated xgoja script for the crash.
- [x] Capture a stack trace by temporarily instrumenting `runtimeowner` panic recovery.
- [x] Identify the first Geppetto frame as `ensureInferenceSettingsProviderDefaults` in `api_engines.go`.
