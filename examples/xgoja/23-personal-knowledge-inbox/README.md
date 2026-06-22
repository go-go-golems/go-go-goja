# Personal Knowledge Inbox tutorial

This example is built as a sequence of complete, runnable steps. Each step lives in its own subdirectory and contains the full files needed for that point in the tutorial. Later steps copy the previous step and add one new concept.

This layout is intentional: a new developer can read and run each directory in order without mentally subtracting later features.

## Steps

1. `01-minimal-jsverb/` — minimal `xgoja.yaml`, one JavaScript verb, generated CLI binary, and smoke test.

Run the current first step:

```bash
make smoke
```

Or run it directly:

```bash
make -C 01-minimal-jsverb smoke
```

Future steps will add a hello-world web server, a separate CLI verb, SQLite-backed inbox state, generated hostauth, device login, and programmatic capture.
