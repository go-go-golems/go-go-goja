# Tasks

## TODO

- [ ] Review GC-05 design deep-dive and choose final EngineFactory module lifecycle (constructor-only vs builder Build freeze)
- [ ] Define EngineModule contract (ID, DependsOn, Register) and registration context shape
- [ ] Implement deterministic dependency resolver with cycle/missing-dependency diagnostics
- [ ] Add conflict detection for duplicate module IDs and registration keys with explicit policy
- [ ] Introduce WithModules(...) API on EngineFactory builder and keep compatibility shim with current EnableAll path
- [ ] Add tests for ordering, dependency failures, and deterministic module install plans
