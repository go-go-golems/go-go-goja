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

## Supported metadata

### `__package__({...})`

Use this to set file-level grouping metadata such as a package name or extra parent verbs.

### `__section__("slug", {...})`

Use this to define shared Glazed sections with reusable fields. Commands opt into those sections through `sections: ["slug"]` or by binding a parameter to that section.

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

## Binding modes

`bind` changes how a JavaScript parameter is populated.

- `bind: "all"` passes every resolved Glazed value as one object.
- `bind: "context"` passes execution metadata such as the command path, module path, root directory, and section maps.
- `bind: "filters"` passes the values from the named section as one object.

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

## See Also

- `glaze help jsverbs-example-overview`
- `glaze help jsverbs-example-developer-guide`
