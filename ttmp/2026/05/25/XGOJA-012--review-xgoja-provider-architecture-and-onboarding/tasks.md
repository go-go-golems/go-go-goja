# Tasks

## Architecture review

- [x] Create XGOJA-012 ticket workspace.
- [x] Inventory xgoja source, provider API, examples, and Discord adapter touchpoints.
- [x] Review core abstractions introduced in XGOJA-008 through XGOJA-011.
- [x] Map runtime flow for built-in commands and provider-owned commands.
- [x] Identify solid foundations and confusing/over-architected/messy areas.
- [x] Include concrete file/API references and cleanup sketches.

## Report and delivery

- [x] Write intern-oriented review report in ticket docs.
- [x] Include prose explanations, bullets, pseudocode, diagrams, and API references.
- [x] Upload report bundle to reMarkable.
- [x] Run `docmgr doctor --ticket XGOJA-012 --stale-after 30`.
- [x] Commit ticket artifacts.

## Follow-up implementation plan for tomorrow

### Phase 0: Re-read and confirm target shape

- [x] Re-read `design/01-xgoja-provider-architecture-review-and-onboarding-guide.md` sections 8, 12, and 13.
- [x] Confirm the intended naming changes before editing APIs:
  - [x] Keep the general capability concept understood, but avoid overusing it in docs.
  - [x] Rename current package-scoped `ModuleCapability` concept to `PackageCapability` (Option A from section 12.3).
  - [x] Remove `ComponentInitializerCapability` and `InitializedModule` unless a real provider needs them.
- [x] Write a short migration note in the XGOJA-012 diary before making code changes.

### Phase 1: Explain and type `RuntimeFactory`

- [x] Inspect current `RuntimeFactory` creation path:
  - [x] `pkg/xgoja/app/factory.go` — concrete `RuntimeFactory` type and `NewRuntime` method.
  - [x] `pkg/xgoja/app/host.go` / root setup — where `Host` stores and passes the factory.
  - [x] `pkg/xgoja/app/command_providers.go` — where `CommandSetContext.RuntimeFactory` is populated.
  - [x] `discord-bot/pkg/xgoja/provider/provider.go` — current type assertion from `any`.
- [x] Add a typed provider-facing runtime factory interface to `providerapi`, for example:
  - [x] `type RuntimeFactory interface { NewRuntime(ctx context.Context, profile string, opts ...require.Option) (*engine.Runtime, error) }` if accepting `engine` dependency is OK.
  - [ ] Or a smaller `RuntimeFactory` / `Runtime` facade if avoiding concrete `engine.Runtime` in providerapi is preferred.
- [x] Change `CommandSetContext.RuntimeFactory any` to the typed interface.
- [x] Update `pkg/xgoja/app/command_providers.go` to pass the concrete factory through the typed field.
- [x] Update `discord-bot/pkg/xgoja/provider/provider.go` to remove the local type assertion and use the typed providerapi interface directly.
- [ ] Add or update tests proving command providers receive a non-nil typed runtime factory.
- [ ] Document concrete RuntimeFactory examples in the report/docs:
  - [ ] Built-in xgoja app runtime factory for `eval`/`run`/`repl`.
  - [ ] Discord adapter `xgojaBotRuntimeFactory` wrapping xgoja factory for `botcli`.
  - [ ] Generated command-provider example/testprovider path.

### Phase 2: Move shared section/init helpers to providerapi-adjacent utility

- [x] Create a reusable package, likely `pkg/xgoja/providerutil` rather than `providerapi` if imports would make `providerapi` too heavy.
- [x] Move/copy generic helpers from app into the utility:
  - [x] collect config sections from `[]providerapi.ModuleDescriptor`.
  - [x] duplicate slug detection.
  - [x] runtime initializer invocation.
  - [x] standard error wrapping with package/module/capability IDs.
- [x] Replace app usage in `pkg/xgoja/app/module_sections.go` with providerutil helpers.
- [x] Replace Discord adapter usage in `discord-bot/pkg/xgoja/provider/provider.go` with providerutil helpers.
- [x] Add unit tests for:
  - [x] duplicate section slug rejection.
  - [x] nil section rejection.
  - [x] empty slug rejection.
  - [x] runtime initializer error wrapping.
  - [x] no-op behavior when no matching capabilities exist.

### Phase 3: Rename package-scoped capabilities to PackageCapability

- [x] Rename `ModuleCapability` to `PackageCapability` in `pkg/xgoja/providerapi/capabilities.go`.
- [x] Rename `ModuleDescriptor.Capabilities []ModuleCapability` to `[]PackageCapability` or `PackageCapabilities`.
- [x] Rename helper internals and registry APIs as needed:
  - [x] `WithCapability` can remain as a compatibility helper or become `WithPackageCapability`.
  - [x] `ResolveCapabilities` can remain or become `ResolvePackageCapabilities`.
- [x] Update all implementors:
  - [x] `pkg/xgoja/providers/http`.
  - [x] `pkg/xgoja/testprovider`.
  - [x] app tests.
  - [x] discord-bot provider tests.
- [x] Run focused tests after each batch.
- [x] Consider leaving type aliases temporarily if churn is too high.

### Phase 4: Remove unused component initializer abstraction

- [x] Remove `ComponentInitializerCapability` from `providerapi/capabilities.go`.
- [x] Remove `InitializedModule` if no non-test code uses it.
- [x] Remove or simplify testprovider fixtures that exist only to exercise component initializers.
- [x] Search with `rg "ComponentInitializer|InitializedModule"` across the workspace.
- [ ] Update XGOJA docs/report so the concept no longer appears as a public abstraction.

### Phase 5: Clarify discovery-vs-execution side effects

- [ ] Find all places where runtime initializers can be called with nil parsed values.
- [ ] Document the exact convention:
  - [ ] `vals == nil` means runtime construction is happening without command parsed values, usually for discovery/help/list or host preloading.
  - [ ] Providers must not start irreversible side effects in this mode.
- [ ] Add tests for the HTTP provider:
  - [ ] nil values keep HTTP disabled.
  - [ ] non-nil values enable HTTP by default.
  - [ ] explicit `--http-enabled=false` suppresses HTTP.
- [ ] Consider replacing the implicit nil convention with an explicit phase later, but do not expand the API unless tests show the need.

### Phase 6: Fix provider documentation signatures and concepts

- [ ] Update `cmd/xgoja/doc/04-providers.md` stale signatures:
  - [ ] `ConfigSections(providerapi.SectionContext)`.
  - [ ] `InitRuntimeFromSections(context.Context, *values.Values, providerapi.RuntimeHandle)`.
- [ ] Update terminology after renaming:
  - [ ] package capability vs module config vs runtime initializer.
- [ ] Add a decision table:
  - [ ] simple module.
  - [ ] static module config.
  - [ ] command-time config section.
  - [ ] runtime initializer.
  - [ ] runtime closer.
  - [ ] command set provider.
- [ ] Add a concrete RuntimeFactory explanation and examples.

### Phase 7: Number and reorganize examples

- [ ] Rename or copy examples into a numbered learning path:
  - [ ] `01-core-provider` — safe modules and simple runtime profile.
  - [ ] `02-host-provider` — guarded host modules.
  - [ ] `03-multiple-runtimes` — separate runtime profiles per command.
  - [ ] `04-module-sections` — config sections and runtime initializers.
  - [ ] `05-command-provider` — provider-owned Glazed command sets.
  - [ ] `06-runtime-filesystem` — runtime disk JS verb discovery, if still relevant here.
  - [ ] `07-embedded-jsverbs` — embedded JS verbs, if still relevant here.
  - [ ] `08-provider-shipped-jsverbs` — provider-shipped JS verb sources.
- [ ] Update `examples/xgoja/README.md` as a numbered curriculum, not just a list.
- [ ] Update Makefile smoke loops to use new names.
- [ ] Decide whether to keep compatibility directories or only rename in one breaking pass.
- [ ] Run all example smoke tests.

### Phase 8: Reorganize xgoja docs

- [ ] Restructure docs into:
  - [ ] `overview` — what xgoja is and when to use it.
  - [ ] `user-guide` — extensive guide and reference for generated binaries, APIs, and file format.
  - [ ] `tutorials/using-xgoja-yaml` — building a generated binary from YAML.
  - [ ] `tutorials/providing-package-and-modules` — writing a provider package and modules.
  - [ ] `tutorials/providing-commands` — writing a command set provider.
- [ ] Preserve existing useful content from:
  - [ ] `cmd/xgoja/doc/01-overview.md`.
  - [ ] `cmd/xgoja/doc/02-buildspec.md`.
  - [ ] `cmd/xgoja/doc/03-tutorial.md`.
  - [ ] `cmd/xgoja/doc/04-providers.md`.
- [ ] Update embedded doc registration if filenames/slugs change.
- [ ] Run doc/help smoke commands if available.

### Phase 9: Validation and closeout

- [ ] Run focused tests after each phase.
- [ ] Run broad xgoja tests:
  - [ ] `go test ./pkg/xgoja/... ./cmd/xgoja/... -count=1`.
- [ ] Run Discord adapter tests after providerapi changes:
  - [ ] `go test ./pkg/xgoja/provider ./internal/jsdiscord ./pkg/botcli -count=1` in `discord-bot`.
- [ ] Run generated example smoke tests.
- [ ] Update XGOJA-012 diary after each phase.
- [ ] Update XGOJA-012 report if the implementation changes the recommendations.
- [ ] Upload final bundle to reMarkable.
- [ ] Run `docmgr doctor --ticket XGOJA-012 --stale-after 30`.
- [ ] Commit at appropriate intervals.
