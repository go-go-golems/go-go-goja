# Tasks

## TODO

- [x] Write issue analysis doc covering reported bugs and similar-pattern sweep findings
- [x] Add/extend inspector app regression tests for drawer unresolved-symbol safety (`ctrl+d`, `ctrl+g`)
- [x] Fix nil-binding guards in `cmd/inspector/app/model.go` drawer go-to-definition/highlight paths
- [x] Run focused validation (`go test ./cmd/inspector/... -count=1`, `go build ./cmd/inspector`) and commit inspector fixes
- [x] Add resolver regression tests for `for-in/for-of` target resolution (`ForIntoExpression`) and `for (var ...)` declaration coverage (`ForIntoVar`)
- [x] Fix resolver handling of `ForInStatement`/`ForOfStatement` into-targets and var-declaration collection
- [x] Add resolver regression tests for parameter-default initializer scope semantics in function literals and arrow functions
- [x] Fix resolver ordering/coverage for parameter default initializers (before body hoisting; include arrow functions)
- [x] Run resolver/full validation (`go test ./pkg/jsparse -count=1`, `go test ./... -count=1`, `make lint`) and commit resolver fixes
- [x] Check off tasks and update diary/changelog after each completed implementation step
- [ ] Upload analysis document(s) for this ticket to reMarkable and verify remote presence
