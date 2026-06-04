__package__({ name: "pinocchio" });

function profileSmoke(sessionId) {
  const gp = require("geppetto");
  const settings = gp.inferenceProfiles.resolve();
  const snapshot = settings.toJSON();
  const agent = gp.agent()
    .name("xgoja-pinocchio-profile-smoke")
    .inference(settings)
    .build();
  const session = agent.session().id(sessionId).build();

  return {
    source: "pinocchio/examples/js/runner-profile-smoke.js",
    migration: "pure-geppetto-session-construction",
    profile: snapshot.provenance?.profileSlug || "",
    registry: snapshot.provenance?.registrySlug || "",
    model: snapshot.chat?.engine || "",
    apiType: snapshot.chat?.api_type || "",
    session: session.id(),
    hasSessionNext: typeof session.next === "function",
  };
}

__verb__("profileSmoke", {
  name: "profile-smoke",
  short: "Port of Pinocchio runner-profile-smoke.js using pure Geppetto APIs",
  fields: {
    sessionId: {
      argument: true,
      default: "xgoja-pinocchio-profile-smoke",
      help: "Session ID used to prove the agent can build a session"
    }
  }
});

function profileDemo(sessionId, prompt) {
  const gp = require("geppetto");
  const settings = gp.inferenceProfiles.resolve();
  const snapshot = settings.toJSON();
  const agent = gp.agent()
    .name("xgoja-pinocchio-profile-demo")
    .inference(settings)
    .build();
  const session = agent.session().id(sessionId).build();

  const result = session.next()
    .system("Answer in one short sentence.")
    .user(prompt)
    .run({ timeoutMs: 120000 });

  return {
    source: "pinocchio/examples/js/runner-profile-demo.js",
    migration: "pure-geppetto-live-inference",
    profile: snapshot.provenance?.profileSlug || "",
    registry: snapshot.provenance?.registrySlug || "",
    model: snapshot.chat?.engine || "",
    apiType: snapshot.chat?.api_type || "",
    session: session.id(),
    text: String(result.text() || "").trim(),
  };
}

__verb__("profileDemo", {
  name: "profile-demo",
  short: "Port of Pinocchio runner-profile-demo.js using pure Geppetto APIs",
  fields: {
    sessionId: {
      argument: true,
      default: "xgoja-pinocchio-profile-demo",
      help: "Session ID"
    },
    prompt: {
      type: "string",
      default: "Say hello in one short sentence.",
      help: "User prompt for the live inference run"
    }
  }
});
