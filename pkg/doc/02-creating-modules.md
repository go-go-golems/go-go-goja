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
- repl
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

## Module Registration

Add your module to the import list in `engine/runtime.go` to ensure it's loaded:

```go
import (
    _ "github.com/go-go-golems/go-go-goja/modules/fs"
    _ "github.com/go-go-golems/go-go-goja/modules/http"
    _ "github.com/go-go-golems/go-go-goja/modules/yourmodule" // Add here
)
```

The blank import ensures the module's `init()` function runs, registering it with the module system.

For asynchronous operations and Promise-based APIs, see:

    glaze help async-patterns
