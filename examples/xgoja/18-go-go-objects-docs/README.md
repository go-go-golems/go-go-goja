# 18 — go-go-objects docs (HelpSource + docs verb)

This example generates a binary that bundles the `go-go-objects` Durable Objects documentation as a provider `HelpSource` and exposes it through the `docs` `CommandSetProvider` verb.

It demonstrates the two documentation surfaces added by the durableobjects provider:

- **Flow A — bundled help:** the `kind: help` provider source loads the embedded Glazed pages into the root help system, so `help <slug>` renders them.
- **Flow B — the docs verb:** the `provider.command-set` command mounts `durableobjects docs list/show/serve`, which read the same embedded pages.

## Buildspec highlights

```yaml
providers:
  - id: go-go-objects-durableobjects
    import: github.com/go-go-golems/go-go-objects/pkg/xgoja/providers/durableobjects
    register: Register
    module:
      replace: ../../../../go-go-objects
sources:
  - id: go-go-objects-help
    kind: help
    from:
      provider:
        provider: go-go-objects-durableobjects
        source: go-go-objects
commands:
  - id: go-go-objects-docs
    type: provider.command-set
    provider: go-go-objects-durableobjects
    name: docs
    mount: durableobjects
```

The `module.replace` points at a local checkout of `go-go-objects` as a sibling of this repository. Adjust the path if your checkout layout differs.

## Run

```bash
make smoke
```

This runs `doctor`, `list-modules`, builds the binary, and exercises both flows:

```bash
./dist/go-go-objects-docs help go-go-objects-overview
./dist/go-go-objects-docs durableobjects docs list --output json
./dist/go-go-objects-docs durableobjects docs show go-go-objects-js-api
```

To serve the docs over HTTP:

```bash
make serve
curl -sf http://127.0.0.1:8788/docs
curl -sf http://127.0.0.1:8788/docs/go-go-objects-js-api
```

## Notes

- The `docs` verb builds its own help system from the same embedded filesystem as the `HelpSource`, so it works even without the `kind: help` source selected. Selecting both gives `help <slug>` and the verb from one binary.
- The JavaScript `require("docs")` docaccess module is available in `goja-repl` but is not yet wired into generated xgoja runtimes; see the durableobjects `04-docs-verb.md` help page.
