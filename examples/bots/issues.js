__section__("filters", {
  title: "Filters",
  description: "Issue filtering flags",
  fields: {
    state: {
      type: "choice",
      choices: ["open", "closed"],
      default: "open",
      help: "Issue state"
    },
    labels: {
      type: "stringList",
      help: "Labels to include"
    }
  }
});

function list(repo, filters, meta) {
  return [{
    repo,
    state: filters.state,
    labelCount: (filters.labels || []).length,
    sourceFile: meta.sourceFile,
    rootDir: meta.rootDir
  }];
}

__verb__("list", {
  short: "Demonstrate bound sections and bound context",
  sections: ["filters"],
  fields: {
    repo: {
      argument: true,
      help: "Repository slug"
    },
    filters: {
      bind: "filters"
    },
    meta: {
      bind: "context"
    }
  }
});
