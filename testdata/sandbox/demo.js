const { defineBot } = require("sandbox")

module.exports = defineBot(({ command, event, configure }) => {
  configure({ name: "demo-bot", purpose: "sandbox-smoke-test" })

  command("ping", (ctx) => {
    const current = ctx.store.get("count", 0)
    ctx.store.set("count", current + 1)
    ctx.reply(`pong:${current}`)
    return current
  })

  event("ready", (ctx) => {
    ctx.store.set("ready", true)
    return "ready"
  })
})
