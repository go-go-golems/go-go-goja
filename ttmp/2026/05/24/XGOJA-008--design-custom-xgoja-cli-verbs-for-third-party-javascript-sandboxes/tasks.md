# Tasks

## Completed design/research

- [x] Add tasks here
- [x] Investigate current xgoja generated command architecture and jsverbs support
- [x] Inventory third-party sandbox command patterns in loupedeck discord-bot css-visual-diff and go-minitrace
- [x] Design extension patterns for custom generated CLI verbs and host services
- [x] Write intern-oriented implementation guide with diagrams APIs pseudocode and file references
- [x] Validate ticket docs and upload bundle to reMarkable
- [x] Revise command-provider design to return Glazed commands instead of Cobra commands
- [x] Revise design for module-provided Glazed config sections and DecodeSectionInto initialization
- [x] Revise design so built-in run repl jsverbs aggregate module-provided Glazed sections

## Implementation phase 0 — planning and checkpoints

- [x] P0.1 Expand this task list into granular implementation phases
- [x] P0.2 Add diary entry for implementation kickoff and scope boundaries
- [x] P0.3 Commit planning-only ticket updates

## Implementation phase 1 — provider API capabilities

- [x] P1.1 Add `providerapi.SectionContext`, `ModuleDescriptor`, and `ModuleCapability`
- [x] P1.2 Add `ConfigSectionCapability` for module-provided Glazed sections
- [x] P1.3 Add `RuntimeHandle` and `RuntimeInitializerCapability` for built-in runtime initialization
- [x] P1.4 Add `InitializedModule` and `ComponentInitializerCapability` for command-provider domain objects
- [x] P1.5 Extend `providerapi.Package` to store package-level capabilities
- [x] P1.6 Add validation for duplicate or empty capability IDs
- [ ] P1.7 Add registry helpers to resolve runtime module descriptors from app runtime specs
- [x] P1.8 Add providerapi unit tests for capability registration, cloning, and validation errors
- [x] P1.9 Run focused providerapi tests
- [x] P1.10 Commit provider API capability slice

## Implementation phase 2 — built-in runtime-profile section aggregation helpers

- [ ] P2.1 Add app helper to collect module descriptors for a runtime profile
- [ ] P2.2 Add app helper to collect `ConfigSectionCapability` sections for a runtime profile
- [ ] P2.3 Add app helper to call `RuntimeInitializerCapability.InitRuntimeFromSections`
- [ ] P2.4 Add section slug/prefix collision checks with useful error messages
- [ ] P2.5 Add tests for aggregation order and collision behavior
- [ ] P2.6 Run focused app helper tests
- [ ] P2.7 Commit built-in aggregation helper slice

## Implementation phase 3 — apply module sections to built-in `run`

- [ ] P3.1 Extend `runCommand` to store selected module descriptors
- [ ] P3.2 Append runtime-profile module sections to `run` command description
- [ ] P3.3 Call runtime initializers before executing the script
- [ ] P3.4 Add fixture provider/module for section exposure and runtime initialization
- [ ] P3.5 Add app tests proving `run --help` exposes fixture flags
- [ ] P3.6 Add app tests proving `run` decodes fixture settings via `DecodeSectionInto`
- [ ] P3.7 Run focused `run` tests
- [ ] P3.8 Commit `run` built-in module-section slice

## Implementation phase 4 — apply module sections to `repl` / TUI

- [ ] P4.1 Extend `tuiCommand` to store selected module descriptors
- [ ] P4.2 Append runtime-profile module sections to `repl` command description
- [ ] P4.3 Thread parsed values into TUI runtime creation
- [ ] P4.4 Call runtime initializers before starting the REPL/TUI session
- [ ] P4.5 Add tests for `repl --help` exposing fixture flags without launching TUI
- [ ] P4.6 Run focused TUI command tests
- [ ] P4.7 Commit `repl` module-section slice

## Implementation phase 5 — apply module sections to `jsverbs`

- [ ] P5.1 Extend jsverb command construction to append runtime-profile module sections
- [ ] P5.2 Ensure verb-declared sections and module sections are collision checked
- [ ] P5.3 Call runtime initializers before `registry.InvokeInRuntime`
- [ ] P5.4 Add fixture jsverb integration test exposing module flags
- [ ] P5.5 Add fixture jsverb integration test proving initializer runs
- [ ] P5.6 Run focused jsverbs tests
- [ ] P5.7 Commit `jsverbs` module-section slice

## Implementation phase 6 — command set providers

- [ ] P6.1 Add `providerapi.CommandSetProvider`, `CommandSet`, and `CommandSetContext`
- [ ] P6.2 Extend registry package storage with command set providers
- [ ] P6.3 Add duplicate/missing factory validation and unit tests
- [ ] P6.4 Add app spec/buildspec `commandProviders` support
- [ ] P6.5 Add generated main wiring for command providers
- [ ] P6.6 Add `Host.AttachCommandProviders`
- [ ] P6.7 Implement mount-parent prefix application or document command-owned parents
- [ ] P6.8 Add fixture command provider returning a `BareCommand`
- [ ] P6.9 Add fixture command provider returning `WriterCommand`/`GlazeCommand` examples if practical
- [ ] P6.10 Add generated app tests for command provider attachment
- [ ] P6.11 Run focused command provider tests
- [ ] P6.12 Commit command provider slice

## Implementation phase 7 — generated examples and docs

- [ ] P7.1 Add generated example for built-in `run` module sections
- [ ] P7.2 Add generated example for `jsverbs` module sections
- [ ] P7.3 Add generated example for custom command provider module-section composition
- [ ] P7.4 Update `cmd/xgoja/doc/02-buildspec.md`
- [ ] P7.5 Update `cmd/xgoja/doc/04-providers.md`
- [ ] P7.6 Smoke generated examples
- [ ] P7.7 Commit docs/examples slice

## Implementation phase 8 — final validation and delivery

- [ ] P8.1 Run focused xgoja package tests
- [ ] P8.2 Run broader go-go-goja tests as time permits
- [ ] P8.3 Update design doc with implementation deltas if needed
- [ ] P8.4 Update diary with final validation results
- [ ] P8.5 Run `docmgr doctor --ticket XGOJA-008 --stale-after 30`
- [ ] P8.6 Upload final implementation bundle to reMarkable
- [ ] P8.7 Commit final ticket updates
- [ ] P8.8 Close XGOJA-008 if all requested implementation work is complete
