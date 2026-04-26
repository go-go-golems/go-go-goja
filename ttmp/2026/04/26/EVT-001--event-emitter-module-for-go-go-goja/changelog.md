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

