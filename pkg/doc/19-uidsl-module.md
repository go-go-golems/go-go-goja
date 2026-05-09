---
Title: ui.dsl Module
Slug: uidsl-module
Short: Render safe server-side HTML nodes from Goja JavaScript
Topics:
- ui-dsl
- modules
- html
- javascript
Commands:
- goja-repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `ui.dsl` module provides a server-side HTML node DSL for Goja runtimes. It exports safe tag builders, document rendering, and internal-tool components such as tables, code blocks, badges, and tabs.

The module is available through both aliases when registered with `uidsl.NewRegistrar()`:

```javascript
const ui = require("ui.dsl");
// or
const ui = require("ui");
```

## Go setup

```go
factory, err := engine.NewBuilder().
    WithRuntimeModuleRegistrars(uidsl.NewRegistrar()).
    Build()
```

For HTTP applications, pair it with `pkg/gojahttp` and `modules/express`:

```go
host := gojahttp.NewHost(gojahttp.HostOptions{
    Renderer: uidsl.RenderAny,
})
```

## Core API

```javascript
ui.page({ title: "Title" }, ...children)
ui.fragment(...children)
ui.text(value)
ui.raw(html)
ui.render(value)
```

The module exports common HTML tag helpers such as `div`, `span`, `h1`, `p`, `a`, `form`, `input`, `table`, `tr`, `td`, `script`, `style`, and SVG helpers. Tag helpers accept an optional first attribute object followed by children:

```javascript
ui.div({ class: "card" },
  ui.h2("Result"),
  ui.p("Safe text")
)
```

## Safety model

Normal text and attribute values are escaped. `ui.raw(html)` intentionally bypasses escaping and should only be used with trusted HTML or CSS.

```javascript
ui.p("<script>")      // renders escaped text
ui.raw("<strong>ok</strong>") // renders raw HTML
```

## Tables

```javascript
ui.table.fromRows("users", [
  { id: 1, name: "Ada", active: true },
  { id: 2, name: "Grace", active: false },
])
```

The richer table builder supports columns, filters, sorting, pagination, money/date/tags/badge cell kinds, links, and empty states. These components are designed for server-rendered internal tools and database browsers.

## Inspection components

```javascript
ui.codeBlock("sql", "select * from users")
ui.sql("select * from users")
ui.js("console.log('hello')")
ui.jsonBlock({ ok: true })
ui.badge("active", { tone: "success" })
ui.tabs("details", [
  { id: "summary", label: "Summary", content: ui.p("Overview") },
  { id: "raw", label: "Raw", content: ui.jsonBlock({ ok: true }) },
])
```

These helpers render static HTML/CSS and do not require client-side JavaScript for their basic behavior.
