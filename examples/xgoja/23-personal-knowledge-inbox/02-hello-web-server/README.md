# Step 02: hello web server

This step copies Step 01 and adds one new idea: a generated xgoja binary can expose an HTTP `serve` command contributed by the HTTP provider.

The same generated binary still has the `verbs inbox hello` CLI command from Step 01. It now also has:

```bash
./dist/personal-knowledge-inbox-hello-web-server serve inbox server --http-listen 127.0.0.1:18790
```

The `server` jsverb registers two public Express routes:

- `GET /` returns a plain text greeting.
- `GET /healthz` returns JSON for smoke tests.

Run:

```bash
make doctor
make smoke
```

Later steps will add separate server/CLI binaries and persistent inbox state.
