---
Title: "Guarded fetch client API"
Slug: guarded-fetch-client-api
Short: "Call HTTP APIs from generated xgoja JavaScript with explicit host policy and Go-owned credential sources."
Topics:
- xgoja
- fetch
- http-client
- auth
- javascript
- agents
Commands:
- xgoja
- xgoja build
- xgoja doctor
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Application
---

The `fetch` host module gives generated xgoja JavaScript a guarded outbound HTTP client. It is meant for agents, smoke tests, and controlled integrations that need to call APIs from embedded goja code without shelling out to `curl` or bringing a Node HTTP library.

Use this page when writing client-side JavaScript that calls an authenticated xgoja route, configuring outbound host policy, or reviewing the programmatic agent example.

## Why fetch is guarded

Outbound HTTP is a host capability. A generated binary should not be able to call arbitrary networks unless its `xgoja.yaml` says so. The host provider therefore requires explicit `allow: true` and supports origin allow-lists, timeouts, response-size limits, and credential-source policy.

```yaml
providers:
  - id: go-go-goja-host
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/host
    register: Register
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

Production agents should narrow both `allowedOrigins` and credential source permissions. A local smoke may allow `http://127.0.0.1:*`; a deployed agent should use exact origins and exact credential file paths where possible.

## Configuration fields

| Field | Meaning |
| --- | --- |
| `allow` | Required boolean gate for enabling outbound HTTP. |
| `allowedOrigins` | Optional list of allowed origins such as `https://api.example.test` or `http://127.0.0.1:*`. Empty means no origin restriction. |
| `timeout` | Default request timeout as a Go duration. Defaults to 30 seconds. |
| `maxResponseBytes` | Maximum buffered response body size. Defaults to 4 MiB. |
| `credentials.allowEnv` | Allows `fetch.auth.bearer().fromEnv(...)`. |
| `credentials.allowFiles` | Allows `fetch.auth.bearer().fromFile(...)`. |
| `credentials.allowedFiles` | Optional exact file allow-list for credential files. |

The first implementation buffers response bodies for `text()` and `json()`. It does not implement browser streaming, CORS, service workers, cache modes, or a complete WHATWG `Request`/`Response` object model.

## Low-level fetch

Use `fetch.fetch(...)` when you want a small browser-like API and explicit options.

```javascript
const fetch = require("fetch")

const res = await fetch.fetch("https://api.example.test/healthz", {
  method: "GET",
  headers: { "Accept": "application/json" },
  timeout: "2s"
})

if (!res.ok) {
  throw new Error(`health check failed: ${res.status}`)
}

const body = await res.json()
```

Supported options:

```typescript
interface FetchOptions {
  method?: string
  headers?: Record<string, string>
  body?: string | Uint8Array
  json?: unknown
  timeout?: string
}
```

The response exposes:

```typescript
interface FetchResponse {
  url: string
  status: number
  statusText: string
  ok: boolean
  headers: Record<string, string[]>
  text(): Promise<string>
  json(): Promise<unknown>
}
```

## Fluent client

Use `fetch.client()` for application and agent code. It centralizes base URL handling, default headers, expected response shape, and credential injection.

```javascript
const client = fetch.client()
  .baseUrl("https://api.example.test")
  .timeout("5s")
  .acceptJson()
  .expectJson()

const report = await client.get("/reports/daily")
  .query("format", "summary")
  .run()
```

Request builders are created with `.get(...)`, `.post(...)`, `.put(...)`, `.patch(...)`, `.delete(...)`, or `.request(method, path)`. Each request can add query parameters, headers, JSON bodies, raw bodies, and per-request response expectations.

```javascript
const created = await client.post("/reports")
  .json({ id: "weekly", title: "Weekly report" })
  .expectJson()
  .run()
```

Expectation methods control `run()` output:

| Method | `run()` result |
| --- | --- |
| `expectJson()` | Parsed JSON body; non-2xx responses reject with status metadata. |
| `expectText()` | Text body; non-2xx responses reject with status metadata. |
| `expectResponse()` | Response object for manual status/body handling. |

## Bearer credential sources

Use `fetch.auth` builders instead of manually concatenating `Authorization` headers for framework-owned credentials.

```javascript
const client = fetch.client()
  .baseUrl(baseUrl)
  .auth(fetch.auth.bearer().fromFile(tokenFile).jsonPath("token.value"))
  .acceptJson()
  .expectJson()
```

Supported credential sources:

```javascript
fetch.auth.none()
fetch.auth.bearer().token("ggpat_...")
fetch.auth.bearer().fromEnv("API_TOKEN")
fetch.auth.bearer().fromFile("/run/secrets/api-token.json").jsonPath("token.value")
```

Credential source builders are Go-owned objects. The client rejects plain JavaScript auth maps for sensitive input so policy checks and redaction stay in Go.

## Agent-to-server example

The programmatic-agent example uses two generated binaries:

- server: provisions an automation agent, issues an API token, and exposes an `express.agent()` route;
- agent: reads the token file with `fetch.auth.bearer().fromFile(...).jsonPath(...)` and calls the protected route with `fetch.client()`.

Run the full smoke:

```bash
make -C examples/xgoja/22-programmatic-agent-auth smoke
```

Run manually in two terminals after `make -C examples/xgoja/22-programmatic-agent-auth build`:

```bash
# terminal 1
examples/xgoja/22-programmatic-agent-auth/dist/programmatic-agent-auth-server \
  serve agentauth server \
  --http-listen 127.0.0.1:18789 \
  --token-file /tmp/xgoja-agent-auth-demo-token.json
```

```bash
# terminal 2
examples/xgoja/22-programmatic-agent-auth/dist/programmatic-agent-auth-agent \
  verbs agentauth call-report \
  --base-url http://127.0.0.1:18789 \
  --token-file /tmp/xgoja-agent-auth-demo-token.json \
  --report-id daily
```

Expected output includes:

```json
{
  "ok": true,
  "reportId": "daily",
  "authMethod": "apiToken",
  "principalKind": "agent",
  "sessionOnlyStatus": 403
}
```

The smoke harness uses `curl` only as an external black-box assertion tool. The JavaScript agent itself uses the `fetch` module, not `exec` or `curl`.

## Security notes

- Keep `allowedOrigins` as narrow as the deployment allows.
- Prefer `credentials.allowedFiles` for production agents that read token files.
- Do not log token files or issuance results containing `token.value`.
- Prefer `fetch.auth.bearer().fromFile(...).jsonPath(...)` or `fromEnv(...)` over passing raw tokens through many JavaScript variables.
- Use `expectResponse()` when a caller must handle expected non-2xx statuses, such as a deliberate `403` check.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `require("fetch")` fails | The host provider fetch module is not selected. | Add `go-go-goja-host` and `runtime.modules[].name: fetch`. |
| Fetch module refuses to load | `allow: true` is missing. | Enable the module explicitly in `xgoja.yaml`. |
| Request fails with target origin not allowed | `allowedOrigins` does not match the URL scheme, host, and port. | Add an exact origin or a local pattern such as `http://127.0.0.1:*` for smoke tests. |
| `fromFile(...)` is rejected | File credential sources are disabled or the file is not allow-listed. | Set `credentials.allowFiles: true` and, if used, include the exact path in `credentials.allowedFiles`. |
| `fromEnv(...)` is rejected | Env credential sources are disabled. | Set `credentials.allowEnv: true`. |
| `expectJson()` rejects a known error response | Non-2xx statuses reject in JSON/text expectation modes. | Use `expectResponse()` when the caller intentionally inspects error statuses. |

## See also

- `xgoja help programmatic-auth-javascript-apis`
- `xgoja help express-route-auth-requirements`
- `xgoja help http-serve-command-reference`
- `xgoja help provider-runtime-config-and-host-services`
- `examples/xgoja/22-programmatic-agent-auth`
