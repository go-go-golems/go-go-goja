---
Title: Implementation Logbook
Ticket: XGOJA-PERSONAL-INBOX-TUTORIAL
Status: active
Topics:
    - xgoja
    - auth
    - security
    - examples
    - jsverbs
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: examples/xgoja/23-personal-knowledge-inbox/01-minimal-jsverb/Makefile
      Note: Step 01 validation commands and xgoja replace pattern
    - Path: examples/xgoja/23-personal-knowledge-inbox/01-minimal-jsverb/verbs/hello.js
      Note: Step 01 hello-world JavaScript verb
    - Path: examples/xgoja/23-personal-knowledge-inbox/01-minimal-jsverb/xgoja.yaml
      Note: Step 01 minimal generated xgoja spec
    - Path: examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server/Makefile
      Note: Step 02 long-running server smoke pattern
    - Path: examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server/verbs/hello.js
      Note: Step 02 hello CLI and public web server routes
    - Path: examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server/xgoja.yaml
      Note: Step 02 generated HTTP serve spec
    - Path: examples/xgoja/23-personal-knowledge-inbox/Makefile
      Note: Top-level smoke target delegates to individual step directories
    - Path: examples/xgoja/23-personal-knowledge-inbox/README.md
      Note: Top-level step index for the incremental tutorial workspace
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---



# Implementation Logbook

## Goal

This logbook records the detailed construction history of the Personal Knowledge Inbox tutorial example. It is intentionally more expansive than the diary: it preserves intermediate file shapes, command output, mistakes, reasoning, and teaching notes so the final tutorial can explain the path rather than only the destination.

## Context

The tutorial is being built incrementally. The design document already sketches a full generated xgoja server plus generated xgoja CLI with device login, OIDC browser approval, SQLite-backed programauth stores, and inbox persistence. The implementation should not jump directly to that final state. Each chapter should introduce the smallest new idea that can be validated independently.

The first implementation slice is therefore not a web server and not an auth example. It is the minimal generated xgoja shape needed to run one JavaScript verb from a generated binary.

## Entry 1: Chapter 1A — Minimal xgoja spec and hello-world jsverb

### User request

The user asked to keep a detailed diary and an extended logbook while implementing the tutorial. The immediate implementation task was to start Chapter 1 by creating a minimal `xgoja.yaml` and a simple hello-world JavaScript verb. The user also emphasized that intermediate versions of files should be smaller and less complete than the finished tutorial design, because those intermediate states are educational.

### Teaching intent

This slice teaches the smallest useful unit of xgoja:

```text
xgoja.yaml
  -> declares a generated binary
  -> points at local jsverbs source files
  -> exposes the built-in jsverbs command set

verbs/hello.js
  -> declares a package
  -> declares one verb
  -> implements one JavaScript function

make smoke
  -> validates the spec
  -> builds the generated binary
  -> runs the generated command
```

A new developer should not start with OIDC, device authorization, SQLite, embedded assets, and HTTP serving all at once. Those systems matter, but they become understandable only after the reader has seen a generated xgoja binary run one command.

### Files created

```text
examples/xgoja/23-personal-knowledge-inbox/
  README.md
  Makefile
  xgoja.yaml
  verbs/
    hello.js
```

The first `xgoja.yaml` is deliberately small:

```yaml
schema: xgoja/v2
name: personal-knowledge-inbox
app:
  name: personal-knowledge-inbox
go:
  module: xgoja.generated/personal-knowledge-inbox
  version: "1.26"
workspace:
  mode: auto
sources:
  - id: inbox-verbs
    kind: jsverbs
    from:
      dir: ./verbs
    include:
      - hello.js
    language: javascript
commands:
  - id: verbs
    type: builtin.jsverbs
    name: verbs
    sources:
      - inbox-verbs
artifacts:
  - id: binary
    type: binary
    output: dist/personal-knowledge-inbox
    sources:
      - inbox-verbs
```

The important detail is that no providers are selected yet. This keeps the first generated binary focused on the built-in jsverbs command path. Later chapters will add providers when the example needs host modules, HTTP serving, Express routes, auth, and fetch.

The first verb is equally small:

```javascript
__package__({
  name: "inbox",
  short: "Personal Knowledge Inbox tutorial commands"
});

__verb__("hello", {
  name: "hello",
  output: "text",
  short: "Say hello from the first Personal Knowledge Inbox xgoja verb",
  fields: {
    name: {
      type: "string",
      default: "world",
      help: "Name to greet"
    }
  }
});

function hello(name) {
  return `Hello, ${name || "world"}! This is the Personal Knowledge Inbox tutorial.`;
}
```

The generated command path is:

```bash
./dist/personal-knowledge-inbox verbs inbox hello --name tutorial
```

This command path is worth preserving in the tutorial. It shows how `__package__({ name: "inbox" })` and `__verb__("hello", ...)` become CLI structure under the generated `verbs` command set.

### Validation commands

The first validation command succeeded:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox doctor
```

The relevant result was:

```text
schema      ok  xgoja/v2
source-plan ok  inbox-verbs jsverbs 1
```

This confirms that the spec is parseable and that xgoja can discover one JavaScript verb source file.

### Failure: relative `--xgoja-replace .`

The first `make smoke` failed during `go mod tidy` in the generated build workspace:

```text
Error: go mod tidy failed: exit status 1
go: xgoja.generated/personal-knowledge-inbox imports
	github.com/go-go-golems/go-go-goja/pkg/xgoja/app: module github.com/go-go-golems/go-go-goja@latest found (v0.10.1, replaced by .), but does not contain package github.com/go-go-golems/go-go-goja/pkg/xgoja/app
go: xgoja.generated/personal-knowledge-inbox imports
	github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi: module github.com/go-go-golems/go-go-goja@latest found (v0.10.1, replaced by .), but does not contain package github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi
```

The cause was the Makefile using:

```make
--xgoja-replace .
```

The build command is launched from the repository root, but the generated module runs `go mod tidy` from a temporary build workspace. A relative replacement path is interpreted from the generated module context, not from the source repository in the way the tutorial needs. Existing examples use an absolute repository root.

The fix was to rewrite the Makefile with absolute paths:

```make
REPO_ROOT := $(abspath ../../..)
EXAMPLE_DIR := $(abspath .)
BIN := $(EXAMPLE_DIR)/dist/personal-knowledge-inbox
XGOJA := cd $(REPO_ROOT) && GOWORK=off go run ./cmd/xgoja

build:
	$(XGOJA) build -f $(EXAMPLE_DIR)/xgoja.yaml --output $(BIN) --xgoja-replace $(REPO_ROOT) --keep-work
```

This is a useful teaching moment. Generated xgoja builds happen in a generated module workspace. If the generated module should use the current checkout rather than a released module version, pass an absolute path to `--xgoja-replace`.

### Successful smoke

After fixing the Makefile, the full smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

The command did three things:

1. Ran `xgoja doctor` on the spec.
2. Built `dist/personal-knowledge-inbox`.
3. Ran the hello verb and checked the output.

The generated build reported:

```text
validated xgoja/v2 plan for .../examples/xgoja/23-personal-knowledge-inbox/xgoja.yaml
generated build workspace: /tmp/xgoja-build-4291803857
generated module: xgoja.generated/personal-knowledge-inbox
xgoja build ok: .../examples/xgoja/23-personal-knowledge-inbox/dist/personal-knowledge-inbox
```

The smoke command then ran:

```bash
./dist/personal-knowledge-inbox verbs inbox hello --name tutorial
```

and verified that the output contained:

```text
Hello, tutorial!
```

### What this unlocks

The example now has a stable baseline. Future chapters can modify this directory incrementally rather than introducing the final architecture all at once.

The next planned steps are:

1. Convert the example from a CLI-only generated binary into a hello-world generated HTTP server.
2. Add a separate CLI verb once the server exists.
3. Add SQLite-backed inbox storage through the guarded `database` module.
4. Add generated hostauth and session-protected browser routes.
5. Add device login and token-backed CLI capture.

### Notes for final tutorial refinement

The final tutorial should keep this chapter small. It should explain only:

- what `schema`, `name`, `app`, `go`, and `workspace` do at a high level;
- what a `jsverbs` source is;
- why `commands[].type: builtin.jsverbs` creates a `verbs` command;
- how `__package__` and `__verb__` map to CLI structure;
- why `xgoja doctor` comes before `xgoja build`;
- why `--xgoja-replace` should use an absolute repository path during local tutorial development.

It should not introduce auth, HTTP, SQLite, or device flow yet. Those concepts belong to later chapters.

## Entry 2: Restructure the example into step directories

### User request

After the first minimal example existed, the user changed the structure: every tutorial step should live in its own subdirectory. Each later step should copy the previous step and add one concept. This gives the finished tutorial a sequence of concrete snapshots rather than one directory that is repeatedly mutated beyond recognition.

### Teaching intent

This is a better tutorial structure. A new developer can read the sequence like this:

```text
01-minimal-jsverb      -> what is the smallest generated xgoja command?
02-hello-web-server    -> what changes when we add HTTP serve?
03-cli-server-sqlite   -> what changes when we add app state and a second binary?
04-hostauth-session    -> what changes when browser sessions enter?
05-device-login        -> what changes when the CLI acquires tokens?
```

Each directory is runnable. The learner does not have to inspect git history or mentally remove features from the final app. The code on disk matches the chapter being read.

### Files moved and created

The first implementation was moved from the tutorial root into:

```text
examples/xgoja/23-personal-knowledge-inbox/01-minimal-jsverb/
  README.md
  Makefile
  xgoja.yaml
  verbs/
    hello.js
```

The top-level tutorial directory now contains an index and dispatcher:

```text
examples/xgoja/23-personal-knowledge-inbox/
  README.md
  Makefile
  01-minimal-jsverb/
    ...
```

The top-level `README.md` explains that each step is a complete runnable snapshot. The top-level `Makefile` delegates `make smoke` to `01-minimal-jsverb` for now:

```make
.DEFAULT_GOAL := smoke

.PHONY: smoke step-01 clean

smoke: step-01

step-01:
	$(MAKE) -C 01-minimal-jsverb smoke

clean:
	$(MAKE) -C 01-minimal-jsverb clean
```

The nested step Makefile had to adjust its repository-root calculation because it moved one directory deeper:

```make
REPO_ROOT := $(abspath ../../../..)
EXAMPLE_DIR := $(abspath .)
```

### Validation

After the move, the top-level smoke still passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

The command delegated to `01-minimal-jsverb`, ran `xgoja doctor`, built the generated binary, and verified the hello command output.

### What changed conceptually

The tutorial is no longer one evolving example directory. It is a step-indexed corpus. That means future work should use this pattern:

```bash
cp -a 01-minimal-jsverb 02-hello-web-server
# modify only 02-hello-web-server for the next chapter
```

When a later chapter needs two binaries, that step can introduce `server.xgoja.yaml` and `inboxctl.xgoja.yaml` without rewriting the earlier steps. When a later chapter needs SQLite or hostauth, those files appear only in the step that teaches them.

### Notes for final tutorial refinement

The final tutorial should explicitly teach why the repository is structured this way. The step directories are not just source organization. They are part of the pedagogy: each directory is an executable checkpoint.

The chapter instructions should include commands like:

```bash
cd examples/xgoja/23-personal-knowledge-inbox/01-minimal-jsverb
make smoke
```

and later:

```bash
cd ../02-hello-web-server
make smoke
```

The comparison between adjacent directories can become an exercise. For example, after Chapter 2 the reader can diff `01-minimal-jsverb/xgoja.yaml` against `02-hello-web-server/xgoja.yaml` to see exactly what the HTTP provider adds.

## Entry 3: Step 02 — Add a generated hello-world web server

### User request

The user confirmed that the accidental `verbs` to `verbs-foo` edit was reverted and asked to continue with `02-hello-web-server`.

### Teaching intent

Step 01 taught the smallest generated xgoja command: one `xgoja.yaml`, one jsverb, and the built-in `verbs` command. Step 02 copies that whole directory and adds exactly one new concept: a generated binary can also expose a provider-contributed HTTP `serve` command.

The step deliberately keeps the same `hello` CLI verb from Step 01. This is useful because the reader can see that adding a provider does not replace the existing command surface. It extends the generated binary.

### Files created

Step 02 lives at:

```text
examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server/
  README.md
  Makefile
  xgoja.yaml
  verbs/
    hello.js
```

The top-level tutorial `README.md` and `Makefile` were updated so `make smoke` now runs both implemented steps:

```text
01-minimal-jsverb
02-hello-web-server
```

### xgoja.yaml changes from Step 01

The Step 02 spec adds the HTTP provider:

```yaml
providers:
  - id: go-go-goja-http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
    register: Register
```

It selects the Express-compatible runtime module:

```yaml
runtime:
  modules:
    - provider: go-go-goja-http
      name: express
      config:
        reject-raw-routes: true
        dev-errors: false
```

It keeps the built-in jsverbs command and adds the provider command set:

```yaml
commands:
  - id: verbs
    type: builtin.jsverbs
    name: verbs
    sources:
      - inbox-verbs
  - id: http-serve
    type: provider.command-set
    name: serve
    mount: serve
    provider: go-go-goja-http
    sources:
      - inbox-verbs
```

The generated binary now has two useful command paths:

```bash
./dist/personal-knowledge-inbox-hello-web-server verbs inbox hello --name tutorial
./dist/personal-knowledge-inbox-hello-web-server serve inbox server --http-listen 127.0.0.1:18790
```

The first path runs a normal CLI verb. The second path invokes a jsverb once to register Express routes, then keeps the HTTP server alive.

### JavaScript changes

The same `verbs/hello.js` file still declares the `inbox` package and `hello` verb. Step 02 adds a second verb named `server`:

```javascript
__verb__("server", {
  name: "server",
  output: "text",
  short: "Register a public hello-world web server"
});

function server() {
  const express = require("express");
  const app = express.app();

  app.get("/")
    .public()
    .audit("inbox.hello.view")
    .handle((_ctx, res) => {
      res.send("Hello from the Personal Knowledge Inbox web server.");
    });

  app.get("/healthz")
    .public()
    .audit("inbox.health")
    .handle((_ctx, res) => {
      res.json({ ok: true, step: "02-hello-web-server" });
    });

  return "personal inbox hello web server routes registered\n";
}
```

This introduces the planned-route style without introducing authentication yet. Both routes are explicitly public. That is the correct first HTTP shape because the reader can focus on route registration and serving before learning session or agent requirements.

### Validation

The focused Step 02 smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server smoke
```

The top-level tutorial smoke also passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

The Step 02 smoke verifies three things:

1. `xgoja doctor` accepts the spec and resolves the HTTP provider.
2. The generated CLI command from Step 01 still works.
3. The generated HTTP server starts, `/healthz` returns `{"ok":true,"step":"02-hello-web-server"}`, and `/` returns the text greeting.

### Implementation detail: avoid rebuilding twice

The first Step 02 Makefile made `serve-smoke` depend on `build`, while `smoke` already ran `build` before invoking `serve-smoke`. That caused a redundant second generated build. I removed the `build` dependency from `serve-smoke` so the smoke path builds once and then starts the server.

### What this unlocks

The tutorial now has both sides of the first xgoja learning boundary:

- Step 01: generated CLI from jsverbs only.
- Step 02: generated CLI plus provider-backed HTTP serving.

The next step can copy Step 02 and introduce either a separate CLI/server split or SQLite-backed app state. The user previously suggested “CLI verb + serve with sqlite backend,” so the likely next step is to teach app persistence and command separation before auth.

### Notes for final tutorial refinement

The final tutorial should ask the reader to diff Step 01 and Step 02:

```bash
diff -u 01-minimal-jsverb/xgoja.yaml 02-hello-web-server/xgoja.yaml
```

The meaningful additions are the provider declaration, runtime module selection, and provider command-set mounting. That diff is a better teaching artifact than a long prose explanation alone.

## Entry 4: Step 03 — Add SQLite-backed CLI inbox verbs and annotated xgoja specs

### User request

The user asked for Step 03 to add a SQLite inbox and CLI verbs, without adding a REST API yet. During the step, the user also asked that the tutorial xgoja YAML files teach the schema inline: whenever a new section or feature appears, the YAML should include concise comments explaining the fields, optionality, and possible shapes. Finally, the user clarified that `capture` should fail unless both `title` and `url` are present, and that this step should introduce reusable parameter sections for shared flags such as the database path.

### Teaching intent

Step 03 introduces local application state without introducing HTTP API design or authentication. The learner already knows how to build a generated CLI and how to start a generated hello web server. This step adds the next concept: a CLI command can use a guarded host capability, open SQLite, and manipulate domain data.

The step also introduces jsverbs sections. The `storage` section defines one reusable `--db` flag and each inbox command opts into it. This is the first place where command metadata starts to look like a small CLI schema rather than a list of unrelated flags.

### Files created and changed

Step 03 lives at:

```text
examples/xgoja/23-personal-knowledge-inbox/03-sqlite-cli-inbox/
  README.md
  Makefile
  xgoja.yaml
  scripts/
    cli_smoke.py
  verbs/
    inbox.js
```

The top-level tutorial README and Makefile now include Step 03. The tutorial-level `.gitignore` ignores local `*.sqlite` files so smoke/manual runs do not commit database artifacts.

### YAML teaching comments

All current tutorial `xgoja.yaml` files now contain explanatory comments:

```text
01-minimal-jsverb/xgoja.yaml
02-hello-web-server/xgoja.yaml
03-sqlite-cli-inbox/xgoja.yaml
```

The comments explain each section when it first matters:

- `schema`, `name`, `app`, `go`, and `workspace` in Step 01;
- `providers`, `runtime.modules`, and `provider.command-set` in Step 02;
- multiple providers and the guarded host `database` module in Step 03.

This makes the YAML files part of the tutorial text. A reader can open a step directory and understand the spec without jumping immediately to the long reference documentation.

### Step 03 xgoja changes

Step 03 adds the host provider:

```yaml
providers:
  - id: go-go-goja-host
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/host
    register: Register
```

It adds the guarded database runtime module:

```yaml
runtime:
  modules:
    - provider: go-go-goja-host
      name: database
      as: database
      config:
        allowConfigure: true
```

`allowConfigure: true` is acceptable here because this is a local tutorial CLI that deliberately accepts `--db`. Later production-shaped steps should narrow capabilities more aggressively.

### jsverbs section for shared database parameters

The new `inbox.js` begins with a reusable section:

```javascript
__section__("storage", {
  title: "Storage",
  description: "Shared SQLite database settings used by inbox commands",
  fields: {
    db: {
      type: "string",
      default: "personal-inbox.sqlite",
      help: "SQLite database path"
    }
  }
});
```

Commands opt into the section and bind it into a function parameter:

```javascript
__verb__("capture", {
  sections: ["storage"],
  fields: {
    title: { type: "string", required: true },
    url: { type: "string", required: true },
    storage: { bind: "storage" }
  }
});

function capture(title, url, note, source, storage) {
  const database = openInbox(storage.db);
  // ...
}
```

This pattern avoids repeating the `db` field in `capture`, `list`, and `archive`. It also creates a natural teaching point: sections describe reusable flag groups, while bound parameters decide how JavaScript receives those parsed values.

### Required capture fields

`capture` now requires both title and URL:

```javascript
title: {
  type: "string",
  required: true,
  help: "Capture title"
},
url: {
  type: "string",
  required: true,
  help: "URL to capture"
}
```

The smoke test now checks the negative cases first:

```bash
if ./dist/... verbs inbox capture --db "$db" --title 'Missing URL'; then
  echo 'capture without url unexpectedly succeeded'
  exit 1
fi

if ./dist/... verbs inbox capture --db "$db" --url https://example.com/missing-title; then
  echo 'capture without title unexpectedly succeeded'
  exit 1
fi
```

This is important for the tutorial because validation should be visible. The CLI should fail at the command boundary when required capture fields are missing.

### Output mode issue

The first Step 03 version used `output: "json"`. That is not a supported jsverbs output mode. The observed error was:

```text
Error: inbox.js#archive has unsupported output mode "json"
```

For now the Step 03 verbs use `output: "text"` and return `JSON.stringify(...)`. A follow-up task records that we should revisit proper structured/Glazed output. This keeps Step 03 runnable without losing the improvement idea.

### Validation

Focused Step 03 smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox/03-sqlite-cli-inbox smoke
```

Top-level smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

The Step 03 smoke verifies:

1. `xgoja doctor` accepts the spec and resolves both host and HTTP providers.
2. The original CLI hello command still works.
3. The hello web server still starts and serves `/` plus `/healthz`.
4. `capture` fails without `--url`.
5. `capture` fails without `--title`.
6. Two captures can be inserted into SQLite.
7. `list` returns two active items.
8. `archive` archives the first item.
9. A second `list` returns only the remaining active item.

### What this unlocks

The tutorial now has local app data before it has REST routes. That order is useful. The next step can move inbox operations behind HTTP routes while preserving the same schema and command behavior. When auth arrives later, the reader will already understand the domain model and persistence layer.

### Notes for final tutorial refinement

The final tutorial should include three small exercises at this point:

1. Run `capture` without `--url` and inspect the error.
2. Run `capture` without `--title` and inspect the error.
3. Open `inbox.js` and identify which commands reuse the `storage` section.

It should also explicitly say that `output: "json"` is not supported, and that the tutorial will revisit structured/Glazed output after the SQLite CLI behavior is stable.

## Entry 5: Step 04 — Split reusable JavaScript, add public API routes, and move CLI verbs to fetch

### User request

The user asked for Step 04 to separate reusable JavaScript into a `lib/` directory, separate server route registration into `server.js`, add API routes to the server, and add `client.js` CLI verbs that hit the API.

### Teaching intent

Step 03 taught local application state: CLI verbs opened SQLite directly. Step 04 introduces the API boundary without adding authentication yet. This is the right order because the reader can learn the transport boundary before learning session cookies, agent tokens, CSRF, or device authorization.

The new architecture is:

```text
server.js
  -> registers public HTTP routes
  -> uses lib/inbox_store.js for SQLite

client.js
  -> exposes CLI verbs
  -> uses lib/api_client.js and guarded fetch
  -> no longer opens SQLite directly

lib/inbox_store.js
  -> shared database schema and data operations

lib/api_client.js
  -> shared fetch client helpers
```

This step also makes local `require("./lib/...")` part of the tutorial. The source list includes both jsverb entry files and helper files so they are embedded in the generated binary.

### Files created

```text
examples/xgoja/23-personal-knowledge-inbox/04-api-client-server/
  README.md
  Makefile
  xgoja.yaml
  scripts/
    api_smoke.py
  verbs/
    server.js
    client.js
    lib/
      inbox_store.js
      api_client.js
```

### xgoja.yaml changes

Step 04 keeps `database` and `express` from Step 03 and adds the guarded `fetch` module:

```yaml
runtime:
  modules:
    - provider: go-go-goja-host
      name: database
      as: database
      config:
        allowConfigure: true
    - provider: go-go-goja-host
      name: fetch
      as: fetch
      config:
        allow: true
        allowedOrigins:
          - http://127.0.0.1:*
        timeout: 5s
        maxResponseBytes: 1048576
    - provider: go-go-goja-http
      name: express
```

The source list now includes multiple files:

```yaml
sources:
  - id: inbox-verbs
    kind: jsverbs
    from:
      dir: ./verbs
    include:
      - server.js
      - client.js
      - lib/*.js
```

This is a useful teaching point. `server.js` and `client.js` declare verbs. `lib/*.js` files do not declare verbs, but they must still be embedded so `require("./lib/inbox_store")` and `require("./lib/api_client")` work in the generated binary.

### API routes

The server registers public routes:

```text
GET  /
GET  /healthz
GET  /api/inbox
POST /api/capture
POST /api/inbox/:id/archive
```

There is still no auth. All API routes are `.public()`. That is deliberate. The step teaches that the CLI now crosses an HTTP boundary; later steps will secure that boundary.

`POST /api/capture` performs server-side validation as well as CLI-side validation:

```javascript
if (!body.title || !body.url) {
  res.status(400).json({ error: "title and url are required" });
  return;
}
```

The duplication is intentional. Client validation improves CLI UX, but server validation remains necessary because HTTP clients are not trusted.

### Client verbs

`client.js` declares package `inboxctl` and provides:

```text
verbs inboxctl capture
verbs inboxctl list
verbs inboxctl archive
```

The reusable `api` section defines `--base-url` once and binds it into each client command. This mirrors the Step 03 `storage` section but for the API boundary.

The client helpers use guarded fetch:

```javascript
return fetch.client()
  .baseUrl(trimRight(baseUrl || "http://127.0.0.1:18792", "/"))
  .acceptJson()
  .expectJson();
```

This prepares the tutorial for later authenticated fetch calls without introducing credentials yet.

### Validation

Focused Step 04 smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox/04-api-client-server smoke
```

Top-level tutorial smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

The Step 04 smoke verifies:

1. `xgoja doctor` sees four source files.
2. The generated server starts and serves `/` plus `/healthz`.
3. Client-side `capture` fails without a URL.
4. Client-side `capture` fails without a title.
5. Two captures through `verbs inboxctl capture` reach the API and persist into SQLite.
6. `verbs inboxctl list` returns two items through the API.
7. `verbs inboxctl archive` archives one item through the API.
8. A second list returns one active item.

### What changed conceptually

Step 03 had this path:

```text
CLI verb -> database module -> SQLite
```

Step 04 changes it to:

```text
CLI verb -> fetch module -> HTTP API route -> database module -> SQLite
```

That extra boundary is what later auth will protect. The tutorial now has a realistic client/server shape before adding credentials.

### Notes for final tutorial refinement

The final tutorial should ask the reader to compare Step 03 and Step 04:

- `verbs/inbox.js` became `server.js`, `client.js`, and `lib/*`.
- `--db` is now a server concern.
- `--base-url` is now a client concern.
- CLI verbs no longer depend on the `database` module directly.
- `fetch` appears for the first time.

This step is also a good place to explain why server-side validation remains necessary even when jsverbs marks CLI fields as required.
