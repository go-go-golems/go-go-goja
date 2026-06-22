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
