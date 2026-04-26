# Changelog

## 2026-04-26

- Initial workspace created


## 2026-04-26

Created EVT-001 ticket, imported event-emitter source brief, gathered runtime/module evidence, wrote detailed implementation guide and diary.

### Related Files

- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/design-doc/01-event-emitter-module-implementation-guide.md — Primary intern-oriented design and implementation guide.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/reference/01-diary.md — Chronological work diary.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/scripts/01-gather-event-emitter-evidence.sh — Reproducible evidence capture script.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/sources/local/01-event-emitter.md — Imported source brief.


## 2026-04-26

Validated EVT-001 with docmgr doctor after adding vocabulary entries and normalizing imported source frontmatter/numeric prefix.

### Related Files

- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/sources/local/01-event-emitter.md — Imported brief renamed with numeric prefix and docmgr frontmatter.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/tasks.md — Marked setup
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/vocabulary.yaml — Added event-emitter


## 2026-04-26

Uploaded EVT-001 document bundle to reMarkable under /ai/2026/04/26/EVT-001 and verified the cloud listing.

### Related Files

- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/design-doc/01-event-emitter-module-implementation-guide.md — Included in the uploaded reMarkable bundle.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/reference/01-diary.md — Included in the uploaded reMarkable bundle.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/sources/local/01-event-emitter.md — Included in the uploaded reMarkable bundle as source material.


## 2026-04-26

Revised EVT-001 design to use JS-called Go factory functions returning connected EventEmitter instances; Watermill is now an opt-in helper, not a default emitter.

### Related Files

- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/design-doc/01-event-emitter-module-implementation-guide.md — Updated primary design around connected emitters.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/reference/01-diary.md — Recorded design revision prompted by user clarification.


## 2026-04-26

Revised EVT-001 design so the events module is implemented natively in Go, with Go-backed EventEmitter objects and adoption of JS-created native emitters.

### Related Files

- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/design-doc/01-event-emitter-module-implementation-guide.md — Updated primary design for Go-native EventEmitter implementation.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/reference/01-diary.md — Recorded Go-native EventEmitter design revision.


## 2026-04-26

Implemented Go-native EventEmitter module, jsverbs examples, TypeScript declarations, docs, and validation (commits b37c256, 12c497d, a905896).

### Related Files

- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/cmd/bun-demo/js/src/types/goja-modules.d.ts — Generated declarations for events and node:events.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/modules/events/events.go — Go-native EventEmitter module implementation.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/modules/events/events_test.go — EventEmitter runtime integration tests.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/testdata/jsverbs/events.js — jsverbs EventEmitter example scripts.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/reference/01-diary.md — Implementation diary updated with commands


## 2026-04-26

Implemented connected-emitter manager and opt-in Watermill helper for JS-provided EventEmitter instances (commit 0a5f322).

### Related Files

- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/jsevents/manager.go — Connected-emitter manager and EmitterRef implementation.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/jsevents/manager_test.go — Connected-emitter manager tests.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/jsevents/watermill.go — Opt-in Watermill helper exposing watermill.connect(topic
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/jsevents/watermill_test.go — Watermill helper ack/nack and validation tests.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/reference/01-diary.md — Recorded connected-emitter implementation step.


## 2026-04-26

Created fsnotify connected-emitter helper tasks and a detailed implementation guide; implementation tasks remain open.

### Related Files

- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/design-doc/02-fsnotify-connected-emitter-helper-implementation-guide.md — New fsnotify design and implementation guide.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/reference/01-diary.md — Recorded fsnotify planning step.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/tasks.md — Added FSWATCH-001 through FSWATCH-010 tasks.


## 2026-04-26

Implemented fsnotify connected-emitter helper and jsverbs example (commits 33819cb and 28058a7); updated docs and guide.

### Related Files

- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/doc/03-async-patterns.md — connected-emitter pattern docs.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/doc/16-nodejs-primitives.md — fswatch helper docs.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/jsevents/fswatch.go — FSWatchHelper implementation.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/jsevents/fswatch_test.go — FSWatchHelper tests.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/jsverbs/jsverbs_test.go — fswatch jsverbs runtime integration test.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/testdata/jsverbs/fswatch.js — fswatch jsverbs example.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/design-doc/02-fsnotify-connected-emitter-helper-implementation-guide.md — fsnotify design guide updated to match implementation.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/reference/01-diary.md — Step 14 implementation diary.


## 2026-04-26

Created recursive fswatch/debounce/glob design guide and implementation task set; implementation tasks remain open.

### Related Files

- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/design-doc/03-fswatch-recursion-debouncing-and-glob-filtering-guide.md — New recursive fswatch/debounce/glob design guide.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/reference/01-diary.md — Recorded recursive fswatch planning step.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/tasks.md — Added FSWATCH-RDG-001 through FSWATCH-RDG-009 tasks.


## 2026-04-26

Implemented recursive fswatch, glob filtering, debounce, and typed Go payloads (commits bc01d99 and d67b767); updated docs and diary.

### Related Files

- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/doc/03-async-patterns.md — async docs update.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/doc/16-nodejs-primitives.md — primitive docs update.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/jsevents/fswatch.go — recursive/debounce/glob fswatch implementation.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/jsevents/fswatch_test.go — recursive/debounce/glob fswatch tests.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/jsverbs/jsverbs_test.go — extended jsverbs integration coverage.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/testdata/jsverbs/fswatch.js — extended jsverbs example.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/design-doc/03-fswatch-recursion-debouncing-and-glob-filtering-guide.md — recursive/debounce/glob design guide.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/reference/01-diary.md — Step 16 diary entry.


## 2026-04-26

Added embedded Glazed connected EventEmitter developer documentation and updated related public docs.

### Related Files

- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/README.md — connected EventEmitter/fswatch overview.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/doc/03-async-patterns.md — connected emitter cross-links.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/doc/08-jsverbs-example-overview.md — fswatch fixture caveat.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/doc/11-jsverbs-example-reference.md — connected-helper fixture section.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/doc/16-nodejs-primitives.md — connected emitter guide cross-link.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/doc/17-connected-eventemitters-developer-guide.md — new embedded Glazed developer guide.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/reference/01-diary.md — Step 17 documentation diary.


## 2026-04-26

Added systematic node: aliases for Node-compatible modules while keeping custom modules unprefixed (commit 3e2f797).

### Related Files

- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/cmd/bun-demo/js/src/types/goja-modules.d.ts — generated alias declarations.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/engine/granular_modules_test.go — alias availability tests.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/engine/module_specs.go — node alias expansion and node:process registration.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/engine/nodejs_primitives_test.go — node:process and goja_nodejs alias tests.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/modules/crypto/crypto.go — node:crypto alias.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/modules/fs/fs.go — node:fs alias.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/modules/os/os.go — node:os alias.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/modules/path/path.go — node:path alias.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/doc/16-nodejs-primitives.md — node alias docs.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/reference/01-diary.md — Step 18 diary entry.


## 2026-04-26

Addressed PR #31 review comments: guarded emit() with no event name, preserved symbol event identity, and fixed single-file fswatch relativeName filtering (commit 972a9ab).

### Related Files

- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/modules/events/events.go — eventName key type and emit() guard.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/modules/events/events_test.go — EventEmitter review regression tests.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/jsevents/fswatch.go — watched-file basename relativeName fix.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/pkg/jsevents/fswatch_test.go — single-file glob regression test.
- /home/manuel/workspaces/2026-04-26/add-event-emitter-module/go-go-goja/ttmp/2026/04/26/EVT-001--event-emitter-module-for-go-go-goja/reference/01-diary.md — Step 19 diary entry.

