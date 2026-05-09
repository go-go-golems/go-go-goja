__package__({
  name: "builtin",
  parents: ["examples"],
  short: "Built-in goja smoke-test verbs",
});

function yamlKeys(text) {
  const yaml = require("yaml");
  const value = yaml.parse(text || "{}");
  return Object.keys(value || {}).map(key => ({ key }));
}

__verb__("yamlKeys", {
  short: "Parse YAML text and emit top-level keys",
  fields: {
    text: { type: "string", required: true, help: "YAML text to parse" },
  },
});
