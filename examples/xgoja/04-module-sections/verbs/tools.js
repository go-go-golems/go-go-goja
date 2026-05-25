__package__({ name: "tools" })

__verb__("checkFixture", {
  name: "check-fixture",
  output: "text"
})
function checkFixture() {
  if (globalThis.fixtureValue !== "verb-ok") {
    throw new Error("fixtureValue=" + globalThis.fixtureValue)
  }
  const hello = require("hello")
  return hello.greet("verb")
}
