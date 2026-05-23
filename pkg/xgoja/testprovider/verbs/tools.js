__package__({ name: "tools" })
__verb__("providerGreet", {
  name: "provider-greet",
  output: "text",
  fields: {
    name: { type: "string", required: true }
  }
})
function providerGreet(name) {
  const hello = require("hello")
  return hello.greet(name)
}

__verb__("ownerPing", {
  name: "owner-ping",
  output: "text"
})
function ownerPing() {
  const owner = require("owner-check")
  return owner.pingAsync()
}
