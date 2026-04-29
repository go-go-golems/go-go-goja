---
Title: Using the Interactive REPL
Slug: repl-usage
Short: Interactive JavaScript evaluation with native module access
Topics:
- repl
- interactive
- javascript
- console
Commands:
- goja-repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Example
---

The REPL (Read-Eval-Print Loop) provides an interactive JavaScript environment with immediate access to all registered native modules. This enables rapid experimentation, debugging, and prototyping without the overhead of creating separate script files.

## Basic REPL Operations

The REPL accepts any valid JavaScript expression and immediately evaluates it within the full go-go-goja runtime environment.

### Starting the REPL

```bash
# Canonical Bobatea-based JS REPL with completion/help widgets
go run ./cmd/goja-repl tui

# Add an extra plugin directory explicitly
go run ./cmd/goja-repl --plugin-dir /tmp/goja-plugins tui

# Restrict loading to specific plugin module names
go run ./cmd/goja-repl --allow-plugin-module plugin:examples:greeter tui

# With debug logging
go run ./cmd/goja-repl --log-level debug tui
```

When you do not pass `--plugin-dir`, `goja-repl tui` scans the default per-user plugin tree under `~/.go-go-goja/plugins/...`. If you do pass one or more `--plugin-dir` flags, those explicit directories are used instead.

### Basic JavaScript Evaluation

```javascript
js> 2 + 3
5

js> const name = "World"
js> `Hello, ${name}!`
Hello, World!

js> Math.sqrt(16)
4

js> [1, 2, 3].map(x => x * 2)
[2, 4, 6]
```

### Running Script Files

Use `goja-repl run <file>` when you want a one-shot runtime for a JavaScript file instead of a persistent REPL session:

```bash
go run ./cmd/goja-repl run ./testdata/yaml.js
```

`run` creates a fresh runtime, enables the default native modules, derives module roots from the script path, executes the file, and then closes the runtime. It does not require `goja-repl create`, a `session-id`, or a SQLite database.

Root-level plugin flags still apply:

```bash
go run ./cmd/goja-repl --plugin-dir ./plugins run ./scripts/with-plugins.js
```

### Module Security Flags

By default, `run` (and all other `goja-repl` commands) load **all** registered native modules. You can restrict the module sandbox using persistent flags:

```bash
# Safe mode: load only data-only modules (crypto, events, path, time, timer)
go run ./cmd/goja-repl --safe-mode run ./script.js

# Whitelist: load only specific modules
go run ./cmd/goja-repl --enable-module fs,path run ./script.js

# Whitelist in the TUI; db is an alias for the database module
go run ./cmd/goja-repl tui --enable-module db

# Blacklist: load all except specific modules
go run ./cmd/goja-repl --disable-module fs,exec run ./script.js
```

| Flag | Effect | Example |
|------|--------|---------|
| `--safe-mode` | Only data-safe modules (no filesystem, process, or DB access) | `--safe-mode` |
| `--enable-module` | Whitelist — only these modules are loaded | `--enable-module fs,path` |
| `--disable-module` | Blacklist — all modules except these are loaded | `--disable-module exec,os` |

These flags are persistent (available on all subcommands) and are applied when the engine runtime is built. They affect `run`, `tui`, `eval`, `create`, and all other commands that construct a runtime.

**Module categories:**

- **Safe (data-only):** `crypto`, `events`, `path`, `time`, `timer`
- **Dangerous (host-access):** `fs`, `os`, `exec`, `database`/`db`, `yaml`
- **Process exposure:** `process` (requires `WithProcess()`) 

### REPL Commands

The REPL recognizes special commands prefixed with `:`:

- `:help` - Display available commands and usage information
- `:plugins` - Display plugin discovery directories, discovered candidates, and loaded modules
- `:quit` or `:exit` - Exit the REPL
- `:clear` - Clear the current context (if implemented)

## Module Usage Examples

All registered native modules are immediately available via `require()`:

### File System Operations

```javascript
js> const fs = require("fs")
js> fs.writeFileSync("/tmp/demo.txt", "Hello from REPL!")
js> fs.readFileSync("/tmp/demo.txt")
Hello from REPL!

js> fs.existsSync("/tmp/demo.txt")
true
```

### Plugin-backed Modules

Plugin-backed modules load through the same `require()` API as in-process native modules:

```javascript
js> const echo = require("plugin:echo")
js> echo.ping("hello")
hello

js> echo.math.add(2, 3)
5

js> const kv = require("plugin:examples:kv")
js> kv.store.set("name", "Manuel")
{key: "name", value: "Manuel", size: 1}

js> kv.store.get("name")
Manuel
```

Use `goja-repl help goja-plugin-user-guide` for the full plugin reference and example catalog, and `goja-repl help plugin-tutorial-build-install` for the step-by-step build/install flow.

### Unified Documentation Access

The runtime now also exposes a built-in `docs` module that lets JavaScript inspect embedded help pages, jsdoc stores when they are attached, and loaded plugin metadata:

```javascript
js> const docs = require("docs")

js> docs.sources()
[
  { id: "default-help", kind: "glazed-help", ... },
  { id: "plugin-manifests", kind: "plugin", ... }
]

js> docs.bySlug("default-help", "repl-usage").title
REPL Usage

js> docs.byID("plugin-manifests", "plugin-method", "plugin:examples:kv/store.get").body
Get a key, returning null if it is absent
```

Use `goja-repl help goja-docs-module-guide` for the full API reference and examples.

### HTTP Requests

```javascript
js> const http = require("http")
js> const response = http.get("https://httpbin.org/json")
js> response.status
200

js> JSON.parse(response.body).slideshow.title
Sample Slide Show
```

### Async Operations with Promises

```javascript
js> const timer = require("timer")
js> timer.sleep(1000).then(() => console.log("Done!"))
Promise { <pending> }
js> // "Done!" appears after 1 second
Done!

js> async function demo() { await timer.sleep(500); return "Finished!"; }
js> demo().then(console.log)
Promise { <pending> }
js> // "Finished!" appears after 500ms
Finished!
```

### YAML Parsing and Serialization

The `yaml` module provides native YAML support for configuration files, API payloads, and data interchange:

```javascript
js> const yaml = require("yaml")

js> const config = yaml.parse("name: goja\nversion: 1.0")
js> config.name
goja

js> const out = yaml.stringify({ host: "localhost", port: 8080 })
js> console.log(out)
host: localhost
port: 8080

js> yaml.validate("[bad").valid
false

js> yaml.validate("hello: world").valid
true
```

Use `goja-repl help yaml-module` for the full API reference.

## Multi-line Input

The REPL supports multi-line JavaScript constructs:

```javascript
js> function fibonacci(n) {
...   if (n <= 1) return n;
...   return fibonacci(n-1) + fibonacci(n-2);
... }

js> fibonacci(10)
55

js> const users = [
...   { name: "Alice", age: 30 },
...   { name: "Bob", age: 25 }
... ]

js> users.filter(u => u.age > 27).map(u => u.name)
["Alice"]
```

## Error Handling and Debugging

The REPL displays detailed error information for invalid JavaScript or module errors:

```javascript
js> undefinedVariable
ReferenceError: undefinedVariable is not defined

js> const fs = require("fs")
js> fs.readFileSync("/nonexistent/file")
Error: open /nonexistent/file: no such file or directory

js> JSON.parse("invalid json")
SyntaxError: Unexpected token i in JSON at position 0
```

### Debug Mode

Enable debug logging to see module registration and runtime details:

```bash
go run ./cmd/goja-repl --log-level debug tui
2024/01/15 10:30:45 engine initialised, modules: [fs, http, timer, uuid, yaml]
js> const fs = require("fs")
2024/01/15 10:30:47 module loaded: fs
```

## Advanced REPL Techniques

The REPL maintains state across interactions, enabling sophisticated development workflows including variable persistence, API exploration, and iterative logic refinement.

### Variable Persistence

Variables and functions persist across REPL sessions:

```javascript
js> let counter = 0
js> function increment() { return ++counter; }
js> increment()
1
js> increment()
2
js> counter
2
```

### Module Experimentation

Use the REPL to explore module APIs:

```javascript
js> const uuid = require("uuid")
js> Object.getOwnPropertyNames(uuid)
["v4"]

js> typeof uuid.v4
function

js> uuid.v4()
123e4567-e89b-12d3-a456-426614174000
```

### Testing Complex Logic

Prototype and test JavaScript logic interactively before moving it into a Go test, module implementation, or a higher-level automation flow:

```javascript
js> function processData(items) {
...   return items
...     .filter(item => item.active)
...     .map(item => ({ ...item, processed: true }))
...     .sort((a, b) => a.priority - b.priority);
... }

js> const testData = [
...   { id: 1, active: true, priority: 2 },
...   { id: 2, active: false, priority: 1 },
...   { id: 3, active: true, priority: 1 }
... ]

js> processData(testData)
[
  { id: 3, active: true, priority: 1, processed: true },
  { id: 1, active: true, priority: 2, processed: true }
]
```

## Console Integration

The global `console` object provides familiar debugging capabilities:

```javascript
js> console.log("Simple message")
Simple message

js> console.log("Multiple", "arguments", 123)
Multiple arguments 123

js> const obj = { name: "test", value: 42 }
js> console.log("Object:", obj)
Object: [object Object]

js> console.error("This is an error message")
This is an error message
```

For structured output and advanced formatting, combine with native modules:

```javascript
js> const data = { users: ["Alice", "Bob"], count: 2 }
js> console.log(JSON.stringify(data, null, 2))
{
  "users": [
    "Alice",
    "Bob"
  ],
  "count": 2
}
```
