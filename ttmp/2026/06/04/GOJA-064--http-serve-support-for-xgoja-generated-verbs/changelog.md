# Changelog

## 2026-06-04

- Initial workspace created


## 2026-06-04

Created GOJA-064 design and diary for HTTP serve support in xgoja generated verbs; mapped jsverbs, command providers, HTTP provider, express, gojahttp, and goja-site reference architecture.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-064--http-serve-support-for-xgoja-generated-verbs/design-doc/01-http-serve-support-for-xgoja-generated-verbs.md — Primary design and implementation guide
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-064--http-serve-support-for-xgoja-generated-verbs/reference/01-diary.md — Chronological investigation diary


## 2026-06-04

Validated GOJA-064 with docmgr doctor, resolved missing topic vocabulary, uploaded the design/diary/task/changelog bundle to reMarkable at /ai/2026/06/04/GOJA-064.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-064--http-serve-support-for-xgoja-generated-verbs/design-doc/01-http-serve-support-for-xgoja-generated-verbs.md — Included in uploaded bundle
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-064--http-serve-support-for-xgoja-generated-verbs/reference/01-diary.md — Included in uploaded bundle and updated with delivery evidence


## 2026-06-04

Added GOJA-064 research logbook covering consulted source files, docs, examples, external goja-site resources, stale paths, and update needs; uploaded updated bundle including the logbook to reMarkable.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-064--http-serve-support-for-xgoja-generated-verbs/reference/01-diary.md — Updated with logbook creation and upload step
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-064--http-serve-support-for-xgoja-generated-verbs/reference/02-research-logbook.md — New resource usefulness and freshness logbook


## 2026-06-04

Updated GOJA-064 serve design for the simplified single-runtime xgoja.yaml schema: top-level modules list, no command runtime fields, no commandProviders runtimeProfile in recommended examples.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-064--http-serve-support-for-xgoja-generated-verbs/design-doc/01-http-serve-support-for-xgoja-generated-verbs.md — Plan updated for single-runtime xgoja schema
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/ttmp/2026/06/04/GOJA-064--http-serve-support-for-xgoja-generated-verbs/reference/01-diary.md — Recorded schema simplification update


## 2026-06-04

Implemented first GOJA-064 code slice: command-provider jsverb source access plus go-go-goja-http serve command provider with long-lived runtime invocation.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/app/jsverb_sources.go — Centralized jsverb source scanning for command providers
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/providerapi/commands.go — Added JSVerbSourceSet API
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/providers/http/serve.go — Implemented HTTP serve command provider


## 2026-06-04

Added generated-binary HTTP serve jsverb smoke test and examples/xgoja/13-http-serve-jsverbs runnable example.

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/internal/generate/generate_test.go — Generated serve command smoke coverage
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/examples/xgoja/13-http-serve-jsverbs/verbs/sites.js — New example setup verb
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/examples/xgoja/13-http-serve-jsverbs/xgoja.yaml — New example buildspec


## 2026-06-04

Step 9: hardened HTTP serve startup errors with synchronous net.Listen binding and added xgoja help documentation for HTTP serve jsverbs (commit 9af57aabb02a554c746b2ea29c14503bed9373f3)

### Related Files

- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/cmd/xgoja/doc/12-tutorial-http-serve-jsverbs.md — New tutorial
- /home/manuel/workspaces/2026-06-03/goja-runtime-flags/go-go-goja/pkg/xgoja/providers/http/http.go — Synchronous bind before serving

