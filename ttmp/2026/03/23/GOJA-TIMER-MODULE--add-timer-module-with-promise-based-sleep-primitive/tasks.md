# Tasks

## TODO

- [x] Create the ticket workspace under `go-go-goja/ttmp`
- [x] Add the implementation-plan design doc and diary doc
- [x] Capture the initial architecture notes for event loop ownership and default module registration
- [ ] Implement `modules/timer/timer.go` with Promise-based `sleep(ms)`
- [ ] Register the timer module in the default shipped runtime
- [ ] Add focused unit tests for the timer module behavior
- [ ] Add runtime integration coverage proving `require("timer")` works in a fresh runtime
- [ ] Update README and async docs to reflect the new shipped module instead of a doc-only example
- [ ] Run `gofmt` and `go test ./...`
- [ ] Record implementation steps in the diary, update changelog, and validate the ticket with `docmgr doctor`
