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

- [x] Add `providerapi.HelpSource` with `Name`, `Description`, `FS`, and `Root` fields.
- [x] Add `HelpSources` storage to `providerapi.Package` snapshots and registry clones.
- [x] Add `Registry.ResolveHelpSource(packageID, sourceName)`.
- [x] Validate duplicate help source names and missing help source filesystems.
- [x] Extend providerapi tests for successful help source resolution and invalid entries.
- [x] Run `go test ./pkg/xgoja/providerapi -count=1`.
- [x] Commit provider API support.

### Phase 2: Buildspec/runtime spec support

- [x] Add `help.sources` YAML schema to `cmd/xgoja/internal/buildspec`.
- [x] Add matching JSON runtime spec types to `pkg/xgoja/app`.
- [x] Add `validateHelp(...)` for duplicate IDs, missing IDs, missing paths, mixed provider/path sources, unknown package IDs, and embedded path existence.
- [x] Add buildspec tests for valid and invalid help source declarations.
- [x] Run `go test ./cmd/xgoja/internal/buildspec -count=1`.
- [x] Commit buildspec/runtime spec support.

### Phase 3: Generator embedding support

- [x] Add generated runtime JSON rendering for `help.sources`.
- [x] Add embedded local help root rewriting under `xgoja_embed/help/<source-id>`.
- [x] Add `copyEmbeddedHelpSources(...)` to copy local help docs into generated workspaces.
- [x] Update generated `main.go` template to embed JS verbs, help docs, or both without unused imports/variables.
- [x] Pass `EmbeddedHelp` to `app.NewRootCommand` and `app.NewHostWithOptions` in all target modes.
- [x] Add generator tests for template output, collision-free root rewriting, copied help files, and generated binary help smoke coverage.
- [x] Run `go test ./cmd/xgoja/internal/generate -count=1`.
- [x] Commit generator embedding support.

### Phase 4: Generated root loading support

- [x] Add `EmbeddedHelp` to `app.Options`, `HostOptions`, and `Host`.
- [x] Pass provider registry and embedded help FS into `installRootFramework`.
- [x] Load built-in generated docs, provider help sources, embedded local help sources, and optional runtime filesystem sources into one Glazed `HelpSystem`.
- [x] Keep `help_cmd.SetupCobraRootCommand(...)` called exactly once.
- [x] Add app tests for provider help source lookup, embedded local help lookup, and missing provider source errors.
- [x] Run `go test ./pkg/xgoja/app -count=1`.
- [x] Commit generated root loading support.

### Phase 5: Loupedeck provider documentation support

- [x] Export `FS() fs.FS` from `loupedeck/docs/help` without breaking `AddDocToHelpSystem`.
- [x] Register a `providerapi.HelpSource{Name: "runtime-api"}` in `loupedeck/runtime/js/provider.Register`.
- [x] Add Loupedeck provider tests that resolve `loupedeck.runtime-api` and verify its filesystem contains the API reference.
- [x] Run `go test ./runtime/js/provider ./pkg/xgoja/provider -count=1` in `loupedeck`.
- [x] Commit Loupedeck provider docs support.

### Phase 6: End-to-end docs, diary, and validation

- [x] Update xgoja help docs with the new `help.sources` buildspec reference and provider docs guidance.
- [x] Add or update examples showing provider-shipped and embedded local help docs.
- [x] Run focused go-go-goja tests: `go test ./pkg/xgoja/providerapi ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate ./pkg/xgoja/app ./cmd/xgoja/... -count=1`.
- [x] Run focused loupedeck tests after provider docs wiring.
- [x] Update diary after each phase with exact commands, failures, tricky parts, and review instructions.
- [x] Update docmgr changelog and related files after each meaningful milestone.
- [x] Run `docmgr doctor --ticket XGOJA-015 --stale-after 30`.
- [x] Upload final implementation bundle to reMarkable.
- [x] Commit final ticket documentation updates.
