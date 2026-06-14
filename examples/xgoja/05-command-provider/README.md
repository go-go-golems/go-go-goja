# command-provider xgoja example

This example demonstrates a provider-shipped Glazed command set.

The fixture provider registers `CommandSetProvider{Name: "tools"}`. The buildspec mounts that provider under `fixture`, so the generated binary exposes:

- `fixture bare`, a `cmds.BareCommand`
- `fixture write`, a `cmds.WriterCommand`
- `fixture rows`, a `cmds.GlazeCommand`
- `fixture asset`, a `cmds.WriterCommand` that reads `assets/message.txt` through `CommandSetContext.Host.AssetResolver()`

The same provider also exposes the `fixture` module section, so the commands accept `--fixture-value` in addition to their command-owned `--message` flag where applicable.

The `fixture asset` command is the important provider-authoring example: provider command sets are not JavaScript modules, but they still receive the generated host's `HostServices` through `providerapi.CommandSetContext.Host`. That lets provider-owned CLI commands resolve embedded assets or discover host-provided services without parsing the embedded runtime plan directly.

Run:

```bash
make -C examples/xgoja/05-command-provider smoke
```
