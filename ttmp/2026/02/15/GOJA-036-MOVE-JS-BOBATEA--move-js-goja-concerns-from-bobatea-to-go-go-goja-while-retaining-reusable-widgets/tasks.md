# Tasks

## TODO

- [x] Baseline current behavior and tests for bobatea JS evaluator + go-go-goja REPL paths
- [x] Create go-go-goja JS evaluator package by porting bobatea/pkg/repl/evaluators/javascript
- [x] Add/port evaluator unit tests in go-go-goja and make them pass
- [x] Create go-go-goja adapter package exposing bobatea/repl evaluator capability interfaces
- [x] Add new go-go-goja cmd/js-repl command using bobatea REPL + go-go-goja adapter
- [x] Move/retire bobatea/examples/js-repl and update references/docs to point to go-go-goja cmd/js-repl
- [x] Remove JS evaluator package from bobatea and clean module dependencies
- [x] Integrate bobatea suggest widget into smalltalk-inspector REPL pane
- [x] Integrate bobatea contextbar widget into smalltalk-inspector REPL pane
- [x] Integrate bobatea contextpanel widget into smalltalk-inspector REPL pane
- [x] Run cross-repo regression tests and manual smoke runs
- [ ] Finalize diary/changelog/task checkoffs and commit code/docs in logical chunks
