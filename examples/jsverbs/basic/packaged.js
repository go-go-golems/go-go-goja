__package__({
  name: "pkg-demo",
  parents: ["meta"],
  short: "Package metadata demo"
});

function ping() {
  return { ok: true };
}

__verb__("ping", {
  short: "Ping package metadata demo"
});
