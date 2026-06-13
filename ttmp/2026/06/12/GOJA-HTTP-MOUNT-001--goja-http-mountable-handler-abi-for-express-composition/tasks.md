# Tasks

## TODO

- [ ] Add tasks here

- [x] Analyze current gojahttp routing, static mount, express, and xgoja HTTP serve behavior, including wildcard and :param route semantics
- [x] Add shared gojahttp mountable handler ABI with hidden non-enumerable http.Handler refs and extraction helpers
- [x] Add gojahttp Host explicit handler mount API with prefix matching and strip-prefix options separate from static asset naming
- [x] Add express app.mount/app.mountHandler JavaScript API that accepts Go http.Handler-backed objects and registers them on the host
- [x] Add unit/integration tests for hidden handler refs, express mounting, prefix behavior, route ordering, :param routes, and wildcard routes
- [x] Update TypeScript declarations and documentation for mountable handlers and route pattern semantics
- [ ] Run targeted and full validation for gojahttp, express, xgoja HTTP provider, and go-go-goja
