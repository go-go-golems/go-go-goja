__section__("db", {
  title: "Database",
  description: "This section is intentionally defined in another file",
  fields: {
    db: {
      help: "SQLite database path",
      default: ":memory:"
    }
  }
});

exports.driver = "sqlite3";
