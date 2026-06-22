# Tasks

## Completed

- [x] Write textbook-style personal knowledge inbox tutorial
- [x] Include resource map for deeper device-flow and xgoja study
- [x] Chapter 1A: create minimal xgoja.yaml and hello-world jsverb
- [x] Restructure tutorial example into per-step subdirectories
- [x] Chapter 1B: create 02-hello-web-server step with generated HTTP serve

## Current tutorial implementation

- [x] Step 03: add SQLite inbox + CLI verbs, with no REST API yet
- [ ] Revisit Step 03 output mode: replace JSON-as-text with proper jsverbs structured/Glazed output once the cleanest return shape is confirmed

## Future phase: revisit xgoja developer experience from tutorial findings

- [x] Investigate better ways to mount verbs so tutorials/apps do not need to expose everything under a top-level `verbs` node
- [ ] Investigate better ergonomics for starting web servers with the provider `serve` verb
- [ ] Clarify and improve structured output, Glazed output modes, and Glazed flags for jsverbs commands
- [ ] Hide `log-*` global flags by default in generated command help, or move them behind long/advanced help
- [ ] Adapt generated help text so new developers see app-specific commands before framework/global machinery
- [x] Step 04: separate reusable JS lib, server routes, and API client CLI verbs
- [x] Step 04: add API server routes and API client CLI verbs
- [x] Add Step 05 embedded retro browser UI assets
- [x] Step 06: add local Keycloak login with Alice and Bob
- [x] Step 07: scope browser inbox data to the authenticated user
- [x] Step 08: add device authorization and programmatic capture
