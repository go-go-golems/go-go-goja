# Step 03: SQLite inbox CLI verbs

This step copies Step 02 and adds local application state. There is still no REST API for inbox data. The inbox is manipulated through CLI verbs that open a SQLite database directly through the guarded host `database` module.

The generated binary now demonstrates four ideas:

- the original `verbs inbox hello` command still works;
- the public hello-world web server from Step 02 still works;
- new CLI verbs can create, list, and archive inbox items in SQLite;
- a reusable jsverbs `storage` section contributes the shared `--db` flag to multiple commands.

The `capture` command requires both `--title` and `--url`. This is intentional: this step teaches CLI validation at the command boundary before any REST API exists.

Run:

```bash
make doctor
make smoke
```

Manual commands after `make build`:

```bash
./dist/personal-knowledge-inbox-sqlite-cli verbs inbox capture \
  --db /tmp/personal-inbox.sqlite \
  --title "Example article" \
  --url https://example.com \
  --note "Read later"

./dist/personal-knowledge-inbox-sqlite-cli verbs inbox list \
  --db /tmp/personal-inbox.sqlite
```

Later steps will move inbox access behind HTTP routes and then protect those routes with session and programmatic auth.
