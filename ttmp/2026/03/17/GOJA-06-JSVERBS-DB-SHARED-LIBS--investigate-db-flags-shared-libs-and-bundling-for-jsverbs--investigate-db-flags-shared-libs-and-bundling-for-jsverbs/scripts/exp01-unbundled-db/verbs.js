__package__({
  name: "db-demo",
  short: "Unbundled jsverbs database demo"
});

__section__("db", {
  title: "Database",
  description: "Connection flags shared by db-backed verbs in this file",
  fields: {
    db: {
      help: "SQLite database path",
      default: ":memory:"
    }
  }
});

function listUsers(prefix, dbConfig, meta) {
  const database = require("database");
  const sql = require("./lib/sql");

  database.configure("sqlite3", dbConfig.db);
  database.exec(sql.createUsersTable);
  database.exec(sql.resetUsers);
  database.exec(sql.insertUser, "Ada");
  database.exec(sql.insertUser, "Alan");
  database.exec(sql.insertUser, "Grace");

  const rows = database.query(sql.selectPrefix, prefix + "%");
  database.close();

  return rows.map((row) => ({
    id: row.id,
    name: row.name,
    db: dbConfig.db,
    rootDir: meta.rootDir
  }));
}

__verb__("listUsers", {
  short: "List seeded users using a --db flag and the native database module",
  sections: ["db"],
  fields: {
    prefix: {
      argument: true,
      help: "Name prefix to filter on"
    },
    dbConfig: {
      bind: "db"
    },
    meta: {
      bind: "context"
    }
  }
});
