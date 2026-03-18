exports.createUsersTable = "CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL)";
exports.resetUsers = "DELETE FROM users";
exports.insertUser = "INSERT INTO users (name) VALUES (?)";
exports.selectPrefix = "SELECT id, name FROM users WHERE name LIKE ? ORDER BY id";
