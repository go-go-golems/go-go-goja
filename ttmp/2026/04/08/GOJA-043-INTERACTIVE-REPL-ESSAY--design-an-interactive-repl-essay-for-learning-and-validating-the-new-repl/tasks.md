# Tasks

## Done

- [x] Create the ticket workspace and primary documents
- [x] Map the current REPL architecture, HTTP handler, and session/evaluation types
- [x] Identify which parts of the desired interactive article can already use the real API
- [x] Identify missing product surfaces needed for the best possible article
- [x] Write the detailed intern-oriented analysis, design, and implementation guide
- [x] Write the chronological investigation diary
- [x] Upload the completed ticket bundle to reMarkable

## Deferred Implementation

- [ ] Build the actual interactive article UI
- [x] Decide whether the first implementation should be HTMX/server-rendered, a bundled frontend, or a browser-side runtime experiment
- [ ] Add any missing HTTP/API capabilities needed by the final article experience

## Frontend Foundation: `web/` (Redux + RTK Query + Storybook + MSW)

### Completed in this slice

- [x] Create `web/` frontend workspace using React + TypeScript + Vite + pnpm
- [x] Add Redux Toolkit store and RTK Query API slice for the section-1 article routes
- [x] Extract primitive components from the imported JSX mock:
  - [x] `Button`
  - [x] `Typography`
  - [x] `Card`
  - [x] `Tag`
  - [x] `Divider`
  - [x] `CodeBlock`
  - [x] `JsonViewer`
- [x] Extract section-level components for section 1:
  - [x] `SectionIntro`
  - [x] `SessionSummaryCard`
  - [x] `PolicyCard`
  - [x] `SessionJsonPanel`
  - [x] `SectionShell`
- [x] Compose `MeetSessionPage` with empty/loading/success/error request states
- [x] Add tokenized theme files and `data-part` styling hooks
- [x] Add MSW handlers for bootstrap/create/snapshot endpoints
- [x] Add Storybook stories for primitives and the full page
- [x] Validate frontend toolchain:
  - [x] `pnpm -C web check`
  - [x] `pnpm -C web build`
  - [x] `pnpm -C web build-storybook`

### Remaining integration tasks

- [x] Serve built `web/dist/public` assets from Go under `/static/essay/*`
- [x] Replace inline section-1 HTML template with a React mount shell
- [x] Keep existing article API routes unchanged while switching UI rendering path
- [x] Add end-to-end smoke coverage for static mount + section-1 happy path
- [x] Rewrite Section 1 copy into a technical field guide with prose, diagrams, pseudocode, API references, and source-file references
- [ ] Port section 2 and section 3 from imported mock into feature modules and wire them to real or article-scoped backend data

## Section 1: "Meet a Session"

### Scope

- [x] Keep the first slice focused on one live session only
- [x] Defer raw-vs-interactive-vs-persistent comparison to Section 2
- [x] Defer restore/history/docs/export panels to later sections

### UX and Content

- [x] Create the section shell with title, intro prose, and one short explanation of what a session is
- [x] Add a primary "Create session" action
- [ ] Add a compact session summary card showing:
  - [x] session id
  - [x] profile
  - [x] createdAt
  - [x] cellCount
  - [x] bindingCount
- [x] Add a policy card showing the current `eval`, `observe`, and `persist` settings in human-readable form
- [x] Add a raw JSON inspector for the returned `SessionSummary`
- [x] Add empty-state copy for "no session yet"
- [x] Add error-state copy for failed session creation or snapshot fetch
- [x] Rewrite the section as a longer technical essay for a new engineer
- [x] Split the long essay body into smaller article-section components for Storybook inspection
- [x] Match the key Section 1 typography/chrome values back to the imported mock

### Backend / API

- [x] Decide whether Section 1 uses the existing `goja-repl serve` handler directly or a small article-specific wrapper
- [x] Implement the article page route and static asset serving for the first section
- [x] Add a small article-specific CLI verb: `goja-repl essay`
- [x] Add an article bootstrap endpoint for Section 1 metadata and panel definitions
- [x] Wire session creation to the real API:
  - [x] article-scoped `POST /api/essay/sections/meet-a-session/session`
  - [x] article-scoped `GET /api/essay/sections/meet-a-session/session/:id`
  - [x] raw `/api/sessions` routes remain mounted for debugging and trust
- [x] Keep the first version on the default persistent profile unless a clean create-session override is introduced as part of this slice
- [ ] If create-session override support is added now, keep it minimal:
  - [ ] support explicit profile selection
  - [ ] do not add full arbitrary policy editing yet

### State Management

- [x] Define the minimal client-side state model for Section 1:
  - [x] `sessionID`
  - [x] `sessionSummary`
  - [x] `createStatus`
  - [x] `loadStatus`
  - [x] `error`
- [x] Make refresh/reload re-fetch the live session snapshot when a session id is already present
- [x] Ensure the UI clearly distinguishes:
  - [x] no session created yet
  - [x] session exists and was fetched successfully
  - [x] fetch failed

### Validation

- [x] Manual test: create a session and confirm the summary card matches the JSON inspector
- [x] Manual test: reload the page and confirm the session snapshot can be re-fetched when the session id is preserved
- [x] Manual test: confirm the displayed policy matches the backend `SessionSummary.Policy`
- [x] Manual test: confirm the section works against the real running server, not mocked data
- [x] Add one automated route/UI smoke test if the chosen implementation style makes that practical

### Documentation

- [x] Add a short implementation note to the ticket diary once Section 1 begins
- [x] Update the changelog when the first section ships
- [ ] Link the implementation back to the design doc section:
  - [ ] `Section 1: "Meet a session"`
  - [ ] `Phase 1: build the teaching skeleton`

## Next Slice: Section 2 "Profiles Change Behavior"

### Scope

- [ ] Add a Section 2 feature module rather than extending `MeetSessionPage` inline
- [ ] Keep Section 2 focused on profile comparison, not on evaluation rewrite details
- [ ] Decide whether Section 2 uses article-scoped mock comparison data first or real create-session profile overrides immediately

### UX and Content

- [ ] Port the Section 2 structure from `repl-essay(1).jsx`
- [ ] Add the profile selector control with `raw`, `interactive`, and `persistent`
- [ ] Add a profile comparison table covering eval, observe, and persist differences
- [ ] Add intern-oriented explanatory prose for what a profile is and why it matters
- [ ] Add a compact “what changes if I pick this?” summary block

### Backend / API

- [ ] Decide the data source for Section 2:
  - [ ] article-scoped static comparison payload from bootstrap, or
  - [ ] article-scoped create-session override route, or
  - [ ] raw API expansion for profile selection
- [ ] If profile override support is introduced, keep the contract minimal and explicit
- [ ] Keep the existing Section 1 create/snapshot flow unchanged while adding Section 2 support

### Storybook

- [ ] Add standalone stories for the profile selector and comparison table
- [ ] Add page-level Section 2 stories for each active profile
- [ ] Use Storybook to tune the typography and spacing to match the imported mock

### Validation

- [ ] Manual test: confirm switching profiles changes the displayed policy explanation correctly
- [ ] Manual test: if real profile creation is supported, confirm the created session summary matches the selected profile
- [ ] Add at least one focused frontend test or route smoke for the selected Section 2 data contract

## Next Slice: Section 3 "What Happened To My Code?"

### Scope

- [ ] Add a Section 3 feature module centered on one evaluation and its visible transformation pipeline
- [ ] Keep Section 3 focused on rewrite/execution visibility, not persistence/history

### UX and Content

- [ ] Port the Section 3 structure from `repl-essay(1).jsx`
- [ ] Add the source editor / canned source input UI
- [ ] Add side-by-side or sequential original-source and transformed-source views
- [ ] Add the rewrite operations list
- [ ] Add the execution result summary block
- [ ] Add explanatory prose about instrumented evaluation, helper insertion, and last-expression capture

### Backend / API

- [ ] Map the exact real API data needed for Section 3:
  - [ ] evaluate route
  - [ ] rewrite report
  - [ ] execution result
  - [ ] static/runtime reports
- [ ] Decide whether Section 3 should call the real evaluate endpoint directly or use an article-scoped wrapper
- [ ] If needed, add a narrow article route that returns the evaluation report in the shape the section needs

### Storybook

- [ ] Add stories for original/transformed source panes
- [ ] Add a story for the rewrite operations list
- [ ] Add a story for the execution result summary
- [ ] Add a page-level Section 3 story using the canned evaluation fixture from the imported mock

### Validation

- [ ] Manual test: submit one simple expression and confirm the transformed source matches the reported rewrite operations
- [ ] Manual test: confirm the execution result block matches the backend evaluation payload
- [ ] Add at least one focused frontend/backend smoke test for the Section 3 evaluation flow
