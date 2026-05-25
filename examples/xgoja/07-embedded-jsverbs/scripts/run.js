const hello = require("hello")
const label = "embedded-jsverbs".replace(/-jsverbs$/, "").replace("runtime-filesystem", "filesystem").replace("provider-shipped", "provider")
if (hello.greet(label) !== "hello " + label) {
  throw new Error("unexpected greeting")
}
