# Tasks

## TODO

- [x] Create ticket and document the scoped-path design

### Phase 1: Extractor API
- [ ] Add `extract.ParseFSFile(fsys fs.FS, path string)`
- [ ] Add extractor tests for FS-scoped parsing

### Phase 2: Batch builder
- [ ] Add a configurable path-parser hook to batch options
- [ ] Keep inline content support unchanged
- [ ] Add batch tests for custom path parsing

### Phase 3: Server refactor
- [ ] Stop converting accepted API paths into absolute host paths
- [ ] Keep validated paths relative after normalization
- [ ] Use `os.DirFS(s.dir)` + `extract.ParseFSFile` for batch API parsing
- [ ] Extend server tests for scoped relative-path flow

### Phase 4: Validation and docs
- [ ] Run focused tests for extract/batch/server
- [ ] Update diary and changelog with each completed step
- [ ] Mark ticket complete and run `docmgr doctor`
