---
Title: Exec Module
Slug: exec-module
Short: Run external host commands from JavaScript
Topics:
- exec
- modules
- goja
- javascript
Commands:
- goja-repl
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `exec` module wraps `os/exec` so JavaScript runtimes can run external host processes. It is simple by design: one function, command plus arguments, returning the combined standard output and standard error as a string.

## JavaScript usage

```javascript
const exec = require("exec");

const output = exec.run("echo", ["hello", "world"]);
console.log(output);
// "hello world\n"
```

## Module API

### `run(cmd, args)`

Runs `cmd` with `args` on the host operating system and returns the combined stdout and stderr as a single string. If the command exits with a non-zero status or cannot be started, the function throws an error.

- `cmd` — executable name or absolute path.
- `args` — array of string arguments passed to the executable.

## Security notes

This module exists to let trusted scripts call local tools, compilers, or system utilities. Because it executes arbitrary host commands, it should only be enabled in runtimes you trust. In untrusted environments, disable the `exec` module through engine middleware.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| "executable file not found in $PATH" error | The command name is not on the host PATH | Use the absolute path to the executable |
| Command exits with an error | The process returned a non-zero exit code | Check the command arguments and environment before calling |
| Empty output despite command success | The command wrote everything to stderr | Rethrow or log errors to see stderr content |
