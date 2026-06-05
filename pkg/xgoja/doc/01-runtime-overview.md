---
Title: "generated xgoja runtime overview"
Slug: runtime-overview
Short: "How generated xgoja binaries create runtimes and expose commands."
Topics:
- xgoja
- runtime
- cli
Commands:
- eval
- run
- repl
- modules
- verbs
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Application
---

A generated xgoja binary is a normal Go command-line program built from an `xgoja.yaml` file. It imports selected provider packages, registers their native Goja modules, embeds a normalized runtime specification, and creates commands that execute JavaScript against the top-level generated module set.

A top-level `modules` selects which provider modules are available to a command invocation. Commands such as `eval`, `run`, `repl`, and mounted JavaScript verbs create a fresh runtime for the generated module set, execute the requested JavaScript, and close the runtime afterwards. The `repl` command is the rich interactive terminal REPL, while `eval` is one-shot string evaluation.

## Common commands

- `modules` lists provider modules compiled into the binary.
- `eval` or the configured evaluation command evaluates one JavaScript source string.
- `run` executes a JavaScript file with the generated module set and script-local module roots.
- `repl` starts an interactive Bubble Tea REPL backed by the generated module set.
- `verbs` or the configured verbs command exposes JavaScript verbs from configured sources.
- `help` lists bundled runtime help topics.

## Module set selection

Commands that execute JavaScript use the top-level module set. The module list controls `require()` visibility. If a module is compiled into the binary but is not selected by top-level `modules`, JavaScript code in generated runtime-backed commands cannot require it.

A runtime may select the same provider module more than once under different `as` names. `as` is the actual `require()` name, not an additional alias. For example, generated binaries that embed assets commonly register `name: fs` twice: `as: fs:assets` for read-only embedded files and `as: fs:host` for explicitly allowed host filesystem access. In that setup `require("fs")` is unavailable unless the runtime also registers an instance whose alias is exactly `fs`.
