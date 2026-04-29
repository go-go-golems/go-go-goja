const fs = require("fs");
const path = require("path");
fs.writeFileSync("/tmp/goja-test.txt", "hello from goja");
console.log("written:", fs.readFileSync("/tmp/goja-test.txt"));
console.log("path.join:", path.join("a", "b"));
