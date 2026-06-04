# Glossary

## `*Spec` pattern

A `*Spec` type is a declarative data structure: it describes what should exist, but it should not itself perform runtime work.

Use `*Spec` for parsed or embedded configuration shapes such as build files, runtime profiles, selected modules, generated command settings, assets, and code-generation inputs. A `*Spec` may have small normalization helpers, but it should not own lifecycle, scheduling, I/O, registration side effects, or mutable runtime state.

Examples:

- `buildspec.BuildSpec`: parsed `xgoja.yaml` build model.
- `buildspec.RuntimeSpec`: declarative runtime profile in `xgoja.yaml`.
- `buildspec.ModuleInstanceSpec`: declarative selected provider module inside a runtime profile.
- `buildspec.CommandProviderInstanceSpec`: declarative selected provider command set in `xgoja.yaml`.
- `app.RuntimeSpec`: normalized embedded runtime model decoded by generated xgoja binaries.
- `app.RuntimeProfileSpec`: declarative runtime profile in the embedded runtime model.
- `app.ModuleInstanceSpec`: declarative selected provider module in the embedded runtime model.
- `app.CommandProviderInstanceSpec`: declarative selected provider command set in the embedded runtime model.

Contrast with non-spec patterns:

- `engine.Runtime`: concrete VM/event-loop/require runtime with lifecycle.
- `app.RuntimeFactory`: factory that creates concrete runtimes from runtime profiles.
- `providerapi.Module`: provider module definition with a `NewModuleFactory` setup hook.
- `providerapi.ModuleSetupContext`: setup-time inputs passed while creating a selected module's CommonJS loader.
- `providerapi.SectionRequest`: request metadata passed when collecting provider configuration sections.
- `providerapi.RuntimeInitializerHandle`: runtime handle passed to provider runtime initializer capabilities; exposes the owned `*engine.Runtime`.
- `providerapi.ProviderRegistry`: active registry of provider packages, modules, command sets, help sources, and capabilities.
- `require.ModuleLoader`: CommonJS loader that populates `module.exports`.

Rule of thumb: if the type says what to build, select, embed, or configure, `*Spec` is appropriate. If it creates, registers, runs, schedules, resolves, or closes something, it should not be named `*Spec`.
