# Personal Knowledge Inbox tutorial

This example is built as a sequence of complete, runnable steps. Each step lives in its own subdirectory and contains the full files needed for that point in the tutorial. Later steps copy the previous step and add one new concept.

This layout is intentional: a new developer can read and run each directory in order without mentally subtracting later features.

## Steps

1. `01-minimal-jsverb/` — minimal `xgoja.yaml`, one JavaScript verb, generated CLI binary, and smoke test.
2. `02-hello-web-server/` — copies Step 01 and adds the HTTP provider, `serve` command, and public Express routes.
3. `03-sqlite-cli-inbox/` — copies Step 02 and adds SQLite-backed CLI verbs for capture, list, and archive.
4. `04-api-client-server/` — copies Step 03, moves reusable JavaScript into `lib/`, adds public REST API routes, and changes CLI verbs to call the API with guarded fetch.
5. `05-embedded-retro-ui/` — copies Step 04 and adds embedded HTML/CSS/browser JS assets with a restrained monochrome retro UI.
6. `06-browser-login-keycloak/` — copies Step 05, adds local Keycloak OIDC login with Alice and Bob, protects browser API routes with session auth/CSRF, and returns CLI commands to direct SQLite access.
7. `07-user-scoped-inbox/` — copies Step 06 and scopes browser API list/capture/archive behavior to the current authenticated session user.
8. `08-device-authorization/` — copies Step 07, adds device-code start/poll CLI verbs, browser approval UI, and a token-authenticated programmatic capture route.

Run all currently implemented steps:

```bash
make smoke
```

Or run an individual step directly:

```bash
make -C 01-minimal-jsverb smoke
make -C 02-hello-web-server smoke
make -C 03-sqlite-cli-inbox smoke
make -C 04-api-client-server smoke
make -C 05-embedded-retro-ui smoke
make -C 06-browser-login-keycloak smoke
make -C 06-browser-login-keycloak keycloak-smoke
make -C 06-browser-login-keycloak tinyidp-smoke
make -C 07-user-scoped-inbox smoke
make -C 07-user-scoped-inbox tinyidp-smoke
make -C 07-user-scoped-inbox keycloak-smoke
make -C 08-device-authorization smoke
make -C 08-device-authorization keycloak-smoke
make -C 08-device-authorization tinyidp-smoke
```

## tinyidp OIDC smoke

Steps 06, 07, and 08 also have `tinyidp-smoke` targets. They are the fast mock-IdP replacement for the Keycloak-backed tutorial path and prove the generated hostauth login/session flow without starting a Keycloak container:

```bash
make tinyidp-smoke
# or run one step at a time
make -C 06-browser-login-keycloak tinyidp-smoke
make -C 07-user-scoped-inbox tinyidp-smoke
make -C 08-device-authorization tinyidp-smoke
```

The smoke matrix gets stricter as the tutorial progresses:

- Step 06 proves OIDC browser login, app-session creation, and CSRF token exposure for Alice.
- Step 07 logs in as Alice and Bob with separate browser sessions, captures one row per user, and verifies inbox isolation.
- Step 08 approves one device token as Alice and one as Bob, captures with both programmatic tokens, and verifies each capture remains visible only to the approving user.

The Keycloak smoke remains available as a compatibility check. The tinyidp smokes currently use root issuer URLs rather than Keycloak realm-path issuers. They also pass `tinyidp-users.yaml` so Alice and Bob have stable seeded subjects and app-specific claims. Step 08 uses tinyidp only for browser login; device authorization remains implemented by the generated xgoja host.

By default the step Makefiles expect the tinyidp checkout in the workspace sibling directory `../2026-06-22--mock-oidc-idp`. Override that with `TINYIDP_ROOT=/path/to/tinyidp make tinyidp-smoke` when running from a different checkout layout.
