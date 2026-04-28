# Tasks

## TODO

- [x] Create ticket workspace and import /tmp/event-emitter.md
- [x] Read imported brief and gather repository evidence
- [x] Write detailed intern-oriented event-emitter implementation guide
- [x] Maintain diary and ticket bookkeeping
- [x] Validate ticket with docmgr doctor
- [x] Upload document bundle to reMarkable and verify listing
- [x] Implement Go-native events module with EventEmitter constructor and core listener methods
- [x] Add runtime integration tests for require("events"), listener semantics, and Go adoption of JS-created emitters
- [x] Add jsverbs example scripts demonstrating EventEmitter usage
- [x] Update docs/declarations for the events module
- [x] Run targeted and full validation, update diary, and commit implementation
- [x] Implement connected-emitter manager with EmitterRef adoption, owner-thread emit, and close lifecycle
- [x] Implement opt-in Watermill helper that connects JS-provided EventEmitter instances
- [x] Add connected-emitter and Watermill helper tests
- [x] Validate connected-emitter slice, update diary, and commit
- [x] FSWATCH-001: Write fsnotify connected-emitter design and implementation guide
- [x] FSWATCH-002: Define FSWatchOptions, helper initializer API, and JavaScript contract
- [x] FSWATCH-003: Implement FSWatchHelper that installs fswatch.watch(path, emitter, options?)
- [x] FSWATCH-004: Validate and normalize watched paths with AllowPath and optional Root policy
- [x] FSWATCH-005: Adopt JS-created Go-native EventEmitter instances via Manager.AdoptEmitterOnOwner
- [x] FSWATCH-006: Implement watcher goroutine lifecycle, connection close, and runtime cancellation
- [x] FSWATCH-007: Emit fsnotify events with bitmask-derived booleans and local error/close events
- [x] FSWATCH-008: Add fswatch tests for event delivery, denied paths, invalid emitters, close, and add failures
- [x] FSWATCH-009: Add jsverbs or example script demonstrating fswatch helper usage in an embedding runtime
- [x] FSWATCH-010: Update docs and diary, run validation, and commit fsnotify helper slice
- [x] FSWATCH-RDG-001: Design recursive watching, debouncing, and glob filtering with typed Go structs
- [x] FSWATCH-RDG-002: Replace fswatch JS option decoding and event/error payload maps with typed Go structs
- [x] FSWATCH-RDG-003: Refactor fswatch watcher loop into an fsWatchState that owns watcher, path policy, and cleanup
- [x] FSWATCH-RDG-004: Implement recursive initial directory walk, dynamic new-directory registration, and watched-directory bookkeeping
- [x] FSWATCH-RDG-005: Implement glob include/exclude filtering for emitted events and recursive directory traversal
- [x] FSWATCH-RDG-006: Implement trailing debounce with merged fsnotify ops, event counts, and timer cleanup on close
- [x] FSWATCH-RDG-007: Add tests for typed payloads/options, recursion, glob filtering, debouncing, and close cleanup
- [x] FSWATCH-RDG-008: Extend fswatch jsverb example with recursive/debounce/glob options
- [x] FSWATCH-RDG-009: Update docs, diary, changelog, relationships, run validation, and commit final fswatch recursion/debounce/glob slice
