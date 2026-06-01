const render = (prefix, target) => {
  const helper = require("./sub/helper");
  return { value: helper.render(prefix, target) };
};

__verb__("render", {
  short: "Use a relative require",
  fields: {
    prefix: { argument: true },
    target: { argument: true }
  }
});
