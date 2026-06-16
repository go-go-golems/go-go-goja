# Tasks

## TODO

- [x] Add tasks here

- [x] Gather issue and code evidence for generated OIDC host support
- [x] Write intern-oriented design and implementation guide
- [x] Validate docs and upload bundle to reMarkable
- [x] Inventory current HTTP provider, Express loader, hot reload, and generated host call paths before changing code
- [x] Add regression coverage showing require("express") does not start a listener when an external host is provided
- [x] Refactor the HTTP provider so serve owns listener, http.Server, top-level handler, and graceful shutdown
- [x] Keep Express as a pure route-registration module and remove serve-time reliance on express.WithOnUse startup
- [x] Adapt hot reload to one stable listener/top-level handler that swaps only app runtime snapshots
- [x] Add top-level auth fields to xgoja v2 spec and generated runtime planning
- [x] Build hostauth services from generated serve configuration using Glazed/env-backed settings
- [x] Mount native OIDC handlers before the generated app host in the serve-owned mux
- [x] Generate a self-contained xgoja.yaml OIDC example to replace the temporary hand-written auth host
- [ ] Add unit, generated-example, and smoke tests for OIDC serve behavior
- [ ] Update permanent xgoja docs and runbooks after implementation
- [ ] Run full validation, update diary/changelog, and upload final implementation bundle
