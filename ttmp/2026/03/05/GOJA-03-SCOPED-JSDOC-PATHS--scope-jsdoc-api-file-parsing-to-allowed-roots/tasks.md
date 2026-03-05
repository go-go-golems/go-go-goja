# Tasks

## TODO

- [x] Create ticket and document the scoped-path design

### Phase 1: Extractor API
- [x] Add `extract.ParseFSFile(fsys fs.FS, path string)`
- [x] Add extractor tests for FS-scoped parsing
- [x] Add a symlink-safe scoped filesystem wrapper for untrusted paths

### Phase 2: Batch builder
- [x] Add a configurable path-parser hook to batch options
- [x] Keep inline content support unchanged
- [x] Add batch tests for custom path parsing

### Phase 3: Server refactor
- [x] Stop converting accepted API paths into absolute host paths
- [x] Keep validated paths relative after normalization
- [x] Use `extract.NewScopedFS(s.dir)` + `extract.ParseFSFile` for batch API parsing
- [x] Reject mixed `path` + `content` request items as bad requests
- [x] Extend server tests for scoped relative-path flow
- [x] Add server coverage for symlink escape rejection

### Phase 4: Validation and docs
- [x] Run focused tests for extract/batch/server
- [x] Update diary and changelog with each completed step
- [x] Mark ticket complete and run `docmgr doctor`
