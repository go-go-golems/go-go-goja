const multiply = async (a, b) => {
  return { product: a * b };
};

__verb__("multiply", {
  short: "Multiply asynchronously in the dedicated botcli fixture",
  fields: {
    a: { type: "int", argument: true },
    b: { type: "int", argument: true }
  }
});
