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

## Field naming

Top-level JavaScript function parameters and verb fields are exposed as idiomatic kebab-case CLI flags while the JavaScript invocation still receives the original parameter names.

For example:

```javascript
function indexQuery(profilePath, docsJson, foo_bar) {
  return { profilePath, docsJson, foo_bar };
}

__verb__("indexQuery", {
  fields: {
    profilePath: { help: "Profile registry path" },
    docsJson: { help: "JSON document list" },
    foo_bar: { help: "Example snake_case field" }
  }
});
```

The generated command exposes `--profile-path`, `--docs-json`, and `--foo-bar`, but `indexQuery` is still called with `profilePath`, `docsJson`, and `foo_bar` argument values.

Section fields currently preserve their declared names because bound sections are passed to JavaScript as objects. This avoids changing object keys such as `filters.localOnly` into `filters["local-only"]`.
