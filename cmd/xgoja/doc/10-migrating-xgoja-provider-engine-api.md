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

Create a module provider entry:

```go
providerapi.Module{
    Name: "my-module",
    NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
        return loader, nil
    },
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

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `undefined: engine.NewBuilder` | The builder constructor was renamed. | Use `engine.NewRuntimeFactoryBuilder`. |
| `undefined: providerapi.NewRegistry` | The provider constructor was renamed. | Use `providerapi.NewProviderRegistry`. |
| `provider.New undefined` | `CommandSetProvider.New` was renamed. | Use `provider.NewCommandSet`. |
| `handle.Runtime undefined` | Runtime initializer handles now expose the engine runtime explicitly. | Use `handle.EngineRuntime()` and then `.VM` for the raw Goja VM. |
| `RuntimeCloserRegistry` is missing | Cleanup registration moved onto `engine.Runtime`. | Use `handle.EngineRuntime().AddCloser(...)`. |
| `NativeModuleSpec` is missing | Registration-performing types no longer use `*Spec`. | Use `engine.NativeModuleRegistrar`. |

## See also

- `xgoja help migrating-runtime-context-api`
- `xgoja help tutorial-providing-package-and-modules`
- `xgoja help tutorial-providing-commands`
- `xgoja help buildspec-reference`
