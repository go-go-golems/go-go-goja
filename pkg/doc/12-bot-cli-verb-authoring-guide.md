---
Title: "bot CLI verb authoring guide"
Slug: bot-cli-verb-authoring-guide
Short: "How to author JavaScript bot scripts for the new `go-go-goja bots` command surface."
Topics:
- goja
- glazed
- javascript
- cli
- bots
Commands:
- go-go-goja bots list
- go-go-goja bots run
- go-go-goja bots help
Flags:
- --bot-repository
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This page explains how to write JavaScript files that work well with the `go-go-goja bots` CLI.

The most important rule in v1 is simple: **a bot CLI command is an explicit `jsverbs` verb**. The scanner for the bot CLI runs with `IncludePublicFunctions = false`, so plain public functions are not exposed automatically. If you want a function to be runnable through `go-go-goja bots run ...`, add `__verb__(...)` metadata for it.

## Minimal example

```js
function greet(name, excited) {
  return {
    greeting: excited ? `Hello, ${name}!` : `Hello, ${name}`
  };
}

__verb__("greet", {
  short: "Greet one person",
  fields: {
    name: {
      argument: true,
      help: "Person name"
    },
    excited: {
      type: "bool",
      short: "e",
      help: "Add excitement"
    }
  }
});
```

If this code lives in `discord.js`, the full bot path becomes:

```text
discord greet
```

You can then run it like this:

```bash
go-go-goja bots run discord greet --bot-repository ./path/to/repo Manuel --excited
```

## How paths are formed

By default, `jsverbs` builds parents from the file path and file name.

Examples:

- `discord.js` + `__verb__("greet", ...)` -> `discord greet`
- `nested/relay.js` + `__verb__("send", ...)` -> `nested relay send`
- `alerts/index.js` + `__verb__("ping", ...)` -> `alerts ping`

That means you should think of:

- directories as command groups,
- file names as command groups,
- verb names as the final action.

## Output modes

The bot CLI supports the standard `jsverbs` output modes.

### Structured output (default)

If you return an object or array and do not set `output: "text"`, the CLI prints JSON in v1.

```js
function status() {
  return { ok: true, service: "discord" };
}

__verb__("status", {
  short: "Show status"
});
```

### Text output

If the result is naturally plain text, set `output: "text"`.

```js
function banner(name) {
  return `*** ${name} ***\n`;
}

__verb__("banner", {
  short: "Render a banner",
  output: "text",
  fields: {
    name: { argument: true }
  }
});
```

## Async handlers

Async functions work too.

```js
const multiply = async (a, b) => {
  return { product: a * b };
};

__verb__("multiply", {
  short: "Multiply asynchronously",
  fields: {
    a: { type: "int", argument: true },
    b: { type: "int", argument: true }
  }
});
```

The host waits for the Promise to settle before printing the result.

## Relative require

Relative imports are supported when the bot repository is scanned and run through the `bots` CLI.

```js
const relay = (prefix, target) => {
  const helper = require("./sub/helper");
  return { value: helper.render(prefix, target) };
};

__verb__("relay", {
  short: "Use a helper",
  fields: {
    prefix: { argument: true },
    target: { argument: true }
  }
});
```

## Recommended workflow

1. Put bot scripts in a dedicated repository or folder.
2. Add explicit `__verb__(...)` metadata for every runnable function.
3. Use `go-go-goja bots list --bot-repository <dir>` to verify discovery.
4. Use `go-go-goja bots help <path> --bot-repository <dir>` to inspect flags.
5. Use `go-go-goja bots run <path> --bot-repository <dir> ...` to execute.

## Relationship to sandbox `defineBot(...)`

The `sandbox` module is still useful for runtime-managed bot behavior, but the `bots` CLI does not scan `defineBot(...)` registrations directly in v1. If you need sandbox functionality from the CLI, write a scanner-visible wrapper function and expose it with `__verb__(...)`.
