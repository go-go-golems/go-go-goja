__package__({ name: "agentauth", short: "Programmatic agent auth client demo" });

__verb__("callReport", {
  name: "call-report",
  output: "text",
  short: "Read the bootstrap API token and call the agent-protected report route",
  fields: {
    options: { bind: "all" },
    baseUrl: {
      help: "Base URL of the demo server, for example http://127.0.0.1:18789",
      default: "http://127.0.0.1:18789"
    },
    tokenFile: {
      help: "Path containing the bootstrap token metadata written by the server verb",
      default: "/tmp/xgoja-agent-auth-demo-token.json"
    },
    reportId: {
      help: "Report id to read",
      default: "daily"
    }
  }
});

async function callReport(options) {
  const fetch = require("fetch");
  const baseUrl = trimRight(options.baseUrl || "http://127.0.0.1:18789", "/");
  const tokenFile = options.tokenFile || "/tmp/xgoja-agent-auth-demo-token.json";
  const reportId = encodeURIComponent(options.reportId || "daily");

  const client = fetch.client()
    .baseUrl(baseUrl)
    .auth(fetch.auth.bearer().fromFile(tokenFile).jsonPath("token.value"))
    .acceptJson()
    .expectJson();

  const report = await client.get(`/agent/reports/${reportId}`).run();

  let sessionOnlyStatus = 0;
  try {
    await client.get("/session-only").run();
  } catch (err) {
    sessionOnlyStatus = err.status || 0;
  }

  return JSON.stringify({
    ok: report.ok,
    reportId: report.report.id,
    reportTitle: report.report.title,
    authMethod: report.auth.method,
    principalKind: report.auth.principalKind,
    principalId: report.auth.principalId,
    credentialHint: report.auth.credentialHint,
    sessionOnlyStatus
  }, null, 2);
}

function trimRight(value, suffix) {
  while (value.endsWith(suffix)) {
    value = value.slice(0, -suffix.length);
  }
  return value;
}
