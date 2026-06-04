# Tasks

## TODO

### Phase 4: Tests and docs

- [ ] Add providerutil unit tests for static config parsing, override merging, and raw JSON conversion helpers.
- [ ] Add docs/help updates for provider authors covering `GlazedConfigSectionCapability`, `XGojaConfigSectionCapability`, and `NewRuntimeFromSections`.

## DONE

- [x] Review existing GOJA-053 design docs and write independent implementation guide
- [x] Research Glazed SectionValues as the xgoja module config merge layer
- [x] Document xgoja codegen and generated script execution runthrough
- [x] Create codegen/runtime runthrough research logbook and upload to reMarkable
- [x] Inventory generic Service/Context/Capability/Runtime/Module/Spec symbols and upload glossary
- [x] Analyze Sobek ECMAScript Modules versus xgoja require/native module machinery
- [x] Create Sobek ESM research logbook and upload to reMarkable
- [x] Clarify build/runtime spec names and app/runtime DTO names
- [x] Move reusable runtime engine code to `pkg/engine`
- [x] Expose `engine.Runtime` through runtime initializer handles
- [x] Clarify engine runtime factory, registrar, and context names
- [x] Add migration help for the provider/engine API cleanup
- [x] Rename the public command/config/env flag capability from `ConfigSectionCapability` to `GlazedConfigSectionCapability`
- [x] Update `providerutil.CollectGlazedConfigSections` and current providers/tests/docs to use `GlazedConfigSectionCapability`
- [x] Add `XGojaConfigSectionCapability` for internal xgoja module configuration sections that must not be exposed as CLI flags by default
- [x] Add `XGojaConfigRequest` with descriptor/profile context and already-parsed static config values
- [x] Add provider API for mapping parsed Glazed command values into internal xgoja module config `SectionValues` overrides
- [x] Add providerutil helper to parse static `ModuleInstanceSpec.Config` maps into internal `values.SectionValues` using `XGojaConfigSectionCapability`
- [x] Add providerutil helper to merge static internal config values with provider-produced runtime overrides
- [x] Add providerutil helper to convert final internal `SectionValues` back into `json.RawMessage` for existing `ModuleSetupContext.Config`
- [x] Avoid global capability dedupe for module config mapping; call config mapping per selected module descriptor/instance
- [x] Add `RuntimeFactory.NewRuntimeFromSections(ctx, profile, vals, opts...)`
- [x] Keep `RuntimeFactory.NewRuntime(ctx, profile, opts...)` as the path for static config only
- [x] Update built-in `eval`, `run`, `tui`, and jsverbs runtime creation paths to call `NewRuntimeFromSections` before runtime initializers execute
- [x] Extend `providerapi.RuntimeFactory` for command providers with the parsed-values runtime creation path
- [x] Update command-provider runtime factory interface so provider-owned commands can opt into the same runtime config patching
- [x] Add app runtime tests proving CLI/config/env-derived values patch module setup config before `NewModuleFactory` runs
- [x] Add regression test for selecting the same provider package twice under different aliases; each module instance receives independent config
- [x] Update Geppetto provider to current xgoja provider API names (`ProviderRegistry`, `NewModuleFactory`, `ModuleSetupContext`).
- [x] Remove Geppetto provider config fields `Profile`, `AllowRegistryLoad`, `AllowNetwork`, `AllowTools`, `EnableStorage`, and nested `Turns`.
- [x] Rename Geppetto provider config `ProfileRegistries` to `DefaultProfileRegistries`.
- [x] Add Geppetto `XGojaConfigSectionCapability` for internal module config.
- [x] Add Geppetto `GlazedConfigSectionCapability` flags for supported runtime overrides (`default-profile-registries`, `default-profile`); turn-store flags were intentionally omitted after removing provider storage config.
- [x] Add Geppetto config mapping from Glazed values into internal xgoja module config before module setup.
- [x] Update Geppetto provider tests for simplified config and renamed registry field.
