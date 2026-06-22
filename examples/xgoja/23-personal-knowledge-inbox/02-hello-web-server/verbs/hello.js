__package__({
  name: "inbox",
  short: "Personal Knowledge Inbox tutorial commands"
});

__verb__("hello", {
  name: "hello",
  output: "text",
  short: "Say hello from the Personal Knowledge Inbox command line",
  fields: {
    name: {
      type: "string",
      default: "world",
      help: "Name to greet"
    }
  }
});

function hello(name) {
  return `Hello, ${name || "world"}! This is the Personal Knowledge Inbox tutorial.`;
}

__verb__("server", {
  name: "server",
  output: "text",
  short: "Register a public hello-world web server"
});

function server() {
  const express = require("express");
  const app = express.app();

  app.get("/")
    .public()
    .audit("inbox.hello.view")
    .handle((_ctx, res) => {
      res.send("Hello from the Personal Knowledge Inbox web server.");
    });

  app.get("/healthz")
    .public()
    .audit("inbox.health")
    .handle((_ctx, res) => {
      res.json({ ok: true, step: "02-hello-web-server" });
    });

  return "personal inbox hello web server routes registered\n";
}
