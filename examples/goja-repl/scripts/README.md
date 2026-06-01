# goja-repl runnable scripts

Run these from the repository root with `cmd/goja-repl run`.

```bash
GOWORK=off go run ./cmd/goja-repl --enable-module yaml run ./examples/goja-repl/scripts/yaml.js
GOWORK=off go run ./cmd/goja-repl --enable-module fs --enable-module exec run ./examples/goja-repl/scripts/hello.js
GOWORK=off go run ./cmd/goja-repl --enable-module database run ./examples/goja-repl/scripts/database.js
```

## Files

- `yaml.js` — data-only YAML module example.
- `hello.js` — trusted host filesystem plus process execution example.
- `database.js` — in-memory SQLite example.

The examples are intentionally small and double as manual smoke tests. If one fails, either the script has drifted from the module API or the command is missing the required `--enable-module` flags.
