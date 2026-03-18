const shared = require("./shared");

__package__({
  name: "bundled-db",
  short: "Bundled jsverbs demo"
});

__section__("db", {
  title: "Database",
  description: "Database flags kept in the bundled entry file",
  fields: {
    db: {
      help: "SQLite database path",
      default: ":memory:"
    }
  }
});

function countUsers(prefix, dbConfig) {
  const database = require("database");
  const dbPath = shared.normalizeDbPath(dbConfig.db);

  database.configure("sqlite3", dbPath);
  shared.seed(database);

  const rows = database.query(
    "SELECT id, name FROM users WHERE name LIKE ? ORDER BY id",
    prefix + "%"
  );
  database.close();

  return rows.map((row) => ({
    id: row.id,
    name: row.name,
    db: dbPath
  }));
}

exports.countUsers = countUsers;

__verb__("countUsers", {
  short: "Count users from a bundled CommonJS jsverb",
  sections: ["db"],
  fields: {
    prefix: {
      argument: true,
      help: "Name prefix to filter on"
    },
    dbConfig: {
      bind: "db"
    }
  }
});
