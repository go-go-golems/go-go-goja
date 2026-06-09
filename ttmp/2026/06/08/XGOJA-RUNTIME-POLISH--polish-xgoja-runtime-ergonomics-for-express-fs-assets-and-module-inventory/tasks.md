# Tasks

## Runtime ergonomics polish

- [x] Make `require("express")` side-effect-light: it must not eagerly bind the HTTP listener merely because the module is required.
- [x] Preserve current `run server.js --keep-alive` behavior: route/static registration should still start serving when HTTP is enabled, or an explicit `app.listen()` path should be provided.
- [x] Add `fs:assets`/embedded filesystem capability metadata, for example `isReadOnly` and `capabilities()`.
- [x] Keep read-only mutation failures unchanged (`EROFS`), but make read-only behavior discoverable from JavaScript and TypeScript docs/declarations.
- [x] Clarify generated runtime module inventory: distinguish compiled provider catalog from selected runtime aliases.
- [x] Ensure selected module inventory is a Glazed structured command that supports normal output flags such as `--output json`.
- [x] Add tests for Express require-only behavior, fs capabilities, and selected-module output.
- [ ] Update docs/examples and diary as implementation proceeds.
