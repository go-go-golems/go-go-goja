---
Title: "Migrating xgoja provider and engine APIs"
Slug: migrating-xgoja-provider-engine-api
Short: "Concise replacement guide for the provider and engine API naming cleanup."
Topics:
- xgoja
- migration
- provider-api
- engine
Commands:
- xgoja
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

Use this page when updating code written against the older xgoja provider API or the older engine factory API. The cleanup is intentionally breaking: prefer the new names directly instead of adding compatibility wrappers.

## Direct replacements

| Old | New |
| --- | --- |
| `engine.NewBuilder(...)` | `engine.NewRuntimeFactoryBuilder(...)` |
| `engine.FactoryBuilder` | `engine.RuntimeFactoryBuilder` |
| `engine.Factory` | `engine.RuntimeFactory` |
| `engine.RuntimeModuleSpec` | `engine.RuntimeModuleRegistrar` |
| `engine.RuntimeModuleContext` | `engine.RuntimeModuleRegistrationContext` |
| `engine.RuntimeContext` | `engine.RuntimeInitializationContext` |
| `engine.NativeModuleSpec` | `engine.NativeModuleRegistrar` |
| `providerapi.NewRegistry()` | `providerapi.NewProviderRegistry()` |
| `providerapi.Registry` | `providerapi.ProviderRegistry` |
| `providerapi.SectionContext` | `providerapi.SectionRequest` |
| `providerapi.ModuleContext` | `providerapi.ModuleSetupContext` |
| `providerapi.Module.New` | `providerapi.Module.NewModuleFactory` |
| `providerapi.RuntimeHandle` | `providerapi.RuntimeInitializerHandle` |
| `RuntimeInitializerHandle.Runtime()` | `RuntimeInitializerHandle.EngineRuntime()` |
| `CommandSetProvider.New` | `CommandSetProvider.NewCommandSet` |
| `app.Spec` | `app.RuntimeSpec` |

Removed aliases:

- `providerapi.ModuleFactory`: inline `func(providerapi.ModuleSetupContext) (require.ModuleLoader, error)`.
- `providerapi.CommandSetProviderFactory`: inline `func(providerapi.CommandSetContext) (*providerapi.CommandSet, error)`.
- `providerapi.RuntimeCloserRegistry`: call `handle.EngineRuntime().AddCloser(...)`.

## Common before and after snippets

Build an engine runtime factory:

```go
factory, err := engine.NewRuntimeFactoryBuilder().
    WithModules(engine.NativeModuleRegistrar{ModuleName: "my-module", Loader: loader}).
    Build()
```

Register xgoja providers:

```go
registry := providerapi.NewProviderRegistry()
_ = myprovider.Register(registry)
```

Create a minimal package provider with one native module:

```go
func Register(registry *providerapi.ProviderRegistry) error {
    return registry.Package("my-package",
        providerapi.Module{
            Name:        "my-module",
            DefaultAs:   "my-module",
            Description: "Module exposed through require(\"my-module\")",
            NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
                return loader, nil
            },
        },
    )
}
```

Create a module provider entry when you are already inside a larger package definition:

```go
providerapi.Module{
    Name: "my-module",
    NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
        return loader, nil
    },
}
```

Decode embedded runtime specs in generated or generated-like binaries:

```go
func decodeSpec() *app.RuntimeSpec {
    spec := &app.RuntimeSpec{}
    must(json.Unmarshal([]byte(embeddedSpecJSON), spec))
    return spec
}
```

Initialize a runtime and register cleanup:

```go
func (Capability) InitRuntimeFromSections(ctx context.Context, vals *values.Values, handle providerapi.RuntimeInitializerHandle) error {
    rt := handle.EngineRuntime()
    if rt == nil || rt.VM == nil {
        return fmt.Errorf("runtime is nil")
    }
    return rt.AddCloser(func(ctx context.Context) error {
        return cleanup(ctx)
    })
}
```

Register package-owned commands:

```go
providerapi.CommandSetProvider{
    Name: "verbs",
    NewCommandSet: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
        return &providerapi.CommandSet{Commands: []cmds.Command{command}}, nil
    },
}
```

## Migration checklist for generated binaries

Generated xgoja binaries can contain stale API names even when the provider package itself has been updated. Check the generated command module as a separate module if it has its own `go.mod`.

Search active source and docs for old names:

```bash
rg -n "providerapi\\.(Registry|ModuleContext|NewRegistry)|\\.New\\(providerapi\\.ModuleContext|app\\.Spec" \
  --glob '!ttmp/**' \
  .
```

Validate both workspace and standalone module modes:

```bash
# From the provider package root.
go test ./...
GOWORK=off go test ./...

# If a checked-in generated command module exists.
cd cmd/<generated-binary>
GOWORK=off go test ./...
```

If workspace mode passes but `GOWORK=off` fails, inspect the relevant `go.mod`. A nested generated command module may still pin an older `github.com/go-go-golems/go-go-goja` version.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `undefined: engine.NewBuilder` | The builder constructor was renamed. | Use `engine.NewRuntimeFactoryBuilder`. |
| `undefined: providerapi.NewRegistry` | The provider constructor was renamed. | Use `providerapi.NewProviderRegistry`. |
| `undefined: providerapi.ProviderRegistry` after migrating source | The module still depends on an older `go-go-goja` release. | Upgrade `github.com/go-go-golems/go-go-goja` in the active `go.mod`; check nested generated command modules too. |
| `unknown field New in struct literal of type providerapi.Module` | `providerapi.Module.New` was renamed. | Use `NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) { ... }`. |
| `undefined: providerapi.ModuleContext` | Module setup context was renamed. | Use `providerapi.ModuleSetupContext`. |
| `undefined: app.Spec` | Generated or generated-like xgoja source still references the old runtime spec DTO. | Use `app.RuntimeSpec`, or regenerate the binary with the current xgoja generator. |
| `provider.New undefined` | `CommandSetProvider.New` was renamed. | Use `provider.NewCommandSet`. |
| `handle.Runtime undefined` | Runtime initializer handles now expose the engine runtime explicitly. | Use `handle.EngineRuntime()` and then `.VM` for the raw Goja VM. |
| `RuntimeCloserRegistry` is missing | Cleanup registration moved onto `engine.Runtime`. | Use `handle.EngineRuntime().AddCloser(...)`. |
| `NativeModuleSpec` is missing | Registration-performing types no longer use `*Spec`. | Use `engine.NativeModuleRegistrar`. |

## See also

- `xgoja help migrating-runtime-context-api`
- `xgoja help tutorial-providing-package-and-modules`
- `xgoja help tutorial-providing-commands`
- `xgoja help xgoja-v2-reference`
