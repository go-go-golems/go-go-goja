---
Title: Kebab-case section flags while preserving JavaScript object keys
Ticket: GOJA-JSVERBS-SECTION-FIELD-CLI-NAMES
Status: active
Topics:
  - goja
  - xgoja
  - jsverbs
DocType: index
Intent: implementation
LastUpdated: 2026-06-07
---

# Kebab-case section flags while preserving JavaScript object keys

This ticket tracks the follow-up to jsverb top-level field-name normalization. The target behavior is that every CLI-facing field name is kebab-case, including fields in named sections, while JavaScript-facing values keep the author-declared names.

## Documents

- `design-doc/01-kebab-case-section-flags-preserve-js-object-keys.md` — design and implementation guide.
- `reference/01-diary.md` — chronological implementation diary.
- `changelog.md` — concise commit and validation log.
- `tasks.md` — task checklist.

## Key files

- `/home/manuel/workspaces/2026-05-27/rag-evaluation-system/go-go-goja/pkg/jsverbs/command.go`
- `/home/manuel/workspaces/2026-05-27/rag-evaluation-system/go-go-goja/pkg/jsverbs/runtime.go`
- `/home/manuel/workspaces/2026-05-27/rag-evaluation-system/go-go-goja/pkg/jsverbs/binding.go`
- `/home/manuel/workspaces/2026-05-27/rag-evaluation-system/go-go-goja/pkg/jsverbs/jsverbs_test.go`
