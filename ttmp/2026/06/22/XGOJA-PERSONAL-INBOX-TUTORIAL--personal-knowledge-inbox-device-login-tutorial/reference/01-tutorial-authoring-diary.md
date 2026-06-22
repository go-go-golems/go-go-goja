---
DocType: reference
Title: Tutorial Authoring Diary
Ticket: XGOJA-PERSONAL-INBOX-TUTORIAL
Status: active
Intent: long-term
Topics:
  - xgoja
  - auth
  - security
  - examples
  - jsverbs
  - documentation
Created: 2026-06-22
Updated: 2026-06-22
RelatedFiles:
  - /home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/ttmp/2026/06/22/XGOJA-PERSONAL-INBOX-TUTORIAL--personal-knowledge-inbox-device-login-tutorial/design/01-personal-knowledge-inbox-tutorial.md:Primary tutorial draft
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
