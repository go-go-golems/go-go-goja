---
Title: Introduction to go-go-goja
Slug: introduction
Short: A Node.js-style JavaScript runtime sandbox with native Go modules
Topics:
- goja
- javascript
- modules
- runtime
Commands:
- goja-repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

go-go-goja bridges the gap between Go's performance and JavaScript's flexibility by providing a runtime where you can implement native modules in Go and seamlessly use them from JavaScript. This sandbox environment enables rapid prototyping and experimentation with the [goja](https://github.com/dop251/goja) JavaScript engine while maintaining full control over the module ecosystem.

## Core Concept

The project's architecture centers around a module registration system that automatically makes Go-implemented functionality available to JavaScript via Node.js-style `require()` calls. When you drop a properly structured Go file into the `modules/` directory, it becomes immediately accessible from JavaScript without any build configuration or explicit binding code.

## Key Capabilities

These core features distinguish go-go-goja from other JavaScript runtimes by prioritizing developer productivity and seamless Go integration.

**Zero-configuration module system:** Write a Go module once and use it immediately from JavaScript without webpack, bundlers, or manual registration steps.

**Full JavaScript environment:** Complete ES5+ support with `console` object, `require()` function, and async primitives including Promises.

**Bidirectional type conversion:** Automatic marshaling between Go types (strings, numbers, maps, slices) and JavaScript values.

**Interactive development:** REPL environment for testing JavaScript code with immediate access to all registered modules.

## Quick Start Example

This example demonstrates the complete workflow from implementing a native module to using it in JavaScript, showcasing how quickly you can extend JavaScript with Go functionality.

Create a native module:
```go
// modules/uuid/uuid.go
package uuidmod

import (
    "github.com/dop251/goja"
    "github.com/go-go-golems/go-go-goja/modules"
    "github.com/google/uuid"
)

type m struct{}
var _ modules.NativeModule = (*m)(nil)

func (m) Name() string { return "uuid" }
func (m) Loader(vm *goja.Runtime, mod *goja.Object) {
    exports := mod.Get("exports").(*goja.Object)
    exports.Set("v4", func() string { return uuid.NewString() })
}
func init() { modules.Register(&m{}) }
```

Use it from JavaScript:
```javascript
const { v4 } = require("uuid");
console.log(v4());
// Output: 123e4567-e89b-12d3-a456-426614174000
```

Run the code:
```bash
go run ./cmd/goja-repl tui
js> const { v4 } = require("uuid"); console.log(v4());
```

For comprehensive guidance on creating modules, using async patterns, using built-in Node.js-style primitives, and integrating with the runtime, explore the topics below.

## Built-in Node.js-Style Primitives

go-go-goja includes a practical set of built-in primitives for scripts that need common Node.js capabilities without custom Go bindings. Data-only primitives such as `require("path")`, `require("crypto")`, `require("time")`, and `performance.now()` are available in every engine runtime. Host-access primitives such as `require("fs")` and `require("os")` are enabled explicitly through the engine factory.

See `nodejs-primitives` for the complete reference, including granular module selection, the `process.env` opt-in policy, and security notes for embedders.
