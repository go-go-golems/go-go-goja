---
Title: Tutorial Authoring Diary
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
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/go-go-parc/Projects/2026/06/22/ARTICLE - tinyidp as a Keycloak Replacement for go-go-goja Auth Testing.md
      Note: source handoff article for tinyidp replacement constraints
    - Path: examples/xgoja/23-personal-knowledge-inbox/01-minimal-jsverb/Makefile
      Note: Chapter 1A validation targets and absolute xgoja replace pattern
    - Path: examples/xgoja/23-personal-knowledge-inbox/01-minimal-jsverb/verbs/hello.js
      Note: Chapter 1A hello-world jsverb
    - Path: examples/xgoja/23-personal-knowledge-inbox/01-minimal-jsverb/xgoja.yaml
      Note: Chapter 1A minimal generated xgoja spec
    - Path: examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server/Makefile
      Note: Step 02 CLI and server smoke validation
    - Path: examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server/verbs/hello.js
      Note: Step 02 CLI hello plus HTTP route registration
    - Path: examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server/xgoja.yaml
      Note: Step 02 generated HTTP serve xgoja spec
    - Path: examples/xgoja/23-personal-knowledge-inbox/05-embedded-retro-ui/assets/public/styles.css
      Note: Retro monochrome UI styling for Step 05
    - Path: examples/xgoja/23-personal-knowledge-inbox/05-embedded-retro-ui/verbs/server.js
      Note: Serves embedded root HTML and static assets
    - Path: examples/xgoja/23-personal-knowledge-inbox/05-embedded-retro-ui/xgoja.yaml
      Note: Step 05 embedded asset source and fs runtime mount
    - Path: examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/Makefile
      Note: tinyidp-smoke process orchestration for first OIDC tutorial step
    - Path: examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/keycloak/realm-personal-inbox.json
      Note: Alice and Bob Keycloak tutorial users
    - Path: examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/scripts/tinyidp_login_smoke.py
      Note: standard-library OIDC login smoke driver
    - Path: examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/verbs/client.js
      Note: Direct SQLite CLI verbs replacing fetch-backed client verbs
    - Path: examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/verbs/server.js
      Note: Session-protected browser API routes
    - Path: examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/xgoja.yaml
      Note: Step 06 generated hostauth OIDC configuration
    - Path: examples/xgoja/23-personal-knowledge-inbox/07-user-scoped-inbox/verbs/lib/inbox_store.js
      Note: Owner-filtered inbox SQL helpers
    - Path: examples/xgoja/23-personal-knowledge-inbox/07-user-scoped-inbox/verbs/server.js
      Note: Step 07 session actor scoping for browser API routes
    - Path: examples/xgoja/23-personal-knowledge-inbox/07-user-scoped-inbox/xgoja.yaml
      Note: Step 07 generated app and Keycloak port configuration
    - Path: examples/xgoja/23-personal-knowledge-inbox/08-device-authorization/assets/public/app.js
      Note: Browser device approval UI
    - Path: examples/xgoja/23-personal-knowledge-inbox/08-device-authorization/verbs/client.js
      Note: Step 08 device CLI verbs
    - Path: examples/xgoja/23-personal-knowledge-inbox/08-device-authorization/verbs/server.js
      Note: Programmatic capture route protected by agent auth
    - Path: examples/xgoja/23-personal-knowledge-inbox/Makefile
      Note: Top-level smoke dispatcher for step directories
    - Path: examples/xgoja/23-personal-knowledge-inbox/README.md
      Note: Top-level step-index for the incremental tutorial workspace
    - Path: ttmp/2026/06/22/XGOJA-PERSONAL-INBOX-TUTORIAL--personal-knowledge-inbox-device-login-tutorial/design/01-personal-knowledge-inbox-tutorial.md
      Note: Primary tutorial draft
    - Path: ttmp/2026/06/22/XGOJA-PERSONAL-INBOX-TUTORIAL--personal-knowledge-inbox-device-login-tutorial/reference/02-implementation-logbook.md
      Note: Extended step-by-step construction log
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---








# Diary

## Goal

Capture the creation of a new docmgr ticket and a textbook-style tutorial for learning xgoja device login through a real personal knowledge inbox application.

## Step 1: Create the personal inbox tutorial ticket and draft

This step turns the tutorial idea into a structured docmgr ticket and a full tutorial draft. The tutorial is written as an implementation guide for a new developer who needs the concepts, route shapes, generated xgoja specs, JavaScript APIs, token flow, and source references explained before building the runnable example.

The most important design decision is to keep the main tutorial as generated xgoja rather than a custom Go application. The server is a generated xgoja HTTP `serve` binary with jsverbs routes, generated hostauth, OIDC browser login, native device endpoints, and durable programauth stores. The CLI is a second generated xgoja binary with jsverbs commands that use the guarded `fetch` module.

### Prompt Context

**User prompt (verbatim):** "ok, make a new ticket and write out the full tutorial. I want if possible to havae the whole tutorial be a xgoja app with serve and jsverbs for the CLI and all. Let me know if you think that currently a go-based app (with xgoja augmentation) makes more sense. 

Use a textbook writing style for the tutorial, thinking about it being for a new developer / intern that needs all the concepts and apis and fundamentals and workings explained."

**Additional user prompt (verbatim):** "the tutorial should also link to all the resources that allow us to make this more indepth."

**Assistant interpretation:** Create a new docmgr ticket and write a detailed tutorial design for the personal knowledge inbox app, with generated xgoja server and CLI flows where feasible, explicitly assess whether a Go-based host would be better, and include a resource map for deeper study.

**Inferred user intent:** Produce a learning artifact that can later become a runnable example and that teaches device login/programmatic token acquisition from first principles.

**Commit (code):** N/A — documentation-only step.

### What I did

- Created ticket `XGOJA-PERSONAL-INBOX-TUTORIAL`.
- Created the primary design document `design/01-personal-knowledge-inbox-tutorial.md`.
- Created this diary document.
- Added and completed tasks for drafting the tutorial and including a resource map.
- Wrote the tutorial in a textbook style with:
  - the architecture decision on generated xgoja versus Go host,
  - server and CLI xgoja spec sketches,
  - Express route declarations,
  - device login sequence,
  - CLI polling and token-cache examples,
  - durable `programauth` storage explanation,
  - exercises and key points,
  - an in-depth resource map.

### Why

- The previous programmatic-auth work added the core capability, but learning it from code alone would force a new developer to jump across examples, help docs, native handlers, SQL stores, and generated host configuration.
- A real personal inbox app makes the security boundaries concrete: browser users manage items, automation agents capture items, and device login connects a terminal client to the user's account without putting browser credentials in the terminal.
- The user specifically asked for the tutorial to link to deeper resources, so the draft includes runnable examples, help docs, implementation files, and standards references.

### What worked

- The existing example and help-doc set is now rich enough to support the tutorial without inventing a new architecture.
- `examples/xgoja/21-generated-host-auth` supplies the generated OIDC host pattern needed for browser approval.
- `examples/xgoja/22-programmatic-agent-auth` supplies the generated server-plus-agent pattern and guarded fetch usage.
- The newly completed durable `programauth` store configuration gives the tutorial a realistic persistence path.

### What didn't work

- No command failures occurred while creating the ticket and writing the tutorial.
- One product caveat surfaced: generated `auth.mode=dev` does not currently provide a complete browser login/approval UX for device approval. The tutorial therefore recommends generated OIDC as the main path and reserves Go `devauth` for an appendix/no-OIDC variant.

### What I learned

- The full tutorial can be all xgoja if it uses generated OIDC for browser sessions. A custom Go host is not needed for the core learning path.
- The CLI can be a generated xgoja jsverbs binary rather than a Go CLI, as long as it uses guarded `fetch` and `fs:host` for HTTP calls and token caching.
- The remaining implementation work is mostly example construction: assets, local OIDC smoke, refresh endpoint exposure, and runnable CLI commands.

### What was tricky to build

- The tutorial had to be honest about current hostauth capabilities. Device approval requires a browser session; generated OIDC supplies that session, while generated dev mode alone does not.
- The resource map needed to cover both conceptual learning and implementation lookup. It links to runnable examples, xgoja help docs, source files, SQL store files, and OAuth/OIDC standards.
- The tutorial sketches refresh-token behavior but notes that a runnable chapter should verify or expose the exact refresh endpoint before presenting it as copy/paste complete.

### What warrants a second pair of eyes

- Whether the tutorial should require local Keycloak from the start, or whether it should first teach unauthenticated route skeletons and add OIDC later.
- Whether the JavaScript `database` module is the right app-data persistence layer for the tutorial, or whether the eventual runnable example should use a tiny Go-owned app store.
- Whether the generated CLI should use raw token strings in memory after reading the token cache or use `fetch.auth.bearer().fromFile(...).jsonPath(...)` for more calls.

### What should be done in the future

- Implement the runnable example under `examples/xgoja/23-personal-knowledge-inbox`.
- Add local OIDC/Keycloak smoke coverage for browser approval.
- Verify and document the refresh-token endpoint path before making the refresh chapter copy/paste runnable.

### Code review instructions

- Start with `design/01-personal-knowledge-inbox-tutorial.md`.
- Review the section "Should this be all xgoja, or a Go app with xgoja augmentation?" for the key architecture recommendation.
- Review the resource map to ensure it links to all important examples, docs, implementation files, and standards.
- Validate doc hygiene with:

```bash
docmgr doctor --ticket XGOJA-PERSONAL-INBOX-TUTORIAL --stale-after 30
```

### Technical details

Key commands:

```bash
docmgr ticket create-ticket --ticket XGOJA-PERSONAL-INBOX-TUTORIAL --title "Personal Knowledge Inbox Device Login Tutorial" --topics xgoja,auth,security,examples,jsverbs,documentation

docmgr doc add --ticket XGOJA-PERSONAL-INBOX-TUTORIAL --doc-type design --title "Personal Knowledge Inbox Tutorial"

docmgr doc add --ticket XGOJA-PERSONAL-INBOX-TUTORIAL --doc-type reference --title "Tutorial Authoring Diary"
```

## Step 2: Start Chapter 1 with a minimal generated xgoja verb

This step begins the runnable tutorial example with the smallest validated xgoja artifact: a generated CLI binary that exposes one JavaScript verb. The point is not to build the inbox yet. The point is to give the learner a concrete baseline for how `xgoja.yaml`, jsverbs sources, generated binaries, and command paths fit together.

The implementation deliberately stays smaller than the full tutorial design. It does not include HTTP serving, SQLite, browser sessions, device login, or programauth. Those concepts will be layered in later chapters so each intermediate file state can become part of the final educational narrative.

### Prompt Context

**User prompt (verbatim):** "Let's keep a detailed logbook / diary as we work through it, so that we can not only refine the tutorial later on, and have a nice step by step history, but also to write a detailed report / technical dive as we go. So fill your diary, but also keep an extended version of it in a logbook.

Let's start with chapter 1, by first creating a minmal xgoja.yaml and a simple hello world js verb. 

(after that we'll expand to a hello world web server, then to CLI verb + serve with sqlite backend, etc...)

We are tyring to do things incrementally, so we will have intermediate versions of the files that are not as complete in your current tutorial write up, which is good for educational purposes where we do things in smaller validated steps. We will use that for a fully fleshed out tutorial."

**Assistant interpretation:** Begin implementing the tutorial incrementally by creating the first runnable xgoja example with only a minimal spec and hello-world jsverb, while recording both a standard diary and a more detailed logbook.

**Inferred user intent:** Build a step-by-step implementation history that can later be turned into a more complete tutorial and technical deep dive.

**Commit (code):** 8991ac9 — "examples: start personal inbox tutorial"

### What I did

- Created `examples/xgoja/23-personal-knowledge-inbox/`.
- Added a minimal `xgoja.yaml` with:
  - one `jsverbs` source,
  - one built-in `verbs` command set,
  - one generated binary artifact.
- Added `verbs/hello.js` with package `inbox` and verb `hello`.
- Added a small `Makefile` with `doctor`, `build`, `smoke`, and `clean` targets.
- Added a short `README.md` explaining the Chapter 1 baseline.
- Created `reference/02-implementation-logbook.md` as the extended logbook and recorded the detailed construction narrative.
- Added and completed task 3: `Chapter 1A: create minimal xgoja.yaml and hello-world jsverb`.

### Why

- The final tutorial should teach xgoja and device login through incremental validated steps rather than presenting a large final app all at once.
- A hello-world jsverb teaches the generated command path before the reader has to learn HTTP serving, host providers, auth, database persistence, or token acquisition.
- The logbook preserves exact intermediate states and mistakes so the final tutorial can explain why the implementation evolved.

### What worked

- `make -C examples/xgoja/23-personal-knowledge-inbox doctor` passed.
- After fixing the Makefile, `make -C examples/xgoja/23-personal-knowledge-inbox smoke` passed.
- The generated command path works:

```bash
examples/xgoja/23-personal-knowledge-inbox/dist/personal-knowledge-inbox verbs inbox hello --name tutorial
```

### What didn't work

- The first smoke attempt failed because the Makefile used a relative replacement path:

```make
--xgoja-replace .
```

The generated module runs `go mod tidy` from a temporary build workspace, so the relative replacement did not point at the repository checkout. The failure was:

```text
Error: go mod tidy failed: exit status 1
go: xgoja.generated/personal-knowledge-inbox imports
	github.com/go-go-golems/go-go-goja/pkg/xgoja/app: module github.com/go-go-golems/go-go-goja@latest found (v0.10.1, replaced by .), but does not contain package github.com/go-go-golems/go-go-goja/pkg/xgoja/app
go: xgoja.generated/personal-knowledge-inbox imports
	github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi: module github.com/go-go-golems/go-go-goja@latest found (v0.10.1, replaced by .), but does not contain package github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi
```

I fixed this by following the existing examples and using absolute paths:

```make
REPO_ROOT := $(abspath ../../..)
EXAMPLE_DIR := $(abspath .)
...
--xgoja-replace $(REPO_ROOT)
```

### What I learned

- The minimal generated jsverbs binary does not need runtime providers. It only needs a `jsverbs` source and the built-in jsverbs command set.
- The generated CLI command structure is useful teaching material: `verbs inbox hello` comes from the command set name, the `__package__` name, and the `__verb__` name.
- Local generated builds should use an absolute `--xgoja-replace` path so the generated temporary module resolves the current checkout correctly.

### What was tricky to build

- The subtlety was not the JavaScript verb or xgoja spec; it was the generated module's working directory during `go mod tidy`. Relative replacement paths are easy to write but fragile in generated build workflows.
- The example needed to remain intentionally incomplete. It is tempting to add HTTP and database modules immediately, but doing so would weaken the educational value of this first chapter.

### What warrants a second pair of eyes

- Whether the minimal spec should include built-in `eval`, `run`, and `repl` commands for discoverability, or whether keeping only `verbs` is better for Chapter 1.
- Whether the hello verb should accept a single `name` argument, as currently implemented, or bind `options` from the beginning to match later CLI verbs.

### What should be done in the future

- Chapter 1B: expand from the CLI hello verb to a hello-world generated web server.
- Keep the logbook updated with every intermediate file state and validation command.
- Later, revise the main tutorial draft so it incorporates this smaller first step before the full server/CLI/auth architecture.

### Code review instructions

- Review `examples/xgoja/23-personal-knowledge-inbox/xgoja.yaml` first.
- Review `examples/xgoja/23-personal-knowledge-inbox/verbs/hello.js` for the jsverbs package and verb declaration.
- Review `examples/xgoja/23-personal-knowledge-inbox/Makefile` for the absolute `--xgoja-replace` pattern.
- Validate with:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

### Technical details

Validation commands and outcomes:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox doctor
# schema ok; source-plan ok

make -C examples/xgoja/23-personal-knowledge-inbox smoke
# build ok; hello command output matched
```

Primary files:

```text
examples/xgoja/23-personal-knowledge-inbox/xgoja.yaml
examples/xgoja/23-personal-knowledge-inbox/verbs/hello.js
examples/xgoja/23-personal-knowledge-inbox/Makefile
examples/xgoja/23-personal-knowledge-inbox/README.md
ttmp/2026/06/22/XGOJA-PERSONAL-INBOX-TUTORIAL--personal-knowledge-inbox-device-login-tutorial/reference/02-implementation-logbook.md
```

## Step 3: Restructure the tutorial example into per-step directories

This step changes the example layout to match the intended teaching format. Instead of one directory that is repeatedly mutated as the app grows, each tutorial chapter will have its own complete runnable subdirectory.

That structure lets the tutorial preserve intermediate versions of the files. A new developer can run Step 01, read only the files that exist in Step 01, then move to Step 02 and see exactly what changed.

### Prompt Context

**User prompt (verbatim):** "in fact, we are going to make subdirectories for each step, and just copy the material to the next step as we work, that way we have the full thing laid out and new developers can read the individual steps as they go."

**Assistant interpretation:** Reorganize the tutorial workspace so every step is a standalone subdirectory, starting by moving the minimal hello-world jsverb example into `01-minimal-jsverb` and adding a top-level step index.

**Inferred user intent:** Make the implementation history directly readable from the filesystem, not only from git history or the diary.

**Commit (code):** e3041bd — "examples: split personal inbox tutorial into steps"

### What I did

- Moved the first runnable example into `examples/xgoja/23-personal-knowledge-inbox/01-minimal-jsverb/`.
- Added a top-level `examples/xgoja/23-personal-knowledge-inbox/README.md` that explains the step-directory approach.
- Added a top-level `Makefile` that delegates `make smoke` to the current first step.
- Updated the nested step `Makefile` because it moved one directory deeper and now needs `REPO_ROOT := $(abspath ../../../..)`.
- Updated the tutorial design document to describe the per-step corpus layout.
- Extended `reference/02-implementation-logbook.md` with a detailed entry explaining the restructuring and why it improves the tutorial.

### Why

- The final tutorial should not ask readers to infer earlier states from the final files.
- Per-step directories make each chapter independently runnable and reviewable.
- The future workflow becomes simple: copy the previous step directory, add one new concept, validate, document the delta.

### What worked

- The top-level smoke passed after the move:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

- The command delegated to `01-minimal-jsverb`, ran `xgoja doctor`, built the generated binary, and verified the hello verb output.

### What didn't work

- No failures occurred during the restructuring.

### What I learned

- The step-directory layout is a better fit for a tutorial than a single evolving example directory.
- The nested step Makefile needs to compute repository root relative to the step directory, not the tutorial root.
- The top-level Makefile can serve as a stable entry point while individual steps stay self-contained.

### What was tricky to build

- The main sharp edge was path depth. Moving the example from `23-personal-knowledge-inbox/` to `23-personal-knowledge-inbox/01-minimal-jsverb/` changes the relative repository root from `../../..` to `../../../..`.
- The documentation had to be updated so it no longer implies the final server and CLI files live directly at the tutorial root.

### What warrants a second pair of eyes

- Whether step directory names should be numeric plus descriptive, as currently chosen, or nested under a `steps/` directory.
- Whether the top-level `make smoke` should eventually run every step smoke or only the latest step.

### What should be done in the future

- For Chapter 1B, copy `01-minimal-jsverb` to `02-hello-web-server` and add only the HTTP provider/server concept.
- Keep adding logbook entries that explain both the file delta and the teaching reason for each step.

### Code review instructions

- Review the top-level `examples/xgoja/23-personal-knowledge-inbox/README.md` and `Makefile` first.
- Review `examples/xgoja/23-personal-knowledge-inbox/01-minimal-jsverb/` to confirm it remains self-contained.
- Validate with:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

### Technical details

The validated step layout is:

```text
examples/xgoja/23-personal-knowledge-inbox/
  README.md
  Makefile
  01-minimal-jsverb/
    README.md
    Makefile
    xgoja.yaml
    verbs/hello.js
```

## Step 4: Add Step 02 hello-world generated web server

This step copies the Step 01 minimal jsverb example into a second runnable snapshot and adds the HTTP provider. The generated binary still exposes the original `verbs inbox hello` command, but it now also exposes a provider-backed `serve inbox server` command that registers public Express routes and keeps an HTTP listener alive.

The tutorial now has its first visible progression: Step 01 is a generated CLI, and Step 02 is the same generated CLI plus generated HTTP serving. The difference between the two directories is intentionally small enough for a new developer to inspect directly.

### Prompt Context

**User prompt (verbatim):** "i reverted it, continue with 02-hello-web-server"

**Assistant interpretation:** Proceed after the user reverted the accidental command-name edit by creating the next tutorial snapshot, `02-hello-web-server`, based on Step 01 plus a hello-world HTTP server.

**Inferred user intent:** Continue the incremental tutorial implementation with the next smallest validated concept: generated xgoja HTTP serving.

**Commit (code):** 2107ba3 — "examples: add personal inbox hello web server step"

### What I did

- Copied `01-minimal-jsverb` to `02-hello-web-server`.
- Updated Step 02 `xgoja.yaml` to add:
  - the `go-go-goja-http` provider,
  - the `express` runtime module,
  - the provider-backed `serve` command set.
- Extended `verbs/hello.js` in Step 02 with a `server` verb that registers:
  - `GET /`, public text response,
  - `GET /healthz`, public JSON health response.
- Added a Step 02 `README.md` and `Makefile`.
- Updated the top-level tutorial `README.md` and `Makefile` so the top-level smoke runs Step 01 and Step 02.
- Added a detailed logbook entry for the Step 02 implementation.

### Why

- HTTP serving is the next foundational xgoja concept after a CLI-only jsverb.
- Keeping the original `hello` CLI verb in Step 02 demonstrates that provider-backed commands extend the generated binary; they do not replace built-in jsverb commands.
- Public routes are the right first HTTP route shape because auth, sessions, CSRF, and agents would distract from the mechanics of route registration and serving.

### What worked

- Focused Step 02 validation passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server smoke
```

- Top-level validation passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

- The smoke verified both CLI and HTTP behavior.

### What didn't work

- The first Step 02 Makefile made `serve-smoke` depend on `build`, while `smoke` already ran `build`. This caused a redundant second generated build. I removed the extra dependency so `serve-smoke` uses the already-built binary.

### What I learned

- Adding a provider-backed command set requires three linked xgoja changes: provider declaration, runtime module selection, and `commands[].type: provider.command-set`.
- The HTTP provider's `serve` command invokes the selected jsverb to register routes and then owns the long-running listener.
- This step produces a useful tutorial diff against Step 01 because the HTTP-specific additions are isolated.

### What was tricky to build

- The implementation itself was straightforward; the main issue was keeping the step educationally small. It would be easy to add SQLite or auth now, but that would make the Step 01 to Step 02 diff harder to understand.
- Smoke testing a long-running server requires starting the generated binary in the background, polling `/healthz`, asserting the root response, and cleaning up the process reliably.

### What warrants a second pair of eyes

- Whether Step 02 should keep CLI `hello` and HTTP `server` in the same JavaScript file or split them into separate files before Step 03.
- Whether the top-level smoke should always run every implemented step, as it does now, or eventually only run the latest step by default.

### What should be done in the future

- Step 03 should copy Step 02 and add the next concept, likely SQLite-backed inbox state and/or separate generated server and CLI binaries.
- The final tutorial should include a Step 01 to Step 02 diff exercise.

### Code review instructions

- Review `examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server/xgoja.yaml` for the HTTP provider and serve command additions.
- Review `examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server/verbs/hello.js` for the public Express route registration.
- Review `examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server/Makefile` for the server smoke pattern.
- Validate with:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

### Technical details

Important command paths:

```bash
examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server/dist/personal-knowledge-inbox-hello-web-server \
  verbs inbox hello --name tutorial

examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server/dist/personal-knowledge-inbox-hello-web-server \
  serve inbox server --http-listen 127.0.0.1:18790
```

Primary files:

```text
examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server/xgoja.yaml
examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server/verbs/hello.js
examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server/Makefile
examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server/README.md
```

## Step 5: Add SQLite CLI inbox verbs and reusable storage section

This step completes the third runnable tutorial snapshot. Step 03 copies the hello web server step, adds the guarded host database module, and introduces SQLite-backed CLI verbs for capturing, listing, and archiving inbox items. It still has no REST API; the point is to teach local application state and command schemas before moving data behind HTTP routes.

The step also improves the tutorial source itself by adding explanatory comments to each current `xgoja.yaml`. Those comments define the sections as they appear, so new developers can learn the spec from the step files without constantly switching to reference docs.

### Prompt Context

**User prompt (verbatim):** "step 3: add sqlite inbox + CLI verbs (no REST API yet)."

**Additional user prompt (verbatim):** "go over the xgoja.yaml files so far, and each time a new feature / section is introduced, add a docstring above it that gives a concise explanation of each field, and a short schema of what is possible, what is optional, etc... That way people can understand what the sections are for and allow."

**Additional user prompt (verbatim):** "capture verb should fail if title / url not present. (required true). also introduce the concept of a section of parameters that are going to be reused across commands (like the db)"

**Assistant interpretation:** Finish Step 03 by adding local SQLite inbox commands, annotating tutorial YAML files as teaching material, making capture require title and URL, and introducing reusable jsverbs sections for shared database parameters.

**Inferred user intent:** Build a more pedagogical intermediate step that teaches command validation and reusable CLI metadata before adding REST or auth.

**Commit (code):** 67473dd — "examples: add personal inbox sqlite cli step"

### What I did

- Added `03-sqlite-cli-inbox` by copying Step 02.
- Added the host provider and `database` runtime module to Step 03.
- Replaced `hello.js` with `inbox.js` containing:
  - the existing `hello` CLI verb,
  - the existing public `server` route-registration verb,
  - a `storage` section with shared `--db`,
  - `capture`, `list`, and `archive` CLI verbs.
- Made `capture` require both `title` and `url`.
- Added negative smoke checks for missing `--title` and missing `--url`.
- Added `scripts/cli_smoke.py` to keep Makefile validation readable.
- Added YAML teaching comments to Steps 01, 02, and 03.
- Added `.gitignore` for local SQLite files under the tutorial workspace.

### Why

- SQLite gives the tutorial its first durable application state, but keeping access local through CLI verbs avoids introducing REST design too early.
- Required CLI fields make validation visible at the command boundary.
- A reusable `storage` section teaches how shared flags can be declared once and reused across commands.
- Annotated YAML files turn each step directory into self-contained teaching material.

### What worked

- Step 03 smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox/03-sqlite-cli-inbox smoke
```

- Top-level tutorial smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

### What didn't work

- The first Step 03 Makefile embedded a Python heredoc inside a Make recipe and failed with:

```text
Makefile:54: *** missing separator.  Stop.
```

I moved the Python validation code into `scripts/cli_smoke.py`.

- The first Step 03 verb metadata used unsupported output mode `json`, producing:

```text
Error: inbox.js#archive has unsupported output mode "json"
```

I changed the current Step 03 verbs to `output: "text"` with `JSON.stringify(...)` and kept a task to revisit proper structured/Glazed output.

- After making `url` required, the existing smoke's second capture failed because it intentionally had no URL. I updated the smoke so missing title/URL are explicit negative tests, then made both successful captures include URLs.

### What I learned

- `__section__` is the right mechanism for file-local reusable flag groups. Commands opt in with `sections: ["storage"]` and receive values through a bound field such as `storage: { bind: "storage" }`.
- Required jsverbs fields fail before the JavaScript function runs, which is the right behavior for CLI validation.
- `output: "json"` is not a supported jsverbs output alias; structured output needs a separate follow-up.

### What was tricky to build

- The main tricky part was balancing current functionality with future improvements. Proper structured output is desirable, but Step 03's teaching goal is SQLite state and reusable sections. Keeping JSON-as-text for now avoids expanding the scope.
- Make recipes are fragile for multiline embedded scripts. Moving the smoke assertions into a script made the validation easier to read and maintain.
- SQLite files are runtime artifacts. A smoke run created `personal-inbox.sqlite`, which I removed from the commit and prevented with a tutorial-level `.gitignore`.

### What warrants a second pair of eyes

- Whether requiring both `title` and `url` is the right product rule long term; notes without URLs may be useful later, but the current rule is good for teaching required fields.
- Whether Step 03 should introduce structured output now or keep the current follow-up task.
- Whether `allowConfigure: true` is acceptable for this local tutorial step, or whether the next step should show a narrower host-owned database configuration.

### What should be done in the future

- Step 04 should copy Step 03 and move inbox operations behind HTTP routes.
- Revisit proper jsverbs structured/Glazed output for `capture`, `list`, and `archive`.
- Use the `storage` section pattern in later CLI steps for token cache paths and base URLs.

### Code review instructions

- Review `03-sqlite-cli-inbox/xgoja.yaml` for the host provider and database module.
- Review `03-sqlite-cli-inbox/verbs/inbox.js` for the `storage` section, required capture fields, and SQLite helpers.
- Review `03-sqlite-cli-inbox/Makefile` and `scripts/cli_smoke.py` for validation coverage.
- Validate with:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

### Technical details

Primary files:

```text
examples/xgoja/23-personal-knowledge-inbox/01-minimal-jsverb/xgoja.yaml
examples/xgoja/23-personal-knowledge-inbox/02-hello-web-server/xgoja.yaml
examples/xgoja/23-personal-knowledge-inbox/03-sqlite-cli-inbox/xgoja.yaml
examples/xgoja/23-personal-knowledge-inbox/03-sqlite-cli-inbox/verbs/inbox.js
examples/xgoja/23-personal-knowledge-inbox/03-sqlite-cli-inbox/Makefile
examples/xgoja/23-personal-knowledge-inbox/03-sqlite-cli-inbox/scripts/cli_smoke.py
```

## Step 6: Add API server and API client CLI verbs

This step creates the fourth runnable tutorial snapshot. It separates the Step 03 single-file implementation into server entrypoints, client entrypoints, and reusable library modules. The CLI no longer opens SQLite directly; it calls public API routes with the guarded `fetch` module.

The step intentionally keeps the API public. Authentication comes later. The purpose here is to introduce the client/server boundary and make the future security boundary concrete.

### Prompt Context

**User prompt (verbatim):** "step 04: separate out a lib/ for reusable js. separate out server.js for the server. Add API routes to the server, add a client.js for CLI verbs that hit the API."

**Assistant interpretation:** Copy Step 03 into Step 04, split reusable SQLite and fetch code into `lib/`, move HTTP route registration into `server.js`, and move CLI behavior into `client.js` where commands call the API instead of direct SQLite.

**Inferred user intent:** Teach the API boundary before adding authentication/device login, while preserving the per-step runnable tutorial structure.

**Commit (code):** 0263363 — "examples: add personal inbox API client-server step"

### What I did

- Added `04-api-client-server` as a new step directory.
- Added `verbs/server.js` with public API routes:
  - `GET /`,
  - `GET /healthz`,
  - `GET /api/inbox`,
  - `POST /api/capture`,
  - `POST /api/inbox/:id/archive`.
- Added `verbs/client.js` with API client CLI verbs:
  - `verbs inboxctl capture`,
  - `verbs inboxctl list`,
  - `verbs inboxctl archive`.
- Added reusable libraries:
  - `verbs/lib/inbox_store.js`,
  - `verbs/lib/api_client.js`.
- Added the guarded `fetch` runtime module to Step 04.
- Updated the Step 04 smoke to start the API server, run client verbs against it, and validate persisted state.
- Updated the top-level tutorial README and Makefile to include Step 04.

### Why

- The tutorial needs a clear transition from local CLI state to client/server architecture before auth is introduced.
- Separating `server.js`, `client.js`, and `lib/` makes code ownership clearer for new developers.
- The API boundary is the future authorization boundary. This step lets later chapters say: now we protect these routes.

### What worked

- Focused Step 04 smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox/04-api-client-server smoke
```

- Top-level tutorial smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

### What didn't work

- No validation failures occurred after the initial implementation.
- I removed the copied Step 03 `cli_smoke.py` from Step 04 before finalizing the commit, because Step 04 uses `api_smoke.py` instead.

### What I learned

- Helper files under `lib/` should be included in the jsverbs source set even when they do not declare verbs. They need to be embedded so local `require("./lib/...")` works in the generated binary.
- Step 04 cleanly separates server-owned `--db` from client-owned `--base-url`, which will make later auth/token-cache sections easier to teach.
- Public API routes still need server-side validation. CLI required fields are useful UX but not a trust boundary.

### What was tricky to build

- The main design choice was whether Step 04 should split into two generated binaries immediately. I kept one binary for now because the requested concept was API/client separation in JavaScript, not binary packaging. Later device-login steps can split server and CLI binaries when that distinction becomes educationally useful.
- The source include list needed to include both verb files and helper files without accidentally treating helper files as commands. The existing jsverbs scanner tolerates helper files without `__verb__` declarations.

### What warrants a second pair of eyes

- Whether the package names `inbox` for server and `inboxctl` for client are the right command UX before the broader xgoja verb-mounting follow-up is addressed.
- Whether Step 05 should split server and client into separate binaries, or wait until device login/token cache makes that split necessary.

### What should be done in the future

- Step 05 should add either a minimal browser UI or unauthenticated REST route cleanup before auth.
- Later steps should protect these API routes with session and programmatic auth.
- Revisit structured output so client commands can return proper Glazed rows instead of JSON text.

### Code review instructions

- Review `04-api-client-server/xgoja.yaml` for the `fetch` module and multi-file source set.
- Review `verbs/server.js` for public API route shape and server-side validation.
- Review `verbs/client.js` and `verbs/lib/api_client.js` for guarded fetch usage.
- Review `verbs/lib/inbox_store.js` for reusable SQLite logic.
- Validate with:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

### Technical details

Primary files:

```text
examples/xgoja/23-personal-knowledge-inbox/04-api-client-server/xgoja.yaml
examples/xgoja/23-personal-knowledge-inbox/04-api-client-server/verbs/server.js
examples/xgoja/23-personal-knowledge-inbox/04-api-client-server/verbs/client.js
examples/xgoja/23-personal-knowledge-inbox/04-api-client-server/verbs/lib/inbox_store.js
examples/xgoja/23-personal-knowledge-inbox/04-api-client-server/verbs/lib/api_client.js
examples/xgoja/23-personal-knowledge-inbox/04-api-client-server/Makefile
examples/xgoja/23-personal-knowledge-inbox/04-api-client-server/scripts/api_smoke.py
```

## Step 7: Add embedded retro browser UI

This step creates the fifth runnable tutorial snapshot by copying Step 04 and adding embedded browser assets. The server still exposes the same public API and the CLI still calls that API with guarded fetch; the new concept is static asset embedding and serving from the generated xgoja binary.

The visual implementation follows the requested direction: a monochrome, retro Macintosh-influenced page without menu bars, window chrome, or Chicago-style fonts. The CSS uses modern system Swiss typography, restrained weights, thin rules instead of heavy panels, and muted foreground-only accent colors.

### Prompt Context

**User prompt (verbatim):** "Add embedded HTML + CSS assets styled in a retro monochrome macOS 1-inspired look: no menu bar, no window chrome, no Chicago font. Use modern clean Swiss typography, restrained font sizes/weights, minimal boxes. Muted accent colors are allowed only for foreground text accents."

**Assistant interpretation:** Continue the Personal Knowledge Inbox tutorial by adding a new step that embeds and serves static HTML/CSS/JS assets, with a restrained monochrome retro UI that avoids literal Mac window chrome or Chicago fonts.

**Inferred user intent:** Introduce embedded assets and a browser-facing surface before adding auth, while keeping the tutorial's visual language polished, minimal, and appropriate for the requested design constraints.

**Commit (code):** f3e6bca — "examples: add personal inbox embedded UI step"

### What I did

- Copied `04-api-client-server` to `05-embedded-retro-ui`.
- Added embedded assets under `assets/public/`:
  - `index.html`,
  - `styles.css`,
  - `app.js`.
- Updated Step 05 `xgoja.yaml` to add:
  - an `fs:assets` runtime module with an embedded mount,
  - a `ui-assets` asset source,
  - an `embedded-assets` artifact.
- Updated `verbs/server.js` so `/` returns the embedded HTML and `/static/*` serves embedded CSS/JS.
- Updated the Step 05 smoke test to verify `/`, `/static/styles.css`, `/static/app.js`, `/healthz`, and the existing API/client flow.
- Updated the top-level tutorial README and Makefile to include Step 05.

### Why

- The tutorial needs a browser-facing app surface before it teaches browser sessions and device login.
- Embedded assets keep the generated binary self-contained and mirror how a small production tool can ship a UI without a separate frontend build pipeline.
- The UI gives later auth steps concrete pages and flows to protect.

### What worked

- Focused Step 05 smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox/05-embedded-retro-ui smoke
```

- Top-level tutorial smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

### What didn't work

- No validation failures occurred in this step.

### What I learned

- The existing xgoja embedded asset path is a good fit for tutorial UI chapters: declare an `assets` source, mount it through `fs:assets`, and use `app.staticFromAssetsModule` for browser files.
- The generated binary can keep serving the existing API while adding a static UI with a very small JavaScript layer.
- The visual requirements were easier to preserve by using typographic hierarchy and horizontal rules, rather than simulating a Macintosh desktop or window frame.

### What was tricky to build

- The main design constraint was avoiding literal retro UI elements while still making the page feel intentionally monochrome and early-computing-inspired. I kept the retro reference in the density, rule work, and simple forms, while using modern system fonts and no chrome.
- The asset source and runtime module are two separate pieces: `sources` embeds files into the generated build, while the `fs:assets` runtime mount is what JavaScript uses to read and serve them.

### What warrants a second pair of eyes

- Whether the CSS strikes the right balance between retro austerity and modern readability.
- Whether Step 05 should keep the API public, as currently implemented, or if the next step should immediately introduce browser session protection.
- Whether the smoke should also use a browser automation check later, once the UI grows beyond simple fetch calls.

### What should be done in the future

- Step 06 should introduce browser/session routes or the first auth boundary around the UI/API.
- Later steps should teach device login and programmatic capture tokens using this browser surface as the approval/interaction context.
- Consider adding a screenshot-based visual regression only if the tutorial UI becomes stable enough to justify it.

### Code review instructions

- Review `05-embedded-retro-ui/xgoja.yaml` for the embedded asset source, `fs:assets` mount, and `embedded-assets` artifact.
- Review `05-embedded-retro-ui/verbs/server.js` for static asset mounting and root HTML serving.
- Review `05-embedded-retro-ui/assets/public/styles.css` against the visual constraints.
- Validate with:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

### Technical details

Primary files:

```text
examples/xgoja/23-personal-knowledge-inbox/05-embedded-retro-ui/xgoja.yaml
examples/xgoja/23-personal-knowledge-inbox/05-embedded-retro-ui/verbs/server.js
examples/xgoja/23-personal-knowledge-inbox/05-embedded-retro-ui/assets/public/index.html
examples/xgoja/23-personal-knowledge-inbox/05-embedded-retro-ui/assets/public/styles.css
examples/xgoja/23-personal-knowledge-inbox/05-embedded-retro-ui/assets/public/app.js
examples/xgoja/23-personal-knowledge-inbox/05-embedded-retro-ui/Makefile
examples/xgoja/23-personal-knowledge-inbox/README.md
examples/xgoja/23-personal-knowledge-inbox/Makefile
```

## Step 8: Add local Keycloak browser login

This step creates the sixth runnable tutorial snapshot. It introduces the first human login boundary with local Keycloak as the OIDC identity provider, while keeping human credentials outside xgoja. The generated host owns app sessions, CSRF checks, and app-local user upsert after OIDC login.

During the step, the CLI commands changed direction. Once the browser API became session-protected, the fetch-backed CLI from Step 04/05 no longer made sense as the default local CLI path. Step 06 therefore replaces those CLI verbs with direct SQLite access again, leaving the protected HTTP API for browser sessions and future device/programmatic credentials.

### Prompt Context

**User prompt (verbatim):** "ok, create 2 users: alice and bob. go ahead with step 06."

**Assistant interpretation:** Add the next tutorial snapshot with local Keycloak OIDC login, seed two tutorial users, and validate generated hostauth login wiring.

**Inferred user intent:** Teach how first human users log in without xgoja owning passwords, and prepare the browser session foundation required for later device authorization approval.

**Commit (code):** 5c259f6 — "examples: add personal inbox keycloak login step"

### What I did

- Added `06-browser-login-keycloak` as a new runnable tutorial step.
- Added `compose.yaml` for local Keycloak.
- Added `keycloak/realm-personal-inbox.json` with:
  - realm `personal-inbox`,
  - OIDC client `personal-inbox-local`,
  - user `alice` / `alice-password`,
  - user `bob` / `bob-password`.
- Added hostauth provider and top-level OIDC `auth:` config in Step 06 `xgoja.yaml`.
- Protected `/api/inbox`, `/api/capture`, and `/api/inbox/:id/archive` with `express.sessionUser()`.
- Added `.csrf()` to unsafe browser mutation routes.
- Updated the browser UI to call `/auth/session`, show login state, and send `X-CSRF-Token` for mutations/logout.
- Replaced API-fetch CLI verbs with direct SQLite verbs that share `verbs/lib/inbox_store.js`.
- Added `make keycloak-smoke` to start Keycloak, verify the login redirect, and verify unauthenticated API requests return `401`.

### Why

- xgoja should not manage human passwords; OIDC/Keycloak owns user credentials.
- The app still needs app-local users and sessions, which hostauth/appauth creates after OIDC login.
- Device authorization later requires an existing browser session to approve CLI/device requests.
- The CLI cannot call session+CSRF browser APIs unless it has a browser session; direct local CLI verbs are clearer until device/access-token auth is introduced.

### What worked

- Fast Step 06 smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak smoke
```

- Keycloak/OIDC smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak keycloak-smoke
```

- Full tutorial smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

### What didn't work

- The first Step 06 fast smoke failed because selecting the hostauth `auth` JavaScript runtime module made direct CLI verbs try to construct `require("auth")` without hostauth services:

```text
Error: register module "xgoja:go-go-goja-hostauth.auth:auth": create module go-go-goja-hostauth.auth: auth module requires hostauth services
```

I fixed this by keeping the `go-go-goja-hostauth` provider and top-level `auth:` config, but not selecting the `auth` JavaScript module in this step. Native auth endpoints still mount through the generated serve host.

- The first Keycloak smoke failed with a planned route chain error after adding `.auth(express.sessionUser())` without an explicit authorization action:

```text
Error: TypeError: Object has no member 'handle' at server (/server.js:61:12(72))
```

I fixed this by adding `.allow("user.self.read")` before `.audit(...).handle(...)` on the session-protected API routes.

- The next Keycloak smoke failed because the smoke used `curl -I /auth/login`, but the native login endpoint does not accept `HEAD`:

```text
curl: (22) The requested URL returned error: 405
```

I fixed the smoke to use `GET` with `curl -D headers -o /dev/null`.

### What I learned

- Generated hostauth can mount native auth routes without exposing the JavaScript `auth` module to every command.
- Session-protected planned routes should declare an explicit `.allow(...)` action before audit/handler chaining.
- Once an API is browser-session-only, CLI access should either be direct local access or a real programmatic credential flow; using unauthenticated fetch would teach the wrong boundary.

### What was tricky to build

- The main boundary issue was separating host-level auth services from JavaScript module exposure. The top-level `auth:` block enables generated serve-time hostauth, while `runtime.modules[].name: auth` exposes JavaScript auth builders. Direct CLI verbs do not have serve-time hostauth services, so this step must not load that module globally.
- The tutorial also has two useful validation layers: fast smoke that avoids Docker, and Keycloak smoke that proves the local OIDC redirect and protected API behavior.

### What warrants a second pair of eyes

- Whether using `.allow("user.self.read")` as the temporary authorization action for inbox API routes is acceptable until app-specific actions/resources are introduced.
- Whether direct CLI verbs should remain under `inboxctl`, or be renamed in a later UX cleanup once programmatic/device auth is added.
- Whether the next step should add first-owner bootstrap or go directly to device authorization.

### What should be done in the future

- Add a browser-login Playwright smoke that logs in as Alice and Bob, not just a redirect/protected-route smoke.
- Add first-owner/bootstrap semantics so the first logged-in user owns a personal inbox tenant.
- Introduce device authorization after browser login is fully interactive.

### Code review instructions

- Review `06-browser-login-keycloak/xgoja.yaml` for hostauth/OIDC config and the absence of the JavaScript `auth` module.
- Review `06-browser-login-keycloak/keycloak/realm-personal-inbox.json` for Alice/Bob seed users and OIDC client redirect URI.
- Review `06-browser-login-keycloak/verbs/server.js` for session-only API routes and CSRF on unsafe methods.
- Review `06-browser-login-keycloak/verbs/client.js` for direct SQLite CLI commands.
- Validate with:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak smoke
make -C examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak keycloak-smoke
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

### Technical details

Primary files:

```text
examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/xgoja.yaml
examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/compose.yaml
examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/keycloak/realm-personal-inbox.json
examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/verbs/server.js
examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/verbs/client.js
examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/assets/public/app.js
examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/Makefile
```

## Step 9: Add user-scoped inbox ownership

This step creates the seventh runnable tutorial snapshot. It keeps the Step 06 Keycloak login foundation, but changes the browser API from “any logged-in user can see the global inbox” to “each logged-in user sees only rows owned by that session user.”

That is the next security lesson after login: authentication gives the application an identity, but handlers must still scope application data by that identity or by an app-owned tenant/resource boundary. Step 07 uses the simplest possible boundary, `ctx.actor.id`, before later device/programmatic credentials inherit the same boundary.

### Prompt Context

**User prompt (verbatim):** "go ahead."

**Assistant interpretation:** Implement the proposed Step 07 user-scoped inbox tutorial snapshot.

**Inferred user intent:** Continue the tutorial by making authenticated identity meaningful for application data ownership before adding device authorization.

**Commit (code):** 5d10630 — "examples: add personal inbox user-scoped step"

### What I did

- Added `07-user-scoped-inbox` as a new runnable tutorial step copied from Step 06.
- Moved Step 07 to its own local ports:
  - app: `127.0.0.1:18794`,
  - Keycloak: `127.0.0.1:18087`.
- Updated Step 07 Keycloak realm redirects and post-logout redirect settings for port `18794`.
- Updated `verbs/lib/inbox_store.js` with:
  - `listInboxItemsForUser(database, userID, includeArchived)`,
  - `archiveInboxItemForUser(database, id, userID)`.
- Updated `verbs/server.js` so browser API routes use `ctx.actor.id`:
  - capture stores `submittedByKind: "sessionUser"` and `submittedById: ctx.actor.id`,
  - list returns only `sessionUser` rows for `ctx.actor.id`,
  - archive updates only rows for `ctx.actor.id`.
- Updated the UI copy and owner label to explain the user-scoped inbox.
- Updated the tutorial root README and Makefile so top-level smoke includes Step 07.

### Why

- Login alone is not a data-isolation policy.
- The tutorial needs an explicit bridge from OIDC identity to application-owned data boundaries.
- User-scoped rows are easier to teach than tenants/resources before the device authorization chapters.
- Future device/access-token steps can request or inherit access to this same user-owned inbox boundary.

### What worked

- Step 07 fast smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox/07-user-scoped-inbox smoke
```

- Step 07 Keycloak smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox/07-user-scoped-inbox keycloak-smoke
```

- Full tutorial smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

### What didn't work

- No validation failures occurred while implementing Step 07.

### What I learned

- The Step 06 route protection was necessary but incomplete: list/archive still operated on global rows until the handler queries included `ctx.actor.id`.
- A per-user row filter is the smallest useful authorization/data ownership concept before adding richer app tenants/resources.
- Keeping Step 07 on separate ports from Step 06 makes both snapshots easier to run independently during manual review.

### What was tricky to build

- The main sharp edge was preserving the direct CLI behavior while changing browser behavior. Direct CLI commands still use local SQLite for tutorial/debugging access; browser API routes now use session-scoped access.
- The route action remains the generic `user.self.read` action. That is acceptable for this teaching step, but app-specific actions should be introduced when the tutorial adds richer authorization semantics.

### What warrants a second pair of eyes

- Whether Step 07 should use a dedicated `inbox.read` / `inbox.capture` / `inbox.archive` action set instead of temporary `user.self.read`.
- Whether archive should report when no row was affected. The current database helper returns `{id, archivedAt}` even if the row did not belong to the user.
- Whether Playwright should be added next to prove Alice/Bob isolation end-to-end through the browser.

### What should be done in the future

- Add a browser automation smoke that logs in as Alice and Bob and proves row isolation visually/API-wise.
- Introduce app-specific grants/actions before device/programmatic credentials.
- Add device authorization so a CLI can obtain scoped tokens for the logged-in user's inbox.

### Code review instructions

- Review `07-user-scoped-inbox/verbs/server.js` for `ctx.actor.id` usage in list/capture/archive.
- Review `07-user-scoped-inbox/verbs/lib/inbox_store.js` for owner-filtered SQL helpers.
- Review `07-user-scoped-inbox/xgoja.yaml` for Step 07 ports and command-set listen defaults.
- Validate with:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox/07-user-scoped-inbox smoke
make -C examples/xgoja/23-personal-knowledge-inbox/07-user-scoped-inbox keycloak-smoke
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

### Technical details

Primary files:

```text
examples/xgoja/23-personal-knowledge-inbox/07-user-scoped-inbox/xgoja.yaml
examples/xgoja/23-personal-knowledge-inbox/07-user-scoped-inbox/verbs/server.js
examples/xgoja/23-personal-knowledge-inbox/07-user-scoped-inbox/verbs/lib/inbox_store.js
examples/xgoja/23-personal-knowledge-inbox/07-user-scoped-inbox/assets/public/app.js
examples/xgoja/23-personal-knowledge-inbox/07-user-scoped-inbox/README.md
examples/xgoja/23-personal-knowledge-inbox/Makefile
examples/xgoja/23-personal-knowledge-inbox/README.md
```

## Step 10: Add device authorization for programmatic capture

This step creates the eighth runnable tutorial snapshot. It keeps the user-scoped browser inbox from Step 07 and adds a device authorization flow so a CLI can request programmatic access without receiving the user's browser cookie or password.

The browser session remains the approval authority. A CLI starts device authorization and receives a raw `device_code` plus human-readable `user_code`; the logged-in browser user approves the `user_code`; the CLI polls for access/refresh tokens; and the access token can call an agent-only programmatic capture route.

### Prompt Context

**User prompt (verbatim):** "cool, it works manually. continue"

**Assistant interpretation:** Continue the tutorial after the user manually confirmed Step 07 by adding the next logical feature: device authorization and a token-authenticated programmatic capture path.

**Inferred user intent:** Move from browser-only user-scoped data access toward the tutorial's core device-login/programmatic-token acquisition goal.

**Commit (code):** 2588034 — "examples: add personal inbox device authorization step"

### What I did

- Added `08-device-authorization` as a new runnable tutorial step copied from Step 07.
- Moved Step 08 to separate local ports:
  - app: `127.0.0.1:18795`,
  - Keycloak: `127.0.0.1:18088`.
- Added guarded `fetch` runtime config for CLI calls to native device endpoints.
- Added `verbs/lib/device_client.js` for device start, token polling, and token-authenticated capture.
- Added CLI verbs:
  - `verbs inboxctl device-start`,
  - `verbs inboxctl device-token`,
  - `verbs inboxctl token-capture`.
- Added browser UI for device approval via `/auth/device/approve` with CSRF.
- Added `/api/programmatic/capture`, protected with `express.agent()` and `.allow("user.self.read")`.
- Programmatic capture stores rows under the approving user's owner id from the device-created agent (`ctx.actor.claims.ownerUserId`).
- Updated top-level tutorial README and Makefile to include Step 08.

### Why

- The tutorial is about device login and programmatic token acquisition; Step 08 introduces those concepts after browser login and user-scoped data boundaries are already established.
- Device authorization keeps human credentials in Keycloak and keeps browser sessions separate from CLI credentials.
- The programmatic route demonstrates why access tokens are route credentials and why refresh tokens/device codes are not.

### What worked

- Step 08 fast smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox/08-device-authorization smoke
```

- Step 08 Keycloak/device smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox/08-device-authorization keycloak-smoke
```

- Full tutorial smoke passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

### What didn't work

- No validation failures occurred while implementing Step 08.

### What I learned

- The native device endpoints can be introduced without exposing the JavaScript `auth` module; the generated host mounts them from hostauth services.
- A device-created agent carries `ownerUserId`, which is the bridge back to the approving browser user for user-scoped programmatic capture.
- A useful first automated device smoke can verify start and pending poll before adding full browser automation for approval.

### What was tricky to build

- The route boundary had to stay explicit: browser routes use `express.sessionUser()` and CSRF, while programmatic capture uses `express.agent()` and bearer-token auth.
- The CLI now has two categories of verbs: direct SQLite tutorial/debug verbs and HTTP device/programmatic verbs. The README calls this out so readers do not confuse local DB access with production-style API access.

### What warrants a second pair of eyes

- Whether `user.self.read` is the right temporary grant for programmatic capture, or whether Step 09 should introduce app-specific actions such as `inbox.capture`.
- Whether the programmatic capture route should return less actor/auth detail once the tutorial moves past diagnostics.
- Whether token persistence should be added before refresh-token rotation is taught.

### What should be done in the future

- Add a Playwright/device integration smoke that logs in, approves a user code, polls tokens, and captures a row end-to-end.
- Introduce app-specific grants/actions for inbox operations.
- Add token cache and refresh-token usage for the CLI.

### Code review instructions

- Review `08-device-authorization/xgoja.yaml` for `fetch` config and hostauth settings.
- Review `08-device-authorization/verbs/client.js` and `verbs/lib/device_client.js` for device CLI behavior.
- Review `08-device-authorization/verbs/server.js` for `express.agent()` and owner-user scoping.
- Review `08-device-authorization/assets/public/app.js` for browser approval and CSRF handling.
- Validate with:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox/08-device-authorization smoke
make -C examples/xgoja/23-personal-knowledge-inbox/08-device-authorization keycloak-smoke
make -C examples/xgoja/23-personal-knowledge-inbox smoke
```

### Technical details

Primary files:

```text
examples/xgoja/23-personal-knowledge-inbox/08-device-authorization/xgoja.yaml
examples/xgoja/23-personal-knowledge-inbox/08-device-authorization/verbs/client.js
examples/xgoja/23-personal-knowledge-inbox/08-device-authorization/verbs/lib/device_client.js
examples/xgoja/23-personal-knowledge-inbox/08-device-authorization/verbs/server.js
examples/xgoja/23-personal-knowledge-inbox/08-device-authorization/assets/public/app.js
examples/xgoja/23-personal-knowledge-inbox/08-device-authorization/Makefile
examples/xgoja/23-personal-knowledge-inbox/08-device-authorization/README.md
```

## Step 11: Promote tinyidp smoke for the first browser-login step

This step promotes the tinyidp proof from the Obsidian handoff article into the active `go-go-goja` tutorial tree. It starts with Step 06 because that is the first personal-inbox step that uses Keycloak/OIDC and it isolates generated hostauth browser login before later user-scoping and device-authorization concepts.

The new target keeps Keycloak available as the compatibility check, but adds a faster mock-IdP path that runs without Docker Compose Keycloak. The smoke uses tinyidp as a standards-shaped OIDC issuer, drives the real browser-facing authorization-code flow through an HTML form, and asserts that the generated app creates a normal xgoja app session.

### Prompt Context

**User prompt (verbatim):** "Read \" /home/manuel/code/wesen/go-go-golems/go-go-parc/Projects/2026/06/22/ARTICLE - tinyidp as a Keycloak Replacement for go-go-goja Auth Testing.md      \" and set up tinyidp as a keycloak replacement in our current tests, going back to the first keycloak use."

**Assistant interpretation:** Read the prior tinyidp replacement report, then move its working Step 06 proof into the active personal-inbox tutorial tests as a maintained target alongside Keycloak.

**Inferred user intent:** Make local OIDC smoke tests faster and lighter by using tinyidp for the first Keycloak-backed tutorial step while preserving Keycloak as a real-provider compatibility target.

**Commit (code):** 874993d — "examples: add tinyidp smoke for personal inbox login"

### What I did

- Read the Obsidian article describing the previous tinyidp replacement proof and its root-issuer caveat.
- Added `scripts/tinyidp_login_smoke.py` to Step 06.
- Added a Step 06 `tinyidp-smoke` Makefile target.
- Added top-level personal-inbox `make tinyidp-smoke`, which delegates to Step 06.
- Updated Step 06 and top-level tutorial README files to document the target, `TINYIDP_ROOT`, and the root issuer URL shape.
- Left the existing `keycloak-smoke` target intact.

### Why

- Step 06 is the first personal-inbox Keycloak/OIDC use and the smallest generated hostauth login target.
- tinyidp can prove the OIDC relying-party behavior without starting a Keycloak container.
- Keeping both targets lets tinyidp serve as the fast local smoke while Keycloak remains the slower compatibility check.

### What worked

- The promoted top-level target passed:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox tinyidp-smoke
```

- The script completed `/auth/login` → tinyidp `/authorize` form → app `/auth/callback` → `/auth/session` and printed:

```text
ok tinyidp step06 full login smoke; session email=alice@example.test
ok tinyidp replacement smoke
```

### What didn't work

- No implementation failure occurred in this step. The known limitation from the handoff article remains: the smoke uses a root tinyidp issuer (`http://127.0.0.1:19087`), not a Keycloak realm-path issuer (`/realms/personal-inbox`).

### What I learned

- The existing generated hostauth OIDC client is provider-generic enough for tinyidp; no xgoja OIDC client code changes were needed.
- A standard-library Python login driver is sufficient for this smoke because tinyidp's login page is simple and the generated app's callback/session behavior is server-side.

### What was tricky to build

- Process cleanup needed care: the Makefile target starts both tinyidp and the generated app, so the trap now avoids assuming both PIDs are initialized before failures.
- The target intentionally uses a separate app port (`19794`) and IdP port (`19087`) to avoid collisions with the Keycloak tutorial ports.

### What warrants a second pair of eyes

- Whether `TINYIDP_ROOT` should remain an absolute local default or move to a repository/tooling convention once tinyidp is installed as a normal binary.
- Whether Step 07 should reuse the same smoke after Step 06 proves hostauth login, or whether Step 07 should wait for an Alice/Bob isolation browser smoke.
- Whether tinyidp should gain Keycloak realm-path compatibility before more examples are migrated.

### What should be done in the future

- Add tinyidp base-path/realm-path support in the tinyidp repo if tests need Keycloak-looking issuer URLs.
- Consider adding tinyidp smoke for Step 07 once it can assert user-scoped isolation end-to-end.
- Keep Step 08 on Keycloak or native hostauth device tests until tinyidp implements device authorization.

### Code review instructions

- Start with `examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/Makefile`, especially the `tinyidp-smoke` process startup and cleanup.
- Review `examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/scripts/tinyidp_login_smoke.py` for the OIDC form-driving assertions.
- Review README updates for the root-issuer caveat.
- Validate with:

```bash
make -C examples/xgoja/23-personal-knowledge-inbox tinyidp-smoke
```

### Technical details

The tinyidp command shape is:

```bash
GOWORK=off go run ./cmd/tinyidp serve \
  --addr 127.0.0.1:19087 \
  --issuer http://127.0.0.1:19087 \
  --client-id personal-inbox-local \
  --redirect-uris http://127.0.0.1:19794/auth/callback
```

The generated app is started with matching OIDC flags and a separate SQLite auth store:

```bash
./dist/personal-knowledge-inbox-browser-login-keycloak serve inbox server \
  --http-listen 127.0.0.1:19794 \
  --auth-oidc-issuer-url http://127.0.0.1:19087 \
  --auth-oidc-client-id personal-inbox-local \
  --auth-oidc-public-base-url http://127.0.0.1:19794 \
  --auth-session-cookie-allow-insecure-http=true \
  --auth-default-store-driver sqlite \
  --auth-default-store-apply-schema=true
```
