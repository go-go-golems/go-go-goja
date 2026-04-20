function echoAll(options) {
  return {
    repo: options.repo,
    dryRun: options.dryRun,
    count: (options.names || []).length,
    names: options.names || []
  };
}

__verb__("echoAll", {
  short: "Demonstrate bind: all",
  fields: {
    options: {
      bind: "all"
    },
    repo: {
      help: "Repository name"
    },
    dryRun: {
      type: "bool",
      help: "Whether this is a dry run"
    },
    names: {
      type: "stringList",
      help: "Names to echo"
    }
  }
});
