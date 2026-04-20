const relay = (prefix, target) => {
  const helper = require("./sub/helper");
  return { value: helper.render(prefix, target) };
};

__verb__("relay", {
  short: "Use a relative require from the dedicated botcli fixture",
  fields: {
    prefix: { argument: true },
    target: { argument: true }
  }
});
