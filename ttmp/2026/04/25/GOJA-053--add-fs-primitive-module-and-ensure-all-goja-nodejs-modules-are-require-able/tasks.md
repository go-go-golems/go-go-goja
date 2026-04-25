---
Title: Tasks
Ticket: GOJA-053
LastUpdated: 2026-04-25T09:05:00-04:00
---

# Tasks

## Track A: Imported goja_nodejs primitives

- [x] 1. Create `engine/nodejs_init.go` with blank imports for `buffer`, `process`, `url`, and `util` so all are require-able.
- [x] 2. Update `engine/factory.go` to enable global `Buffer`, `URL`, and `URLSearchParams` by default.
- [x] 3. Add `engine.ProcessEnv()` runtime initializer that enables global `process` only when callers opt in.
- [x] 4. Add smoke tests proving default `Buffer`, `URL`, `URLSearchParams`, `require("util")`, and `require("process")` work.
- [x] 5. Add smoke tests proving global `process` is absent by default and present when `ProcessEnv()` is configured.
- [x] 6. Run `go test ./engine -count=1`.
- [x] 7. Commit Track A.

## Track B: Timing APIs

- [x] 8. Add a runtime helper that installs global `performance.now()` using monotonic elapsed milliseconds.
- [x] 9. Add console timing helpers: `console.time(label)`, `console.timeLog(label)`, and `console.timeEnd(label)`.
- [x] 10. Create `modules/time/time.go` exposing `require("time").now()` and `require("time").since(startMs)`.
- [x] 11. Add TypeScript declarations and docs for the `time` module.
- [x] 12. Add smoke tests for `performance.now()` monotonicity and elapsed positive values.
- [x] 13. Add smoke tests for `console.time*` functions being callable from JS.
- [x] 14. Add smoke tests for `require("time")`.
- [x] 15. Run `go test ./engine ./modules/time -count=1`.
- [x] 16. Commit Track B.

## Track C: Promise-based fs module

- [x] 17. Rewrite `modules/fs/fs.go` to wire async and sync APIs and update docs/TypeScript declarations.
- [x] 18. Create `modules/fs/fs_async.go` with promise-based functions: `readFile`, `writeFile`, `exists`, `mkdir`, `readdir`, `stat`, `unlink`, `appendFile`, `rename`, `copyFile`.
- [x] 19. Create `modules/fs/fs_sync.go` with sync counterparts: `readFileSync`, `writeFileSync`, `existsSync`, `mkdirSync`, `readdirSync`, `statSync`, `unlinkSync`, `appendFileSync`, `renameSync`, `copyFileSync`.
- [x] 20. Add real async smoke tests using a live runtime and temp files.
- [x] 21. Add real sync smoke tests using a live runtime and temp files.
- [x] 22. Run `go test ./modules/fs -count=1`.
- [x] 23. Commit Track C.

## Track D: Final validation and documentation

- [ ] 24. Update diary after each track with commands, errors, commits, and review notes.
- [ ] 25. Update changelog after each track with commit hashes.
- [ ] 26. Run `go test ./engine ./modules/fs ./modules/time -count=1`.
- [ ] 27. Run broader `go test ./... -count=1` if feasible; document any pre-existing failures.
- [ ] 28. Run `docmgr doctor --ticket GOJA-053 --stale-after 30`.
- [ ] 29. Upload updated docs to reMarkable.
