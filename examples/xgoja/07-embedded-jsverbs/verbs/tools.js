__package__({ name: "tools" })
__verb__("embeddedGreet", {
  name: "embedded-greet",
  output: "text",
  fields: {
    name: { type: "string", required: true }
  }
})
function embeddedGreet(name) {
  const hello = require("hello")
  return hello.greet(name)
}
