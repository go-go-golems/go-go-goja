function add(a, b) {
  return { sum: a + b };
}

__verb__("add", {
  short: "Add two integers",
  fields: {
    a: { type: "int", argument: true },
    b: { type: "int", argument: true }
  }
});

const multiply = async (a, b) => {
  return { product: a * b };
};

__verb__("multiply", {
  short: "Multiply asynchronously",
  fields: {
    a: { type: "int", argument: true },
    b: { type: "int", argument: true }
  }
});

function listNames(...names) {
  return names.map((name, index) => ({ index, name }));
}

__verb__("listNames", {
  short: "Expand a list argument",
  fields: {
    names: { type: "stringList", argument: true }
  }
});
