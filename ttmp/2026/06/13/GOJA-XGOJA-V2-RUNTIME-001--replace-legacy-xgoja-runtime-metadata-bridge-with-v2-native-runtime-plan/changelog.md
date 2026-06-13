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

