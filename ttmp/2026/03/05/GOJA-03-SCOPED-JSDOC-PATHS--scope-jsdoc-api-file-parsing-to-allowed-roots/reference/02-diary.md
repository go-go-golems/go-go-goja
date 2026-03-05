---
Title: Diary
Ticket: GOJA-03-SCOPED-JSDOC-PATHS
Status: active
Topics:
    - goja
    - tooling
    - security
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-05T14:55:00-05:00
WhatFor: |
    Chronological implementation diary for the scoped jsdoc path refactor, including
    prompts, code changes, validation steps, and follow-up review notes.
WhenToUse: |
    Use when continuing GOJA-03 work or reviewing how the scoped filesystem design was
    implemented step by step.
---

# Diary

## Goal

Keep a detailed chronological record of the GOJA-03 refactor so a new intern can understand why the CodeQL warning exists, how the path-scoping design works, and how to review the resulting implementation.

## Step 1: Create GOJA-03 ticket and write the implementation plan

This step creates a dedicated ticket for the path-scoping refactor and writes the design/implementation plan before changing the code. The goal is to avoid “security by ad hoc patching” and instead make the filesystem trust boundary explicit in the design first.

### Prompt Context

**User prompt (verbatim):** "yes, craete a new docmgr ticket for this, add an implementation plan document and tasks, then go work on the tasks one by one, committing at appropriate intervals, keeping a diary."

**Assistant interpretation:** Create a new docmgr ticket for the CodeQL/path-scoping work, document the plan and task breakdown, then implement the refactor incrementally with commits and diary entries.

**Inferred user intent:** Turn the security discussion into a tracked engineering task with explicit design rationale, reviewable implementation steps, and durable documentation.

**Commit (code):** N/A

### What I did
- Created ticket `GOJA-03-SCOPED-JSDOC-PATHS`.
- Added this diary and a design/implementation plan document.
- Decided the target design should use `fs.FS` scoping for untrusted API paths rather than pushing more policy into the extractor itself.

### Why
- CodeQL is flagging a structural problem, not just a missing string check. The design should fix the abstraction boundary.

### What worked
- Existing GOJA-01 and GOJA-02 ticket structure provided a clear template for the new ticket.

### What didn't work
- N/A

### What I learned
- N/A

### What was tricky to build
- The main design tradeoff is not “can we validate paths?” because we already do; it is “where should the security boundary live so the code structure itself is convincing?”

### What warrants a second pair of eyes
- Whether `ParseFile(path)` should remain as a trusted/local helper or be deprecated immediately.

### What should be done in the future
- Implement the scoped extractor and refactor server/batch to use it.

### Code review instructions
- Start with the design document in `reference/01-design-implementation-plan-scoped-jsdoc-paths.md`.

### Technical details
- N/A
