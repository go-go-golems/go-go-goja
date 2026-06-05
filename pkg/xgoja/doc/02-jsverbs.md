---
Title: "generated xgoja jsverbs"
Slug: jsverbs
Short: "How generated xgoja binaries mount JavaScript verbs as Glazed commands."
Topics:
- xgoja
- jsverbs
- glazed
Commands:
- verbs
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

Generated xgoja binaries can expose JavaScript functions as CLI commands. The build specification declares one or more jsverb sources. At startup, the generated runtime support scans those sources, builds Glazed commands from discovered verb metadata, and attaches them under the configured verbs root command.

A jsverb command executes inside the generated xgoja runtime module set. The command builds field values from CLI flags and arguments, creates a runtime with the top-level `modules` list, injects the source loader for the scanned JavaScript files, invokes the captured JavaScript function, and prints the Glazed command result.

## Source modes

- Runtime filesystem sources read JavaScript files from disk when the generated binary runs.
- Embedded sources are copied into the generated Go module at build time and embedded with `go:embed`.
- Provider-shipped sources come from a provider package's embedded filesystem.

Use the `sources` subcommand under the verbs root to list configured source IDs.
