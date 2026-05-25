# xgoja host provider example

This example builds a generated xgoja binary using the guarded first-party host provider package at `pkg/xgoja/providers/host`.

The runtime selects host-capability modules with explicit config:

- `fs` with `config.allow: true`
- `exec` with `config.allow: true` and an `allowedCommands` allow-list containing only `echo`
- `database` with `config.allowConfigure: true` so the script can open an in-memory sqlite database

Run:

```bash
make smoke
```

Security notes:

- `fs` is acknowledged but not path-sandboxed.
- `exec` can run host processes; this example restricts exact command names with `allowedCommands`.
- `database.configure()` can open caller-selected driver/data-source pairs only when `allowConfigure` is true.
