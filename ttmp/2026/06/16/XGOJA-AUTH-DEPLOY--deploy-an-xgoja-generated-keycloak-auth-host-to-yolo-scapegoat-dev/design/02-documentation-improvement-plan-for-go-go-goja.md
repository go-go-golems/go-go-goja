---
Title: Documentation improvement plan for go-go-goja
Ticket: XGOJA-AUTH-DEPLOY
Status: active
Topics:
    - goja
    - documentation
    - xgoja
    - gojahttp
    - auth
    - keycloak
    - deployment
    - gitops
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: README.md
      Note: |-
        Has no pointer to either help tree (37 + 10 pages undiscoverable)
        Has no pointer to either help tree (47 pages undiscoverable)
    - Path: cmd/goja-repl/root.go
      Note: Wires pkg/doc help into goja-repl via newSharedHelpSystem + SetupCobraRootCommand
    - Path: cmd/xgoja/doc/18-go-planned-auth-api.md
      Note: |-
        Existing Go-side planned auth API doc (in tree 2) — refines the gap analysis
        Existing Go-side planned auth API doc (tree 2) — refines the gap analysis
    - Path: cmd/xgoja/doc/doc.go
      Note: |-
        Embeds cmd/xgoja/doc/* into the xgoja help system (tree 2 loader)
        Tree 2 loader — separate embed.FS served by xgoja only
    - Path: cmd/xgoja/root.go
      Note: Wires cmd/xgoja/doc help into xgoja
    - Path: pkg/doc/29-express-auth-user-guide.md
      Note: Existing JS-side planned auth guide (tree 1) — not cross-linked to tree 2
    - Path: pkg/doc/doc.go
      Note: |-
        Embeds pkg/doc/* into the goja-repl help system (tree 1 loader)
        Tree 1 loader (//go:embed *
    - Path: pkg/xgoja/hostauth/config.go
      Note: hostauth.Config — rich schema with zero user-facing reference doc
    - Path: reference/02-research-logbook.md
      Note: Research logbook whose Part H produced the navigation findings this plan extends
ExternalSources: []
Summary: 'Intern-facing analysis, design, and implementation guide for closing the go-go-goja documentation gaps discovered during the XGOJA-AUTH-DEPLOY investigation: the two-tree help system, the JS-documented/Go-underserved asymmetry, the missing hostauth/stores/deployment references, and the README navigation map.'
LastUpdated: 2026-06-16T10:45:00-04:00
WhatFor: Use when planning or executing doc-cleanup work across go-go-goja so the next engineer does not have to re-derive the doc landscape by hand.
WhenToUse: Before writing any new pkg/doc or cmd/xgoja/doc page, and before editing README.md for navigation.
---


# Documentation improvement plan for go-go-goja

## 1. Executive summary

This document is the write-up of the documentation-improvement findings from the
`XGOJA-AUTH-DEPLOY` investigation (recorded in Part H of
`reference/02-research-logbook.md` and in the doc-plan discussion that
followed). It is written for a new intern: it explains how documentation
currently works in `go-go-goja`, where the gaps are, and exactly which new
documents and README edits to make.

The headline finding is more nuanced than "the Go side is undocumented." During
this analysis we discovered that `go-go-goja` has **two separate Glazed help
trees**, served by different binaries:

1. **`pkg/doc/`** (37 pages) — loaded by `goja-repl`, `jsverbs-example`, and the
   in-REPL help resolver. This is the "module and JS authoring" tree. It
   contains the well-developed Express-auth **JavaScript** trilogy (`29`, `30`,
   `31`).
2. **`cmd/xgoja/doc/`** (10 pages) — loaded only by the `xgoja` binary. This is
   the "code generation and Go host" tree. It already contains a solid Go-side
   planned-auth API page (`18-go-planned-auth-api.md`).

The real problem is therefore **threefold**:

- **Asymmetry of depth, not of existence.** The JS route DSL (`29/30/31`) is
  documented end-to-end; the Go planned-auth *route* API (`cmd/xgoja/doc/18`)
  is documented; but the **host-composition integration story** (how to build a
  full binary that wires OIDC + Postgres stores + the host, i.e. "how to promote
  example 19") is documented only inside example `main.go` files.
- **Missing references.** `hostauth.Config` (a rich YAML schema), the auth
  store drivers, and the OIDC `ModeOIDC` hard-stop have no user-facing
  reference page at all.
- **Zero discoverability.** The two trees are not cross-linked, `README.md`
  does not mention either tree, and the 47 total help pages are reachable only
  by `goja-repl help` / `xgoja help` or by `ls`. An intern reading
  `pkg/doc/29-express-auth-user-guide.md` has no path to discover
  `cmd/xgoja/doc/18-go-planned-auth-api.md`.

This plan proposes (a) a small set of new reference/tutorial pages that fill the
real depth gaps, and (b) a navigation layer (README "Documentation" map +
cross-links) that makes the existing 47 pages discoverable. The navigation work
is the highest-leverage, lowest-effort part.

## 2. Problem statement and scope

### 2.1 Problem

During the deploy investigation we had to reverse-engineer, by reading source
and example `main.go` files, several things that should have had a doc page:

- how to compose `gojahttp.NewHost` with the full auth stack (example 19's
  `main.go`, ~250 lines, was the only source);
- what `hostauth.Config` accepts (`pkg/xgoja/hostauth/config.go`);
- that `auth.mode=oidc` is an explicit hard-stop (`resolve.go` error), not
  merely undocumented;
- how the `serve` command builds its host (`pkg/xgoja/providers/http/serve.go`);
- and how to deploy any of it (now in `design/01-...`).

We also found that `pkg/doc/` and `cmd/xgoja/doc/` are never linked together,
and `README.md` never tells a reader that 47 Glazed help pages even exist.

### 2.2 In scope

- Map and explain the two-tree Glazed help system.
- Produce a refined, accurate gap analysis (correcting the earlier "Go side
  undocumented" claim).
- Specify the new doc pages to write and the README/cross-link edits to make,
- with file-level guidance and acceptance criteria.

### 2.3 Out of scope

- The actual *deployment* design (that is `design/01-...`).
- The `ModeOIDC` *implementation* (that is GitHub issue #82 / `XGOJA-AUTH-KEYCLOAK-MFA`).
- Writing the docs themselves — this plan specifies them; writing is the
  implementation phase.

### 2.4 Relationship to prior work in this ticket

- `reference/02-research-logbook.md` Part H lists seven README/navigation
  improvements (H1–H7). This plan adopts H1–H3 (in-repo) verbatim and refines
  them, and records H4–H7 (out-of-repo) for cross-repo follow-up.
- The earlier doc-plan discussion proposed docs `32`–`37`. This plan corrects
  that numbering: because `cmd/xgoja/doc/18` already exists, several proposed
  pages merge or move. See §5.

## 3. Current-state architecture: the two-tree help system

### 3.1 How Glazed help pages work

`go-go-goja` uses the Glazed help system (`github.com/go-go-golems/glazed/pkg/help`).
Each page is a Markdown file with YAML frontmatter that declares `Slug`,
`Title`, `Short`, `Topics`, `SectionType` (`GeneralTopic` / `Tutorial` /
`Example` / `Application`), `IsTopLevel`, and `ShowPerDefault`. At build time the
pages are embedded with `//go:embed *` and loaded via
`helpSystem.LoadSectionsFromFS(...)`. The Cobra root command is then wired with
`help_cmd.SetupCobraRootCommand(helpSystem, root)`, which registers the `help`
command and the `--help` integration.

The critical consequence: **the pages a binary can show are determined entirely
by which `embed.FS` its root command loads.** go-go-goja loads two different
`embed.FS` trees from two different binaries, so there are two disjoint help
namespaces.

### 3.2 Tree 1 — `pkg/doc/` (the module/JS-authoring tree)

- **Loader:** `pkg/doc/doc.go` — `//go:embed *` over `pkg/doc`, exposed as
  `doc.AddDocToHelpSystem(helpSystem)`.
- **Loaded by:** `cmd/goja-repl/root.go` (`newSharedHelpSystem()` at line ~183)
  and `cmd/jsverbs-example/main.go`. The in-REPL help resolver
  (`pkg/repl/evaluators/javascript/docs_resolver.go`) also reads this tree.
- **Served by binaries:** `goja-repl`, `jsverbs-example`.
- **Size:** 37 pages.
- **Coverage:** introduction, creating modules, async patterns, REPL, jsparse,
  inspector, jsverbs, plugins, the docs module, Node primitives, every native
  module (yaml/crypto/events/exec/fs/os/path/time/timer/db/uidsl), the Express
  **JavaScript** auth trilogy (`29-express-auth-user-guide`,
  `30-migrate-express-apps-to-planned-auth`, `31-express-auth-examples`),
  protobuf builders, and the bun bundling playbook.

### 3.3 Tree 2 — `cmd/xgoja/doc/` (the generation/Go-host tree)

- **Loader:** `cmd/xgoja/doc/doc.go` — its own `//go:embed *` over
  `cmd/xgoja/doc`, exposed as `doc.AddDocToHelpSystem(helpSystem)`.
- **Loaded by:** `cmd/xgoja/root.go` (line ~61).
- **Served by binary:** `xgoja` only.
- **Size:** 10 pages.
- **Coverage:** xgoja overview, user guide + v2 spec reference, buildspec
  migration, runtime-context-API migration, provider/engine-API migration,
  provider runtime config + host services, protobuf builder provider tutorial,
  migrating to xgoja/v2, v2 config reference, and **`18-go-planned-auth-api.md`**
  (the Go-side planned-auth route API: `NewApp`, `RegisterPlannedHTTP`,
  `PlannedMiddleware`, `Enforcer`, `AuthOptions`).

### 3.4 The wiring diagram

```text
                      go-go-goja documentation surface

  pkg/doc/ (37 pages)                  cmd/xgoja/doc/ (10 pages)
  ┌───────────────────────┐           ┌────────────────────────┐
  │ 29/30/31 express AUTH │           │ 18-go-planned-auth-api │  ← Go host route API
  │   (JavaScript DSL)    │   NO LINK │   (NewApp/Register/    │
  │ 18-express-module     │ ◀────────▶│    Middleware/Enforcer)│
  │ module refs (20-28)   │           │ 06-17 xgoja gen + v2   │
  └──────────┬────────────┘           └───────────┬────────────┘
             │ embed.FS (pkg/doc)                 │ embed.FS (cmd/xgoja/doc)
             ▼                                    ▼
  doc.AddDocToHelpSystem            doc.AddDocToHelpSystem
             │                                    │
   cmd/goja-repl/root.go            cmd/xgoja/root.go
   cmd/jsverbs-example/main.go       (SetupCobraRootCommand)
             ▼                                    ▼
   `goja-repl help <slug>`           `xgoja help <slug>`
   (in-REPL :help too)               (xgoja CLI only)

  README.md  ── mentions NEITHER tree ──  (47 pages undiscoverable)
```

### 3.5 Why this matters for an intern

- A reader who starts at `pkg/doc/29-express-auth-user-guide.md` (the obvious
  auth entry point) will learn the JavaScript `.auth().allow()` DSL but will
  **never discover** `cmd/xgoja/doc/18-go-planned-auth-api.md`, which is the
  Go-host counterpart. The two trees describe the same `RoutePlan` contract from
  two sides, with no bridge between them.
- `README.md` has sections for Quick start, Module Security Flags, Runtime API,
  TypeScript declarations, adding a native module, testing, and async APIs —
  but **no "Documentation" section** and no mention of `pkg/doc/` or the
  `help` command. The 47 pages are invisible to a reader who only reads the
  README.

## 4. Gap analysis (refined and corrected)

The earlier doc-plan discussion claimed "the Go host side is not documented at
all." That was **imprecise**. After discovering tree 2, the accurate picture is:

| # | Topic | Documented? | Where | Gap |
| --- | --- | --- | --- | --- |
| G1 | JS planned-auth route DSL | ✅ well | `pkg/doc/29,30,31` | none |
| G2 | Go planned-auth route API (NewApp/Register/Middleware/Enforcer) | ✅ well | `cmd/xgoja/doc/18` | not discoverable from tree 1 |
| G3 | **Full host composition** (NewHost + AuthOptions + stores + keycloakauth.New + ServeMux) | ❌ | only example 19 `main.go` | **biggest real gap** |
| G4 | `hostauth.Config` reference (Mode/Session/Stores + Glazed section) | ❌ | only `config.go`/`glazed.go` | no reference page |
| G5 | Auth store drivers (memory/sqlite/postgres, ApplySchema, DSNs) | ❌ | only `sqlstore/*.go` | no reference page |
| G6 | `ModeOIDC` hard-stop + hybrid decision | ❌ | only `resolve.go` error + example 21 README | no page; misleading (`glazed.go` offers the `oidc` choice but it errors) |
| G7 | Deployment (image/GitOps/Keycloak/Vault) | ❌ | only `design/01-...` ticket | no permanent `pkg/doc` page |
| G8 | `serve` command internals (factory discovery, host build, hot reload) | ⚠️ partial | `serve.go` + example READMEs | no reference page |
| G9 | **Cross-tree navigation** | ❌ | — | two trees never linked |
| G10 | **README navigation map** | ❌ | — | README ignores both trees |

So the genuine content gaps are **G3, G4, G5, G6, G7, G8**; the navigation gaps
are **G9, G10** (and they are the cheapest, highest-leverage fixes).

## 5. Proposed solution

### 5.1 Document placement rule

New pages go into the tree that matches the binary their reader uses:

- Pages for **module authors / JS route authors / REPL users** → `pkg/doc/`
  (served by `goja-repl`).
- Pages for **xgoja generation / Go host integration / `serve`** →
  `cmd/xgoja/doc/` (served by `xgoja`).

Because the host-composition and hostauth topics are about building a Go host,
most new pages land in **tree 2** (`cmd/xgoja/doc/`), which is also where the
existing Go auth page `18` lives — keeping the Go-host story in one place.

### 5.2 New pages to write

| ID (proposed file) | Tree | Closes gap | Type | Title / slug |
| --- | --- | --- | --- | --- |
| `cmd/xgoja/doc/19-express-auth-host-integration-guide.md` | 2 | G3 | GeneralTopic | Express auth host integration guide (`express-auth-host-integration-guide`) |
| `cmd/xgoja/doc/20-hostauth-config-reference.md` | 2 | G4, G6 | GeneralTopic | hostauth configuration reference (`hostauth-config-reference`) |
| `cmd/xgoja/doc/21-auth-stores-reference.md` | 2 | G5 | GeneralTopic | Auth stores and persistence reference (`auth-stores-reference`) |
| `cmd/xgoja/doc/22-http-serve-command-reference.md` | 2 | G8 | GeneralTopic | The generated `serve` command reference (`http-serve-command-reference`) |
| `pkg/doc/32-deploying-an-express-auth-host.md` | 1 | G7 | Tutorial | Deploying an Express auth host (`deploying-an-express-auth-host`) |

Per-page rationale (audience + what it must contain + file anchors):

#### Page A — `19-express-auth-host-integration-guide.md` (closes G3, highest content priority)

- **Audience:** a Go engineer building a binary that embeds the auth host (the
  "promote example 19" reader).
- **Contains:** how to compose `gojahttp.NewHost(HostOptions{Dev, RejectRawRoutes, Auth{Authenticator, CSRF, Resources, Authorizer, Audit}})`;
  mounting the express module via
  `engine.NewRuntimeFactoryBuilder().WithModules(express.NewRegistrar(host)).Build()`;
  running the JS route script; mounting OIDC `/auth/login|callback|logout` on an
  `http.ServeMux`; serving with probes and graceful shutdown; and a diagram of
  how a JS route declaration maps to Go enforcement.
- **Must bridge to** `cmd/xgoja/doc/18` (route API) and `pkg/doc/29` (JS DSL).
- **Anchors:** `pkg/gojahttp/host.go`, `examples/xgoja/19-.../cmd/host/main.go`,
  `pkg/gojahttp/auth/keycloakauth/keycloakauth.go`.

#### Page B — `20-hostauth-config-reference.md` (closes G4, G6)

- **Audience:** anyone running a generated host with `--auth-*` flags or a YAML config.
- **Contains:** the full `hostauth.Config` schema (`Mode none|dev|oidc`,
  `SessionConfig`/`CookieConfig`, `StoresConfig` with per-store inheritance,
  `ResolvedConfig`); the Glazed section fields (`auth-mode`, `auth-default-store-*`);
  and a **prominent, explicit note that `mode: oidc` currently returns
  `ErrOIDCNotImplemented`** (with a link to GitHub issue #82).
- **Why it matters:** `glazed.go` currently *offers* the `oidc` choice in the
  CLI, which silently errors at runtime — the page must prevent that surprise.
- **Anchors:** `pkg/xgoja/hostauth/{config.go,glazed.go,resolve.go,builder.go}`.

#### Page C — `21-auth-stores-reference.md` (closes G5)

- **Audience:** operators/devs choosing persistence backends.
- **Contains:** the four stores (session/audit/appauth/capability); drivers
  `memory|sqlite|postgres`; `ApplySchema` semantics; DSN formats; when to use
  which; store↔route relationship.
- **Anchors:** `pkg/gojahttp/auth/{sessionauth,audit,appauth,capability}/sqlstore/*.go`,
  `pkg/xgoja/hostauth/stores.go`.

#### Page D — `22-http-serve-command-reference.md` (closes G8)

- **Audience:** users of the generated `serve` command.
- **Contains:** how `serve` discovers `hostauth.LookupServiceFactory`, builds
  `gojahttp.NewHost(hostOptionsWithAuth(...))`, mounts the runtime, hot-reload
  flags, and the `http.*` / `hot-reload.*` Glazed sections.
- **Anchors:** `pkg/xgoja/providers/http/{serve.go,http.go}`.

#### Page E — `pkg/doc/32-deploying-an-express-auth-host.md` (closes G7)

- **Audience:** operators deploying to the cluster.
- **Contains:** the `pkg/doc` rendering of `design/01-...` — image build,
  `deploy/gitops-targets.json`, Keycloak client, Vault/VSO secrets, probes,
  TLS/`Secure`-cookie, multi-instance transaction store. Lands in tree 1 because
  it pairs with the `pkg/doc/31` examples page.
- **Anchors:** `Dockerfile`, `deploy/gitops-targets.json`, `.github/workflows/publish-image.yaml`.

> Note on numbering: the earlier discussion proposed `32`–`37`. After finding
> tree 2's existing `18`, the plan is tighter: five pages, with the Go-host
> pages (`19`–`22`) sitting next to the existing `18` in `cmd/xgoja/doc/`.

### 5.3 Navigation fixes (cheapest, highest leverage)

These close G9/G10 and need **no new content**, only links.

#### N1 — `README.md`: add a "Documentation" section (closes G10)

Insert after the Folder layout, before Quick start:

```markdown
## Documentation

go-go-goja ships two Glazed help trees. Browse them with:

- `goja-repl help` — module authoring, JS route DSL, REPL (37 pages in `pkg/doc/`).
- `xgoja help` — code generation, Go host integration, the `serve` command (10 pages in `cmd/xgoja/doc/`).

Key entry points:

- JS planned-auth routes: `goja-repl help express-auth-user-guide`
- Go planned-auth API:    `xgoja help go-planned-auth-api`
- Deploying an auth host: see ticket `XGOJA-AUTH-DEPLOY` (`design/01-...`).
```

#### N2 — cross-link the two auth pages (closes G9)

- In `pkg/doc/29-express-auth-user-guide.md`, add a "See also" pointing to
  `xgoja help go-planned-auth-api` (the Go-host counterpart) and to the new
  `xgoja help express-auth-host-integration-guide`.
- In `cmd/xgoja/doc/18-go-planned-auth-api.md`, add a "See also" back to
  `goja-repl help express-auth-user-guide` and forward to the host-integration
  guide.

#### N3 — `examples/xgoja/README.md`: flag production-template vs smoke-only

Annotate the learning path (research-logbook H3):
- `19-express-keycloak-auth-host` — **production template**.
- `21-generated-host-auth` — **generated-seam template** (note OIDC = follow-up).
- `20-express-hello-world` — smoke-only.

## 6. Decision records

### Decision D1: put Go-host pages in `cmd/xgoja/doc/`, not `pkg/doc/`

- **Context:** Two disjoint help trees exist. The Go planned-auth page already
  lives in `cmd/xgoja/doc/18`.
- **Options:** (a) put new Go-host pages in `pkg/doc/` next to the JS trilogy;
  (b) put them in `cmd/xgoja/doc/` next to `18`.
- **Decision:** (b).
- **Rationale:** keeps the entire Go-host story in one tree and one `help`
  namespace; `xgoja` is the binary a host-author runs; avoids splitting the
  host story across two `embed.FS` trees.
- **Consequences:** the deployment page (Page E) still goes in `pkg/doc/`
  because it pairs with the JS examples page `31`. The cross-links (N2) are
  therefore mandatory, not optional.
- **Status:** accepted

### Decision D2: navigation fixes before new content

- **Context:** The two trees and the README gap (G9/G10) hide 47 existing pages.
- **Decision:** implement N1/N2/N3 before (or alongside) writing Pages A–E.
- **Rationale:** highest leverage, lowest effort; immediately improves
  discoverability of *existing* good docs (including `cmd/xgoja/doc/18`, which
  many readers do not know exists).
- **Consequences:** a reader may follow a link to a not-yet-written page;
  stub the target pages (frontmatter + "coming soon" + ticket link) when adding
  the links.
- **Status:** accepted

### Decision D3: document the `ModeOIDC` hard-stop explicitly

- **Context:** `glazed.go` offers `auth-mode` choice `oidc`, but `resolve.go`
  returns `ErrOIDCNotImplemented` for it. This is a silent runtime trap.
- **Decision:** Page B must call this out prominently and link GitHub issue #82.
- **Rationale:** prevents users from selecting `oidc` and hitting an opaque error.
- **Consequences:** when issue #82 ships, Page B must be updated (remove the
  warning; document the new OIDC config block).
- **Status:** accepted

## 7. Implementation plan

Phases are ordered by leverage. Each page is a single Markdown file with Glazed
frontmatter; no Go changes are required (the `//go:embed *` picks up new files
automatically). Re-run `go generate ./...` only if logcopter generation is wired
to the doc dirs (it is, via `logcopter.go` in each tree).

### Phase 0 — Navigation (do first; closes G9, G10)

1. **N1** — add a "Documentation" section to `README.md` (text in §5.3).
2. **N2** — add "See also" cross-links in `pkg/doc/29-express-auth-user-guide.md`
   and `cmd/xgoja/doc/18-go-planned-auth-api.md`.
3. **N3** — annotate `examples/xgoja/README.md` (production / generated-seam /
   smoke-only).

Validate:

```bash
goja-repl help            # confirm help lists render
xgoja help                # confirm tree 2 lists render
grep -n "help" README.md  # confirm the new section
```

### Phase 1 — Content: host-integration guide (closes G3)

Write `cmd/xgoja/doc/19-express-auth-host-integration-guide.md`. Frontmatter
shape (pseudocode — fill real values):

```yaml
---
Title: "Express auth host integration guide"
Slug: express-auth-host-integration-guide
Short: "Compose the gojahttp host with OIDC, sessions, stores, and a Go ServeMux."
Topics: [xgoja, gojahttp, auth, keycloak, net-http]
Commands: [xgoja, goja-repl]
IsTopLevel: true
SectionType: GeneralTopic
---
```

Sections: intent-vs-infrastructure recap → `NewHost` + `AuthOptions` → mounting
the express module → OIDC handlers on the mux → probes/shutdown → "See also"
(`18`, `pkg/doc/29`). Anchor every claim to `pkg/gojahttp/host.go` and example 19.

### Phase 2 — Content: hostauth + stores + serve references (closes G4, G5, G6, G8)

- `cmd/xgoja/doc/20-hostauth-config-reference.md` — include the **`oidc` →
  `ErrOIDCNotImplemented` warning** and a link to issue #82 (Decision D3).
- `cmd/xgoja/doc/21-auth-stores-reference.md`.
- `cmd/xgoja/doc/22-http-serve-command-reference.md`.

Validate: `xgoja help <slug>` for each new slug resolves.

### Phase 3 — Content: deployment tutorial (closes G7)

- `pkg/doc/32-deploying-an-express-auth-host.md` — distil `design/01-...` into a
  permanent, non-ticket page. Cross-link `pkg/doc/31-express-auth-examples.md`.

Validate: `goja-repl help deploying-an-express-auth-host` resolves.

### Phase 4 — Cross-repo doc health (out of scope here; record for follow-up)

These are the out-of-repo findings from research-logbook Part H (H4–H7), logged
here for continuity, **not** implemented in this ticket:

- **H4** terraform `docs/shared-keycloak-platform-playbook.md` is stale
  (smailnail-centric; realm claim wrong).
- **H5** fix stale `corporate-headquarters` → `go-go-golems/infra-tooling` links
  in cluster + infra-tooling docs.
- **H6** infra-tooling GitOps-PR doc: add a "current adopters" table.
- **H7** cluster README: name the copy-template packages explicitly.

## 8. Validation strategy

- **Build:** `go build ./...` and `go generate ./...` (ensures the new
  `//go:embed` pages compile; logcopter generation stays green).
- **Help render:** `goja-repl help` and `xgoja help` list the new slugs; each
  `help <slug>` renders without frontmatter leaking.
- **Slug uniqueness:** `goja-repl help <slug>` and `xgoja help <slug>` must not
  collide; the glazed-lint help validator (if wired) should pass. Confirm with:
  ```bash
  grep -h "^Slug:" pkg/doc/*.md cmd/xgoja/doc/*.md | sort | uniq -d
  ```
  (expect no duplicates).
- **Link liveness:** every "See also" target slug must resolve in its tree.
- **Doctor:** `docmgr doctor --ticket XGOJA-AUTH-DEPLOY` stays green after
  relating the new page.

## 9. Risks, alternatives, open questions

### Risks
- **Drift between the two trees.** Because they are separate `embed.FS`, a
  concept documented in one can diverge from the other. Mitigation: the N2
  cross-links and the single "Go-host story in tree 2" rule (D1).
- **Page B becomes wrong when issue #82 lands.** Mitigation: D3 makes the
  warning explicit and ties it to the issue, so the update is discoverable.
- **Discoverability of tree 2 for non-xgoja users.** A `goja-repl` user cannot
  `goja-repl help go-planned-auth-api` (it is in the other tree). Mitigation:
  N1 names both binaries and N2 bridges the auth pages.

### Alternatives considered
- **Merge the two trees into one.** Rejected for now: they are served by
  different binaries with different audiences and different `embed.FS`; merging
  would couple `goja-repl` to xgoja-generation docs. Cross-links (N2) achieve
  discoverability without coupling.
- **Generate docs from code.** Rejected for the auth pages: the value is the
  integration narrative, which does not fall out of godoc.

### Open questions
1. Should the deployment tutorial (Page E) live in `pkg/doc` or also be mirrored
   into `cmd/xgoja/doc` for `xgoja help` users? (Default: `pkg/doc`, paired with
   `31`; revisit if `xgoja` users report they can't find it.)
2. Is there appetite for a single top-level `docs/index.md` that indexes both
   trees for web/GitHub readers (outside the Glazed help system)?

## 10. References

### Help-system wiring (this repo)
- `pkg/doc/doc.go` — tree 1 loader (`//go:embed *`, `AddDocToHelpSystem`).
- `cmd/xgoja/doc/doc.go` — tree 2 loader.
- `cmd/goja-repl/root.go` (~line 183, `newSharedHelpSystem`) — wires tree 1.
- `cmd/xgoja/root.go` (~line 61) — wires tree 2.
- `pkg/repl/evaluators/javascript/docs_resolver.go` — in-REPL help resolver (tree 1).
- `github.com/go-go-golems/glazed/pkg/help` + `pkg/help/cmd` — the Glazed help framework.

### Existing auth docs to cross-link
- `pkg/doc/29-express-auth-user-guide.md`, `30-...`, `31-express-auth-examples.md`.
- `cmd/xgoja/doc/18-go-planned-auth-api.md`.

### Source anchors for new pages
- `pkg/gojahttp/host.go` — `NewHost`, `HostOptions`, `AuthOptions`.
- `pkg/gojahttp/auth/keycloakauth/keycloakauth.go` — `Config`, `New`, OIDC handlers.
- `pkg/xgoja/hostauth/{config.go,glazed.go,resolve.go,builder.go,stores.go}`.
- `pkg/xgoja/providers/http/{serve.go,http.go}`.
- `examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go` — integration reference.
- `examples/gojahttp/01-planned-auth/` — Go-only planned-auth example (referenced by `18`).

### Companion documents in this ticket
- `design/01-deploy-xgoja-keycloak-auth-host-to-yolo.md` — the deployment design (Page E source).
- `reference/02-research-logbook.md` Part H — original navigation findings (H1–H7).
- GitHub issue #82 — the `ModeOIDC` implementation that Page B's warning tracks.
