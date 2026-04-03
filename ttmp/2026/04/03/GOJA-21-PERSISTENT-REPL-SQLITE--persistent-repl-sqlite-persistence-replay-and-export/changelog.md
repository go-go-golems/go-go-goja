# Changelog

## 2026-04-03

- Initial workspace created
- Replaced the placeholder scaffold with a concrete implementation plan, ordered task list, and working diary for the SQLite persistence phase
- Added `pkg/repldb` with SQLite open/bootstrap helpers, initial schema creation, and bootstrap tests for the durable REPL store
- Added transactional store APIs plus `replsession` persistence hooks for session lifecycle, evaluations, binding versions, REPL-authored docs, and read-side export/replay loading (commit `35fcb4a`)
- Updated the ticket diary, related-file links, and task tracking to reflect the completed Phase 1 persistence implementation
