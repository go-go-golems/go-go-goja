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
- [ ] FSWATCH-002: Define FSWatchOptions, helper initializer API, and JavaScript contract
- [ ] FSWATCH-003: Implement FSWatchHelper that installs fswatch.watch(path, emitter, options?)
- [ ] FSWATCH-004: Validate and normalize watched paths with AllowPath and optional Root policy
- [ ] FSWATCH-005: Adopt JS-created Go-native EventEmitter instances via Manager.AdoptEmitterOnOwner
- [ ] FSWATCH-006: Implement watcher goroutine lifecycle, connection close, and runtime cancellation
- [ ] FSWATCH-007: Emit fsnotify events with bitmask-derived booleans and local error/close events
- [ ] FSWATCH-008: Add fswatch tests for event delivery, denied paths, invalid emitters, close, and add failures
- [ ] FSWATCH-009: Add jsverbs or example script demonstrating fswatch helper usage in an embedding runtime
- [ ] FSWATCH-010: Update docs and diary, run validation, and commit fsnotify helper slice
