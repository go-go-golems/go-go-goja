---
Title: Path Module
Slug: path-module
Short: Cross-platform filepath manipulation helpers
Topics:
- path
- modules
- goja
- javascript
Commands:
- goja-repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `path` module wraps Go's `path/filepath` package for host-platform path manipulation. It is aliased as both `path` and `node:path`. The separator and behavior match the operating system on which the runtime runs.

## JavaScript usage

```javascript
const path = require("path");

const full = path.join("/tmp", "demo", "file.txt");
// "/tmp/demo/file.txt"

const dir = path.dirname("/tmp/demo/file.txt");
// "/tmp/demo"

const base = path.basename("/tmp/demo/file.txt");
// "file.txt"

const ext = path.extname("/tmp/demo/file.txt");
// ".txt"

const abs = path.isAbsolute("/tmp/demo");
// true

const rel = path.relative("/tmp/demo", "/tmp/demo/output");
// "output"

const resolved = path.resolve("demo", ".."); // resolves against cwd
```

## Module API

### `join(...parts)`

Joins path elements into a single path using the OS-specific separator. Cleans up `..` and `.` segments.

### `resolve(...parts)`

Joins path elements and converts the result to an absolute path relative to the current working directory.

### `dirname(path)`

Returns the directory portion of `path`, excluding the final separator and file name.

### `basename(path)`

Returns the last element of `path`.

### `extname(path)`

Returns the file extension, including the leading dot. Returns an empty string when there is no extension.

### `isAbsolute(path)`

Returns `true` if `path` is absolute on the current platform.

### `relative(from, to)`

Returns the shortest relative path from `from` to `to`.

### `path.separator`

`string` — the OS path separator (`/` on Unix, `\` on Windows).

### `path.delimiter`

`string` — the OS path list delimiter (`:` on Unix, `;` on Windows).

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| Paths contain backslashes on Linux | The script assumes Windows separators | Use `path.join` instead of manual string concatenation |
| `resolve()` returns unexpected root | No arguments provided | Call with explicit starting paths, or accept that it resolves against the process cwd |
