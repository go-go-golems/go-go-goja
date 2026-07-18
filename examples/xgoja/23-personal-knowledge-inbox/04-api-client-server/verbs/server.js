__package__({
  name: "inbox",
  short: "Personal Knowledge Inbox server commands"
});

__section__("storage", {
  title: "Storage",
  description: "SQLite database settings used by the inbox server",
  fields: {
    db: {
      type: "string",
      default: "personal-inbox.sqlite",
      help: "SQLite database path"
    }
  }
});

__verb__("server", {
  name: "server",
  output: "text",
  short: "Register the public inbox API server",
  sections: ["storage"],
  fields: {
    storage: {
      bind: "storage"
    }
  }
});

function server(storage) {
  const express = require("express");
  const store = require("./lib/inbox_store");
  const app = express.app();
  const dbPath = storage.db || "personal-inbox.sqlite";

  app.get("/")
    .public()
    .audit("inbox.hello.view")
    .handle((_ctx, res) => {
      res.send("Hello from the Personal Knowledge Inbox API server.");
    });

  app.get("/healthz")
    .public()
    .audit("inbox.health")
    .handle((_ctx, res) => {
      res.json({ ok: true, step: "04-api-client-server" });
    });

  app.get("/api/inbox")
    .public()
    .audit("inbox.api.list")
    .handle((_ctx, res) => {
      const database = store.openInbox(dbPath);
      try {
        res.json({ ok: true, items: store.listInboxItems(database, false) });
      } finally {
        database.close();
      }
    });

  app.post("/api/capture")
    .public()
    .audit("inbox.api.capture")
    .handle((ctx, res) => {
      const body = ctx.body || {};
      if (!body.title || !body.url) {
        res.status(400).json({ error: "title and url are required" });
        return;
      }
      const database = store.openInbox(dbPath);
      try {
        const item = store.insertInboxItem(database, {
          title: body.title,
          url: body.url,
          note: body.note || "",
          source: body.source || "api",
          submittedByKind: "publicApi",
          submittedById: "anonymous"
        });
        res.status(201).json({ ok: true, item });
      } finally {
        database.close();
      }
    });

  app.post("/api/inbox/:id/archive")
    .public()
    .audit("inbox.api.archive")
    .handle((ctx, res) => {
      const database = store.openInbox(dbPath);
      try {
        const archived = store.archiveInboxItem(database, ctx.params.id);
        res.json({ ok: true, id: archived.id, archivedAt: archived.archivedAt });
      } finally {
        database.close();
      }
    });

  return `personal inbox API server routes registered; db: ${dbPath}\n`;
}
