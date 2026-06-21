# xgoja programmatic agent auth example

This example demonstrates both sides of programmatic API access:

1. a generated xgoja HTTP server that provisions an automation agent, issues a one-time API token, and exposes an agent-only route; and
2. a generated xgoja jsverb agent that reads the issued token through the framework-native `fetch.auth` credential builder and calls the protected route through `fetch.client()`.

The JavaScript agent does **not** use `exec` or `curl`. Outbound HTTP is performed by the guarded host `fetch` module.

## Run

```bash
make -C examples/xgoja/22-programmatic-agent-auth smoke
```

The smoke test builds two generated binaries: `dist/programmatic-agent-auth-server` and `dist/programmatic-agent-auth-agent`. It starts the server on a random localhost port, waits for `/healthz`, verifies the agent route rejects unauthenticated requests, runs the agent jsverb, and verifies the same API token is rejected by a session-user-only route.

## Server-side pattern

The server verb provisions a bot and writes the raw token value once to a local bootstrap file:

```js
const issued = auth.agents.create("daily-report-bot")
  .kind("ci")
  .tenantId("o1")
  .createdBy("server-bootstrap")
  .grants(auth.grants().allow("user.self.read").done())
  .issueApiToken("daily-report-token")
  .run();

fs.writeFileSync(tokenFile, JSON.stringify({ agent: issued.agent, token: issued.token }, null, 2), "utf8");
```

It then exposes an agent-only route:

```js
app.get("/agent/reports/:reportId")
  .auth(express.agent())
  .rateLimit(express.rateLimit("agent-report-read").perMinute(60).byActor().byRoute())
  .allow("user.self.read")
  .audit("agent.report.read")
  .handle((ctx, res) => res.json({ report, actor: ctx.actor, auth: ctx.auth }));
```

## Agent-side pattern

The agent uses the fluent fetch client and a Go-owned bearer credential source:

```js
const client = fetch.client()
  .baseUrl(baseUrl)
  .auth(fetch.auth.bearer().fromFile(tokenFile).jsonPath("token.value"))
  .acceptJson()
  .expectJson();

const report = await client.get(`/agent/reports/${reportId}`).run();
```

The important design point is that the canonical client example does not manually concatenate `Authorization` headers and does not shell out to a process. Credential source parsing and header injection are part of the guarded fetch module.

## xgoja configuration

The server and agent use separate xgoja specs so the agent runtime does not need the server-side `auth` module. The agent spec enables only the host capability it needs:

```yaml
runtime:
  modules:
    - provider: go-go-goja-host
      name: fetch
      as: fetch
      config:
        allow: true
        allowedOrigins:
          - http://127.0.0.1:*
        timeout: 5s
        maxResponseBytes: 1048576
        credentials:
          allowFiles: true
```

Production agents should narrow `allowedOrigins` and `credentials.allowedFiles` to exact values.
