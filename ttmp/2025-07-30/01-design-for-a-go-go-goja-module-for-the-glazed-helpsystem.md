---
Title: Design for Glazed Help System Module
Slug: glazed-help-module-design
Short: Native module exposing multiple Glazed HelpSystem instances to JavaScript via go-go-goja
IsTopLevel: true
SectionType: GeneralTopic
ShowPerDefault: true
---

# Glazed Help System Module – Design Document

## 1. Purpose
The goal is to create a **native Go module** that makes Glazed’s `help.HelpSystem` available inside go-go-goja’s JavaScript runtime.  
Key features:

* Wrap one _or more_ `*help.HelpSystem` instances.
* Address them from JavaScript using a **registry key** (string).
* Expose a minimal, ergonomic API for querying sections and retrieving rendered pages.
* Follow the patterns described in
  * `01-help-system.md` – structure & capabilities of `HelpSystem`.
  * `02-creating-modules.md` – guidelines for implementing native modules.

> We **only design** here – no implementation yet.

## 2. High-Level Architecture
```mermaid
classDiagram
    class help.HelpSystem {
        +QuerySections(q string) ([]*Section, error)
        +GetSectionWithSlug(slug string) (*Section, error)
        +GetTopLevelHelpPage() *Page
    }
    class Registry {
        +Register(key string, hs *help.HelpSystem)
        +Get(key string) (*help.HelpSystem, error)
        -systems map[string]*help.HelpSystem
        -mu sync.RWMutex
    }
    class Module "glazehelp" {
        +Loader(vm *goja.Runtime, module *goja.Object)
    }
    help.HelpSystem <|.. Registry
    Registry <.. Module
```

1. **Registry** – Go-side singleton that stores `HelpSystem` pointers keyed by string.
2. **Module** – Implements `modules.NativeModule`; exposes JS functions that proxy into a selected `HelpSystem` retrieved from the registry.

## 3. Go API Surface
### 3.1 Registry (`modules/glazehelp/registry.go`)
```go
package glazehelp

var (
    systems = map[string]*help.HelpSystem{}
    mu      sync.RWMutex
)

func Register(key string, hs *help.HelpSystem) {
    mu.Lock()
    defer mu.Unlock()
    systems[key] = hs
}

func MustRegister(key string, hs *help.HelpSystem) {
    if _, ok := systems[key]; ok {
        panic(fmt.Sprintf("help system with key %s already registered", key))
    }
    Register(key, hs)
}

func Get(key string) (*help.HelpSystem, error) {
    mu.RLock()
    defer mu.RUnlock()
    hs, ok := systems[key]
    if !ok {
        return nil, fmt.Errorf("help system %s not found", key)
    }
    return hs, nil
}
```
### 3.2 Native Module Skeleton (`modules/glazehelp/glazehelp.go`)
```go
package glazehelp

import (
    "github.com/dop251/goja"
    "github.com/go-go-golems/go-go-goja/modules"
)

type m struct{}

// Compile-time interface check
afterhelp
var _ modules.NativeModule = (*m)(nil)

func (m) Name() string { return "glazehelp" }

func (m) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    exports := moduleObj.Get("exports").(*goja.Object)

    // JS: glazehelp.query(key, dsl) -> []Section (as JS objects)
    exports.Set("query", func(key, dsl string) interface{} {/* impl */})

    // JS: glazehelp.section(key, slug) -> Section or null
    exports.Set("section", func(key, slug string) interface{} {/* impl */})

    // JS: glazehelp.render(key) -> markdown string (top-level page)
    exports.Set("render", func(key string) (string, error) {/* impl */})

    // JS: glazehelp.topics(key) -> []string (distinct topics across all sections)
    exports.Set("topics", func(key string) interface{} {/* impl */})

    // JS: glazehelp.keys() -> []string (registered help system keys)
    exports.Set("keys", func() interface{} {/* impl */})
}

func init() { modules.Register(&m{}) }
```

## 4. JavaScript API
```javascript
const help = require("glazehelp");

// List sections matching DSL query
const results = help.query("default", "type:example AND topic:database");
results.forEach(s => console.log(s.slug, s.short));

// Fetch single section by slug
const sec = help.section("default", "help-system");
console.log(sec.title, sec.content);

// Render the root help page to the console
console.log(help.render("default"));

// List all topics for a help system
console.log(help.topics("default"));

// Show available help systems
console.log(help.keys());
```

* `key` – string identifier of the registered help system.
* Return values are **plain JS objects** generated via goja’s automatic conversion from Go structs / maps.

## 4.1 TypeScript Module Interface

```ts
type GlazeHelpModule = {
  query: (key: string, dsl: string) => any[];
  section: (key: string, slug: string) => any | null;
  render: (key: string) => string;
  topics: (key: string) => string[];
  keys: () => string[];
};
```

## 5. Type Mapping
| Go value                          | JS value |
|----------------------------------|----------|
| `*help.Section` → map            | Object   |
| `[]*help.Section` → `[]any`      | Array    |
| `*help.Page` → `string` (render) | String   |

Conversion relies on goja’s default rules (see `02-creating-modules.md`, Type Conversion Rules section).

## 6. Error Handling
* Go functions return `(value, error)`.
* Errors convert to JS `Error` objects automatically; callers should use `try/catch`.

## 7. Concurrency & Safety
`HelpSystem` itself is safe for concurrent reads; the registry uses a RW-mutex to guard modifications.  
Heavy queries are executed in Go; only results cross the language boundary.

## 8. Registration Workflow
1. Application constructs one or more `*help.HelpSystem` instances (e.g., loading docs from embedded FS).
2. Call `glazehelp.Register("default", hs)` before the JavaScript runtime executes.
3. JavaScript scripts can now access it via `require("glazehelp")`.

Example:
```go
hs := help.NewHelpSystem()
// ... load sections ...
glazehelp.Register("default", hs)

vm := goja.New()
require.Enable(vm) // whatever module system is in use
_, err := vm.RunString(`const help = require("glazehelp"); console.log(help.render("default"));`)
```

## 9. Directory Layout
```
go-go-goja/
  modules/
    glazehelp/
      registry.go
      glazehelp.go
```
`engine/runtime.go` will import the module via blank import:
```go
import (
    _ "github.com/go-go-golems/go-go-goja/modules/glazehelp"
)
```

## 10. Testing Strategy
* **Unit tests** for registry (duplicate keys, retrieval, missing key).
* **Integration tests**: spin up a `goja.Runtime`, register a fake `HelpSystem` with two sections, execute JS snippets, assert results.

## 11. Future Enhancements
1. **Caching** complex query results on the Go side.
2. **Streaming** large rendered pages via async iterator / generator.
3. **DSL Validation** helper exposed to JS (`validate(query)` → bool).
4. **Write support**: allow JS scripts to add or modify sections (requires concurrency guard).

---

**Status:** DESIGN COMPLETE – ready for implementation.
