---
Title: Diary
Ticket: XGOJA-002
Status: active
Topics:
    - xgoja
    - jsverbs
    - goja
    - cli
    - glazed
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/05/22/XGOJA-002--mount-jsverbs-in-generated-xgoja-binaries/design-doc/01-real-jsverbs-mounting-design-and-implementation-guide.md
      Note: Primary design deliverable created in Step 1
ExternalSources: []
Summary: Implementation diary for mounting jsverbs as executable commands in generated xgoja binaries.
LastUpdated: 2026-05-22T19:06:53-04:00
WhatFor: Use this diary to resume or review XGOJA-002 real jsverbs mounting work.
WhenToUse: Read before changing jsverbs invocation, xgoja app runtime command mounting, or generated verb-source behavior.
---


# Diary

## Goal

This diary records the work to close the previous xgoja implementation ticket, create `XGOJA-002`, design real jsverbs mounting, and implement the required runtime and command-mounting changes one task at a time.

## Step 1: Create the jsverbs mounting ticket and design

This step started the follow-up ticket for the missing jsverbs behavior in generated xgoja binaries. The current generated app can list configured verb source IDs, but it does not scan JavaScript files, turn them into commands, or invoke them with the xgoja runtime profile.

I closed the completed previous ticket, created `XGOJA-002`, wrote the design guide, and added implementation tasks. The design intentionally limits the first pass to filesystem jsverb sources and direct invocation on xgoja's minimal runtime; embedded source copying and provider-shipped verb source mounting remain future work.

### Prompt Context

**User prompt (verbatim):** "Close XGOJA_002 and make a new ticket to properly mount jsverbs, add design and implementation guide, and  tasks. THen work on tasks one by one, committing at appropriate intervals, and keeping a diary."

**Assistant interpretation:** Close the completed current xgoja ticket if applicable, create a new ticket for real jsverbs mounting, write a design/implementation guide and task list, then start implementing incrementally with commits and diary updates.

**Inferred user intent:** Move the placeholder generated `verbs` command to a real executable jsverbs command mounting implementation while maintaining the same docmgr/diary/commit discipline as XGOJA-001.

### What I did

- Checked for `XGOJA_002` and `XGOJA-002`; neither existed yet.
- Closed `XGOJA-001`, which had all tasks complete.
- Created `XGOJA-002 — Mount jsverbs in generated xgoja binaries`.
- Added design document `/home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/22/XGOJA-002--mount-jsverbs-in-generated-xgoja-binaries/design-doc/01-real-jsverbs-mounting-design-and-implementation-guide.md`.
- Added this diary document.
- Added tasks covering direct jsverbs invocation, xgoja runtime require options, real command mounting, tests, and documentation/upload.

### Why

- The prior implementation intentionally left jsverbs as a placeholder source-listing command.
- Real mounting requires a small design because the existing jsverbs invocation path expects `engine.Runtime`, while generated xgoja currently uses a minimal `goja` + `require.Registry` runtime.

### What worked

- `docmgr ticket close --ticket XGOJA-001` succeeded.
- `docmgr ticket create-ticket --ticket XGOJA-002 ...` succeeded.
- Design, task, and diary files were created and written.

### What didn't work

- The exact ticket name `XGOJA_002` from the prompt did not exist. I interpreted the request as closing the completed xgoja implementation ticket and creating the new follow-up as `XGOJA-002`.

### What I learned

- Real jsverbs mounting is mostly a runtime adapter problem, not a scanner problem. The scanner and Glazed command conversion already exist; xgoja needs a direct invocation path compatible with its current runtime.

### What was tricky to build

- The design must avoid prematurely switching back to `engine.Factory`, because importing `engine` still hits the existing goja/goja_nodejs mismatch. The first pass therefore targets direct goja runtime invocation.

### What warrants a second pair of eyes

- Confirm that closing `XGOJA-001` was the intended interpretation of "Close XGOJA_002" given no `XGOJA_002` ticket existed.
- Review whether embedded verb sources should be part of this ticket or a follow-up.

### What should be done in the future

- Implement the tasks in order and commit after focused phases.

### Code review instructions

- Start with the design guide, then review changes to `pkg/jsverbs/runtime.go` and `pkg/xgoja/app` as they land.

### Technical details

Ticket paths:

```text
go-go-goja/ttmp/2026/05/22/XGOJA-002--mount-jsverbs-in-generated-xgoja-binaries
```
