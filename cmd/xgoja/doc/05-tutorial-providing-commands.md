---
Title: "Tutorial: providing xgoja commands"
Slug: tutorial-providing-commands
Short: "How provider packages add Glazed command sets and create xgoja runtimes."
Topics:
- xgoja
- providers
- commands
- glazed
Commands:
- xgoja
- xgoja build
Flags:
- --xgoja-replace
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

Provider modules expose JavaScript APIs. Command set providers expose domain-specific CLI commands. Use them when a package needs commands such as `bots run`, `devices list`, or `tools import` instead of only generic `eval`, `run`, `repl`, and `jsverbs`.

## Register a command set provider

A command set provider returns Glazed commands. xgoja mounts them into the generated root command according to `commandProviders[].mount`.

```go
func Register(registry *providerapi.ProviderRegistry) error {
    return registry.Package("my-provider",
        providerapi.Module{...},
        providerapi.CommandSetProvider{
            Name:         "tools",
            DefaultMount: "my-provider",
            NewCommandSet: func(c providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
                cmd, err := cmds.NewBareCommand(
                    cmds.NewCommandDescription("hello", cmds.WithShort("Say hello")),
                    func(ctx context.Context, vals *values.Values) error {
                        return nil
                    },
                )
                if err != nil {
                    return nil, err
                }
                return &providerapi.CommandSet{Commands: []cmds.Command{cmd}}, nil
            },
        },
    )
}
```

## Mount it from xgoja.yaml

```yaml
commandProviders:
  - id: my-tools
    package: my-provider
    name: tools
    mount: tools
    runtimeProfile: main
```

The generated binary will expose commands under the mount path, for example `xapp tools hello`.

## Use selected module sections

When `runtimeProfile` is set, `CommandSetContext.SelectedModules` contains the selected module descriptors for that runtime. If the command should expose the same provider flags as built-in commands, use `providerutil.CollectConfigSections` and attach the sections to the returned Glazed command descriptions.

```go
sections, err := providerutil.CollectGlazedConfigSections(
    c.SelectedModules,
    providerapi.SectionRequest{CommandProviderID: c.Name, RuntimeProfile: c.RuntimeProfile},
    map[string]string{schema.DefaultSlug: "command schema"},
)
if err != nil {
    return nil, err
}
```

## Create runtimes from provider-owned commands

Provider-owned commands can create xgoja runtimes through the typed runtime factory:

```go
runtime, err := c.RuntimeFactory.NewRuntime(ctx, c.RuntimeProfile)
if err != nil {
    return err
}
defer runtime.Close(ctx)
```

If the command parsed provider sections, run selected runtime initializers before executing JavaScript:

```go
handle := yourRuntimeHandleAdapter(runtime)
if err := providerutil.InitRuntimeFromSections(ctx, vals, handle, c.SelectedModules); err != nil {
    return err
}
```

The Discord bot adapter is the concrete pattern: xgoja provides the runtime factory and selected module descriptors, while `discord-bot` owns its `bots` commands and Discord session lifecycle.

## Example

See `examples/xgoja/05-command-provider/` for a generated binary that mounts a provider-owned command set and still reuses module-provided configuration sections.
