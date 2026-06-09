---
Title: Investigation Diary
Ticket: XGOJA-EXTERNAL-HTTP-HOST
Status: active
Topics:
    - xgoja
    - goja
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/changelog.md
      Note: Ticket changelog for guide creation and upload
    - Path: go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/design-doc/01-external-go-http-host-integration-implementation-guide.md
      Note: Primary guide produced by this diary step
    - Path: go-go-goja/ttmp/2026/06/08/XGOJA-EXTERNAL-HTTP-HOST--support-external-go-http-hosts-for-generated-xgoja-runtimes/tasks.md
      Note: Ticket implementation and delivery checklist
ExternalSources:
    - https://github.com/go-go-golems/go-go-goja/issues/65
Summary: Chronological diary for creating the external Go HTTP host integration ticket and implementation guide.
LastUpdated: 2026-06-09T01:25:00-04:00
WhatFor: Use this diary to understand why the first implementation should be non-invasive and how the design guide was assembled.
WhenToUse: Read before resuming XGOJA-EXTERNAL-HTTP-HOST implementation or updating the guide after code changes.
---


# Diary

## Goal

Capture the ticket setup, evidence collection, and delivery work for the non-invasive xgoja external Go HTTP host integration design. This diary records why the guide keeps current `HostServices` names for now, what code paths shaped the proposal, and how the document was validated and uploaded.

## Step 1: Create external Go HTTP host integration ticket and guide

I created a new `go-go-goja` docmgr ticket for the planned non-invasive approach: make generated xgoja packages configurable enough for a Go application to inject an external `gojahttp.Host`, then teach the HTTP provider to register Express routes into that host without owning the listener. This intentionally does not perform the larger `HostService*` to `RuntimeService*` rename because that breaking cleanup is now tracked separately in GitHub issue #65.

The resulting design document is written for a new intern. It explains the current xgoja provider/runtime construction path, the `HostServices` and provider contribution APIs, the generated package templates, `gojahttp.Host`, the Express module, the HTTP provider, the proposed service-injection hook, external-host behavior, route introspection, and runtime-manager validation strategy.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket, Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a focused docmgr ticket for the non-invasive xgoja external HTTP host plan, write a thorough implementation guide for an intern, validate the ticket, and upload the document bundle to reMarkable.

**Inferred user intent:** Preserve the future HostServices rename as a later issue, but give implementers a concrete, evidence-backed plan for external Go HTTP host integration now.

**Commit (code):** N/A — documentation and ticket setup only.

### What I did

- Created docmgr ticket `XGOJA-EXTERNAL-HTTP-HOST` under `go-go-goja/ttmp`.
- Added primary design doc:
  - `design-doc/01-external-go-http-host-integration-implementation-guide.md`
- Added this diary:
  - `reference/01-investigation-diary.md`
- Wrote `tasks.md` with completed documentation tasks and future implementation phases.
- Read the reMarkable upload skill and used the minimized upload workflow.
- Inspected current xgoja/provider/http/express/gojahttp code paths with line-numbered evidence.
- Cross-referenced existing broader design material from the ClubMedMeetup ticket and go-go-goja generated-package / HTTP serve tickets.
- Created GitHub issue #65 immediately before this ticket to track the later breaking HostServices-to-RuntimeServices rename.

### Why

- The current implementation need is not a broad rename. It is a small embedding hook plus HTTP provider external-host support.
- Generated package users need a way to supply live Go services without editing generated code or custom templates.
- The HTTP provider already has most required machinery after Express lazy binding: route registration now triggers the start hook, so external no-listen mode can be implemented as an ownership-aware start no-op.
- A detailed guide reduces the chance that an intern accidentally changes the provider API names, starts with a generic runtime manager too early, or makes the HTTP provider bind a listener in external-host mode.

### What worked

- `docmgr --root go-go-goja/ttmp ticket create-ticket` created the ticket workspace successfully.
- `docmgr doc add` created the design doc and diary skeletons.
- Existing code already provided strong evidence for the guide:
  - `app.HostServices` already has a keyed service map.
  - `RuntimeFactory` already composes provider contributions before module setup.
  - `modules/express` already accepts an external `*gojahttp.Host`.
  - `gojahttp.Host` is already an `http.Handler` and dispatches through `runtimeowner`.
  - Generated package templates expose a clean `Options`/`NewBundle` seam.

### What didn't work

- There was no existing focused go-go-goja ticket for this exact non-invasive external-host approach. The relevant material existed in broader ClubMedMeetup research/design docs and adjacent go-go-goja tickets (`GOJA-064`, `GOJA-065`, `XGOJA-RUNTIME-POLISH`).
- The deliverable checklist mentions routine reMarkable status/account/listing checks, but the current reMarkable upload skill explicitly says not to run those expensive checks unless upload fails. I followed the upload skill and planned dry-run plus real upload only.

### What I learned

- The generated package API is close to sufficient; the main missing piece is `ConfigureServices` on `Options` and `HostOptions`.
- The non-invasive implementation can reuse the current `HostServiceLookup`/`HostServiceValues` model rather than inventing a parallel API.
- The bigger naming cleanup is worthwhile but should be isolated because `ModuleSetupContext.Host` and `HostService*` are already used across several provider packages.

### What was tricky to build

- The main tricky design point was error handling for `ConfigureServices`. `app.NewHostWithOptions` currently returns `*Host`, not `(*Host, error)`, so a non-invasive callback is easier than an error-returning callback. The guide recommends keeping provider payload validation in module setup, where errors can already abort runtime construction.
- Another tricky point was separating reusable xgoja work from app-specific RuntimeManager work. The guide recommends implementing external host support in `go-go-goja`, then proving the runtime manager app-locally before extracting a generic package.

### What warrants a second pair of eyes

- Whether `ConfigureServices func(*app.HostServices)` is sufficient, or whether we should accept a slightly more invasive error-returning constructor.
- Whether `ExternalHostService.OwnsListen` should exist in the first patch or whether the first patch should only support no-listen external mode.
- Whether route introspection should include static mounts in the first implementation or only method/pattern route descriptors.

### What should be done in the future

- Implement the phases in `tasks.md` in order.
- Keep the breaking rename issue #65 separate until downstream providers can be updated together.
- Add a follow-up implementation diary entry with actual commit hashes after code changes begin.

### Code review instructions

- Start with the design doc executive summary and sections 5-8 for the proposed API and validation plan.
- Check that file references match the current post-rebase branch state.
- Validate ticket hygiene with:
  - `docmgr --root go-go-goja/ttmp doctor --ticket XGOJA-EXTERNAL-HTTP-HOST --stale-after 30`
- Validate upload delivery with the `remarquee upload bundle` output.

### Technical details

Ticket creation commands:

```bash
docmgr --root go-go-goja/ttmp ticket create-ticket \
  --ticket XGOJA-EXTERNAL-HTTP-HOST \
  --title "Support external Go HTTP hosts for generated xgoja runtimes" \
  --topics xgoja,goja,http,architecture,runtime

docmgr --root go-go-goja/ttmp doc add \
  --ticket XGOJA-EXTERNAL-HTTP-HOST \
  --doc-type design-doc \
  --title "External Go HTTP Host Integration Implementation Guide"

docmgr --root go-go-goja/ttmp doc add \
  --ticket XGOJA-EXTERNAL-HTTP-HOST \
  --doc-type reference \
  --title "Investigation Diary"
```

Key evidence commands:

```bash
cd go-go-goja
nl -ba pkg/xgoja/providerapi/module.go | sed -n '1,130p'
nl -ba pkg/xgoja/providerapi/capabilities.go | sed -n '1,130p'
nl -ba pkg/xgoja/app/host.go | sed -n '1,160p'
nl -ba pkg/xgoja/app/factory.go | sed -n '1,220p'
nl -ba pkg/xgoja/providers/http/http.go | sed -n '1,220p'
nl -ba modules/express/express.go | sed -n '1,220p'
nl -ba pkg/gojahttp/host.go | sed -n '1,220p'
nl -ba pkg/gojahttp/route_registry.go | sed -n '1,140p'
nl -ba cmd/xgoja/internal/generate/templates/runtime_package.go.tmpl | sed -n '1,140p'
```

### Delivery evidence

Docmgr validation passed after keeping the ticket topics within the shared vocabulary:

```text
## Doctor Report (1 findings)

### XGOJA-EXTERNAL-HTTP-HOST

- ✅ All checks passed
```

reMarkable dry-run succeeded and listed the four bundled files: index, implementation guide, diary, and tasks. The real upload then succeeded:

```text
OK: uploaded XGOJA External HTTP Host Guide.pdf -> /ai/2026/06/09/XGOJA-EXTERNAL-HTTP-HOST
```
