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
- repl
- js-repl
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
# Lightweight line REPL / script runner
go run ./cmd/repl

# With debug logging
go run ./cmd/repl --debug

# Run a script file
go run ./cmd/repl path/to/script.js

# Rich Bobatea-based JS REPL with completion/help widgets
go run ./cmd/js-repl
```

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

### REPL Commands

The REPL recognizes special commands prefixed with `:`:

- `:help` - Display available commands and usage information
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
go run ./cmd/repl --debug
2024/01/15 10:30:45 engine initialised, modules: [fs, http, timer, uuid]
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

Prototype and test JavaScript logic before moving to script files:

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

## Script File Execution

Run complete JavaScript files through the REPL environment:

```bash
# Create a test script
echo 'const fs = require("fs"); console.log("Files in /tmp:", fs.readdirSync("/tmp"));' > test.js

# Execute it
go run ./cmd/repl test.js
Files in /tmp: [demo.txt, .com.apple.launchd.ABC123, ...]
```

This approach combines the benefits of script organization with full access to native modules and proper error reporting.

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
