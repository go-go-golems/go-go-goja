---
Title: Creating Native Modules
Slug: creating-modules
Short: Step-by-step guide to implementing Go modules for JavaScript
Topics:
- modules
- native
- go
- implementation
Commands:
- goja-repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

Native modules bridge Go's capabilities with JavaScript's accessibility by implementing the `NativeModule` interface. Each module becomes a Node.js-style package that JavaScript can import via `require()`, enabling you to expose Go functionality while maintaining familiar JavaScript patterns.

## Module Structure

Every native module follows a consistent pattern that includes interface implementation, automatic registration, and bidirectional type conversion. The module system handles the loading mechanism, while your code focuses purely on the exported functionality.

### Basic Module Template

```go
// modules/example/example.go
package examplemod

import (
    "github.com/dop251/goja"
    "github.com/go-go-golems/go-go-goja/modules"
)

type m struct{}

// Compile-time interface check
var _ modules.NativeModule = (*m)(nil)

func (m) Name() string { 
    return "example" // This becomes the require() name
}

func (m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)
    
    // Export functions to JavaScript
    exports.Set("hello", func(name string) string {
        return "Hello, " + name + "!"
    })
}

func init() {
    modules.Register(&m{}) // Auto-register during import
}
```

### Type Conversion Rules

The goja runtime automatically converts between Go and JavaScript types according to these mappings:

**Go to JavaScript:**
- `string` → String
- `int`, `int64`, `float64` → Number  
- `bool` → Boolean
- `map[string]interface{}` → Object
- `[]interface{}` → Array
- `nil` → null

**JavaScript to Go:**
- String → `string`
- Number → `int`, `int64`, `float64` (as appropriate)
- Boolean → `bool`
- Object → `map[string]interface{}`
- Array → `[]interface{}`
- null/undefined → `nil`

## Real-World Examples

These comprehensive examples demonstrate patterns beyond the basic template, showing how to handle complex data types, error conditions, and multiple function exports within a single module.

### File System Module

```go
// modules/fs/fs.go
package fsmod

import (
    "os"
    "path/filepath"
    "github.com/dop251/goja"
    "github.com/go-go-golems/go-go-goja/modules"
)

type m struct{}
var _ modules.NativeModule = (*m)(nil)

func (m) Name() string { return "fs" }

func (m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)
    
    exports.Set("readFileSync", func(path string) (string, error) {
        data, err := os.ReadFile(path)
        if err != nil {
            return "", err
        }
        return string(data), nil
    })
    
    exports.Set("writeFileSync", func(path, data string) error {
        dir := filepath.Dir(path)
        if err := os.MkdirAll(dir, 0755); err != nil {
            return err
        }
        return os.WriteFile(path, []byte(data), 0644)
    })
    
    exports.Set("existsSync", func(path string) bool {
        _, err := os.Stat(path)
        return err == nil
    })
}

func init() { modules.Register(&m{}) }
```

Usage from JavaScript:
```javascript
const fs = require("fs");

// Write and read files
fs.writeFileSync("/tmp/test.txt", "Hello World!");
const content = fs.readFileSync("/tmp/test.txt");
console.log(content); // "Hello World!"

// Check existence
if (fs.existsSync("/tmp/test.txt")) {
    console.log("File exists!");
}
```

### HTTP Client Module

```go
// modules/http/http.go
package httpmod

import (
    "io"
    "net/http"
    "github.com/dop251/goja"
    "github.com/go-go-golems/go-go-goja/modules"
)

type m struct{}
var _ modules.NativeModule = (*m)(nil)

func (m) Name() string { return "http" }

func (m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)
    
    exports.Set("get", func(url string) (map[string]interface{}, error) {
        resp, err := http.Get(url)
        if err != nil {
            return nil, err
        }
        defer resp.Body.Close()
        
        body, err := io.ReadAll(resp.Body)
        if err != nil {
            return nil, err
        }
        
        return map[string]interface{}{
            "status": resp.StatusCode,
            "body":   string(body),
            "headers": resp.Header,
        }, nil
    })
}

func init() { modules.Register(&m{}) }
```

Usage from JavaScript:
```javascript
const http = require("http");

try {
    const response = http.get("https://api.github.com/users/octocat");
    console.log("Status:", response.status);
    
    const user = JSON.parse(response.body);
    console.log("User:", user.login);
} catch (error) {
    console.error("Request failed:", error);
}
```

## Error Handling

Return Go errors directly from exported functions. The runtime automatically converts them to JavaScript Error objects:

```go
exports.Set("divide", func(a, b float64) (float64, error) {
    if b == 0 {
        return 0, fmt.Errorf("division by zero")
    }
    return a / b, nil
})
```

JavaScript usage with error handling:
```javascript
const math = require("math");

try {
    const result = math.divide(10, 2); // Returns 5
    console.log(result);
} catch (error) {
    console.error("Math error:", error.message);
}
```

## Returning Rich Content (Images)

A function can return a Go value that carries rich content, such as an image, next to ordinary data. The value implements `resultcontent.Provider` (package `github.com/go-go-golems/go-go-goja/pkg/resultcontent`); the host that runs the code decides what to do with it: MCP returns an image, CLIs write a file, tables get a path or placeholder. JavaScript just returns the value, alone or inside an object or array:

```go
type Provider interface {
    ResultContent(ctx context.Context) ([]Content, error) // Content{MIMEType, Data, Caption}
}
type Describer interface { // optional
    ResultPlaceholder() string // shown as "[content N: <this>]"
}
```

A complete module that returns a solid-colour PNG:

```go
package swatch

import (
    "bytes"
    "context"
    "fmt"
    "image"
    "image/color"
    "image/png"

    "github.com/dop251/goja"
    "github.com/go-go-golems/go-go-goja/modules"
    "github.com/go-go-golems/go-go-goja/pkg/resultcontent"
)

// Swatch renders lazily. Its fields are unexported, so JSON.stringify and the
// json.Marshal used for persisted bindings never see image bytes.
type Swatch struct {
    size int
    rgb  color.RGBA
}

var _ resultcontent.Provider = (*Swatch)(nil)
var _ resultcontent.Describer = (*Swatch)(nil)

// ResultContent runs off the runtime owner goroutine and uses only Go state.
func (s *Swatch) ResultContent(ctx context.Context) ([]resultcontent.Content, error) {
    img := image.NewRGBA(image.Rect(0, 0, s.size, s.size))
    for y := 0; y < s.size; y++ {
        for x := 0; x < s.size; x++ {
            img.Set(x, y, s.rgb)
        }
    }
    var buf bytes.Buffer
    if err := png.Encode(&buf, img); err != nil {
        return nil, err
    }
    return []resultcontent.Content{{MIMEType: "image/png", Data: buf.Bytes(), Caption: s.label()}}, nil
}

// ResultPlaceholder runs on the owner goroutine while the result is walked: keep it cheap.
func (s *Swatch) ResultPlaceholder() string { return s.label() }

func (s *Swatch) label() string {
    return fmt.Sprintf("swatch %dx%d #%02x%02x%02x", s.size, s.size, s.rgb.R, s.rgb.G, s.rgb.B)
}

type m struct{}

var _ modules.NativeModule = (*m)(nil)

func (m) Name() string { return "swatch" }
func (m) Doc() string  { return "swatch.solid(size, r, g, b) returns a PNG swatch" }

func (m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)
    _ = exports.Set("solid", func(size int, r, g, b uint8) *Swatch {
        return &Swatch{size: size, rgb: color.RGBA{R: r, G: g, B: b, A: 255}}
    })
}

func init() { modules.Register(&m{}) }
```

JavaScript:

```javascript
const { solid } = require("swatch");
const red = solid(4, 255, 0, 0);
({ size: 4, red, both: [red, solid(2, 0, 0, 255)] })
```

Over MCP (`goja-repl mcp`, `replmcp`) the `eval` tool returns this text followed by two `image/png` items:

```text
{size: 4, red: [content 1: swatch 4x4 #ff0000], both: [[content 1: swatch 4x4 #ff0000], [content 2: swatch 2x2 #0000ff]]}

content:
content 1 swatch 4x4 #ff0000: attached (image/png, 78 bytes)
content 2 swatch 2x2 #0000ff: attached (image/png, 80 bytes)
```

As the last expression of an xgoja `eval` (or the `module.exports` of a `run` script) with `--content-dir out`:

```text
{size: 4, red: [content 1: swatch 4x4 #ff0000], both: [[content 1: swatch 4x4 #ff0000], [content 2: swatch 2x2 #0000ff]]}
# content 1 swatch 4x4 #ff0000: out/content-1.png
# content 2 swatch 2x2 #0000ff: out/content-2.png
```

Rules for providers:

- **Return a Go value.** Only Go-backed objects whose exported value implements `Provider` are found. A provider may sit inside plain objects, arrays, Go maps with string keys and Go slices (`map[string]*Swatch`, `[]*Swatch` work). The walk is bounded (depth 8, 1000 properties), never reads `then` and never calls getters, so thenable proxies are safe.
- **Render lazily.** `ResultContent` is only called when a host delivers the content (MCP, `--content-dir`), off the owner goroutine and with the request context. Do not touch `goja` values there; capture what you need in Go fields when the value is created.
- **Keep bytes unexported.** Exported fields are visible to `JSON.stringify` and to the `json.Marshal` of persisted session bindings.
- **Expect limits.** Hosts cap item count and bytes (`resultcontent.Limits`; replsession defaults 8 items / 8 MiB, MCP 4 MiB per image) and report what they dropped. Providers past the item limit are not called.
- **Name it.** Without `Describer` the placeholder is `[content N]`.

What each surface does with a provider:

| Surface | Result |
| --- | --- |
| `replapi`/`replsession` (Go hosts, `goja-repl eval`/`serve`/TUI) | byte-free refs in `cell.execution.content`, rendered bytes in `EvaluateResponse.Content` (not persisted) |
| MCP (`goja-repl mcp`, `replmcp`, xgoja `mcp serve`) | `ImageContent` items after the text result |
| xgoja `eval`/`run`, `goja-repl run` | placeholders; `--content-dir DIR` writes `DIR/content-N.<ext>` and prints the paths |
| jsverbs | placeholder strings in rows; `--content-dir DIR` puts the file path in the row field |
| `console.log(value)` | prints `[content: <placeholder>]`; never renders |

Hosts and session hooks are covered in `goja-repl help replapi-guide`.

## Module Registration

Add your module to the import list in `pkg/engine/runtime.go` to ensure it's loaded:

```go
import (
    _ "github.com/go-go-golems/go-go-goja/modules/fs"
    _ "github.com/go-go-golems/go-go-goja/modules/http"
    _ "github.com/go-go-golems/go-go-goja/modules/yourmodule" // Add here
)
```

The blank import ensures the module's `init()` function runs, registering it with the module system.

## TypeScript Declarations

If the module is user-facing, add a TypeScript descriptor next to the runtime implementation. This keeps editor completions and generated `.d.ts` files aligned with the actual Go exports.

Implement `modules.TypeScriptDeclarer` on the same module type:

```go
import "github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"

type m struct{}

var _ modules.NativeModule = (*m)(nil)
var _ modules.TypeScriptDeclarer = (*m)(nil)

func (m) TypeScriptModule() *spec.Module {
    return &spec.Module{
        Name: "example",
        Functions: []spec.Function{
            {
                Name: "hello",
                Params: []spec.Param{
                    {Name: "name", Type: spec.String()},
                },
                Returns: spec.String(),
            },
        },
    }
}
```

The canonical declaration-generation workflow for this repository is `go generate` on the bun demo package:

```bash
go generate ./cmd/bun-demo
```

That command is defined in `cmd/bun-demo/generate.go`. If your new module should appear in `cmd/bun-demo/js/src/types/goja-modules.d.ts`, add its module name to the `//go:generate go run ../gen-dts ... --module ...` filter there, run generation, review the generated diff, and commit the updated `.d.ts` together with the module code.

For the full declaration generator reference, see:

    goja-repl help typescript-declaration-generator

For asynchronous operations and Promise-based APIs, see:

    goja-repl help async-patterns

For hosting sessions whose results carry content, see:

    goja-repl help replapi-guide
