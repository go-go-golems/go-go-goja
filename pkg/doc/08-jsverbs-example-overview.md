---
Title: "jsverbs-example overview"
Slug: jsverbs-example-overview
Short: "How the example runner scans JavaScript files, turns them into Glazed verbs, and executes them."
Topics:
- goja
- glazed
- javascript
- commands
Commands:
- jsverbs-example
- jsverbs-example list
Flags:
- --dir
- --log-level
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Application
---

This page explains what `jsverbs-example` does and how to use it to validate JavaScript-backed commands quickly.

The example runner itself still scans from a directory passed through `--dir`, but the underlying `pkg/jsverbs` package is now broader than that one CLI shape. The library can scan a real directory, a generic `fs.FS` such as an `embed.FS`, or raw in-memory source strings. The runner is therefore best thought of as the simplest interactive harness for the package, not as the complete boundary of what `jsverbs` supports.

## What the runner scans

The runner walks a directory recursively and inspects `.js` and `.cjs` files. It extracts:

- top-level function declarations,
- top-level arrow functions assigned to variables,
- `__package__`, `__section__`, and `__verb__` metadata,
- and `doc\`...\`` prose blocks for long help.

Under the hood, the scanner now parses metadata literals directly from the tree-sitter AST instead of rewriting JavaScript object text into JSON. That means metadata support is intentionally strict: literal objects, arrays, strings, numbers, booleans, and `null` are supported, while dynamic expressions inside metadata are rejected. This is a deliberate design choice so command discovery stays deterministic and scanner failures are easier to explain.

By default, public top-level functions become commands even without explicit `__verb__` metadata. Files under subdirectories become nested command groups.

## Command shape

Each discovered function is compiled into an ordinary Glazed command. Scalar parameters become Glazed fields, shared sections become flag groups, and the JavaScript result is converted back into rows:

- object result: one row,
- array of objects: one row per item,
- primitive result: one row with a `value` column,
- `Promise`: awaited before conversion.

Some verbs can opt out of structured output entirely. When a verb declares `output: "text"`, the runner exposes it as a writer-style command and prints the returned string directly instead of building a table.

Positional arguments are treated as required unless a default value is declared. That means `jsverbs-example basics echo` now fails with usage instead of silently producing no rows.

The package also has a stricter error model now. Invalid metadata is recorded as scan diagnostics and, by default, scan functions fail with a `ScanError` instead of silently dropping the broken section or verb. That change matters because it turns missing-command debugging from guesswork into a direct scanner error.

## Typical workflow

List what was discovered first:

```bash
jsverbs-example --dir ./testdata/jsverbs list
```

Run a simple command:

```bash
jsverbs-example --dir ./testdata/jsverbs basics greet Manuel --excited
```

Run a verb that writes plain text instead of structured rows:

```bash
jsverbs-example --dir ./testdata/jsverbs basics banner Manuel
```

Run a command that uses a shared section:

```bash
jsverbs-example --dir ./testdata/jsverbs basics list-issues go-go-golems/go-go-goja --state closed --labels bug --labels docs
```

If you are working inside Go rather than through the example binary, the package-level entrypoints are now:

```go
jsverbs.ScanDir(...)
jsverbs.ScanFS(...)
jsverbs.ScanSource(...)
jsverbs.ScanSources(...)
```

That is worth remembering because the example runner is intentionally simpler than the package API. A future application can embed command files at build time or synthesize them in memory without first writing them to disk.

## Logging

The example runner exposes standard Glazed logging flags on the root command. The default log level is `error` so module-registration debug logs stay out of the way during normal use. Raise the level explicitly when debugging loader behavior:

```bash
jsverbs-example --log-level debug --dir ./testdata/jsverbs list
```

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| A command prints nothing | A required positional argument was omitted earlier and the function returned `undefined` | Re-run with the required positional argument or inspect `--help` |
| Relative `require()` fails | The helper file is outside the scanned directory or uses an unsupported resolution path | Keep helper files under the scanned tree and use relative imports like `./helper` |
| A function is not listed | It is not top-level, starts with `_`, is only defined as an object method, or metadata parsing failed | Export it as a top-level function and check the scanner error output for invalid metadata |
| A `__verb__` block seems to be ignored | The metadata used dynamic JavaScript instead of static literals | Restrict metadata to literal objects, arrays, strings, numbers, booleans, and `null` |

## See Also

- `glaze help jsverbs-example-fixture-format`
- `glaze help jsverbs-example-developer-guide`
- `glaze help jsverbs-example-reference`
