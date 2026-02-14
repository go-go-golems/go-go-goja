# Tasks

## TODO

- [x] Create GOJA-030 ticket workspace
- [x] Add syntax-highlighting improvement implementation plan

- [ ] Baseline benchmarking: add benchmark cases and capture current performance/allocation metrics
- [ ] Implement per-line span index in `pkg/jsparse/highlight.go`
- [ ] Replace global linear span lookup in renderer with indexed lookup path
- [ ] Implement segment-based line rendering to reduce per-character overhead
- [ ] Add styled-line cache with invalidation strategy for source changes
- [ ] Fix REPL fallback path to always rebuild/invalidate syntax spans
- [ ] Expand correctness tests (multiline strings/comments/template literals/operator cases)
- [ ] Add benchmark + profile comparison report (before/after)
- [ ] Validate behavior in both file-source and REPL-source views
- [ ] Document final algorithm choice and follow-up opportunities

## Research Spike (Optional but Recommended)

- [ ] Run a 1-2 day algorithm research spike (tree-sitter highlight architecture, interval indexing, cache invalidation patterns)
- [ ] Produce short recommendation memo and feed decisions back into implementation plan
