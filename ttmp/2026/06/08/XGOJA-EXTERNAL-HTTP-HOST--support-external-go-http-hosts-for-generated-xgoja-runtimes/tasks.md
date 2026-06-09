# Tasks

## Ticket setup and design delivery

- [x] Create `XGOJA-EXTERNAL-HTTP-HOST` docmgr ticket in `go-go-goja/ttmp`.
- [x] Add primary design document for the non-invasive external Go HTTP host integration approach.
- [x] Add investigation diary entry documenting prompt context, evidence collection, and delivery commands.
- [x] Relate key implementation files and source design documents to the ticket docs.
- [x] Validate ticket hygiene with `docmgr doctor`.
- [x] Upload the design bundle to reMarkable.

## Future implementation phases

- [x] Add `app.HostServices` helper methods for host-supplied keyed services.
- [x] Add `ConfigureServices func(*app.HostServices)` to `app.HostOptions`.
- [x] Wire `ConfigureServices` into generated package and bundle-fragment templates.
- [x] Add HTTP provider `HostServiceKey` and `ExternalHostService` payload.
- [x] Make the HTTP provider consume external host services from `ModuleSetupContext.Host`.
- [x] Ensure external no-listen mode does not bind a TCP listener.
- [x] Add `gojahttp` route introspection for registry/host route descriptors.
- [x] Add focused app/provider/gojahttp tests.
- [x] Add generated package smoke coverage for service injection.
- [x] Prototype RuntimeManager behavior with blue/green reload, last-known-good fallback, status, and polling watcher.

## HTTP serve hot reload integration

### Phase 1: design and planning

- [x] Add `HTTP Serve Hot Reload Implementation Guide` design document.
- [x] Add detailed task checklist for `serve --hot-reload`.
- [x] Relate the hot reload serve design to implementation files.
- [x] Commit planning docs.

### Phase 2: runtime factory per-runtime host services

- [x] Add optional `providerapi.RuntimeFactoryWithHostServices` interface.
- [x] Implement per-runtime host service injection in `app.RuntimeFactory`.
- [x] Preserve existing `NewRuntime` / `NewRuntimeFromSections` behavior.
- [x] Add focused tests proving command-time services reach provider module setup.
- [x] Commit Phase 2 implementation and diary update.

### Phase 3: serve hot-reload command flags

- [x] Add serve-specific hot reload Glazed section/flags.
- [x] Decode hot reload settings from parsed command values.
- [x] Test that generated serve commands expose the new flags.
- [x] Commit Phase 3 implementation and diary update.

### Phase 4: serve hot-reload execution path

- [x] Branch `serveVerb` to a hot-reload path when `--hot-reload` is enabled.
- [x] Inject candidate `ExternalHostService{OwnsListen:false}` per reload.
- [x] Start one Go-owned HTTP server around `hotreload.Manager` using `--http-listen`.
- [x] Implement optional status endpoint and optional smoke path.
- [x] Wire watcher roots/extensions/poll/debounce/close grace.
- [x] Add focused provider tests for reload success, last-known-good, status, and smoke failure.
- [x] Commit Phase 4 implementation and diary update.

### Phase 5: generated binary integration test

- [ ] Add/update generated binary integration coverage for `serve --hot-reload`.
- [ ] Verify health endpoint response.
- [ ] Verify file change increments hot reload status version.
- [ ] Verify broken edit keeps last-known-good runtime serving.
- [ ] Commit Phase 5 tests and diary update.

### Phase 6: docs and final validation

- [ ] Update xgoja user docs for `serve --hot-reload`.
- [ ] Run focused tests and `go test ./...`.
- [ ] Run `docmgr doctor`.
- [ ] Mark all hot reload serve tasks complete.
- [ ] Commit final docs and diary update.
