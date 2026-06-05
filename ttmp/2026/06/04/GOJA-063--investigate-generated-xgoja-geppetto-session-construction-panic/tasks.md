# Tasks

## TODO

- [ ] Decide whether generated xgoja profile smokes should include a local fake API key fixture by default, or point at Pinocchio's minimal profile and expect a normal missing-key error for agent construction.
- [ ] Update GOJA-053 docs/example notes if the final fix changes provider-author guidance.

## DONE

- [x] Create GOJA-063 ticket workspace.
- [x] Create analysis document and investigation diary.
- [x] Add a reproducible generated xgoja script for the crash.
- [x] Capture a stack trace by temporarily instrumenting `runtimeowner` panic recovery.
- [x] Identify the first Geppetto frame as `ensureInferenceSettingsProviderDefaults` in `api_engines.go`.
- [x] Add a focused Geppetto unit/regression test for `ensureInferenceSettingsProviderDefaults` with `Chat.ApiType` set and `API` unset.
- [x] Fix `ensureInferenceSettingsProviderDefaults` so it handles nil `InferenceSettings.API` safely.
- [x] Add a Geppetto JS regression test that builds an agent and session from a profile-backed inference settings object.
- [x] Re-run the exact generated xgoja no-inference session-construction repro and confirm it no longer panics; it now returns a normal missing API key error for the minimal Pinocchio profile fixture.
- [x] Restore the `examples/xgoja/12-geppetto-host-services` deterministic `profile-smoke` port to construct a session using a local profile fixture with a dummy API key.
- [x] Run the live generated xgoja Geppetto host-services smoke after the fix to confirm no regression.
- [x] Add optional debug stack traces for recovered runtimeowner panics.
- [x] Expose recovered panic stacks through generated xgoja Glazed flag `--debug-panic-stack`.
