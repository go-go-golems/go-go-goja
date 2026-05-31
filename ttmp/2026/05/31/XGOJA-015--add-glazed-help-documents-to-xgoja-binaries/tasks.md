# Tasks

## Completed research/setup

- [x] Create docmgr ticket XGOJA-015.
- [x] Add primary design/implementation guide document.
- [x] Add investigation diary document.
- [x] Inspect xgoja builder help wiring and generated-binary help wiring.
- [x] Inspect buildspec/generator patterns for embedded source directories.
- [x] Inspect providerapi extension points for provider-owned resources.
- [x] Inspect Loupedeck Glazed help docs and standalone CLI help setup.
- [x] Write intern-friendly design with API sketches, pseudocode, diagrams, implementation phases, tests, and file references.

## Implementation tasks

### Phase 1: Provider API support

- [ ] Add `providerapi.HelpSource` with `Name`, `Description`, `FS`, and `Root` fields.
- [ ] Add `HelpSources` storage to `providerapi.Package` snapshots and registry clones.
- [ ] Add `Registry.ResolveHelpSource(packageID, sourceName)`.
- [ ] Validate duplicate help source names and missing help source filesystems.
- [ ] Extend providerapi tests for successful help source resolution and invalid entries.
- [ ] Run `go test ./pkg/xgoja/providerapi -count=1`.
- [ ] Commit provider API support.

### Phase 2: Buildspec/runtime spec support

- [ ] Add `help.sources` YAML schema to `cmd/xgoja/internal/buildspec`.
- [ ] Add matching JSON runtime spec types to `pkg/xgoja/app`.
- [ ] Add `validateHelp(...)` for duplicate IDs, missing IDs, missing paths, mixed provider/path sources, unknown package IDs, and embedded path existence.
- [ ] Add buildspec tests for valid and invalid help source declarations.
- [ ] Run `go test ./cmd/xgoja/internal/buildspec -count=1`.
- [ ] Commit buildspec/runtime spec support.

### Phase 3: Generator embedding support

- [ ] Add generated runtime JSON rendering for `help.sources`.
- [ ] Add embedded local help root rewriting under `xgoja_embed/help/<source-id>`.
- [ ] Add `copyEmbeddedHelpSources(...)` to copy local help docs into generated workspaces.
- [ ] Update generated `main.go` template to embed JS verbs, help docs, or both without unused imports/variables.
- [ ] Pass `EmbeddedHelp` to `app.NewRootCommand` and `app.NewHostWithOptions` in all target modes.
- [ ] Add generator tests for template output, collision-free root rewriting, copied help files, and generated binary help smoke coverage.
- [ ] Run `go test ./cmd/xgoja/internal/generate -count=1`.
- [ ] Commit generator embedding support.

### Phase 4: Generated root loading support

- [ ] Add `EmbeddedHelp` to `app.Options`, `HostOptions`, and `Host`.
- [ ] Pass provider registry and embedded help FS into `installRootFramework`.
- [ ] Load built-in generated docs, provider help sources, embedded local help sources, and optional runtime filesystem sources into one Glazed `HelpSystem`.
- [ ] Keep `help_cmd.SetupCobraRootCommand(...)` called exactly once.
- [ ] Add app tests for provider help source lookup, embedded local help lookup, and missing provider source errors.
- [ ] Run `go test ./pkg/xgoja/app -count=1`.
- [ ] Commit generated root loading support.

### Phase 5: Loupedeck provider documentation support

- [ ] Export `FS() fs.FS` from `loupedeck/docs/help` without breaking `AddDocToHelpSystem`.
- [ ] Register a `providerapi.HelpSource{Name: "runtime-api"}` in `loupedeck/runtime/js/provider.Register`.
- [ ] Add Loupedeck provider tests that resolve `loupedeck.runtime-api` and verify its filesystem contains the API reference.
- [ ] Run `go test ./runtime/js/provider ./pkg/xgoja/provider -count=1` in `loupedeck`.
- [ ] Commit Loupedeck provider docs support.

### Phase 6: End-to-end docs, diary, and validation

- [ ] Update xgoja help docs with the new `help.sources` buildspec reference and provider docs guidance.
- [ ] Add or update examples showing provider-shipped and embedded local help docs.
- [ ] Run focused go-go-goja tests: `go test ./pkg/xgoja/providerapi ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate ./pkg/xgoja/app ./cmd/xgoja/... -count=1`.
- [ ] Run focused loupedeck tests after provider docs wiring.
- [ ] Update diary after each phase with exact commands, failures, tricky parts, and review instructions.
- [ ] Update docmgr changelog and related files after each meaningful milestone.
- [ ] Run `docmgr doctor --ticket XGOJA-015 --stale-after 30`.
- [ ] Upload final implementation bundle to reMarkable.
- [ ] Commit final ticket documentation updates.
