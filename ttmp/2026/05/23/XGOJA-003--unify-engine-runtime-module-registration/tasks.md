# Tasks

## TODO


- [x] Write design, migration guide, and implementation plan for hard cutover to runtime-aware module registration
- [ ] Replace engine ModuleSpec/RuntimeModuleRegistrar split with one RuntimeModuleSpec contract
- [ ] Update built-in/default/process module specs and engine factory to use RegisterRuntimeModule
- [ ] Update runtime module registrars and call sites to use WithModules
- [ ] Update xgoja runtime factory to use engine.Runtime safely from xgoja spec-selected modules
- [ ] Update tests, examples, docs, and validate focused package set
