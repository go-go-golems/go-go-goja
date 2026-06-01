__section__("filters", {
  title: "Filters",
  description: "Shared filter flags",
  fields: {
    state: {
      type: "choice",
      choices: ["open", "closed"],
      default: "open",
      help: "Issue state"
    },
    labels: {
      type: "stringList",
      help: "Labels to filter on"
    }
  }
});

doc`---
verb: greet
---
Greets one person and optionally adds an exclamation mark.`;

function greet(name, excited) {
  return {
    greeting: excited ? `Hello, ${name}!` : `Hello, ${name}`
  };
}

__verb__("greet", {
  short: "Greet one person",
  fields: {
    name: {
      argument: true,
      help: "Person name"
    },
    excited: {
      type: "bool",
      short: "e",
      help: "Add excitement"
    }
  }
});

const echo = (value) => value;

__verb__("echo", {
  short: "Return a primitive value",
  fields: {
    value: {
      argument: true
    }
  }
});

function banner(name) {
  return `=== ${name} ===\n`;
}

__verb__("banner", {
  short: "Write plain text output",
  output: "text",
  fields: {
    name: {
      argument: true
    }
  }
});

function listIssues(repo, filters, meta) {
  const helper = require("./support/helper");
  return [
    {
      repo,
      state: filters.state,
      labelCount: (filters.labels || []).length,
      helper: helper.decorate(repo),
      rootDir: meta.rootDir
    }
  ];
}

__verb__("listIssues", {
  short: "Use shared sections and context",
  sections: ["filters"],
  fields: {
    repo: {
      argument: true,
      help: "Repository name"
    },
    filters: {
      bind: "filters"
    },
    meta: {
      bind: "context"
    }
  }
});

function summarize(options) {
  return {
    owner: options.owner,
    repo: options.repo,
    joined: options.owner + "/" + options.repo
  };
}

__verb__("summarize", {
  short: "Bind all parsed values into one object",
  fields: {
    options: {
      bind: "all"
    },
    owner: {
      help: "Repository owner"
    },
    repo: {
      help: "Repository name"
    }
  }
});
