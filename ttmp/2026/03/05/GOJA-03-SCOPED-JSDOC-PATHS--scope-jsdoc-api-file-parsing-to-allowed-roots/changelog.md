# Changelog

## 2026-03-05

- Initial workspace created
- Created GOJA-03 ticket and added the scoped-path design/implementation plan.
- Added a scoped extractor filesystem, batch path-parser injection, and server refactor so API paths stay relative and are enforced against symlink escapes (commit 80f6e1b).
- Fixed mixed `path` + `content` API inputs to return bad-request errors instead of surfacing as internal build errors (commit 80f6e1b).
