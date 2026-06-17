---
title: Script Archive README
doc_type: reference
topics:
  - auth
  - xgoja
  - keycloak
  - smoke-testing
status: active
---

# Ticket Scripts

Scripts captured for `XGOJA-ISSUE-85-JS-AUTH-DB-AUDIT`.

- `00-retroactive-generated-keycloak-smoke.py`: the first temporary local Docker Compose Keycloak smoke driver used to validate generated example 21 before a reusable script was committed.
- `01-example21-compose-smoke.sh`: reusable shell runner copied from `examples/xgoja/21-generated-host-auth/scripts/compose_smoke.sh`; starts the example 19 Keycloak/Postgres compose stack, runs the generated example 21 host with Postgres-backed auth stores, and invokes the Python smoke.
- `02-example21-keycloak-compose-smoke.py`: reusable Python flow copied from `examples/xgoja/21-generated-host-auth/scripts/keycloak_compose_smoke.py`; drives browserless Keycloak login, seeds demo appauth rows, exercises JS-owned audit/invite routes, and verifies persisted capability usage.

Prefer the source copies under `examples/xgoja/21-generated-host-auth/scripts/` for active maintenance. These ticket copies preserve the investigation/release validation artifacts.
