# Tasks

## TODO

- [ ] Add ModuleConfigCapability interface to providerapi/capabilities.go
- [ ] Add ModuleConfigPatchFromSections to providerutil/sections.go
- [ ] Add NewRuntimeFromSections to app.RuntimeFactory
- [ ] Update built-in commands (eval, run, repl, jsverbs) to use NewRuntimeFromSections
- [ ] Extend providerapi.RuntimeFactory interface with NewRuntimeFromSections
- [ ] Add tests for ModuleConfigCapability and NewRuntimeFromSections
- [ ] Implement Geppetto ModuleConfigCapability

## DONE

- [x] Review existing GOJA-053 design docs and write independent implementation guide
- [x] Research Glazed SectionValues as the xgoja module config merge layer
- [x] Document xgoja codegen and generated script execution runthrough
- [x] Create codegen/runtime runthrough research logbook and upload to reMarkable
- [x] Inventory generic Service/Context/Capability/Runtime/Module/Spec symbols and upload glossary
- [x] Analyze Sobek ECMAScript Modules versus xgoja require/native module machinery
- [x] Create Sobek ESM research logbook and upload to reMarkable

