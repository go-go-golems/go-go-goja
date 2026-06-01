const { configure, query, exec, close } = require('database');

console.log("database module loaded");

// Configure for an in-memory SQLite database.
configure('sqlite3', ':memory:');
console.log("database configured for in-memory sqlite3");

// Create a table.
let result = exec("CREATE TABLE users (id INT, name TEXT)");
console.log("CREATE TABLE successful:", JSON.stringify(result));

// Insert data.
result = exec("INSERT INTO users (id, name) VALUES (?, ?)", 1, "John Doe");
console.log("INSERT successful:", JSON.stringify(result));

// Query data.
const rows = query("SELECT * FROM users");

if (rows.length !== 1 || rows[0].name !== "John Doe") {
    console.error("Query result mismatch:", JSON.stringify(rows));
    throw new Error("Query result validation failed");
}
console.log("SELECT successful:", JSON.stringify(rows));

// Close the database connection
close();

console.log("database test successful");
console.log("OK"); 