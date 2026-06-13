# Express planned auth route sketch

This example is a JavaScript API sketch for the Go-backed planned auth route builder in `require("express")`. It uses the hard-cutover verb-helper API: `app.get(...)`, `app.patch(...)`, and friends return staged planned route builders, so every route must call `.public()` or `.auth(...).allow(...)` before `.handle(...)`.

It is intentionally not a standalone generated binary smoke test yet, because authenticated planned routes require the embedding Go host to configure `gojahttp.HostOptions.Auth` with an `Authenticator`, `ResourceResolver`, and `Authorizer`.

Use `scripts/server.js` as the route-authoring reference when wiring those host services into an application.
