# gojahttp planned auth example

This example shows the Go-native planned auth API without JavaScript Express.
It registers routes with `gojahttp.NewApp(host)` and uses `gojahttp.Enforcer`
indirectly through `Host`.

Routes:

- `GET /healthz` is public.
- `GET /projects/{id}` requires `X-Demo-User` and authorizes `project.read`.
- `GET /middleware/projects/{id}` uses standard Go 1.22 `http.ServeMux` plus
  `gojahttp.PlannedMiddleware`.

Run the smoke test:

```bash
make smoke
```

Run the server:

```bash
make serve
curl http://127.0.0.1:18810/healthz
curl -H 'X-Demo-User: alice' http://127.0.0.1:18810/projects/p1
curl -H 'X-Demo-User: alice' http://127.0.0.1:18810/middleware/projects/p1
```

The example auth services are intentionally tiny. They are for demonstrating the
API shape only; production hosts should validate real sessions or tokens, load
real domain resources, and implement application-specific authorization.
