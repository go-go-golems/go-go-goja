# Inspector example input

This folder contains JavaScript input for the `cmd/inspector` and Smalltalk inspector examples.

## Run

From the repository root:

```bash
GOWORK=off go run ./cmd/inspector ./examples/inspector/inspector-test.js
```

This launches an interactive TUI, so it is not part of the non-interactive smoke-test set.

## What this showcases

`inspector-test.js` contains classes, inheritance, functions, constants, objects, arrays, and closures. It gives the inspector enough structure to demonstrate AST navigation, symbol lookup, completion, and object-shape inspection.

## Status

The file is a maintained example input. The command itself is interactive; validate it manually in a terminal rather than in automated smoke scripts.
