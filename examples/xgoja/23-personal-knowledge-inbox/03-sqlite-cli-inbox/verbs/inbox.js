__package__({
  name: "inbox",
  short: "Personal Knowledge Inbox tutorial commands"
});

__section__("storage", {
  title: "Storage",
  description: "Shared SQLite database settings used by inbox commands",
  fields: {
    db: {
      type: "string",
      default: "personal-inbox.sqlite",
      help: "SQLite database path"
    }
  }
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
      res.json({ ok: true, step: "03-sqlite-cli-inbox" });
    });

  return "personal inbox hello web server routes registered\n";
}

__verb__("capture", {
  name: "capture",
  output: "text",
  short: "Capture a URL or note into the local SQLite inbox",
  sections: ["storage"],
  fields: {
    title: {
      type: "string",
      required: true,
      help: "Capture title"
    },
    url: {
      type: "string",
      required: true,
      help: "URL to capture"
    },
    note: {
      type: "string",
      default: "",
      help: "Optional note"
    },
    source: {
      type: "string",
      default: "cli",
      help: "Capture source label"
    },
    storage: {
      bind: "storage"
    }
  }
});

function capture(title, url, note, source, storage) {
  const database = openInbox(storage.db);
  try {
    const item = insertInboxItem(database, {
      title,
      url,
      note: note || "",
      source: source || "cli"
    });
    return JSON.stringify({ ok: true, item }, null, 2);
  } finally {
    database.close();
  }
}

__verb__("list", {
  name: "list",
  output: "text",
  short: "List local SQLite inbox items",
  sections: ["storage"],
  fields: {
    includeArchived: {
      type: "bool",
      default: false,
      help: "Include archived items"
    },
    storage: {
      bind: "storage"
    }
  }
});

function list(includeArchived, storage) {
  const database = openInbox(storage.db);
  try {
    return JSON.stringify({
      ok: true,
      items: listInboxItems(database, !!includeArchived)
    }, null, 2);
  } finally {
    database.close();
  }
}

__verb__("archive", {
  name: "archive",
  output: "text",
  short: "Archive one local SQLite inbox item",
  sections: ["storage"],
  fields: {
    id: {
      type: "string",
      required: true,
      help: "Inbox item id to archive"
    },
    storage: {
      bind: "storage"
    }
  }
});

function archive(id, storage) {
  const database = openInbox(storage.db);
  try {
    const archivedAt = new Date().toISOString();
    database.exec(
      "update inbox_items set archived_at = ? where id = ? and archived_at = ''",
      archivedAt,
      id
    );
    return JSON.stringify({ ok: true, id, archivedAt }, null, 2);
  } finally {
    database.close();
  }
}

function openInbox(path) {
  const database = require("database");
  database.configure("sqlite3", path || "personal-inbox.sqlite");
  database.exec(`
    create table if not exists inbox_items (
      id text primary key,
      title text not null,
      url text not null default '',
      note text not null default '',
      source text not null,
      submitted_by_kind text not null,
      submitted_by_id text not null,
      created_at text not null,
      archived_at text not null default ''
    )
  `);
  return database;
}

function insertInboxItem(database, input) {
  const item = {
    id: newItemID(),
    title: String(input.title || "Untitled capture"),
    url: String(input.url || ""),
    note: String(input.note || ""),
    source: String(input.source || "cli"),
    submittedByKind: "localCli",
    submittedById: "local-user",
    createdAt: new Date().toISOString(),
    archivedAt: ""
  };

  database.exec(
    `insert into inbox_items
      (id, title, url, note, source, submitted_by_kind, submitted_by_id, created_at, archived_at)
     values (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
    item.id,
    item.title,
    item.url,
    item.note,
    item.source,
    item.submittedByKind,
    item.submittedById,
    item.createdAt,
    item.archivedAt
  );

  return item;
}

function listInboxItems(database, includeArchived) {
  const where = includeArchived ? "" : "where archived_at = ''";
  return database.query(`
    select id,
           title,
           url,
           note,
           source,
           submitted_by_kind as submittedByKind,
           submitted_by_id as submittedById,
           created_at as createdAt,
           archived_at as archivedAt
      from inbox_items
      ${where}
     order by created_at desc, id desc
  `);
}

function newItemID() {
  return `item_${Date.now()}_${Math.random().toString(16).slice(2)}`;
}
