# command-provider xgoja example

This example demonstrates a provider-shipped Glazed command set.

The fixture provider registers `CommandSetProvider{Name: "tools"}`. The buildspec mounts that provider under `fixture`, so the generated binary exposes:

- `fixture bare`, a `cmds.BareCommand`
- `fixture write`, a `cmds.WriterCommand`
- `fixture rows`, a `cmds.GlazeCommand`

The same provider also exposes the `fixture` module section, so all three commands accept `--fixture-value` in addition to their command-owned `--message` flag.

Run:

```bash
make -C examples/xgoja/05-command-provider smoke
```
