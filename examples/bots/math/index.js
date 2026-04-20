const multiply = async (a, b) => {
  return { product: a * b };
};

__verb__("multiply", {
  short: "Multiply two integers asynchronously",
  fields: {
    a: { type: "int", argument: true },
    b: { type: "int", argument: true }
  }
});

function leaderboard(...names) {
  return names.map((name, index) => ({
    rank: index + 1,
    name,
    shout: name.toUpperCase()
  }));
}

__verb__("leaderboard", {
  short: "Expand a positional string list into ranked rows",
  fields: {
    names: {
      type: "stringList",
      argument: true,
      help: "Names to rank"
    }
  }
});
