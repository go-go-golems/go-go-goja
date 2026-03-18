exports.normalizeDbPath = function(dbPath) {
  return String(dbPath || ":memory:").trim() || ":memory:";
};

exports.seed = function(database) {
  database.exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL)");
  database.exec("DELETE FROM users");
  database.exec("INSERT INTO users (name) VALUES (?)", "Margaret");
  database.exec("INSERT INTO users (name) VALUES (?)", "Marvin");
  database.exec("INSERT INTO users (name) VALUES (?)", "Manual");
};
