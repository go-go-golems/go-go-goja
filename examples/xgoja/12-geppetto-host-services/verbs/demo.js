function run(sessionId) {
  const gp = require("geppetto");

  const toolResult = gp.toolRegistry()
    .addGo("wordCount")
    .call("wordCount", { text: "generated xgoja host services" });

  const settings = gp.inferenceProfiles.resolve();
  const store = gp.turnStores.default();

  const agent = gp.agent()
    .name("generated-host-services-smoke")
    .inference(settings)
    .goMiddleware("addSystemPrompt", { prompt: "Answer with exactly the word: hosted" })
    .defaultStore()
    .build();

  const session = agent.session()
    .id(sessionId)
    .defaultStore()
    .metadata("demo", "host-services")
    .build();

  const result = session.next().user("Say hosted.").run();
  const latest = store.loadLatest({ sessionId, phase: "final" });
  const blocks = latest && latest.turn ? latest.turn.toJSON().blocks : [];
  const textsFor = function(roleOrKind) {
    return blocks
      .filter(block => block.role === roleOrKind || block.kind === roleOrKind)
      .map(block => block.payload && block.payload.text)
      .filter(Boolean)
      .join("\n");
  };

  return {
    sessionId,
    toolCount: toolResult.count,
    text: result.text(),
    listed: store.list({ sessionId, phase: "final" }).length,
    latestText: textsFor("assistant") || textsFor("llm_text"),
    systemText: textsFor("system"),
  };
}

__verb__("run", {
  short: "Run a Geppetto host-services smoke",
  fields: {
    sessionId: { argument: true, help: "Session ID" }
  }
});
