# Changelog

## 2026-04-08

- Initial workspace created
- Mapped the real REPL teaching surface from `replapi`, `replsession`, `replhttp`, and `goja-repl`
- Identified the recommended article shape: a live interactive essay that exercises the real session lifecycle and evaluation APIs
- Documented the most important current gap: the HTTP create-session path is persistent-only and does not let the browser select raw or interactive profiles
- Wrote the detailed design guide, diary, and reference links
- Validated the ticket with `docmgr doctor` and uploaded the bundle to reMarkable

## 2026-04-14

- Added a concrete implementation task breakdown for the first article slice: Section 1 "Meet a session"
- Scoped Section 1 to one live session card plus policy/JSON views, while explicitly deferring profile comparison and persistence theater to later sections
- Added a UX handoff document with section-by-section frontend guidance and data shape for all planned sections
- Implemented the first backend slice: `goja-repl essay`, a Section 1 article page, an article bootstrap endpoint, and article-scoped create/snapshot routes backed by the real REPL app
- Imported the UX mock source (`repl-essay(1).jsx`) into ticket sources for direct extraction work
- Added a new `web/` frontend foundation with React+TypeScript, Redux Toolkit, RTK Query, MSW, and Storybook
- Extracted primitive components and Section 1 components, then composed a `MeetSessionPage` for backend-aligned section behavior
- Added a new frontend implementation guide document and expanded `tasks.md` with completed foundation work plus remaining integration tasks
- Switched `pkg/replessay` page rendering from the large inline script template to a React-shell path backed by static asset serving under `/static/essay/*`
- Added fallback shell behavior when `web/dist/public` is unavailable, while keeping article API routes and backend tests intact

## 2026-04-15

- Rewrote Section 1 copy from short demo text into a technical field guide for a new engineer
- Added deeper explanatory content covering mental model, request flow, pseudocode, API references, file references, and validation exercises
- Updated the backend bootstrap payload and MSW bootstrap fixture so the live page and Storybook/mocked page tell the same story
- Verified the live section through both `vite` dev mode and the backend-served built assets using Playwright against the real article routes
- Split the Section 1 essay body into smaller components so Storybook can inspect individual article sections instead of one large monolith
- Added story coverage for essay-specific primitives and Section 1 building blocks
- Matched the active font stacks back to the original imported artifact CSS and restored the original callout copy/style treatment

## 2026-04-08

Wrote the design-first ticket for a live interactive REPL essay, mapped the real API and report surfaces, identified the profile-selection HTTP gap, and linked the deliverable to existing REPL hardening docs.

### Related Files

- /home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replhttp/handler.go — Current JSON API surface used as the foundation for the article design
- /home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/pkg/replsession/types.go — Response objects that make the article feasible
- /home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/08/GOJA-043-INTERACTIVE-REPL-ESSAY--design-an-interactive-repl-essay-for-learning-and-validating-the-new-repl/design-doc/01-interactive-repl-essay-analysis-design-and-implementation-guide.md — Primary ticket deliverable
