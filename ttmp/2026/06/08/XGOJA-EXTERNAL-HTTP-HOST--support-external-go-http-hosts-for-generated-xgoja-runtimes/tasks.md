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
- [ ] Add HTTP provider `HostServiceKey` and `ExternalHostService` payload.
- [ ] Make the HTTP provider consume external host services from `ModuleSetupContext.Host`.
- [ ] Ensure external no-listen mode does not bind a TCP listener.
- [ ] Add `gojahttp` route introspection for registry/host route descriptors.
- [ ] Add focused app/provider/gojahttp tests.
- [x] Add generated package smoke coverage for service injection.
- [ ] Prototype RuntimeManager behavior app-locally before extracting generic runtime-manager APIs.
