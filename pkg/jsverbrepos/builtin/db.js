__package__({
  name: "builtin",
  parents: ["examples"],
  short: "Built-in goja smoke-test verbs",
});

function tables() {
  const db = require("db");
  return db.query(`
    SELECT name
    FROM sqlite_schema
    WHERE type = 'table'
      AND name NOT LIKE 'sqlite_%'
    ORDER BY name
  `);
}

__verb__("tables", {
  short: "List user tables from the configured SQLite database",
});
