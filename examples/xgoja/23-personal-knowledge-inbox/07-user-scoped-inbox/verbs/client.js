__package__({
  name: "inboxctl",
  short: "Personal Knowledge Inbox direct database commands"
});

__section__("storage", {
  title: "Storage",
  description: "Shared SQLite database settings used by direct inbox commands",
  fields: {
    db: {
      type: "string",
      default: "personal-inbox.sqlite",
      help: "SQLite database path"
    }
  }
});

__verb__("capture", {
  name: "capture",
  output: "text",
  short: "Capture a URL or note directly into the local SQLite inbox",
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
      default: "cli-direct",
      help: "Capture source label"
    },
    storage: {
      bind: "storage"
    }
  }
});

function capture(title, url, note, source, storage) {
  const store = require("./lib/inbox_store");
  const database = store.openInbox(storage.db);
  try {
    const item = store.insertInboxItem(database, {
      title,
      url,
      note: note || "",
      source: source || "cli-direct",
      submittedByKind: "localCli",
      submittedById: "direct-user"
    });
    return JSON.stringify({ ok: true, item }, null, 2);
  } finally {
    database.close();
  }
}

__verb__("list", {
  name: "list",
  output: "text",
  short: "List inbox items directly from SQLite",
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
  const store = require("./lib/inbox_store");
  const database = store.openInbox(storage.db);
  try {
    return JSON.stringify({
      ok: true,
      items: store.listInboxItems(database, !!includeArchived)
    }, null, 2);
  } finally {
    database.close();
  }
}

__verb__("archive", {
  name: "archive",
  output: "text",
  short: "Archive one local SQLite inbox item directly",
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
  const store = require("./lib/inbox_store");
  const database = store.openInbox(storage.db);
  try {
    return JSON.stringify({ ok: true, ...store.archiveInboxItem(database, id) }, null, 2);
  } finally {
    database.close();
  }
}
