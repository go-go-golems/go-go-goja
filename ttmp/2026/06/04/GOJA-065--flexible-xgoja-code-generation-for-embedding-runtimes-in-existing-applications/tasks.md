# Tasks

## TODO

- [x] Map current xgoja generation targets, templates, and runtime APIs
- [x] Study Pinocchio and Geppetto scopedjs integration needs
- [x] Design flexible codegen, library/runtime embedding, and custom template options
- [x] Validate documentation and upload bundle to reMarkable
- [x] Extend buildspec target schema with `package` and `template` fields
- [x] Add validation/defaulting for `target.kind: package`
- [x] Add runtime package template data and renderer
- [x] Add generated package API: `EmbeddedSpecJSON`, `DecodeSpec`, `RegisterProviders`, `NewBundle`, `Bundle.NewRuntime`, `Bundle.NewRuntimeFromSections`, and `Bundle.AttachDefaultCommands`
- [x] Add package-generation writer that copies embedded resources and writes `xgoja_runtime.gen.go` without creating `go.mod` or `main.go`
- [x] Add `xgoja generate` CLI command for source generation without compiling
- [x] Add unit tests for package target validation and package template rendering
- [x] Add generated-package compile/smoke test that imports the generated package from a host module and creates a runtime
- [x] Add runnable example under `examples/xgoja/14-generated-runtime-package`
- [x] Add xgoja help/user-guide/buildspec documentation for package generation
- [x] Run focused and full validation for GOJA-065
- [x] Update GOJA-065 diary, changelog, file relations, and docmgr doctor
- [x] Commit GOJA-065 implementation and documentation changes
- [ ] Implement source-fragment generation mode
- [ ] Implement custom template generation mode
- [ ] Add tests and documentation for source-fragment and custom-template generation
