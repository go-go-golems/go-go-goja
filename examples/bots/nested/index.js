const helper = require("./sub/helper");

function relay(prefix, target) {
  return { value: helper.render(prefix, target) };
}

__verb__("relay", {
  short: "Exercise a relative require from a nested bot example",
  fields: {
    prefix: { argument: true },
    target: { argument: true }
  }
});
