# Tasks

## Done

- [x] Create a new docmgr ticket for the plugin authoring SDK workstream.
- [x] Inspect the current plugin contract, shared transport, host validation/reification path, and author-facing examples.
- [x] Compare the current plugin authoring shape against the built-in native-module authoring model.
- [x] Write a detailed intern-oriented analysis, design, and implementation guide for an author-facing SDK layer.
- [x] Record the investigation in a diary and update the changelog.
- [x] Relate key files, validate the ticket with `docmgr doctor`, and upload the bundle to reMarkable.

## Next

- [x] Narrow the implementation target to the richer but still opinionated v1 SDK surface:
  `sdk.Module`, `sdk.MustModule`, `sdk.Serve`, `sdk.ServeModule`, `sdk.Function`, `sdk.Object`, `sdk.Method`, `sdk.Call`, and minimal metadata helpers.
- [x] Confirm the package location and file layout for the SDK:
  `pkg/hashiplugin/sdk/module.go`, `export.go`, `call.go`, `convert.go`, `dispatch.go`, `serve.go`, `errors.go`, `sdk_test.go`.
- [x] Decide and document the exact public API we will ship in v1, including which of `Version`, `Doc`, and `Capabilities` are in scope immediately versus deferred.

## Phase 1: Skeleton And Definitions

- [x] Create the `pkg/hashiplugin/sdk` package skeleton and keep it depending only on `contract` and `shared`, not on `host`.
- [x] Define the core author-facing types:
  `Module`, `ModuleOption`, `ExportOption`, `ObjectOption`, `Handler`, and the internal function/object/method descriptor types.
- [x] Implement `sdk.Function(...)`, `sdk.Object(...)`, and `sdk.Method(...)` as declarative builders rather than map-only sugar.
- [x] Implement `sdk.NewModule(...)` and `sdk.MustModule(...)` so module definitions become immutable after construction.
- [x] Add SDK-side structural validation for:
  empty module name, duplicate exports, nil handlers, object exports without methods, duplicate method names, and empty method names.

## Phase 2: Manifest Generation

- [x] Implement manifest generation from SDK definitions into `contract.ModuleManifest`.
- [x] Support the chosen metadata subset in v1:
  at minimum module name, and optionally version/doc/capabilities if we keep them in scope.
- [x] Add tests proving SDK-generated manifests match host expectations for function exports and object-method exports.
- [x] Add tests proving invalid SDK definitions fail before they ever reach host validation.

## Phase 3: Dispatch And Conversion

- [x] Implement the internal dispatch table that maps `(exportName, methodName)` to a registered `Handler`.
- [x] Implement `sdk.Call` with raw request context plus convenience helpers for common reads.
- [x] Decide the minimal `sdk.Call` helper set for v1:
  `Len`, `Value`, `String`, `StringDefault`, `Float64`, `Bool`, `Map`, and `Slice`.
- [x] Implement request argument decoding from `[]*structpb.Value` to author-facing values.
- [x] Implement result encoding from plain Go values back to `*structpb.Value`.
- [x] Standardize handler and conversion errors so unsupported args/results fail with clear author-facing messages.
- [x] Add unit tests for dispatch success, unknown exports, unknown methods, bad arg conversions, and bad result conversions.

## Phase 4: Serve Wrapper

- [x] Implement `sdk.Module` so it satisfies `contract.JSModule` directly through `Manifest(...)` and `Invoke(...)`.
- [x] Implement `sdk.Serve(mod contract.JSModule)` as the thin wrapper over `shared.Handshake` and `shared.VersionedServerPluginSets(...)`.
- [x] Implement `sdk.ServeModule(name string, opts ...ModuleOption)` as the convenience happy path.
- [x] Decide whether `sdk.ServeModule(...)` returns an error or whether `sdk.MustModule(...)` + `sdk.Serve(...)` is the preferred explicit path in docs and examples.
- [x] Add tests proving a module built through the SDK can be dispensed through the existing shared gRPC adapter.

## Phase 5: Migrate Examples And Fixtures

- [x] Migrate `plugins/examples/greeter/main.go` to the SDK and keep the example obviously shorter than the current manual version.
- [x] Migrate `plugins/testplugin/echo/main.go` to the SDK if it improves clarity without obscuring integration-test intent.
- [x] Decide whether `plugins/testplugin/invalid/main.go` should remain handwritten to preserve raw-contract coverage, or be replaced by an SDK-driven invalid case that still exercises the intended failure mode.
- [x] Update `plugins/examples/README.md` so the example shows the richer SDK surface rather than the old low-level contract.

## Phase 6: Integration Validation

- [x] Extend `pkg/hashiplugin/host/registrar_test.go` so at least one integration test builds an SDK-authored plugin and loads it through the existing runtime path.
- [x] Confirm that host-side behavior from JavaScript remains unchanged:
  `require("plugin:...")`, top-level functions, object methods, and runtime cleanup after close.
- [x] Add at least one integration test that proves SDK-generated manifests remain compatible with host validation rules.
- [x] Run focused package tests for `pkg/hashiplugin/sdk`, `pkg/hashiplugin/shared`, `pkg/hashiplugin/host`, and the example/test plugin packages.

## Phase 7: Documentation

- [x] Update the GOJA-09 design doc if the shipped v1 API differs from the current proposed API.
- [x] Update `pkg/doc/12-plugin-user-guide.md` to keep the user-facing story stable while pointing plugin authors at the new SDK-based examples.
- [x] Update `pkg/doc/13-plugin-developer-guide.md` to explain the new layering:
  author-facing `sdk`, low-level `contract/shared`, host-side `host`, runtime-side `engine`.
- [x] Rewrite `pkg/doc/14-plugin-tutorial-build-install.md` so the primary authoring path uses `sdk.ServeModule(...)` or `sdk.MustModule(...)`.
- [x] Add one short quickstart snippet showing the richer SDK in the smallest useful form.

## Phase 8: Ticket Closeout

- [x] Update the GOJA-09 diary after each implementation slice with commands, failures, and commit hashes.
- [x] Keep `tasks.md` checked off task by task as the SDK work lands.
- [x] Update `changelog.md` with meaningful reviewable slices rather than one bulk entry.
- [x] Run `docmgr doctor --ticket GOJA-09-PLUGIN-AUTHORING-SDK--create-a-plugin-authoring-sdk-layer-for-hashicorp-goja-plugins --stale-after 30`.
- [x] Upload the refreshed GOJA-09 bundle to reMarkable and verify the remote listing.

## Phase 9: Example Catalog Expansion

- [x] Re-open GOJA-09 to expand the user-facing example catalog for the richer SDK surface.
- [x] Add `plugins/examples/clock` to demonstrate metadata, zero-argument functions, and structured object returns.
- [x] Add `plugins/examples/validator` to demonstrate `sdk.Call` helpers, optional/default handling, and author-facing validation errors.
- [x] Add `plugins/examples/kv` to demonstrate stateful object methods inside one plugin subprocess.
- [x] Add `plugins/examples/system-info` to demonstrate mixed export shapes and nested JSON-like responses.
- [x] Add `plugins/examples/failing` to demonstrate explicit error returns and host-visible failure behavior.
- [x] Rewrite `plugins/examples/README.md` so it acts as a catalog instead of a single-example note.
- [x] Update the plugin user/developer/tutorial help pages to reference the broader example set where useful.
- [x] Add focused runtime or integration coverage for at least the stateful and error-propagation examples.
- [x] Re-run the relevant example, host, and docs validation passes.
- [x] Update the GOJA-09 diary and changelog for the new example slices.
- [ ] Refresh the GOJA-09 bundle on reMarkable after the expanded example catalog lands.
