# Changelog

## 2026-06-04

- Initial workspace created


## 2026-06-04

Created GOJA-063 crash investigation workspace, added repro script, captured runtimeowner stack trace, and documented likely nil API settings root cause.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-063--investigate-generated-xgoja-geppetto-session-construction-panic/analysis/01-session-construction-panic-analysis.md — Initial crash analysis and hypothesis
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-063--investigate-generated-xgoja-geppetto-session-construction-panic/reference/01-investigation-diary.md — Step 1 diary entry
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-063--investigate-generated-xgoja-geppetto-session-construction-panic/scripts/01-reproduce-session-construction-panic.sh — Repro script


## 2026-06-04

Fixed Geppetto nil API settings panic, restored generated xgoja profile-smoke session construction with a local deterministic profile fixture, and validated live host-services smoke.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/geppetto/pkg/js/modules/geppetto/api_agent_profile_test.go — Regression tests (commit 4c975f1b)
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/geppetto/pkg/js/modules/geppetto/api_engines.go — Nil API settings fix (commit 4c975f1b)
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/examples/xgoja/12-geppetto-host-services/verbs/pinocchio_profiles.js — Restored profile smoke session construction (commit 95a9c4a)
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-063--investigate-generated-xgoja-geppetto-session-construction-panic/reference/01-investigation-diary.md — Step 2 diary entry


## 2026-06-04

Added opt-in runtimeowner recovered panic stack traces and engine builder plumbing for future provider crash diagnostics.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/engine/options.go — Engine builder option (commit 2a81564)
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/runtimeowner/runner.go — Recovered panic stack implementation (commit 2a81564)
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-063--investigate-generated-xgoja-geppetto-session-construction-panic/reference/01-investigation-diary.md — Step 3 diary entry

