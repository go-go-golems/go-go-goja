__package__({
  name: "builtin",
  parents: ["examples"],
  short: "Built-in goja smoke-test verbs",
});

function renderSampleTable() {
  const ui = require("ui.dsl");
  return ui.render(ui.table.fromRows("sample", [
    { name: "Alice", role: "admin" },
    { name: "Bob", role: "viewer" },
  ]).features(f => f.pagination().sorting()).render({}));
}

__verb__("renderSampleTable", {
  short: "Render a sample HTML table with ui.dsl",
  outputMode: "text",
});
