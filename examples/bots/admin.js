__package__({
  name: "ops",
  parents: ["meta"],
  short: "Operations-focused example package"
});

function status() {
  return {
    ok: true,
    scope: "meta ops",
    uptimeHint: "simulated"
  };
}

__verb__("status", {
  short: "Demonstrate package metadata affecting the final bot path"
});
