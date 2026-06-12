---
Title: V1 xgoja spec inventory
Ticket: XGOJA-ARCH-001
Status: active
Topics:
    - xgoja
    - tooling
DocType: source
Summary: Inventory of v1 xgoja.yaml examples and likely migration fixture coverage for the hard v2 cutover.
LastUpdated: 2026-06-12T13:35:00-04:00
WhatFor: Use when selecting v1 fixtures and golden migration tests for xgoja/v2 cutover work.
WhenToUse: Before implementing or reviewing migrate-spec fixture coverage.
---

# V1 xgoja spec inventory

Command used:

```bash
find examples/xgoja cmd/xgoja pkg/xgoja -name 'xgoja.yaml' -o -name '*xgoja*.yaml' | sort
```

Discovered specs:

- `examples/xgoja/01-core-provider/xgoja.yaml`
- `examples/xgoja/02-host-provider/xgoja.yaml`
- `examples/xgoja/03-single-runtime-modules/xgoja.yaml`
- `examples/xgoja/04-module-sections/xgoja.yaml`
- `examples/xgoja/05-command-provider/xgoja.yaml`
- `examples/xgoja/06-runtime-filesystem/xgoja.yaml`
- `examples/xgoja/07-embedded-jsverbs/xgoja.yaml`
- `examples/xgoja/08-provider-shipped-jsverbs/xgoja.yaml`
- `examples/xgoja/09-provider-shipped-help-docs/xgoja.yaml`
- `examples/xgoja/10-embedded-assets-fs/xgoja.yaml`
- `examples/xgoja/11-config-env/xgoja.yaml`
- `examples/xgoja/12-geppetto-host-services/xgoja.yaml`
- `examples/xgoja/13-http-serve-jsverbs/xgoja.yaml`
- `examples/xgoja/14-generated-runtime-package/xgoja.yaml`
- `examples/xgoja/15-typescript-jsverbs/xgoja.yaml`

Fixture coverage groups:

| Coverage group | Representative specs |
| --- | --- |
| Minimal generated binary/provider package | `01-core-provider`, `02-host-provider` |
| Runtime modules | `03-single-runtime-modules`, `04-module-sections` |
| Provider command sets | `05-command-provider`, `13-http-serve-jsverbs` |
| Local/embedded jsverbs | `07-embedded-jsverbs`, `13-http-serve-jsverbs`, `15-typescript-jsverbs` |
| Provider-shipped jsverbs | `08-provider-shipped-jsverbs` |
| Provider-shipped help | `09-provider-shipped-help-docs` |
| Assets | `10-embedded-assets-fs` |
| Config file/env prefix | `11-config-env` |
| Host services / extra imports | `12-geppetto-host-services` |
| Runtime package artifact | `14-generated-runtime-package` |
| TypeScript compile settings | `15-typescript-jsverbs` |

Initial hard-cutover rule:

- v1 specs remain valid only as input to `xgoja migrate-spec`.
- Normal v2-era commands should reject v1 specs with a migration diagnostic.
- The planner/runtime implementation should not keep v1 compatibility branches after cutover.

Provider API note:

- Existing provider runtime APIs should be wrapped by the v2 provider graph first.
- Provider API changes should be driven by concrete planner/runtime gaps, not by the schema cutover itself.
