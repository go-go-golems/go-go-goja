---
Title: Personal Knowledge Inbox Tutorial
Ticket: XGOJA-PERSONAL-INBOX-TUTORIAL
Status: active
Topics:
    - xgoja
    - auth
    - security
    - examples
    - jsverbs
    - documentation
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/xgoja/doc/26-express-route-auth-requirements.md
      Note: Route auth requirement reference linked by tutorial
    - Path: cmd/xgoja/doc/27-guarded-fetch-client-api.md
      Note: Guarded fetch client reference linked by tutorial
    - Path: cmd/xgoja/doc/28-device-authorization-programmatic-access.md
      Note: Device authorization endpoint reference linked by tutorial
    - Path: cmd/xgoja/doc/28-device-authorization-programmatic-access.md:Native device authorization endpoint reference
    - Path: examples/xgoja/21-generated-host-auth/README.md
      Note: Generated OIDC host and browser-session pattern used by tutorial
    - Path: examples/xgoja/21-generated-host-auth/README.md:Generated xgoja OIDC host pattern for browser sessions and native auth handlers
    - Path: examples/xgoja/22-programmatic-agent-auth/README.md
      Note: Generated server plus agent programmatic-auth example used by tutorial
    - Path: examples/xgoja/22-programmatic-agent-auth/README.md:Generated server plus agent binary pattern for programmatic auth
    - Path: pkg/gojahttp/auth/programauth/device.go
      Note: Device service implementation referenced for deeper study
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Personal Knowledge Inbox: Device Login and Programmatic Capture with xgoja

## Purpose

This tutorial teaches programmatic token acquisition by building a small application that is useful enough to feel real. The application is a personal knowledge inbox. A browser user can view and manage captured items. A terminal client can capture links and notes after completing device login. The server decides whether each request is allowed by looking at the authenticated principal, the route requirement, and the granted action.

By the end, the reader should understand the complete path from a terminal command to a protected xgoja route: the CLI starts device authorization, the user approves the request in a browser session, the CLI receives an access/refresh-token pair, the CLI calls a protected route with `Authorization: Bearer ...`, and the route sees a redacted `ctx.auth` result for an automation agent.

## Should this be all xgoja, or a Go app with xgoja augmentation?

The best first version should be a generated xgoja app, not a custom Go app. The server can be a generated xgoja HTTP `serve` binary with jsverbs route registration, embedded static assets, generated hostauth, and native Go-owned auth handlers. The CLI can be a second generated xgoja binary whose jsverbs use the guarded `fetch` client and a small token cache file.

There are two caveats, and they are important:

1. Device approval needs a real browser session. Generated `auth.mode=oidc` provides native login/callback/session handlers. Generated `auth.mode=dev` builds stores and services, but it does not currently provide a complete browser login UI for approval. For a real tutorial, use OIDC with local Keycloak or the generated-host OIDC smoke pattern.
2. Application data persistence can be simple in JavaScript for the tutorial. Use the guarded `database` module with SQLite, or start with a JSON file through `fs:host`. A custom Go host becomes more attractive only when the app data model needs a Go-owned repository layer, migrations, custom domain services, or non-trivial production deployment behavior.

The recommended tutorial path is therefore:

```text
Generated xgoja server binary
  - jsverbs define routes
  - generated HTTP serve owns listener
  - generated hostauth owns sessions, OIDC, device endpoints, programauth stores
  - JavaScript owns application routes and inbox persistence

Generated xgoja CLI binary
  - jsverbs define commands such as capture, login, refresh, whoami
  - guarded fetch module calls the server
  - fs:host stores local token cache
```

A Go-based appendix can be added later for local development without Keycloak, using the existing `devauth` package as the browser-session source. That is a useful variant, but it should not be the main teaching path because it moves attention away from generated xgoja hostauth.

## The application we will build

The application has one responsibility: collect small knowledge items from different entry points and let the browser user manage them later.

An inbox item is intentionally small:

```typescript
interface InboxItem {
  id: string
  title: string
  url?: string
  note?: string
  source: "browser" | "cli" | "script"
  submittedByKind: "sessionUser" | "agent"
  submittedById: string
  createdAt: string
  archivedAt?: string
}
```

The distinction between browser and agent routes is the point of the tutorial. A browser user has a session and can manage the inbox. A CLI agent has a token and can submit new items. Those are different capabilities. The application should not treat them as the same principal just because both can make HTTP requests.

## What the reader learns

The tutorial is organized around five concepts:

- A generated xgoja `serve` command mounts native Go auth handlers before JavaScript routes.
- JavaScript route code declares auth requirements; it does not parse cookies, bearer tokens, or CSRF tokens.
- Device authorization is a two-party flow: the CLI owns the device code, and the browser session approves the user code.
- Access tokens authenticate routes; refresh tokens acquire new access tokens but are not route credentials.
- The `programauth` store family makes agents, tokens, refresh families, and device codes durable across restarts.

## Repository layout

The tutorial should become a sequence of complete runnable snapshots. Each step lives in a subdirectory and copies the previous step before adding one new idea:

```text
examples/xgoja/23-personal-knowledge-inbox/
  README.md
  Makefile
  01-minimal-jsverb/
    xgoja.yaml
    verbs/hello.js
  02-hello-web-server/
    xgoja.yaml
    verbs/server.js
  03-cli-and-server-sqlite/
    server.xgoja.yaml
    inboxctl.xgoja.yaml
    verbs/server.js
    verbs/inboxctl.js
  ...
```

This layout is part of the teaching strategy. A new developer can open one directory, read the files that exist at that point in the narrative, run the smoke test, and then compare it with the next directory.

The later server/CLI steps should converge on two generated binaries:

| Binary | Built from | Responsibility |
| --- | --- | --- |
| `personal-inbox-server` | `server.xgoja.yaml` | HTTP server, browser UI, protected API, native auth/device endpoints. |
| `inboxctl` | `inboxctl.xgoja.yaml` | Terminal capture client, device login, token refresh, authenticated API calls. |

## Architecture sequence

The full device-login path is this sequence:

```text
1. inboxctl login starts device authorization
   POST /auth/device/start

2. server returns device_code and user_code
   device_code is secret; user_code is shown to the human

3. user opens browser and logs in through OIDC
   GET /auth/login -> OIDC provider -> GET /auth/callback

4. browser UI approves the device request
   POST /auth/device/approve with session cookie and X-CSRF-Token

5. inboxctl polls until approval completes
   POST /auth/device/token

6. server consumes the device authorization and returns tokens
   access_token = route credential
   refresh_token = token renewal credential

7. inboxctl captures a link
   POST /api/capture with Authorization: Bearer <access_token>

8. route enforcer authenticates the access token and checks route policy
   principal kind must be agent
   grant must include inbox.capture

9. JavaScript handler stores the item
   ctx.auth is available for redacted principal metadata
```

The important boundary is before step 9. Handler code should only run after the host has authenticated the request, checked the route requirement, checked CSRF when needed, and verified the route action grant.

## Chapter 1: Create the generated server

The server spec selects the HTTP provider, host provider, hostauth provider, embedded assets, and JavaScript route source.

```yaml
# server.xgoja.yaml
schema: xgoja/v2
name: personal-knowledge-inbox-server
app:
  name: personal-knowledge-inbox-server
  envPrefix: XGOJA_INBOX_SERVER
go:
  module: xgoja.generated/personal-knowledge-inbox-server
  version: "1.26"
workspace:
  mode: auto
providers:
  - id: go-go-goja-host
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/host
    register: Register
  - id: go-go-goja-hostauth
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/hostauth
    register: Register
  - id: go-go-goja-http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
    register: Register
runtime:
  modules:
    - provider: go-go-goja-host
      name: fs
      as: fs:host
      config:
        allow: true
    - provider: go-go-goja-host
      name: database
      as: database
      config:
        allowConfigure: true
    - provider: go-go-goja-hostauth
      name: auth
      as: auth
      config:
        audit:
          maxLimit: 100
    - provider: go-go-goja-http
      name: express
      config:
        reject-raw-routes: true
        dev-errors: false
auth:
  mode: oidc
  session:
    cookie:
      allow-insecure-http: true
  stores:
    default:
      driver: sqlite
      dsn: file:personal-inbox-auth.sqlite?cache=shared
      apply-schema: true
    programauth:
      driver: sqlite
      dsn: file:personal-inbox-programauth.sqlite?cache=shared
      apply-schema: true
  oidc:
    issuer-url: http://localhost:18080/realms/personal-inbox
    client-id: personal-inbox
    public-base-url: http://127.0.0.1:18789
sources:
  - id: inbox-server-verbs
    kind: jsverbs
    from:
      dir: ./verbs
    include:
      - server.js
  - id: inbox-assets
    kind: assets
    from:
      dir: ./assets
commands:
  - id: http-serve
    type: provider.command-set
    name: serve
    mount: serve
    provider: go-go-goja-http
    sources:
      - inbox-server-verbs
artifacts:
  - id: binary
    type: binary
    output: dist/personal-inbox-server
    sources:
      - inbox-server-verbs
      - inbox-assets
  - id: embedded-assets
    type: embedded-assets
    sources:
      - inbox-assets
```

The `programauth` store is deliberately separate from the default store in this example. That makes the token/device state visible as its own operational concern. If omitted, it would inherit from `default`.

Build and inspect the spec:

```bash
go run ./cmd/xgoja doctor -f examples/xgoja/23-personal-knowledge-inbox/server.xgoja.yaml
go run ./cmd/xgoja build \
  -f examples/xgoja/23-personal-knowledge-inbox/server.xgoja.yaml \
  --output examples/xgoja/23-personal-knowledge-inbox/dist/personal-inbox-server \
  --xgoja-replace .
```

## Chapter 2: Define the server routes

The server route file uses jsverbs to register a `server` verb. That verb is executed by the generated HTTP `serve` command; it registers routes on the Express-compatible app.

```javascript
// verbs/server.js
__package__({ name: "inbox", short: "Personal knowledge inbox server" })

__verb__("server", {
  name: "server",
  output: "text",
  short: "Register personal inbox routes",
  fields: {
    options: { bind: "all" },
    databasePath: {
      help: "SQLite database file for inbox items",
      default: "personal-inbox.sqlite"
    }
  }
})

function server(options) {
  const express = require("express")
  const database = require("database")
  const assets = require("fs:assets")

  const app = express.app()
  configureDatabase(database, options.databasePath || "personal-inbox.sqlite")

  app.staticFromAssetsModule("/static", assets, "/public")

  app.get("/")
    .public()
    .audit("inbox.landing.view")
    .handle((_ctx, res) => {
      res.type("text/html").send(assets.readFileSync("/public/index.html", "utf8"))
    })

  app.get("/healthz")
    .public()
    .audit("inbox.health")
    .handle((_ctx, res) => res.json({ ok: true, app: "personal-inbox" }))

  app.get("/api/inbox")
    .auth(express.sessionUser())
    .allow("inbox.read")
    .audit("inbox.read")
    .handle((_ctx, res) => {
      res.json({ items: listInboxItems(database) })
    })

  app.post("/api/inbox/:id/archive")
    .auth(express.sessionUser())
    .csrf()
    .allow("inbox.archive")
    .audit("inbox.archive")
    .handle((ctx, res) => {
      archiveItem(database, ctx.params.id)
      res.json({ ok: true })
    })

  app.post("/api/capture")
    .auth(express.anyOf(
      express.sessionUser(),
      express.agent()
    ))
    .rateLimit(express.rateLimit("inbox-capture").perMinute(60).byActor().byRoute())
    .allow("inbox.capture")
    .audit("inbox.capture")
    .handle((ctx, res) => {
      const item = createInboxItem(database, ctx)
      res.status(201).json({ ok: true, item })
    })

  app.get("/api/capture/me")
    .auth(express.anyOf(
      express.sessionUser(),
      express.agent()
    ))
    .allow("inbox.capture")
    .audit("inbox.capture.me")
    .handle((ctx, res) => {
      res.json({ auth: ctx.auth, actor: ctx.actor })
    })

  return "personal inbox routes registered\n"
}
```

The route declaration tells the host what is allowed. The capture route is shared because both browser users and agents may submit items. The archive route is session-only because archiving changes the user's personal workspace and should not be available to a generic capture agent.

The persistence helpers are ordinary JavaScript. They are not security-sensitive; the security decision happened before the handler.

```javascript
function configureDatabase(database, path) {
  database.configure("sqlite3", path)
  database.exec(`
    create table if not exists inbox_items (
      id text primary key,
      title text not null,
      url text not null default '',
      note text not null default '',
      source text not null,
      submitted_by_kind text not null,
      submitted_by_id text not null,
      created_at text not null,
      archived_at text not null default ''
    )
  `)
}

function createInboxItem(database, ctx) {
  const body = ctx.body || {}
  const id = `item_${Date.now()}_${Math.random().toString(16).slice(2)}`
  const now = new Date().toISOString()
  const submittedByKind = ctx.auth.principalKind === "agent" ? "agent" : "sessionUser"
  const submittedById = ctx.auth.principalId || ctx.actor.id || "unknown"
  const item = {
    id,
    title: String(body.title || body.url || "Untitled capture"),
    url: String(body.url || ""),
    note: String(body.note || ""),
    source: String(body.source || submittedByKind),
    submittedByKind,
    submittedById,
    createdAt: now,
    archivedAt: ""
  }
  database.exec(
    `insert into inbox_items
      (id, title, url, note, source, submitted_by_kind, submitted_by_id, created_at, archived_at)
     values (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
    item.id, item.title, item.url, item.note, item.source,
    item.submittedByKind, item.submittedById, item.createdAt, item.archivedAt
  )
  return item
}

function listInboxItems(database) {
  return database.query(`
    select id, title, url, note, source,
           submitted_by_kind as submittedByKind,
           submitted_by_id as submittedById,
           created_at as createdAt,
           archived_at as archivedAt
      from inbox_items
     where archived_at = ''
     order by created_at desc
  `)
}

function archiveItem(database, id) {
  database.exec(
    "update inbox_items set archived_at = ? where id = ?",
    new Date().toISOString(),
    id
  )
}
```

The reader should notice that no handler reads the `Authorization` header. The host reads it, authenticates it, and exposes redacted metadata as `ctx.auth`.

## Chapter 3: Add the browser UI and approval page

The browser UI has three jobs:

1. Show the current inbox.
2. Let the user submit or archive items through session-authenticated routes.
3. Let the user approve a device login request.

The generated hostauth OIDC handlers provide:

```text
GET  /auth/login
GET  /auth/callback
GET  /auth/session
POST /auth/logout
POST /auth/device/approve
```

The page should call `/auth/session` after login to get the CSRF token. Unsafe session-backed mutations then send `X-CSRF-Token`.

```javascript
async function loadSession() {
  const res = await fetch("/auth/session")
  if (res.status === 401) {
    document.location.href = "/auth/login"
    return null
  }
  return await res.json()
}

async function approveDevice(userCode, actions) {
  const session = await loadSession()
  if (!session) return

  const res = await fetch("/auth/device/approve", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-CSRF-Token": session.csrfToken
    },
    body: JSON.stringify({
      user_code: userCode,
      tenantId: "o1",
      actions
    })
  })

  if (!res.ok) {
    throw new Error(await res.text())
  }
  return await res.json()
}
```

The approval UI should display what is being granted. The first version can hard-code `inbox.capture`, but the tutorial should still explain why this matters: approval must not be a blind button that grants every action the client requested.

```text
Device request

Client: inboxctl on this computer
Requested action: inbox.capture
User code: ABCD-EFGH

[Approve capture access] [Deny]
```

The service intersects requested and approved grants. If the CLI requested `inbox.capture` and the browser approves only `inbox.capture`, the token receives that grant. If a malicious or buggy CLI requested `inbox.delete`, the approval page should not approve it for this tutorial.

## Chapter 4: Create the CLI xgoja binary

The CLI is also generated by xgoja. It does not need the HTTP provider or hostauth provider. It needs guarded host capabilities: `fetch` for HTTP calls and `fs:host` for the local token cache.

```yaml
# inboxctl.xgoja.yaml
schema: xgoja/v2
name: inboxctl
app:
  name: inboxctl
  envPrefix: INBOXCTL
go:
  module: xgoja.generated/inboxctl
  version: "1.26"
workspace:
  mode: auto
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
        timeout: 10s
        maxResponseBytes: 1048576
        credentials:
          allowFiles: true
    - provider: go-go-goja-host
      name: fs
      as: fs:host
      config:
        allow: true
sources:
  - id: inboxctl-verbs
    kind: jsverbs
    from:
      dir: ./verbs
    include:
      - inboxctl.js
artifacts:
  - id: binary
    type: binary
    output: dist/inboxctl
    sources:
      - inboxctl-verbs
```

The CLI should expose four commands:

| Command | Purpose |
| --- | --- |
| `login` | Start device authorization, print user code, poll until tokens arrive. |
| `capture` | Submit a URL or note, refreshing the access token if needed. |
| `whoami` | Call `/api/capture/me` and print the authenticated principal. |
| `logout` | Delete the local token cache. |

## Chapter 5: Implement device login in jsverbs

The `login` command starts device authorization and polls for tokens. It should not ask the user for a password. The user authenticates in the browser; the CLI only shows a code and waits.

```javascript
// verbs/inboxctl.js
__package__({ name: "inboxctl", short: "Personal knowledge inbox CLI" })

__verb__("login", {
  name: "login",
  output: "text",
  short: "Authorize this terminal client with device login",
  fields: {
    options: { bind: "all" },
    baseUrl: { default: "http://127.0.0.1:18789" },
    tokenFile: { default: "~/.config/inboxctl/tokens.json" },
    clientName: { default: "inboxctl" }
  }
})

async function login(options) {
  const fetch = require("fetch")
  const fs = require("fs:host")
  const baseUrl = trimRight(options.baseUrl, "/")
  const tokenFile = expandHome(options.tokenFile)

  const startClient = fetch.client()
    .baseUrl(baseUrl)
    .acceptJson()
    .expectJson()

  const started = await startClient.post("/auth/device/start")
    .json({
      clientName: options.clientName || "inboxctl",
      tenantId: "o1",
      actions: ["inbox.capture"]
    })
    .run()

  console.log("Open this URL in your browser:")
  console.log(`  ${baseUrl}${started.verification_uri_complete || started.verification_uri}`)
  console.log("")
  console.log("Approve this user code:")
  console.log(`  ${started.user_code}`)
  console.log("")
  console.log("Waiting for approval...")

  const tokens = await pollForTokens(fetch, baseUrl, started)
  writeJSON(fs, tokenFile, {
    baseUrl,
    accessToken: tokens.access_token,
    refreshToken: tokens.refresh_token,
    tokenType: tokens.token_type,
    expiresAt: new Date(Date.now() + tokens.expires_in * 1000).toISOString(),
    scope: tokens.scope
  })

  return `login complete; token cache written to ${tokenFile}\n`
}
```

The polling loop has to treat `authorization_pending` and `slow_down` as expected states. They are not failures. They are the protocol telling the client when to try again.

```javascript
async function pollForTokens(fetch, baseUrl, started) {
  let intervalSeconds = started.interval || 5
  const client = fetch.client()
    .baseUrl(baseUrl)
    .acceptJson()
    .expectResponse()

  while (true) {
    await sleep(intervalSeconds * 1000)

    const res = await client.post("/auth/device/token")
      .json({
        grant_type: "urn:ietf:params:oauth:grant-type:device_code",
        device_code: started.device_code
      })
      .run()

    const body = await res.json()
    if (res.status >= 200 && res.status < 300) {
      return body
    }

    if (body.error === "authorization_pending") {
      intervalSeconds = body.interval || intervalSeconds
      continue
    }

    if (body.error === "slow_down") {
      intervalSeconds = body.interval || (intervalSeconds + 5)
      continue
    }

    throw new Error(`device authorization failed: ${body.error || res.status}`)
  }
}
```

This is the first place where the reader sees that device login is not a browser automation trick. The CLI never handles the user's browser credentials. It receives tokens only after the server has observed a valid browser session and approval request.

## Chapter 6: Capture a link with the access token

The `capture` command reads the token cache and calls the protected capture route. The first implementation can require the access token to still be valid. The next chapter adds refresh.

```javascript
__verb__("capture", {
  name: "capture",
  output: "json",
  short: "Capture a URL or note into the inbox",
  fields: {
    options: { bind: "all" },
    baseUrl: { default: "http://127.0.0.1:18789" },
    tokenFile: { default: "~/.config/inboxctl/tokens.json" },
    url: { help: "URL to capture" },
    title: { help: "Capture title" },
    note: { help: "Optional note" }
  }
})

async function capture(options) {
  const fetch = require("fetch")
  const fs = require("fs:host")
  const tokenFile = expandHome(options.tokenFile)
  const tokens = readJSON(fs, tokenFile)
  const baseUrl = trimRight(options.baseUrl || tokens.baseUrl, "/")

  const client = fetch.client()
    .baseUrl(baseUrl)
    .auth(fetch.auth.bearer().token(tokens.accessToken))
    .acceptJson()
    .expectJson()

  return await client.post("/api/capture")
    .json({
      url: options.url || "",
      title: options.title || options.url || "Untitled capture",
      note: options.note || "",
      source: "cli"
    })
    .run()
}
```

At this point the route should succeed only if the access token carries `inbox.capture`. A token with no grant, a revoked token, a refresh token, or a browser session without the right route action should fail before the handler runs.

## Chapter 7: Add refresh-token rotation

Access tokens should be short-lived. The CLI should refresh before calling the server when the cached access token is near expiry.

The tutorial should add a helper:

```javascript
async function authenticatedClient(fetch, fs, options) {
  const tokenFile = expandHome(options.tokenFile)
  let tokens = readJSON(fs, tokenFile)

  if (isExpiringSoon(tokens.expiresAt)) {
    tokens = await refreshTokens(fetch, tokens)
    writeJSON(fs, tokenFile, tokens)
  }

  return fetch.client()
    .baseUrl(trimRight(options.baseUrl || tokens.baseUrl, "/"))
    .auth(fetch.auth.bearer().token(tokens.accessToken))
    .acceptJson()
    .expectJson()
}
```

The exact refresh endpoint should be implemented or exposed before this chapter becomes runnable. The conceptual rule is already fixed: a refresh token can obtain a new pair, but it must not authenticate `/api/capture` directly. A useful exercise is to deliberately send the refresh token as a bearer credential and observe that the route rejects it.

```text
Access token  -> accepted by protected routes until expiry/revocation
Refresh token -> accepted only by token refresh service code
Device code   -> accepted only by device token polling endpoint
```

This distinction prevents long-lived refresh credentials from becoming general-purpose route credentials.

## Chapter 8: Make programauth durable

The first run can use SQLite for programauth from the beginning. The tutorial should still ask the reader to prove persistence:

1. Run `inboxctl login`.
2. Capture one item.
3. Stop the server.
4. Start the server again with the same `personal-inbox-programauth.sqlite` file.
5. Run `inboxctl whoami` or `inboxctl capture` again.

If the token still works, the agent and token family survived restart. If the server used memory stores, the same token would fail after restart because the stored token hash would be gone.

The store configuration to study is:

```yaml
auth:
  stores:
    default:
      driver: sqlite
      dsn: file:personal-inbox-auth.sqlite?cache=shared
      apply-schema: true
    programauth:
      driver: sqlite
      dsn: file:personal-inbox-programauth.sqlite?cache=shared
      apply-schema: true
```

The reader should understand that SQLite is a local persistence tool here. A multi-process deployment should use PostgreSQL so all server replicas share token revocation, refresh-family state, and device-code consumption.

## Chapter 9: Add an approved-clients screen

A real product lets users inspect and revoke automation access. The first version can expose a browser-only page that lists agents and token metadata. The JavaScript programmatic-auth API already supports API-token list/revoke; access/refresh-token management may need a small additional read API before the UI can be complete.

The UI should show redacted information only:

```text
Connected clients

inboxctl
  principal: agent agt_...
  scopes: inbox.capture
  credential: ggat_...redacted
  last used: 2026-06-22 11:12:13

[Disable client]
```

The teaching point is that user-facing security screens need stable identities. An agent is the identity. A token is one credential for that identity. Revoking a token and disabling an agent are different administrative actions.

## Chapter 10: Smoke test the full app

The smoke test should build both binaries, start the server, exercise unauthenticated failure paths, drive device start/pending polling, optionally complete approval through a browser automation helper, and capture an item through `inboxctl`.

A first smoke can validate the pieces that do not require browser automation:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

Expected checks:

- `xgoja doctor` passes for both specs.
- Both binaries build.
- `/healthz` is public.
- `/api/inbox` returns `401` without a session.
- `/api/capture` returns `401` without a token.
- `/auth/device/start` returns `device_code` and `user_code`.
- `/auth/device/token` returns `authorization_pending` before approval.

The full smoke should add local OIDC login and device approval. It can borrow the browserless OIDC helper pattern from `examples/xgoja/21-generated-host-auth/scripts/keycloak_compose_smoke.py`.

## Exercises

1. Add `inbox.read-own` and let an agent list only items it submitted. Compare this with `inbox.read`, which should remain browser-only.
2. Add a `source` filter to `GET /api/inbox` and verify it is only available to session users.
3. Add token cache corruption handling to `inboxctl`. The CLI should delete the cache and ask for login again when refresh fails.
4. Change `programauth` from SQLite to memory and prove that tokens fail after server restart.
5. Move the auth stores to PostgreSQL and explain which tables belong to session, audit, appauth, capability, and programauth.

## Key points

- Generated xgoja is sufficient for the main tutorial. The server is a generated `serve` app; the CLI is a generated jsverbs binary.
- Device authorization needs a browser-session approval path. In current generated hostauth, OIDC is the real path for that session.
- JavaScript declares route requirements with `express.sessionUser()`, `express.agent()`, and `express.anyOf(...)`; it should not parse auth headers directly.
- Access tokens authenticate routes. Refresh tokens and device codes do not.
- Durable `programauth` storage is required when device approval, token refresh, token revocation, or route authentication may happen across restarts or replicas.
- A Go-based host is useful as an appendix for local dev-auth or custom domain infrastructure, but it is not necessary for the main tutorial.

## Resource map for deeper study

This tutorial should link readers to the material below. The goal is not to duplicate every reference page; it is to give the intern a path from working example to implementation details.

### Runnable examples

| Resource | Why it matters |
| --- | --- |
| `examples/xgoja/13-http-serve-jsverbs` | Minimal generated HTTP `serve` command and jsverbs route registration. |
| `examples/xgoja/21-generated-host-auth` | Generated OIDC hostauth, native login/session handlers, embedded assets, protected routes. |
| `examples/xgoja/22-programmatic-agent-auth` | Generated server plus generated agent binary, API-token auth, guarded fetch client, native device start/poll smoke. |
| `examples/xgoja/02-host-provider` | Guarded host modules including `fs`, `exec`, and `database`. |
| `examples/xgoja/18-express-auth-host` | Go-based dev-auth host. Use this as the appendix path if we want no OIDC dependency. |

### xgoja help docs

| Help page / file | Topic |
| --- | --- |
| `xgoja help xgoja-v2-reference` / `cmd/xgoja/doc/17-xgoja-v2-reference.md` | xgoja spec structure, sources, commands, artifacts. |
| `xgoja help http-serve-command-reference` / `cmd/xgoja/doc/22-http-serve-command-reference.md` | Generated HTTP serve command behavior. |
| `xgoja help hostauth-config-reference` / `cmd/xgoja/doc/20-hostauth-config-reference.md` | Auth modes, OIDC, session cookies, store configuration, programauth store flags. |
| `xgoja help auth-stores-reference` / `cmd/xgoja/doc/21-auth-stores-reference.md` | Store families, drivers, schema application, DB sharing. |
| `xgoja help generated-auth-javascript-apis` / `cmd/xgoja/doc/24-generated-auth-javascript-apis.md` | Browser-session auth module APIs such as audit/capability helpers. |
| `xgoja help programmatic-auth-javascript-apis` / `cmd/xgoja/doc/25-programmatic-auth-javascript-apis.md` | Agents, API tokens, grant builders, raw-token handling. |
| `xgoja help express-route-auth-requirements` / `cmd/xgoja/doc/26-express-route-auth-requirements.md` | `express.agent()`, `express.sessionUser()`, `express.anyOf(...)`, route status behavior. |
| `xgoja help guarded-fetch-client-api` / `cmd/xgoja/doc/27-guarded-fetch-client-api.md` | `fetch.client()`, outbound policy, bearer credential sources. |
| `xgoja help device-authorization-programmatic-access` / `cmd/xgoja/doc/28-device-authorization-programmatic-access.md` | Device start, token polling, approval, durable programauth storage. |
| `xgoja help auth-host-production-runbook` / `cmd/xgoja/doc/23-auth-host-production-runbook.md` | Production-shaped auth deployment checklist. |

### Implementation files

| File | What to read for |
| --- | --- |
| `pkg/gojahttp/auth/programauth/device.go` | Device authorization state machine and service behavior. |
| `pkg/gojahttp/auth/programauth/device_handlers.go` | Native `/auth/device/*` HTTP handlers and protocol responses. |
| `pkg/gojahttp/auth/programauth/oauth_token.go` | Access/refresh token model, refresh rotation, bearer authentication. |
| `pkg/gojahttp/auth/programauth/composite.go` | How API-token and access-token authenticators combine with session fallback. |
| `pkg/gojahttp/auth/programauth/sqlstore/schema.go` | SQL tables for agents, API tokens, access tokens, refresh tokens, and device authorizations. |
| `pkg/gojahttp/auth/programauth/sqlstore/sqlstore.go` | SQL store behavior, refresh rotation transaction, device transition updates. |
| `pkg/xgoja/hostauth/builder.go` | Generated hostauth service construction and native handler mounting. |
| `pkg/xgoja/hostauth/stores.go` | Memory/SQLite/Postgres store construction, including `ProgramAuthStores`. |
| `modules/fetch` | Guarded fetch module implementation. |
| `modules/express` | Express-compatible route declaration surface. |
| `modules/database` | JavaScript database module used for tutorial inbox persistence. |

### Standards and external references

| Reference | Why it matters |
| --- | --- |
| RFC 8628: OAuth 2.0 Device Authorization Grant — `https://www.rfc-editor.org/rfc/rfc8628` | Defines the device authorization flow, `authorization_pending`, and `slow_down`. |
| RFC 6749: OAuth 2.0 Authorization Framework — `https://www.rfc-editor.org/rfc/rfc6749` | Defines the general OAuth model and token endpoint concepts. |
| RFC 6750: OAuth 2.0 Bearer Token Usage — `https://www.rfc-editor.org/rfc/rfc6750` | Defines bearer-token use in `Authorization: Bearer ...`. |
| RFC 9700: Best Current Practice for OAuth 2.0 Security — `https://www.rfc-editor.org/rfc/rfc9700` | Modern OAuth security guidance. |
| OpenID Connect Core — `https://openid.net/specs/openid-connect-core-1_0.html` | Background for generated OIDC browser login. |

## Implementation recommendation

Turn this tutorial into a runnable example in three increments:

1. Build the server and CLI skeleton with public health, protected route failures, and guarded fetch.
2. Add OIDC login and the browser approval page, using local Keycloak or the existing generated OIDC smoke helper.
3. Add durable SQLite `programauth`, token cache, capture flow, and full smoke coverage.

Do not start with a custom Go host unless the immediate goal is to teach local development auth internals. The main learning objective is programmatic token acquisition in generated xgoja, so the first runnable version should stay inside xgoja specs and jsverbs.
