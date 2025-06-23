// hello.js – simple sanity-check for the go-go-goja runtime
//
// Expected console output ends with the line "OK".
// The script exercises:
//   1. fs.readFileSync / writeFileSync
//   2. exec.run

const fs = require("fs");
const exec = require("exec");

fs.writeFileSync("/tmp/goja-test.txt", "Hi from test");
const content = fs.readFileSync("/tmp/goja-test.txt");
if (content !== "Hi from test") {
  throw new Error("fs module failed");
}

const out = exec.run("echo", ["yo"]).trim();
if (out !== "yo") {
  throw new Error("exec module failed");
}

console.log("OK"); 