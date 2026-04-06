---
Title: Investigation diary
Ticket: GOJA-039-REVIEW-REVIEW
Status: active
Topics:
    - goja
    - go
    - review
    - analysis
    - architecture
    - documentation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - /home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/06/GOJA-25-CODE-REVIEW--comprehensive-code-review-of-repl-architecture-work-since-origin-main/design/01-comprehensive-code-review-repl-architecture-delta-since-origin-main.md
    - /home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/service.go
    - /home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repldb/read.go
    - /home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/repldb/store.go
    - /home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replapi/app.go
ExternalSources: []
Summary: "Chronological investigation notes for validating GOJA-25 directly against the code and comparing it against higher-signal branch defects."
LastUpdated: 2026-04-06T16:55:00-04:00
WhatFor: "Provide a concise audit trail for the meta-review."
WhenToUse: "Use when retracing how the GOJA-25 findings were validated."
---

# Investigation diary

## Goal

Determine whether `GOJA-25-CODE-REVIEW` is accurate, whether it missed more important issues, and whether it is too sprawling to serve as a useful action document.

## Working notes

### 1. Locate the intern review and inspect its structure

I first located the existing ticket workspace and confirmed that the main review document lives under:

- `ttmp/2026/04/06/GOJA-25-CODE-REVIEW--comprehensive-code-review-of-repl-architecture-work-since-origin-main/design/01-comprehensive-code-review-repl-architecture-delta-since-origin-main.md`

The document is long and mixes onboarding material with findings. The key finding sections run from `6.1` through `6.27`.

### 2. Validate the "deprecated evaluator" claim

I checked whether `pkg/repl/evaluators/javascript/evaluator.go` was truly just dead leftover code. It is not obviously dead. The Bobatea adapter in `pkg/repl/adapters/bobatea/javascript.go` imports that evaluator directly and adapts it into Bobatea interfaces.

Conclusion:

- the file may be legacy or duplicative
- the review did not establish deprecation
- calling it "deprecated" is too strong

### 3. Validate the `types.go` complaint

I opened `pkg/repldb/types.go` and compared it against GOJA-25 section `6.13`. The claimed duplicate declaration / suspicious-comment issue was not present.

Conclusion:

- `6.13` is unsupported by the actual file
- this is the clearest factual problem in GOJA-25

### 4. Validate timeout and promise-loop concerns

I inspected `waitPromise` in `pkg/replsession/service.go`. The function polls promise state and sleeps for 5ms while pending. The loop does not explicitly check `ctx.Done()`.

Conclusion:

- the intern's concern is real
- the stronger framing is "evaluation interruption story is incomplete"
- this is a useful finding

### 5. Validate the `doc` sentinel collision claim

I inspected `installDocSentinels`, `snapshotGlobals`, `bindingVersionRecord`, `upsertDeclaredBinding`, and summary construction. The name `doc` is marked ignored. That does affect snapshot and binding-version persistence paths. It does not completely erase the binding from in-memory session tracking.

Conclusion:

- the core concern is real
- the effect is narrower than the review implies
- this is an observability/persistence collision, not a total runtime binding loss

### 6. Compare GOJA-25 against higher-signal branch defects

I then compared the intern review's coverage against independently reproduced defects already known on this branch:

- deleted sessions remain listable/restorable
- session IDs collide across separate processes
- SQLite foreign keys are not reliably enabled across pooled connections

These are more important than many GOJA-25 findings and were not covered there.

Conclusion:

- GOJA-25 has meaningful omissions in the highest-signal bug category

## Useful commands run

```bash
docmgr list tickets --with-glaze-output --output csv --with-headers=false --fields ticket,title,path,last_updated | rg '^GOJA-25-CODE-REVIEW,'
find ttmp/2026/04/06/GOJA-25-CODE-REVIEW--comprehensive-code-review-of-repl-architecture-work-since-origin-main -maxdepth 3 -type f | sort
rg -n '^## |^### ' ttmp/2026/04/06/GOJA-25-CODE-REVIEW--comprehensive-code-review-of-repl-architecture-work-since-origin-main/design/01-comprehensive-code-review-repl-architecture-delta-since-origin-main.md
nl -ba pkg/repldb/types.go | sed -n '1,220p'
nl -ba pkg/repl/adapters/bobatea/javascript.go | sed -n '1,240p'
nl -ba pkg/repl/evaluators/javascript/evaluator.go | sed -n '1,260p'
nl -ba pkg/replsession/service.go | sed -n '993,1148p'
nl -ba pkg/replsession/service.go | sed -n '1170,1338p'
nl -ba pkg/replapi/app.go | sed -n '80,140p'
nl -ba pkg/repldb/read.go | sed -n '1,220p'
nl -ba pkg/repldb/store.go | sed -n '1,180p'
nl -ba pkg/replhttp/handler.go | sed -n '1,240p'
```

## Final investigation conclusion

GOJA-25 shows genuine effort and some good review instincts, but it over-prioritizes lower-signal issues and does not meet the standard for an authoritative branch review. It is best treated as:

- a useful onboarding and architecture document
- a source of several valid follow-up issues
- not the final word on branch risk or bug prioritization
