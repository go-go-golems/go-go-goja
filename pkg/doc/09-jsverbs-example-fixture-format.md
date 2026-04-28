---
Title: "jsverbs-example fixture format"
Slug: jsverbs-example-fixture-format
Short: "Metadata and fixture patterns supported by the prototype JavaScript verb runner."
Topics:
- goja
- glazed
- metadata
- fixtures
Commands:
- jsverbs-example
Flags:
- --dir
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Example
---

This page describes the metadata patterns supported by the current prototype so you can create fixture directories intentionally.

The important recent change is that metadata is now parsed as a strict literal subset rather than by rewriting JavaScript object text into JSON. That makes the accepted fixture format narrower, but also far more predictable. If a metadata block uses dynamic JavaScript, the scanner now reports an error instead of silently guessing.

## Supported metadata

### `__package__({...})`

Use this to set file-level grouping metadata such as a package name or extra parent verbs.

### `__section__("slug", {...})`

Use this to define file-local Glazed sections with reusable fields. Commands in the same file can opt into those sections through `sections: ["slug"]` or by binding a parameter to that section.

`__section__` does not import metadata across files through `require()`. If you need one section catalog reused across multiple files, register it from Go on the scanned registry:

```go
registry, err := jsverbs.ScanDir("./verbs")
if err != nil {
    return err
}

err = registry.AddSharedSection(&jsverbs.SectionSpec{
    Slug:  "filters",
    Title: "Filters",
    Fields: map[string]*jsverbs.FieldSpec{
        "state": {Type: "choice", Choices: []string{"open", "closed"}},
    },
})
if err != nil {
    return err
}
```

Registry-level shared sections are resolved after file-local sections. That means a file can still override a shared slug intentionally by declaring its own `__section__("filters", ...)`.

The example runner now includes a concrete demo of that host-side registration path under `testdata/jsverbs-example/registry-shared`. That fixture binds to `filters` without declaring any local `__section__`; `cmd/jsverbs-example` injects the shared section before command compilation when you scan that directory.

### `__verb__("functionName", {...})`

Use this to override the inferred command name, help text, parents, or fields. Field metadata currently supports:

- `type`
- `help`
- `short`
- `default`
- `choices`
- `required`
- `argument`
- `section`
- `bind`

Verb metadata also supports output selection:

- `output: "text"` turns the verb into a plain writer command
- omitted `output` keeps the default structured Glazed row behavior

Metadata values should be static literals. Supported literal shapes are:

- object literals
- array literals
- quoted strings
- template strings without `${...}` substitutions
- numbers
- `true`, `false`, and `null`

Unsupported metadata shapes include:

- function calls
- identifiers used as values
- spreads
- computed object keys
- template substitutions
- other dynamic expressions

That restriction is intentional. Metadata is part of command discovery, so it needs to behave like declarative configuration rather than executable code.

## Binding modes

`bind` changes how a JavaScript parameter is populated.

- `bind: "all"` passes every resolved Glazed value as one object.
- `bind: "context"` passes execution metadata such as the command path, module path, root directory, and section maps.
- `bind: "filters"` passes the values from the named section as one object.

Bindings are now resolved through one shared internal binding plan used by both schema generation and runtime invocation. As a JavaScript author you do not need to know the internal type name, but you do benefit from the result: if a bind is invalid, the failure happens consistently instead of one phase accepting it while another phase mis-invokes the function.

## Example

```js
__section__("filters", {
  fields: {
    state: { type: "choice", choices: ["open", "closed"], default: "open" }
  }
});

function listIssues(repo, filters, meta) {
  return [{ repo, state: filters.state, rootDir: meta.rootDir }];
}

__verb__("listIssues", {
  sections: ["filters"],
  fields: {
    repo: { argument: true },
    filters: { bind: "filters" },
    meta: { bind: "context" }
  }
});
```

File-backed object arguments use Glazed's `objectFromFile` field type. Glazed reads the JSON or YAML file before invoking the JavaScript function, so the verb receives a normal JavaScript object rather than a filename string:

```js
function inspectConfig(config) {
  return [{
    name: config.name,
    nested: config.nested && config.nested.value,
    itemCount: config.items ? config.items.length : 0
  }];
}

__verb__("inspectConfig", {
  short: "Inspect a JSON or YAML config file",
  fields: {
    config: {
      type: "objectFromFile",
      help: "Path to a JSON or YAML config file"
    }
  }
});
```

Given a file like this:

```json
{
  "name": "demo",
  "nested": { "value": 42 },
  "items": ["a", "b", "c"]
}
```

run the command with the file path:

```bash
jsverbs-example --dir ./verbs objectfile inspect-config --config ./config.json
```

Inside JavaScript, `typeof config` is `"object"`, `config.name` is `"demo"`, and `config.items.length` is `3`.

Text-output verbs look like this:

```js
function banner(name) {
  return `=== ${name} ===\n`;
}

__verb__("banner", {
  output: "text",
  fields: {
    name: { argument: true }
  }
});
```

## Non-directory sources

Most examples use directory fixtures because they are easy to inspect and run manually. The package is no longer limited to that one source shape, though. The same metadata format works when files come from:

- `ScanDir(...)`
- `ScanFS(...)`
- `ScanSource(...)`
- `ScanSources(...)`

That means the fixture format described here is also the format for embedded command trees and in-memory synthetic command files.

## Scanner failures

Malformed metadata is now surfaced as scanner diagnostics and usually returned as a `ScanError`. In practice that means fixture authors should expect invalid metadata to fail loudly.

Typical causes:

- using a dynamic expression in metadata,
- binding to a section that does not exist,
- declaring a verb for a function name that is not present in the file.

When shared sections are involved, "does not exist" means "not found either in the file-local `__section__` catalog or in the registry shared-section catalog added from Go."

## See Also

- `glaze help jsverbs-example-overview`
- `glaze help jsverbs-example-developer-guide`
- `glaze help jsverbs-example-reference`
