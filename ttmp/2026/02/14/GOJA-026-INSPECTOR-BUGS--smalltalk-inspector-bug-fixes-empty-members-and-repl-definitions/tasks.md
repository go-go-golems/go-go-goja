# Tasks

## TODO

- [x] Add tasks here

- [ ] Reproduce and document all bugs in a formal bug report
- [ ] Bug 1 fix: Add runtime introspection for value-type globals in buildMembers()
- [ ] Bug 2 fix: Refresh globals list after REPL eval by merging runtime globals
- [ ] Bug 3 fix: Enter on value globals triggers runtime inspection
- [ ] Test all three fixes in tmux and verify no regressions
- [ ] Bug 4 (new): Fix init ordering — move rtSession creation before buildGlobals/buildMembers
- [ ] Bug 5 (new): Show runtime proto chain in members pane footer for non-class globals
- [ ] Bug 6 (new): Clear navStack on new REPL eval to prevent stale drill-in state
