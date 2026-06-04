# Glossary

## `*Spec` pattern

A `*Spec` type is a declarative data structure: it describes what should exist, but it should not itself perform runtime work.

Use `*Spec` for parsed or embedded configuration shapes such as build files, runtime profiles, selected modules, generated command settings, assets, and code-generation inputs. A `*Spec` may have small normalization helpers, but it should not own lifecycle, scheduling, I/O, registration side effects, or mutable runtime state.

Examples:

- `buildspec.Spec`: parsed `xgoja.yaml` build model.
- `buildspec.RuntimeSpec`: declarative runtime profile in `xgoja.yaml`.
- `buildspec.ModuleInstanceSpec`: declarative selected provider module inside a runtime profile.
- `buildspec.CommandProviderInstanceSpec`: declarative selected provider command set.

Contrast with non-spec patterns:

- `engine.Runtime`: concrete VM/event-loop/require runtime with lifecycle.
- `app.RuntimeFactory`: factory that creates concrete runtimes from runtime profiles.
- `providerapi.Module`: provider module definition and setup factory.
- `require.ModuleLoader`: CommonJS loader that populates `module.exports`.

Rule of thumb: if the type says what to build, select, embed, or configure, `*Spec` is appropriate. If it creates, registers, runs, schedules, resolves, or closes something, it should not be named `*Spec`.
