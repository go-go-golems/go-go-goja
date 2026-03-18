---
Title: "Using the docs Module"
Slug: goja-docs-module-guide
Short: "Reference and usage guide for the runtime-scoped docs module that exposes help pages, jsdoc entries, and plugin metadata."
Topics:
- goja
- documentation
- repl
- javascript
- plugins
Commands:
- repl
- js-repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `docs` module gives JavaScript code a uniform way to inspect runtime documentation sources. It is available through `require("docs")` in runtimes that wire the documentation registrar, which now includes both `repl` and `js-repl`.

This guide explains what the module exposes, what kinds of entries it returns, and how to use it in practice.

## What the module aggregates

The module is a view over one or more documentation providers:

- embedded Glazed help pages from the main help system
- attached jsdoc stores, when a runtime registers one
- loaded plugin manifests, exports, and methods

Each result tells you which source it came from. The module does not flatten everything into one opaque blob.

## Basic usage

```javascript
const docs = require("docs")

docs.sources()
```

Typical result:

```javascript
[
  {
    id: "default-help",
    kind: "glazed-help",
    title: "Default Help",
    summary: "Embedded REPL help pages",
    runtimeScoped: false,
    metadata: null,
  },
  {
    id: "plugin-manifests",
    kind: "plugin",
    title: "Plugin Manifests",
    summary: "Runtime-scoped plugin metadata",
    runtimeScoped: true,
    metadata: null,
  },
]
```

## API reference

### `docs.sources()`

Return every registered documentation source descriptor.

Use this first when you want to know what data is available in the current runtime.

### `docs.search(query)`

Search across one or more sources.

Accepted query fields:

- `text`
- `sourceIds`
- `kinds`
- `topics`
- `tags`
- `limit`

Example:

```javascript
docs.search({
  text: "plugin",
  kinds: ["plugin-module", "plugin-method"],
  limit: 10,
})
```

### `docs.get(ref)`

Fetch one entry by explicit reference object:

```javascript
docs.get({
  sourceId: "default-help",
  kind: "help-section",
  id: "repl-usage",
})
```

This returns `null` when the entry does not exist.

### `docs.byID(sourceId, kind, id)`

Convenience form of `docs.get(...)` when you already know the reference components.

Example:

```javascript
docs.byID("plugin-manifests", "plugin-module", "plugin:examples:kv")
```

### `docs.bySlug(sourceId, slug)`

Convenience helper for Glazed help pages:

```javascript
docs.bySlug("default-help", "repl-usage")
```

### `docs.bySymbol(sourceId, symbol)`

Convenience helper for jsdoc symbol entries:

```javascript
docs.bySymbol("workspace-jsdoc", "smoothstep")
```

## Entry shapes

Every returned entry uses the same top-level structure:

```javascript
{
  ref: { sourceId, kind, id },
  title,
  summary,
  body,
  topics,
  tags,
  path,
  kindLabel,
  related,
  metadata,
}
```

The shared shape is stable, but `metadata` is source-specific.

## Help page examples

```javascript
const docs = require("docs")
const replUsage = docs.bySlug("default-help", "repl-usage")

replUsage.title
replUsage.summary
replUsage.body
replUsage.metadata.commands
replUsage.metadata.flags
```

## Plugin metadata examples

Plugin entries are especially useful because they let you inspect manifests without reverse-engineering the JS callable surface.

### Module-level entry

```javascript
docs.byID("plugin-manifests", "plugin-module", "plugin:examples:kv")
```

### Export-level entry

```javascript
docs.byID("plugin-manifests", "plugin-export", "plugin:examples:kv/store")
```

### Method-level entry

```javascript
docs.byID("plugin-manifests", "plugin-method", "plugin:examples:kv/store.get")
```

This is where rich method doc bodies now show up.

## Common workflows

### Discover everything loaded from plugins

```javascript
docs.search({
  sourceIds: ["plugin-manifests"],
})
```

### Find all help pages about the REPL

```javascript
docs.search({
  sourceIds: ["default-help"],
  text: "repl",
})
```

### Show method docs for a plugin object

```javascript
docs.search({
  sourceIds: ["plugin-manifests"],
  kinds: ["plugin-method"],
  text: "store",
})
```

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `require("docs")` fails | The runtime was created without the documentation registrar | Use `repl` or `js-repl`, or attach the GOJA-11 registrar in your own runtime builder |
| `plugin-manifests` source is missing | No plugins were loaded in that runtime | Pass `--plugin-dir` or install plugins under `~/.go-go-goja/plugins/...` |
| `docs.bySymbol(...)` returns `null` | No jsdoc store was attached under that source ID | Attach a jsdoc provider when constructing your runtime or query `docs.sources()` first |
| A plugin method has no body | The plugin did not publish method docs | Add method docs in the plugin SDK declaration |

## See Also

- `repl-usage`
- `goja-plugin-user-guide`
- `goja-plugin-developer-guide`
