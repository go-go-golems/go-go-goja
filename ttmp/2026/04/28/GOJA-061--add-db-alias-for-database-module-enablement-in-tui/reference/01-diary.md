---
title: GOJA-061 Diary — db alias for database module enablement
---

# Diary

## Reproduction

Ran the user command in tmux:

```bash
go run ./cmd/goja-repl tui --alt-screen=false --enable-module db
```

Then evaluated:

```js
const db = require("db"); typeof db
```

The TUI returned:

```text
promise rejected: GoError: Invalid modulepromise rejected: GoError: Invalid module
```

This reproduced the bug. The middleware treated `db` as an exact module name, but the default database module was only registered as `database`, so `--enable-module db` selected no usable database module and `require("db")` failed.

## Fix

- Registered `modules/database` twice: canonical `database` and short alias `db`.
- Added middleware alias expansion in both directions: `database -> db` and `db -> database`.
- Made the database TypeScript declaration use `m.Name()` so the alias declaration name is correct if inspected.
- Added regression tests for `MiddlewareOnly("db")` and `runScriptFile(... EnableModules: []string{"db"})` with `require("db")`.

## Verification

After the fix, reran tmux:

```bash
go run ./cmd/goja-repl tui --alt-screen=false --enable-module db
```

Evaluated:

```js
const db = require("db"); typeof db.query
```

The TUI returned:

```text
"function"
```

Also verified `typeof require("database").query` returned `"function"` in the same TUI session.
