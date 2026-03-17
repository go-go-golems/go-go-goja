function listIssues(repo, filters, meta) {
  return [
    {
      repo,
      state: filters.state,
      labelCount: (filters.labels || []).length,
      rootDir: meta.rootDir
    }
  ];
}

__verb__("listIssues", {
  short: "Use a host-registered shared section",
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
