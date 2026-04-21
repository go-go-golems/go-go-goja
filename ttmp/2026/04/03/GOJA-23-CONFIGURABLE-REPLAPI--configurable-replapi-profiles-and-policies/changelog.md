# Changelog

## 2026-04-03

- Initial workspace created
- Added profile-based `replapi` construction and policy-driven `replsession` execution, including raw, interactive, and persistent session behavior (`de8a47d`)
- Adopted the interactive `replapi` profile in `cmd/repl` so the line REPL now uses the shared configurable session kernel (`d4fa0b5`)
