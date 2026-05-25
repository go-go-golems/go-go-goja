---
Title: go-minitrace xgoja command provider implementation guide
Ticket: XGOJA-014
Status: active
Topics:
  - xgoja
  - command-providers
  - go-minitrace
  - jsverbs
DocType: design-doc
Intent: implementation
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Design for exposing go-minitrace repository-backed query commands through an xgoja CommandSetProvider."
LastUpdated: 2026-05-25T20:05:00-04:00
WhatFor: "Guide the go-minitrace portion of XGOJA-014."
WhenToUse: "When adding or reviewing the go-minitrace xgoja command provider."
---

# go-minitrace xgoja command provider implementation guide

## Goal

Expose go-minitrace's repository-backed query catalog as package-owned Glazed commands in generated xgoja binaries. A buildspec should be able to mount something like `commandProviders: [{ package: go-minitrace, name: queries }]` and get the same SQL/JS catalog commands that the standalone CLI exposes under `go-minitrace query commands`, while still letting xgoja compose the JavaScript runtime profile.

## Current state

- `pkg/minitracejs/provider` already registers the `minitrace` module, but only for host-service runtimes that provide a SQL connection.
- `cmd/go-minitrace/cmds/query` already has reusable Glazed command implementations:
  - `MinitraceCatalogGlazeCommand` executes SQL and JS-backed catalog entries.
  - `NewMinitraceCatalogGlazeCommand(command, catalog)` builds one command.
  - The Cobra command tree currently wraps those commands and creates folder groups manually.
- The catalog lives in `pkg/minitracecmd` and already supports embedded queries plus configured repositories.

## Provider shape

Add a command provider to package ID `go-minitrace`:

```go
providerapi.CommandSetProvider{
  Name:         "queries",
  DefaultMount: "minitrace",
  Description:  "Run go-minitrace repository-backed query commands",
  New:          newQueriesCommandSet,
}
```

Provider config should be JSON/YAML-friendly:

```yaml
commandProviders:
  - package: go-minitrace
    name: queries
    mount: traces
    config:
      appName: go-minitrace
      queryRepositories:
        - ./query-commands/team
```

Implementation notes:

- Decode provider config into a typed struct.
- Load the catalog with `minitracecmd.LoadConfiguredCatalog(appName, queryRepositories)`.
- Convert every catalog entry to `cmds.Command` using the existing Glazed command constructor.
- Preserve nested catalog folders by setting `command.Description().Parents = strings.Split(command.Folder, "/")` before returning it. xgoja will prepend the mount prefix immutably.
- Use `ParserConfig.ShortHelpSections` to include the default command section and `query-runtime`, so catalog command help remains readable.

## Runtime behavior

The existing catalog commands open DuckDB, load archives, and expose `require("minitrace")` to JS commands using their own SQL connection. That means this first command provider can be useful even without a live xgoja host service. Later, if desired, JS command execution can be changed to call `ctx.RuntimeFactory.NewRuntime(ctx, ctx.RuntimeProfile, require.WithLoader(registry.RequireLoader()))`; for the initial provider, reuse the proven catalog command path to minimize risk.

## Tests

Add tests in `pkg/minitracejs/provider`:

1. Registry resolution finds command provider `go-minitrace.queries`.
2. Command provider builds commands from the embedded catalog with no external repository.
3. Returned commands keep nested `Parents` so mounting creates `mount/folder/command` paths.

## Validation

Run:

```bash
go test ./pkg/minitracejs/provider ./cmd/go-minitrace/cmds/query ./pkg/minitracecmd -count=1
```

If dependency updates are needed, use published `github.com/go-go-golems/go-go-goja v0.5.0` instead of workspace-local replaces.
