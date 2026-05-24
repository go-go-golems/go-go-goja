---
Title: "generated xgoja runtime overview"
Slug: runtime-overview
Short: "How generated xgoja binaries create runtimes and expose commands."
Topics:
- xgoja
- runtime
- cli
Commands:
- repl
- modules
- verbs
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Application
---

A generated xgoja binary is a normal Go command-line program built from an `xgoja.yaml` file. It imports selected provider packages, registers their native Goja modules, embeds a normalized runtime specification, and creates commands that execute JavaScript against named runtime profiles.

A runtime profile selects which provider modules are available to a command invocation. Commands such as `repl`, `run`, and mounted JavaScript verbs create a fresh runtime for the selected profile, execute the requested JavaScript, and close the runtime afterwards.

## Common commands

- `modules` lists provider modules compiled into the binary.
- `repl` or the configured evaluation command evaluates one JavaScript source string.
- `verbs` or the configured verbs command exposes JavaScript verbs from configured sources.
- `help` lists bundled runtime help topics.

## Runtime profile selection

Commands that execute JavaScript accept or derive a runtime profile. The profile controls `require()` visibility. If a module is compiled into the binary but is not selected by the runtime profile, JavaScript code in that command cannot require it.
