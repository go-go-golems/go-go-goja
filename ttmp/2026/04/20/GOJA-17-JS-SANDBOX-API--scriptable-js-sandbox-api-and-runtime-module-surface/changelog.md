# Changelog

## 2026-04-20

- Initial workspace created
- Added a detailed design guide for a scriptable JS sandbox host API built on the existing go-go-goja runtime and module seams
- Added a compact JS API reference with example bot scripts
- Added a diary entry that records the current architecture mapping and sandbox API design decisions
- Implemented the sandbox runtime module, host registrar, in-memory store, bot dispatch helpers, runtime tests, and demo CLI harness (`04611cc`)
- Refreshed the reMarkable bundle so the uploaded PDF now includes the implementation diary, updated tasks, and changelog
- Added proper async Promise settlement for bot command and event dispatch, so settled JS handlers now return their resolved values instead of raw Promise objects
- Refreshed the reMarkable bundle again after the async-settlement change so the PDF matches the latest implementation
