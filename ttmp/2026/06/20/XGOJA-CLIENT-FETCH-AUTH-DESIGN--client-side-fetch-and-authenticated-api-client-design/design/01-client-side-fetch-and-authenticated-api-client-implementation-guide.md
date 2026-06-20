---
Title: Client-side Fetch and Authenticated API Client Implementation Guide
Ticket: XGOJA-CLIENT-FETCH-AUTH-DESIGN
Status: active
Topics:
    - xgoja
    - fetch
    - auth
    - javascript-api
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: examples/xgoja/21-generated-host-auth/xgoja.yaml
      Note: Generated-host example structure to follow for future server+agent smoke example
    - Path: modules/express/auth_builders.go
      Note: Go-owned fluent builder-store pattern for security-sensitive route specs
    - Path: modules/fs/fs.go
      Note: Promise-returning module API and TypeScript declaration pattern
    - Path: modules/fs/fs_async.go
      Note: goja Promise plus runtimebridge async host-work pattern
    - Path: pkg/gojahttp/auth/programauth/token.go
      Note: API-token issuance/authentication and redaction model that client auth must respect
    - Path: pkg/jsverbs/runtime.go
      Note: Promise-returning jsverbs support needed by async fetch clients
    - Path: pkg/xgoja/providers/host/host.go
      Note: Guarded host provider module pattern and proposed fetch provider integration
    - Path: pkg/xgoja/providers/hostauth/programmatic.go
      Note: Programauth JavaScript builders and redacted token projection patterns
ExternalSources: []
Summary: Design and implementation guide for a guarded client-side fetch module and authenticated fluent API client for xgoja JavaScript agents.
LastUpdated: 2026-06-20T11:38:12.072209588-04:00
WhatFor: Use when implementing outbound HTTP/fetch support, client-side authentication builders, or server+agent smoke examples for programmatic auth.
WhenToUse: Before adding a fetch/http client module or writing examples that call agent-protected gojahttp routes.
---


# Client-side Fetch and Authenticated API Client Implementation Guide

## Executive summary

This ticket designs a first-class client-side HTTP API for xgoja JavaScript programs. The immediate motivation is the programmatic-auth example: we can already expose server routes that require agent API-token authentication, but the agent side has no good JavaScript-native way to call those routes. Shelling out to `curl` would prove that HTTP works, but it would document the wrong pattern and bypass the framework principles we have been building: host-owned security boundaries, explicit configuration, fluent builders, redacted credentials, and deterministic smoke tests.

The proposed solution is a guarded `fetch` provider module with two layers:

1. A low-level `fetch.fetch(url, options)` function that behaves like a practical subset of browser `fetch` and returns Promise-backed response objects.
2. A fluent `fetch.client()` API for framework-native agents and scripts. The client owns base URL normalization, timeout policy, JSON handling, bearer credential injection, redaction, and request execution.

Authentication support should be general enough to carry bearer tokens or future credential sources, but it should be opinionated for our framework. In particular, it should integrate naturally with existing `auth.agents.create(...).issueApiToken(...).run()` output and server routes declared with `express.agent()` / `express.sessionUser()` / `express.anyOf(...)`.

The first implementation should not try to support every browser `fetch` feature. It should implement the minimum safe and useful client for xgoja jsverbs, generated hosts, and smoke tests:

- guarded module registration in `go-go-goja-host`, because outbound network access is a host capability;
- allow-listing for target origins/hosts;
- request timeouts and response-size limits;
- Promise-returning low-level fetch;
- response helpers for `text()`, `json()`, and headers;
- a fluent client builder with `.baseUrl(...)`, `.bearerToken(...)`, `.bearerFromFile(...).jsonPath(...)`, `.expectJson()`, and request builders such as `.get(...).run()`;
- TypeScript declarations;
- examples and smoke tests that demonstrate both server and agent sides without `exec` or `curl`.

## Problem statement and scope

### Problem

The server-side programmatic-auth work now supports:

- explicit automation agents;
- API tokens issued by host-owned services;
- bearer-token authentication into `AuthResult`;
- route-level principal restrictions such as agent-only routes;
- JavaScript builders for provisioning agents and tokens.

What is missing is the other half of the story: a JavaScript agent that can acquire or read a credential and call an agent-protected API route using a framework-native HTTP client.

Without a client module, examples are forced into bad alternatives:

- shell out to `curl` through `exec`, which is explicitly not the desired pattern;
- ask users to bring their own Node runtime/client library, which is not available in embedded goja;
- implement ad hoc Go smoke tests only, which does not document real JavaScript usage;
- pass raw `Authorization` header strings everywhere, which conflicts with the Go-owned credential-handling direction.

### Scope

In scope:

- a new guarded xgoja provider module named `fetch` or `httpClient`;
- low-level fetch semantics for common HTTP requests;
- a fluent authenticated client builder aligned with current API style;
- bearer-token support suitable for programauth API tokens;
- credential-source primitives for raw tokens, environment variables, and local files;
- JSON request/response helpers;
- examples that show server and agent jsverbs;
- smoke tests that run close to real usage.

Out of scope for the first implementation:

- browser-complete `Request`, `Response`, `Headers`, `AbortController`, streaming bodies, service workers, redirects policy compatibility, cache modes, CORS, credentials modes, and multipart/form-data;
- arbitrary OAuth/OIDC interactive browser flows on the client side;
- refresh-token rotation, device login polling, or token exchange; those are future auth credential sources once their server-side primitives exist;
- retry/backoff policy beyond a simple future extension point;
- automatic global `fetch` injection unless explicitly configured.

## Current-state architecture and evidence

### xgoja host provider owns dangerous host capabilities

The host provider currently exposes modules that can touch the filesystem, process table, or databases only when enabled explicitly in `xgoja.yaml`. The provider registers `fs`, `node:fs`, `exec`, `database`, and `db` in `pkg/xgoja/providers/host/host.go:57-67`. The comment at `pkg/xgoja/providers/host/host.go:57-59` states the pattern directly: guarded host-capability modules require explicit per-module config.

The filesystem module config illustrates how this is done. `fsModule` exposes a JSON schema with `allow: true` for host filesystem access or `embedded.allow: true` for read-only embedded assets at `pkg/xgoja/providers/host/host.go:70-99`. At runtime the module factory rejects missing authorization and only constructs an OS-backed filesystem when `cfg.Allow` is true (`pkg/xgoja/providers/host/host.go:113-126`).

The `exec` module is also guarded. It requires `allow: true` and optionally restricts exact command names (`pkg/xgoja/providers/host/host.go:161-174`). It then checks the allow-list before calling `exec.Command` (`pkg/xgoja/providers/host/host.go:190-200`). This proves the existing pattern, but it also proves why client HTTP should not be implemented as `exec curl`: the framework already treats process execution as a broad and risky host capability. A fetch module can be much narrower, auditable, and easier to test.

### Existing modules already use Promise-returning async APIs

The filesystem module exposes Promise-based async methods in its TypeScript declaration. `modules/fs/fs.go:55-66` declares `readFile`, `writeFile`, `exists`, `mkdir`, `readdir`, `stat`, and other operations as returning `Promise<...>`. The implementation requires runtime services (`modules/fs/fs.go:115-120`) and then binds async methods such as `readFile` to `asyncReadFile` (`modules/fs/fs.go:131-135`).

The async helper pattern is in `modules/fs/fs_async.go:11-39`: create a goja promise, run blocking work in a goroutine, and post resolution or rejection back through `runtimebridge`. Fetch should follow this pattern because network I/O is blocking host work and must not run on the goja runtime goroutine.

### jsverbs can return Promises

The jsverbs runtime already handles Promise-returning verbs. It checks whether the JavaScript function result exports as a `*goja.Promise` at `pkg/jsverbs/runtime.go:90-100`, and waits for Promise completion at `pkg/jsverbs/runtime.go:106-108`. The polling helper at `pkg/jsverbs/runtime.go:242-278` is described as simple but sufficient for the current prototype.

This matters because an agent jsverb should be allowed to write normal async JavaScript:

```javascript
async function callReport(options) {
  const client = fetch.client().baseUrl(options.baseUrl).bearerFromFile(options.tokenFile).jsonPath("token.value").expectJson();
  return await client.get("/agent/reports/daily").run();
}
```

### Server-side route builders already use Go-owned fluent objects

Express route auth builders store Go-owned specs behind JavaScript builder objects. `modules/express/auth_builders.go:13-17` keeps separate maps for auth specs, resource specs, and rate-limit specs. `express.agent()` creates a `SecuritySpec` with `PrincipalKindAgent` at `modules/express/auth_builders.go:26-29`; `express.sessionUser()` creates a session user requirement at `modules/express/auth_builders.go:31-34`; `express.anyOf(...)` combines existing builder-owned auth specs at `modules/express/auth_builders.go:36-56`. The `.auth(...)` call rejects arbitrary objects that are not known builder outputs (`modules/express/auth_builders.go:76-89`).

The fetch client should use the same design language. Security-sensitive configuration such as bearer credential sources should be represented by Go-owned builders, not arbitrary JavaScript object bags.

### Programmatic auth already exposes server-side provisioning builders

The hostauth provider exposes programmatic auth builders in `pkg/xgoja/providers/hostauth/programmatic.go`. `newProgrammaticExports` creates `auth.grants`, `auth.agents`, and `auth.tokens` exports (`pkg/xgoja/providers/hostauth/programmatic.go:32-56`). Grant builders store typed Go state behind objects and reject arbitrary values (`pkg/xgoja/providers/hostauth/programmatic.go:59-120`). Agent creation can issue an API token as part of the builder chain (`pkg/xgoja/providers/hostauth/programmatic.go:123-193`). Token issue/list/revoke builders return redacted projections except for the one-time raw token value at issuance (`pkg/xgoja/providers/hostauth/programmatic.go:197-318`).

This creates the server-side provisioning story. The client-side fetch API should consume that output without normalizing users toward unsafe manual header construction.

### API-token storage and authentication are redaction-sensitive

`APIToken` stores `TokenHash`, `TokenPrefix`, and lifecycle metadata; the comment at `pkg/gojahttp/auth/programauth/token.go:50-51` states that `TokenHash` must not be returned by JavaScript/list APIs. `APITokenView` is the list/detail-safe projection with no raw token or hash (`pkg/gojahttp/auth/programauth/token.go:81-97`). `IssuedAPIToken` includes the raw value only at issuance (`pkg/gojahttp/auth/programauth/token.go:109-112`).

Authentication hashes the raw bearer token, finds candidates by prefix, and uses constant-time comparison (`pkg/gojahttp/auth/programauth/token.go:190-235`). Successful API-token auth returns `AuthResult` with method `apiToken`, principal kind `agent`, a credential hint, grants, scopes, and `CSRFRequired: false` (`pkg/gojahttp/auth/programauth/token.go:223-233`).

The fetch client cannot make the raw token disappear once an agent process has it, but it can keep the recommended path narrow: read from a configured source, inject into the request in Go-owned client code, and redact it from errors/logs.

### Generated hostauth services already bundle programauth services

Generated hostauth services now include `AgentStore`, `APITokenStore`, `Agents`, and `APITokens` fields (`pkg/xgoja/hostauth/services.go:45-69`). The builder constructs in-memory programauth stores and services (`pkg/xgoja/hostauth/builder.go:86-92`) and wires a composite bearer/session authenticator into `gojahttp.AuthOptions` (`pkg/xgoja/hostauth/builder.go:235-243`).

This is sufficient for a generated server example to provision an agent and token when the service starts, then expose agent-only routes. The missing piece is the generated agent/client side.

### Route-level agent restrictions exist

`AuthRequirement` constrains route entry by authentication method and/or principal kind (`pkg/gojahttp/auth_plan.go:75-83`). It is part of `SecuritySpec` (`pkg/gojahttp/auth_plan.go:85-90`). `ValidateRoutePlan` rejects auth requirements on public routes and normalizes requirements on authenticated routes (`pkg/gojahttp/auth_plan.go:262-280`, `pkg/gojahttp/auth_plan.go:305-340`). The enforcer checks requirements immediately after authentication and before CSRF/resource/authorization work (`pkg/gojahttp/enforcer.go:80-103`, `pkg/gojahttp/enforcer.go:245-257`).

This means the target server-side route can be a real agent-only route:

```javascript
app.get("/agent/reports/:reportId")
  .auth(express.agent())
  .allow("report.read")
  .audit("agent.report.read")
  .handle((ctx, res) => res.json({ reportId: ctx.params.reportId, auth: ctx.auth }));
```

## Gap analysis

### What works today

- JavaScript can declare protected server routes with `express.agent()` and related builders.
- JavaScript can provision agents and issue API tokens through `auth.agents.create(...).issueApiToken(...).run()`.
- Generated hostauth services authenticate bearer API tokens and skip CSRF for programmatic calls.
- jsverbs can run async code and wait for Promise completion.
- Host capability modules can be guarded by explicit config in `xgoja.yaml`.

### What is missing

- There is no built-in outbound HTTP client module for goja JavaScript.
- There is no framework-native way to attach an API token to outbound requests.
- There is no redaction policy for client-side bearer values in errors or logs.
- There is no host-level allow-list for outbound network destinations.
- There is no documented server+agent example that exercises programmatic auth end to end.
- There are no smoke tests that prove a generated agent can call a generated server using the JavaScript APIs.

### Why `exec curl` is not acceptable

Using `exec.run("curl", ...)` would conflict with the framework direction for several reasons:

- It requires enabling process execution, a broader host capability than outbound HTTP.
- It hides HTTP behavior from Go-level tests and policy controls.
- It makes token redaction depend on shell command strings and process listing behavior.
- It documents an escape hatch instead of the intended API.
- It cannot easily share future framework features such as device-flow polling, retries, token refresh, or structured error handling.

The example directory created during early exploration was therefore removed before this ticket was written. The new example should wait for the fetch module.

## Proposed architecture

### Module placement

Create a new reusable native module package:

```text
modules/fetch/
  fetch.go
  config.go
  async.go
  response.go
  client_builder.go
  auth_builder.go
  typescript.go
  fetch_test.go
  client_builder_test.go
```

Expose it through the guarded host provider:

```text
pkg/xgoja/providers/host/host.go
pkg/xgoja/providers/host/host_test.go
```

The module belongs under the host provider because outbound HTTP is a host/network capability. It should not be automatically available from the core provider.

### Conceptual diagram

```text
+-------------------------+        +--------------------------+
| agent jsverb            |        | server jsverb            |
|                         |        |                          |
| fetch.client()          |        | auth.agents.create()     |
|   .baseUrl(...)         |        |   .issueApiToken()       |
|   .auth(...)            |        |                          |
|   .get(...).run()       |        | express.agent() route    |
+-----------+-------------+        +-------------+------------+
            |                                    ^
            | HTTPS/HTTP with Authorization      |
            v                                    |
+-----------+-------------+        +-------------+------------+
| modules/fetch           |        | gojahttp.Enforcer        |
|                         |        |                          |
| origin allow-list       |        | BearerFromHeader         |
| timeout                 |        | APITokenService          |
| max body size           |        | AuthRequirement          |
| credential injection    |        | GrantSet                 |
+-----------+-------------+        +-------------+------------+
            |                                    ^
            v                                    |
+-----------+-------------+        +-------------+------------+
| net/http.Client         |        | net/http server/mux      |
+-------------------------+        +--------------------------+
```

### Runtime wiring

`xgoja.yaml` should opt into fetch explicitly:

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
          - http://127.0.0.1:*      # development/smoke only
          - https://api.example.com  # production exact origin
        timeout: 5s
        maxResponseBytes: 1048576
        credentials:
          allowFiles: true
          allowedFiles:
            - /tmp/xgoja-agent-auth-demo-token.json
```

The exact schema can be refined during implementation, but the important contract is explicit host permission plus destination restriction.

## Public API proposal

### Low-level API

```javascript
const fetch = require("fetch");

const response = await fetch.fetch("https://api.example.test/v1/status", {
  method: "GET",
  headers: {
    "Accept": "application/json"
  }
});

if (!response.ok) {
  throw new Error(`request failed: ${response.status}`);
}

const payload = await response.json();
```

Low-level `fetch.fetch(url, options)` should support:

- `method?: string`, default `GET`;
- `headers?: Record<string,string>`;
- `body?: string | Uint8Array | object`;
- `json?: unknown`, a convenience that JSON-encodes and sets `Content-Type: application/json`;
- `timeout?: string`, optional per-request override bounded by module config;
- response fields: `url`, `status`, `statusText`, `ok`, `headers`;
- response methods: `text()`, `json()`, `arrayBuffer()` optional later.

### Fluent client API

```javascript
const fetch = require("fetch");

const client = fetch.client()
  .baseUrl("http://127.0.0.1:18789")
  .timeout("5s")
  .auth(fetch.auth.bearer().fromFile("/tmp/agent-token.json").jsonPath("token.value"))
  .acceptJson()
  .expectJson();

const report = await client
  .get("/agent/reports/daily")
  .header("X-Request-ID", "demo-1")
  .run();
```

This is the preferred pattern for agent scripts. It keeps request construction declarative and lets Go own execution, header injection, and error shaping.

### Convenience credential API

Credential builders should be Go-owned objects. Do not accept arbitrary credential maps for security-sensitive inputs.

```javascript
fetch.auth.bearer().token(rawToken)
fetch.auth.bearer().fromEnv("REPORT_AGENT_TOKEN")
fetch.auth.bearer().fromFile("/tmp/token.json").jsonPath("token.value")
fetch.auth.none()
```

The first implementation can support `token`, `fromEnv`, and `fromFile(...).jsonPath(...)`. Future implementations can add:

```javascript
fetch.auth.device().provider("...").clientId("...").scopes(...).run()
fetch.auth.refreshingBearer().tokenFile("...").refreshWith(...)
```

### Request builder API

```javascript
await client.get(path).run();
await client.post(path).json(body).run();
await client.patch(path).json(body).run();
await client.delete(path).run();

const response = await client
  .post("/raw")
  .body("plain text")
  .expectResponse()
  .run();
```

Request builder methods:

- `.query(name, value)` for query parameters;
- `.header(name, value)` for non-sensitive headers;
- `.json(value)` for JSON request body;
- `.body(value)` for raw string/bytes;
- `.expectJson()` override;
- `.expectText()` override;
- `.expectResponse()` escape hatch.

### Error API

For non-2xx responses in `.expectJson()` / `.expectText()` mode, throw a structured error:

```javascript
try {
  await client.get("/agent/reports/missing").run();
} catch (err) {
  console.error(err.name);       // "HTTPError"
  console.error(err.status);     // 404
  console.error(err.body);       // redacted/truncated response text
  console.error(err.url);        // safe URL, no Authorization header
}
```

The low-level `fetch.fetch` should not throw on HTTP status by default, matching browser fetch. The fluent client may throw on non-2xx when `expectJson()` or `expectText()` is configured because that is better for command-style jsverbs.

## Pseudocode

### Provider registration

```go
// pkg/xgoja/providers/host/host.go
func Register(registry *providerapi.ProviderRegistry) error {
    return registry.Package(PackageID,
        fsModule("fs"),
        fsModule("node:fs"),
        fetchModule("fetch"),
        execModule(),
        databaseModule("database"),
        databaseModule("db"),
    )
}

func fetchModule(name string) providerapi.Module {
    return providerapi.Module{
        Name: name,
        DefaultAs: name,
        Description: "Guarded outbound HTTP client module.",
        TypeScript: fetchmod.New(fetchmod.WithName(name)).TypeScriptModule(),
        ConfigSchema: fetchConfigSchema,
        NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
            cfg := FetchConfig{}
            decodeConfig(ctx.Config, &cfg)
            if !cfg.Allow { return nil, fmt.Errorf("fetch module requires config.allow=true") }
            policy := fetchmod.PolicyFromConfig(cfg)
            return fetchmod.New(fetchmod.WithName(requireName(ctx)), fetchmod.WithPolicy(policy)).Loader, nil
        },
    }
}
```

### Async fetch execution

```go
func (m Module) fetch(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
    promise, resolve, reject := vm.NewPromise()
    reqSpec := parseRequest(vm, call)
    callCtx := runtimebridge.CurrentOwnerContext(vm)
    runtimeServices := runtimebridge.MustLookup(vm)

    go func() {
        result, err := m.execute(callCtx, reqSpec)
        if err != nil {
            runtimeServices.PostWithCustomContext(callCtx, "fetch.reject", func(context.Context, *goja.Runtime) {
                reject(fetchErrorValue(vm, err))
            })
            return
        }
        runtimeServices.PostWithCustomContext(callCtx, "fetch.resolve", func(context.Context, *goja.Runtime) {
            resolve(responseObject(vm, result))
        })
    }()

    return vm.ToValue(promise)
}
```

### Policy check

```go
func (p Policy) CheckURL(raw string) (*url.URL, error) {
    u, err := url.Parse(raw)
    if err != nil { return nil, err }
    if u.Scheme != "http" && u.Scheme != "https" {
        return nil, fmt.Errorf("fetch only supports http and https")
    }
    if !p.AllowsOrigin(u) {
        return nil, fmt.Errorf("fetch target origin is not allowed")
    }
    return u, nil
}
```

### Credential resolution

```go
type CredentialSource interface {
    Apply(ctx context.Context, req *http.Request) error
    RedactedDescription() string
}

type BearerFromFile struct {
    Path string
    JSONPath string
}

func (s BearerFromFile) Apply(ctx context.Context, req *http.Request) error {
    data := os.ReadFile(s.Path)
    token := extractJSONPath(data, s.JSONPath)
    req.Header.Set("Authorization", "Bearer "+token)
    return nil
}
```

The JavaScript object returned by `fetch.auth.bearer().fromFile(...).jsonPath(...)` should store this Go `CredentialSource` in a builder store, similar to Express `authSpecs` and hostauth `grantObjectStore`.

### Fluent client execution

```go
func (c *ClientState) newRequestBuilder(method, path string) *goja.Object {
    req := RequestSpec{Method: method, URL: c.resolve(path), Headers: clone(c.defaultHeaders)}
    obj := vm.NewObject()
    obj.Set("header", func(name, value string) *goja.Object { req.Headers[name] = value; return obj })
    obj.Set("json", func(value goja.Value) *goja.Object { req.JSON = value.Export(); return obj })
    obj.Set("run", func() goja.Value {
        return asyncExecute(vm, c.policy, req, c.credential, c.expectation)
    })
    return obj
}
```

## Decision records

### Decision: implement fetch as a guarded host provider module

- **Context:** Outbound network access can exfiltrate secrets or contact internal services. Existing host modules use explicit per-module config for filesystem/process/database access.
- **Options considered:** Core provider module always available; host provider guarded module; application-specific Go-only helper.
- **Decision:** Add fetch to the host provider as a guarded module.
- **Rationale:** This follows the evidence-backed pattern in `pkg/xgoja/providers/host/host.go` and allows origin allow-lists, timeouts, and response limits.
- **Consequences:** xgoja specs must opt in explicitly. This is slightly more verbose but safer and easier to audit.
- **Status:** proposed

### Decision: provide low-level fetch and high-level fluent client

- **Context:** Developers expect browser-like `fetch`, but framework examples need safer authenticated client behavior.
- **Options considered:** Only browser-like fetch; only fluent client; both layers.
- **Decision:** Provide both `fetch.fetch(...)` and `fetch.client()`.
- **Rationale:** The low-level function is familiar and flexible. The client builder captures our API principles and becomes the recommended path for authenticated automation.
- **Consequences:** More implementation surface, but clearer separation between escape hatch and opinionated framework usage.
- **Status:** proposed

### Decision: use Go-owned credential builders, not JavaScript object bags

- **Context:** Bearer values and future device/refresh credentials are security-sensitive. Existing Express and hostauth builders reject arbitrary JavaScript maps.
- **Options considered:** `{ auth: { type: "bearer", token } }` option bags; direct `headers.Authorization`; Go-owned credential builder objects.
- **Decision:** Add credential builders such as `fetch.auth.bearer().fromFile(...).jsonPath(...)`.
- **Rationale:** This preserves typed state on the Go side and makes redaction/validation possible.
- **Consequences:** The API is more explicit. Tests must ensure `.auth(...)` rejects unrecognized objects.
- **Status:** proposed

### Decision: start with bearer credentials and extensible sources

- **Context:** Programmatic API tokens are already implemented; device flow and refresh tokens are not yet complete.
- **Options considered:** General auth plugin system immediately; only raw bearer token; bearer builder with sources.
- **Decision:** Implement bearer credential source builders first: raw token, environment variable, and file JSON path.
- **Rationale:** This supports immediate programauth examples while leaving room for future device/refresh sources.
- **Consequences:** Some future auth mechanisms will require additional builders, but the request/client architecture will remain stable.
- **Status:** proposed

### Decision: response bodies are buffered with size limits in v1

- **Context:** Streaming is complex in goja and unnecessary for the first smoke/documentation examples.
- **Options considered:** Streaming response bodies; full buffering with max bytes; text-only response.
- **Decision:** Buffer response bodies up to a configured `maxResponseBytes` limit.
- **Rationale:** It is simple, testable, and compatible with `json()` / `text()` helpers.
- **Consequences:** Not suitable for large downloads; add streaming later if needed.
- **Status:** proposed

## Implementation plan

### Phase 1: Create `modules/fetch` low-level fetch

Files:

- `modules/fetch/fetch.go`
- `modules/fetch/async.go`
- `modules/fetch/response.go`
- `modules/fetch/typescript.go`
- `modules/fetch/fetch_test.go`

Tasks:

1. Define `Module`, `Option`, and `Policy` types.
2. Implement `fetch.fetch(url, options)` with Promise-returning async execution.
3. Parse method, headers, `body`, and `json` options.
4. Use `net/http.Client` with timeout.
5. Buffer response body with `io.LimitReader` and return an error if truncated.
6. Build response object with `status`, `ok`, `headers`, `text()`, and `json()`.
7. Add TypeScript declarations.
8. Add tests using `httptest.Server`.

Acceptance criteria:

- low-level fetch can GET JSON from an allowed local server;
- disallowed origin fails before making the request;
- response `json()` parses JSON;
- non-2xx response resolves normally for low-level fetch;
- timeout and max body limit are tested.

### Phase 2: Add guarded host provider integration

Files:

- `pkg/xgoja/providers/host/host.go`
- `pkg/xgoja/providers/host/host_test.go`

Tasks:

1. Add `FetchConfig` with `allow`, `allowedOrigins`, `allowedHosts`, `timeout`, `maxResponseBytes`, and credential file settings.
2. Register `fetchModule("fetch")` in the host provider.
3. Decode config and reject missing `allow: true`.
4. Convert config into `modules/fetch.Policy`.
5. Add provider tests for missing allow, allowed host, and rejected host.

Acceptance criteria:

- `require("fetch")` fails without explicit config;
- configured module can call an allowed `httptest.Server`;
- blocked targets produce safe errors.

### Phase 3: Add fluent client builder

Files:

- `modules/fetch/client_builder.go`
- `modules/fetch/client_builder_test.go`

Tasks:

1. Implement `fetch.client()` returning a Go-owned client builder.
2. Add `.baseUrl(...)`, `.timeout(...)`, `.acceptJson()`, `.expectJson()`, `.expectText()`, `.expectResponse()`.
3. Add `.get`, `.post`, `.put`, `.patch`, `.delete`, and `.request` methods.
4. Add request builder methods `.query`, `.header`, `.json`, `.body`, and `.run`.
5. Make `.run()` return a Promise.
6. In expected JSON/text modes, throw structured `HTTPError` on non-2xx responses.

Acceptance criteria:

- `await client.get("/path").run()` returns parsed JSON when expected;
- request JSON body is encoded and content-type is set;
- non-2xx JSON response throws structured error;
- path resolution prevents accidental base URL loss.

### Phase 4: Add client auth builders

Files:

- `modules/fetch/auth_builder.go`
- `modules/fetch/auth_builder_test.go`

Tasks:

1. Add `fetch.auth` namespace.
2. Implement `fetch.auth.none()` and `fetch.auth.bearer()`.
3. Add `.token(value)`, `.fromEnv(name)`, `.fromFile(path)`, and `.jsonPath(path)`.
4. Store credential sources in Go-owned builder state.
5. Add `.auth(credentialSpec)` to `fetch.client()`.
6. Redact bearer values from errors and debug output.
7. Validate credential files against module policy.

Acceptance criteria:

- `.auth(...)` rejects arbitrary JavaScript objects;
- bearer token source injects `Authorization: Bearer ...`;
- file source reads the existing programauth issuance JSON shape (`token.value`);
- error strings do not contain raw token values.

### Phase 5: Add end-to-end example and smoke tests

Files:

- `examples/xgoja/22-programmatic-agent-auth/xgoja.yaml`
- `examples/xgoja/22-programmatic-agent-auth/verbs/server.js`
- `examples/xgoja/22-programmatic-agent-auth/verbs/agent.js`
- `examples/xgoja/22-programmatic-agent-auth/scripts/smoke.sh`
- `examples/xgoja/22-programmatic-agent-auth/README.md`
- `examples/xgoja/22-programmatic-agent-auth/Makefile`

Server jsverb:

```javascript
function server(options) {
  const express = require("express");
  const auth = require("auth");
  const fs = require("fs:host");
  const app = express.app();

  const issued = auth.agents.create("daily-report-bot")
    .kind("ci")
    .tenantId("o1")
    .createdBy("server-bootstrap")
    .grants(auth.grants().allow("report.read").done())
    .issueApiToken("daily-report-token")
    .run();

  fs.writeFileSync(options.tokenFile, JSON.stringify({ agent: issued.agent, token: issued.token }, null, 2), "utf8");

  app.get("/agent/reports/:reportId")
    .auth(express.agent())
    .allow("report.read")
    .audit("agent.report.read")
    .handle((ctx, res) => res.json({ reportId: ctx.params.reportId, auth: ctx.auth }));
}
```

Agent jsverb:

```javascript
async function callReport(options) {
  const fetch = require("fetch");
  const client = fetch.client()
    .baseUrl(options.baseUrl)
    .auth(fetch.auth.bearer().fromFile(options.tokenFile).jsonPath("token.value"))
    .acceptJson()
    .expectJson();

  return await client.get(`/agent/reports/${options.reportId || "daily"}`).run();
}
```

Smoke test:

1. Build generated binary.
2. Start `serve agentauth server --token-file <tmp>`.
3. Wait for `/healthz`.
4. Verify no-token call to `/agent/reports/daily` returns `401`.
5. Run `verbs agentauth call-report --base-url http://... --token-file <tmp>`.
6. Verify output contains `method: apiToken`, `principalKind: agent`, and expected report id.
7. Verify same token cannot access `.auth(express.sessionUser())` route and receives `403`.

### Phase 6: Documentation and generated DTS validation

Tasks:

1. Add README sections explaining low-level fetch vs fluent authenticated client.
2. Add generated DTS examples or TypeScript fixture if appropriate.
3. Add `xgoja doctor` coverage for the example.
4. Link this ticket from the programmatic auth design ticket after implementation.

## Suggested file-level API references

### `modules/fetch` public Go types

```go
type Policy struct {
    AllowedOrigins []OriginPattern
    Timeout time.Duration
    MaxResponseBytes int64
    Credentials CredentialPolicy
}

type CredentialPolicy struct {
    AllowEnv bool
    AllowFiles bool
    AllowedFiles []string
}

type Module struct {
    name string
    policy Policy
    client *http.Client
}
```

### JavaScript DTS sketch

```typescript
export function fetch(url: string, options?: FetchOptions): Promise<FetchResponse>;
export function client(): FetchClientBuilder;
export namespace auth {
  function none(): AuthSpec;
  function bearer(): BearerAuthBuilder;
}

export interface FetchOptions {
  method?: string;
  headers?: Record<string, string>;
  body?: string | Uint8Array;
  json?: unknown;
  timeout?: string;
}

export interface FetchResponse {
  url: string;
  status: number;
  statusText: string;
  ok: boolean;
  headers: Record<string, string[]>;
  text(): Promise<string>;
  json(): Promise<unknown>;
}

export interface FetchClientBuilder {
  baseUrl(url: string): FetchClientBuilder;
  timeout(duration: string): FetchClientBuilder;
  auth(spec: AuthSpec): FetchClientBuilder;
  acceptJson(): FetchClientBuilder;
  expectJson(): FetchClientBuilder;
  expectText(): FetchClientBuilder;
  expectResponse(): FetchClientBuilder;
  get(path: string): RequestBuilder;
  post(path: string): RequestBuilder;
  put(path: string): RequestBuilder;
  patch(path: string): RequestBuilder;
  delete(path: string): RequestBuilder;
  request(method: string, path: string): RequestBuilder;
}

export interface RequestBuilder {
  query(name: string, value: string | number | boolean): RequestBuilder;
  header(name: string, value: string): RequestBuilder;
  json(value: unknown): RequestBuilder;
  body(value: string | Uint8Array): RequestBuilder;
  expectJson(): RequestBuilder;
  expectText(): RequestBuilder;
  expectResponse(): RequestBuilder;
  run(): Promise<unknown>;
}

export interface BearerAuthBuilder extends AuthSpec {
  token(value: string): BearerAuthBuilder;
  fromEnv(name: string): BearerAuthBuilder;
  fromFile(path: string): BearerAuthBuilder;
  jsonPath(path: string): BearerAuthBuilder;
}

export interface AuthSpec {}
```

## Testing strategy

### Unit tests

- `modules/fetch/fetch_test.go`
  - GET request success;
  - POST JSON body;
  - response headers projection;
  - low-level non-2xx resolves;
  - invalid URL rejects;
  - disallowed origin rejects;
  - timeout rejects;
  - response-size limit rejects.

- `modules/fetch/client_builder_test.go`
  - base URL and relative path resolution;
  - query parameter encoding;
  - default headers;
  - JSON expectation returns parsed data;
  - text expectation returns text;
  - response expectation returns response object;
  - non-2xx structured error.

- `modules/fetch/auth_builder_test.go`
  - bearer token injection;
  - from-env source;
  - from-file JSON path source;
  - arbitrary object rejection;
  - redaction in error messages.

### Provider tests

- `pkg/xgoja/providers/host/host_test.go`
  - module unavailable without `allow: true`;
  - config schema and factory accept allowed origins;
  - rejected origins fail before network;
  - TypeScript module is present.

### Integration tests

- Express server with agent-only route and fetch client call in the same test process.
- Generated xgoja example smoke test that starts a server process and runs an agent verb process.
- Negative tests:
  - no Authorization header -> `401`;
  - API-token agent hits `express.sessionUser()` route -> `403`;
  - session user hits `express.agent()` route -> `403` if test harness can create session.

## Risks and mitigations

### Risk: accidental SSRF capability

Outbound HTTP from scripts can contact internal services. Mitigation: require `allow: true`, require origin/host allow-lists for generated examples and production docs, and reject unsupported schemes.

### Risk: token leakage in errors/logs

Bearer values can appear in headers, command output, or thrown errors. Mitigation: credential builders should redact values and fetch errors should never include request headers. Tests should assert raw token absence.

### Risk: browser-fetch compatibility expectations

A module named `fetch` may make users expect the full browser API. Mitigation: document it as a practical xgoja subset and expose `fetch.client()` as the recommended framework API.

### Risk: async bridge limitations

The current jsverbs Promise wait loop is explicitly simple. Mitigation: follow the existing module async pattern and add tests for jsverbs awaiting client calls.

### Risk: credential file access expands module scope

Reading token files from fetch mixes network and file capabilities. Mitigation: gate file credential sources separately with `credentials.allowFiles` and optional `allowedFiles`. Alternatively, support only `token()` and `fromEnv()` in Phase 4 and add file sources in Phase 5 if policy review agrees.

## Alternatives considered

### Alternative: use `exec curl`

Rejected. It is broader, harder to test, harder to redact, and documents an escape hatch instead of the desired framework API.

### Alternative: only implement browser-like global `fetch`

Rejected for first implementation. It is useful but insufficient for framework-native authenticated agents, because it leaves credential injection and redaction entirely to user JavaScript.

### Alternative: put client auth under the existing `auth` module

Possible later, but not preferred initially. The existing `auth` module is hostauth/server-side oriented: audit, capabilities, agents, and tokens. Client request auth is tightly coupled to outbound HTTP request construction. Keeping it under `fetch.auth` makes the API easier to reason about and avoids implying that client credential sources can authenticate local server routes directly.

### Alternative: implement an OAuth-style client immediately

Deferred. The framework does not yet implement all device/refresh token server-side primitives. Bearer credentials are enough to support programauth examples now, and the builder architecture can add device/refresh sources later.

## Open questions

1. Should the module name be `fetch`, `http`, or `httpClient`? Recommendation: use `fetch` for familiarity and expose `client()` for the opinionated API.
2. Should `fetch.fetch` also be installed as a global `fetch` when configured? Recommendation: not in v1; require explicit `require("fetch")` to keep capability usage visible.
3. Should credential file reading live in `fetch.auth.bearer().fromFile(...)` or require users to read files with `fs`? Recommendation: support file sources behind a separate credential policy so canonical examples do not expose raw token values to JavaScript variables.
4. How strict should origin patterns be? Recommendation: exact origins plus explicit localhost wildcard syntax for smoke tests; do not accept arbitrary globs initially.
5. Should non-2xx throw in the fluent client by default? Recommendation: yes for `.expectJson()` / `.expectText()`, no for low-level `fetch.fetch`.

## New-intern review path

Read these files in order before implementing:

1. `pkg/xgoja/providers/host/host.go` — guarded host capability pattern.
2. `modules/fs/fs.go` and `modules/fs/fs_async.go` — TypeScript declarations and Promise-returning host work.
3. `modules/express/auth_builders.go` — Go-owned fluent builder pattern.
4. `pkg/xgoja/providers/hostauth/programmatic.go` — security-sensitive builder pattern and redacted token projection.
5. `pkg/gojahttp/auth/programauth/token.go` — API-token issuance/authentication and redaction model.
6. `pkg/jsverbs/runtime.go` — Promise support for jsverbs.
7. `examples/xgoja/21-generated-host-auth/xgoja.yaml` — generated host example structure.

## Implementation checklist

- [ ] Create `modules/fetch` with low-level Promise fetch.
- [ ] Add TypeScript declarations for low-level fetch.
- [ ] Add host provider guarded `fetch` module registration.
- [ ] Add origin/host allow-list policy.
- [ ] Add timeout and max-response-body policy.
- [ ] Add fluent `fetch.client()` builder.
- [ ] Add `fetch.auth` credential builders.
- [ ] Add bearer token injection with redaction tests.
- [ ] Add generated xgoja server+agent example.
- [ ] Add smoke test proving real agent API-token flow.
- [ ] Update docs and cross-link with programmatic auth ticket.

## References

- `pkg/xgoja/providers/host/host.go:57-67` — guarded host provider module registration.
- `pkg/xgoja/providers/host/host.go:70-126` — guarded filesystem module pattern.
- `pkg/xgoja/providers/host/host.go:161-204` — guarded exec module pattern and why not to use curl for canonical examples.
- `modules/fs/fs.go:55-80` — TypeScript declarations for Promise-returning module functions.
- `modules/fs/fs.go:115-155` — runtime service lookup and async method binding.
- `modules/fs/fs_async.go:11-39` — goja Promise + goroutine + runtimebridge resolution pattern.
- `modules/express/auth_builders.go:13-90` — Go-owned builder-store pattern and auth builder validation.
- `pkg/xgoja/providers/hostauth/programmatic.go:32-120` — Go-owned grant/auth builder state.
- `pkg/xgoja/providers/hostauth/programmatic.go:123-193` — agent creation and API-token issuance builder.
- `pkg/xgoja/providers/hostauth/programmatic.go:287-318` — redacted JavaScript projections.
- `pkg/gojahttp/auth/programauth/token.go:50-112` — API-token storage and one-time raw issuance model.
- `pkg/gojahttp/auth/programauth/token.go:190-235` — bearer authentication into `AuthResult`.
- `pkg/gojahttp/auth_plan.go:75-90` — route auth requirement model.
- `pkg/gojahttp/enforcer.go:80-103` and `pkg/gojahttp/enforcer.go:245-257` — route auth requirement enforcement.
- `pkg/jsverbs/runtime.go:90-108` and `pkg/jsverbs/runtime.go:242-278` — Promise-returning jsverb support.
- `examples/xgoja/21-generated-host-auth/xgoja.yaml:24-49` — generated runtime module configuration pattern.
