const { configure, query, exec, close } = require('database');

console.log("database module loaded");

// Configure for an in-memory SQLite database
const err = configure('sqlite3', ':memory:');
if (err) {
    console.error("Failed to configure database:", err);
    throw new Error("DB configuration failed");
}
console.log("database configured for in-memory sqlite3");

// Create a table
let result, err = exec("CREATE TABLE users (id INT, name TEXT)");
if (err) {
    console.error("Failed to create table:", err);
    throw new Error("CREATE TABLE failed");
}
console.log("CREATE TABLE successful:", JSON.stringify(result));

// Insert data
result, err = exec("INSERT INTO users (id, name) VALUES (?, ?)", 1, "John Doe");
if (err) {
    console.error("Failed to insert data:", err);
    throw new Error("INSERT failed");
}
console.log("INSERT successful:", JSON.stringify(result));

// Query data
let rows, qErr = query("SELECT * FROM users");
if (qErr) {
    console.error("Failed to query data:", qErr);
    throw new Error("SELECT failed");
}

if (rows.length !== 1 || rows[0].name !== "John Doe") {
    console.error("Query result mismatch:", JSON.stringify(rows));
    throw new Error("Query result validation failed");
}
console.log("SELECT successful:", JSON.stringify(rows));

// Close the database connection
close();

console.log("database test successful");
console.log("OK"); 