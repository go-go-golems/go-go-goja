# Changelog

## 2026-06-13

- Initial workspace created


## 2026-06-13

Created hard-cutover xgoja v2 runtime plan design guide and investigation diary; documented removal of legacy runtime metadata bridge plus docs/migration update requirements.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/ttmp/2026/06/13/GOJA-XGOJA-V2-RUNTIME-001--replace-legacy-xgoja-runtime-metadata-bridge-with-v2-native-runtime-plan/design-doc/01-xgoja-v2-native-runtime-plan-design-and-implementation-guide.md — Primary implementation guide
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/ttmp/2026/06/13/GOJA-XGOJA-V2-RUNTIME-001--replace-legacy-xgoja-runtime-metadata-bridge-with-v2-native-runtime-plan/reference/01-investigation-diary.md — Investigation diary


## 2026-06-13

Validated ticket docs and uploaded GOJA XGOJA V2 Runtime Cutover Guide bundle to reMarkable at /ai/2026/06/13/GOJA-XGOJA-V2-RUNTIME-001.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/ttmp/2026/06/13/GOJA-XGOJA-V2-RUNTIME-001--replace-legacy-xgoja-runtime-metadata-bridge-with-v2-native-runtime-plan/design-doc/01-xgoja-v2-native-runtime-plan-design-and-implementation-guide.md — Uploaded in reMarkable bundle


## 2026-06-13

Expanded the ticket with a detailed 52-task phased implementation backlog covering hard cutover prep, RuntimePlan types, generator rewrite, SourceRegistry, app runtime rewrite, provider API/HTTP serve updates, runtime-package migration, docs/migration guide updates, sessionstream chatbot smoke, legacy removal sweep, validation, and final reMarkable upload.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/ttmp/2026/06/13/GOJA-XGOJA-V2-RUNTIME-001--replace-legacy-xgoja-runtime-metadata-bridge-with-v2-native-runtime-plan/tasks.md — Detailed phased implementation backlog


## 2026-06-13

Phase 0 task 7: created dedicated local branch task/xgoja-v2-runtime-cutover from merged go-go-goja main plus ticket docs; baseline head is 70e98b3.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/ttmp/2026/06/13/GOJA-XGOJA-V2-RUNTIME-001--replace-legacy-xgoja-runtime-metadata-bridge-with-v2-native-runtime-plan/tasks.md — Tracks cutover implementation tasks


## 2026-06-13

Phase 0 task 8: added and ran scripts/01-reproduce-provider-command-source-loss.sh, which proves v2 commands[].sources is dropped in generated legacy commandProviders metadata while all jsverb sources remain global.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/ttmp/2026/06/13/GOJA-XGOJA-V2-RUNTIME-001--replace-legacy-xgoja-runtime-metadata-bridge-with-v2-native-runtime-plan/scripts/01-reproduce-provider-command-source-loss.sh — Reproduces provider command-set source loss


## 2026-06-13

Phase 0: added and passed an xgoja build regression proving provider.command-set sources scope HTTP serve jsverb commands while preserving serve flags.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/xgoja/internal/generate/templates.go — Carries v2 command sources into generated runtime metadata during the interim bridge
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/cmd/xgoja/root_test.go — Generated binary regression for command-scoped HTTP serve jsverb sources
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/xgoja/app/command_providers.go — Passes command-scoped JS verb sources into provider command contexts
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/xgoja/app/jsverb_sources.go — Filters JS verb sources by command source IDs
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/go-go-goja/pkg/xgoja/app/runtime_spec.go — Adds source IDs to command provider runtime metadata for the regression fix

