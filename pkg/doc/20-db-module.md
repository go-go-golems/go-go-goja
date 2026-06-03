---
Title: Database Module
Slug: db-module
Short: Run SQL queries from JavaScript against sqlite3 or any Go database/sql driver
Topics:
- database
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

The `database` (and `db`) modules expose a minimal SQL helper surface for Goja runtimes. They wrap Go's `database/sql` connection pool so JavaScript handlers can query and execute statements without leaving the event-driven flow.

## Go setup

The database module is part of the default registry, so it is available automatically when you build an engine with the default modules:

```go
factory, err := engine.NewBuilder().Build()
```

For production uses, pre-configure the database connection from Go so the runtime does not need to call `configure()`:

```go
db, _ := sql.Open("sqlite3", ":memory:")
module := databasemod.New(
    databasemod.WithPreconfiguredDB(db),
    databasemod.WithCloseFn(db.Close),
    databasemod.WithName("database"),
)
factory, err := engine.NewBuilder().WithModules(module).Build()
```

When you pre-configure the module, `configure()` is disabled and calling it throws.

## JavaScript usage

```javascript
const db = require("database");

db.configure("sqlite3", ":memory:");

db.exec(`
  CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT
  )
`);

try {
  db.exec("INSERT INTO users (name) VALUES (?)", "Ada");
} catch (e) {
  console.error("insert failed:", e);
}

const rows = db.query("SELECT * FROM users WHERE name = ?", "Ada");
for (const row of rows) {
  console.log(row.id, row.name);
}
```

## Module API

### `configure(driverName, dataSourceName)`

Opens a new `database/sql` connection using the named driver and DSN. Only available when the module was not pre-configured from Go. Throws when called on a pre-configured module.

### `query(sql, ...args)`

Executes a `SELECT`-style statement and returns an array of plain objects, one per row. Column names become object keys. `args` are passed as positional query parameters.

If `args` contains a single array, that array is flattened and used as positional parameters. This makes it convenient to spread an existing list.

### `exec(sql, ...args)`

Executes an `INSERT`, `UPDATE`, `DELETE`, or DDL statement and returns a result summary on success:

```javascript
{
  success: true,
  rowsAffected: 1,
  lastInsertId: 3
}
```

If the statement fails, `exec` **throws an exception** with the error message. The underlying Go function returns both a result object and a non-nil error, and Goja raises the error as a JavaScript exception. Use `try/catch` to handle failed SQL executions.

### `close()`

Closes the underlying connection if the module owns it. Safe to call multiple times. Silently does nothing when there is no owned connection.

## Module aliases

Both `require("database")` and `require("db")` resolve to the same module instance.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| "database not configured" error | No `configure()` call and no pre-configured DB | Call `configure()` before queries, or pre-configure the module in Go |
| "preconfigured and does not allow configure" error | Go side used `WithPreconfiguredDB` | Remove the `configure()` call from JavaScript |
| Driver import missing at runtime | The Go binary was not linked against the driver | Add the driver import to your Go main package (for example, `_ "github.com/mattn/go-sqlite3"`) |
| Query parameters do not match | Wrong number of `?` placeholders | Ensure placeholder count equals the number of bound arguments |
