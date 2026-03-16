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

## What the runner scans

The runner walks a directory recursively and inspects `.js` and `.cjs` files. It extracts:

- top-level function declarations,
- top-level arrow functions assigned to variables,
- `__package__`, `__section__`, and `__verb__` metadata,
- and `doc\`...\`` prose blocks for long help.

By default, public top-level functions become commands even without explicit `__verb__` metadata. Files under subdirectories become nested command groups.

## Command shape

Each discovered function is compiled into an ordinary Glazed command. Scalar parameters become Glazed fields, shared sections become flag groups, and the JavaScript result is converted back into rows:

- object result: one row,
- array of objects: one row per item,
- primitive result: one row with a `value` column,
- `Promise`: awaited before conversion.

Some verbs can opt out of structured output entirely. When a verb declares `output: "text"`, the runner exposes it as a writer-style command and prints the returned string directly instead of building a table.

Positional arguments are treated as required unless a default value is declared. That means `jsverbs-example basics echo` now fails with usage instead of silently producing no rows.

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
| A function is not listed | It is not top-level, starts with `_`, or is only defined as an object method | Export it as a top-level function or add explicit metadata around a discoverable function |

## See Also

- `glaze help jsverbs-example-fixture-format`
- `glaze help jsverbs-example-developer-guide`
- `glaze help jsverbs-example-reference`
