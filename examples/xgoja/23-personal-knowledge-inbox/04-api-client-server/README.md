# Step 04: API server and API client verbs

This step copies Step 03 and separates reusable JavaScript into `lib/` files. It also changes the architecture: CLI verbs no longer open SQLite directly. They call public HTTP API routes with the guarded `fetch` module.

There is still no authentication in this step. The goal is to introduce the API boundary before adding sessions, agents, or device login.

## Files

```text
verbs/server.js          # registers public API routes
verbs/client.js          # CLI verbs that call the API
verbs/lib/inbox_store.js # reusable SQLite store helpers
verbs/lib/api_client.js  # reusable fetch client helpers
```

## Commands

Start the API server:

```bash
./dist/personal-knowledge-inbox-api-client-server \
  serve inbox server \
  --http-listen 127.0.0.1:18792 \
  --db /tmp/personal-inbox-api.sqlite
```

Capture through the API client:

```bash
./dist/personal-knowledge-inbox-api-client-server \
  verbs inboxctl capture \
  --base-url http://127.0.0.1:18792 \
  --title "Example article" \
  --url https://example.com \
  --note "Read later"
```

List through the API client:

```bash
./dist/personal-knowledge-inbox-api-client-server \
  verbs inboxctl list \
  --base-url http://127.0.0.1:18792
```

Run the smoke test:

```bash
make smoke
```
