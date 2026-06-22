__package__({
  name: "inboxctl",
  short: "Personal Knowledge Inbox API client commands"
});

__section__("api", {
  title: "API",
  description: "Shared API settings used by inbox client commands",
  fields: {
    baseUrl: {
      type: "string",
      default: "http://127.0.0.1:18792",
      help: "Base URL of the inbox API server"
    }
  }
});

__verb__("capture", {
  name: "capture",
  output: "text",
  short: "Capture a URL or note through the inbox API",
  sections: ["api"],
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
      default: "cli-api",
      help: "Capture source label"
    },
    api: {
      bind: "api"
    }
  }
});

async function capture(title, url, note, source, api) {
  const client = require("./lib/api_client");
  const result = await client.capture(api.baseUrl, {
    title,
    url,
    note: note || "",
    source: source || "cli-api"
  });
  return JSON.stringify(result, null, 2);
}

__verb__("list", {
  name: "list",
  output: "text",
  short: "List inbox items through the API",
  sections: ["api"],
  fields: {
    api: {
      bind: "api"
    }
  }
});

async function list(api) {
  const client = require("./lib/api_client");
  const result = await client.list(api.baseUrl);
  return JSON.stringify(result, null, 2);
}

__verb__("archive", {
  name: "archive",
  output: "text",
  short: "Archive one inbox item through the API",
  sections: ["api"],
  fields: {
    id: {
      type: "string",
      required: true,
      help: "Inbox item id to archive"
    },
    api: {
      bind: "api"
    }
  }
});

async function archive(id, api) {
  const client = require("./lib/api_client");
  const result = await client.archive(api.baseUrl, id);
  return JSON.stringify(result, null, 2);
}
