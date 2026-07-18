# Step 01: minimal jsverb

This is the first runnable step of the Personal Knowledge Inbox tutorial.

It demonstrates the smallest useful generated xgoja shape:

- one `xgoja.yaml`,
- one JavaScript verb source file,
- one generated CLI binary,
- one smoke test that builds and runs the verb.

Run:

```bash
make doctor
make smoke
```

Manual command after `make build`:

```bash
./dist/personal-knowledge-inbox verbs inbox hello --name tutorial
```

Later steps copy this directory and add one concept at a time.
