# Changelog

## 2026-02-14

- Initial workspace created


## 2026-02-14

All 6 bugs fixed: runtime members for value globals, globals refresh after REPL eval, Enter-to-inspect, proto chain footer, navStack leak, init ordering. Also fixed strict-mode panic in InspectObject via safeGet().

### Related Files

- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/model.go — buildValueMembers
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/cmd/smalltalk-inspector/app/update.go — navStack clear
- /home/manuel/workspaces/2026-02-14/smalltalk-inspector/go-go-goja/pkg/inspector/runtime/introspect.go — safeGet

