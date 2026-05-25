if (globalThis.fixtureValue !== "run-ok") {
  throw new Error("fixtureValue=" + globalThis.fixtureValue)
}
const hello = require("hello")
if (hello.greet("module sections") !== "hello module sections") {
  throw new Error("unexpected hello module output")
}
