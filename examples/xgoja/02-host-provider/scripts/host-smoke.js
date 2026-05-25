const fs = require("fs")
const exec = require("exec")
const database = require("database")

const file = "host-provider-smoke.txt"
fs.writeFileSync(file, "host provider ok", "utf8")
if (fs.readFileSync(file, "utf8") !== "host provider ok") {
  throw new Error("fs read/write failed")
}
fs.unlinkSync(file)

const out = exec.run("echo", ["exec-ok"]).trim()
if (out !== "exec-ok") {
  throw new Error("exec allow-list failed: " + out)
}

database.configure("sqlite3", ":memory:")
database.exec("create table smoke (name text)")
database.exec("insert into smoke (name) values (?)", "database-ok")
const rows = database.query("select name from smoke")
if (!rows || rows.length !== 1 || rows[0].name !== "database-ok") {
  throw new Error("database smoke failed")
}
database.close()

console.log("host provider ok")
