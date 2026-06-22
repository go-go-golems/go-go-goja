# Personal Knowledge Inbox tutorial

This example is built as a sequence of complete, runnable steps. Each step lives in its own subdirectory and contains the full files needed for that point in the tutorial. Later steps copy the previous step and add one new concept.

This layout is intentional: a new developer can read and run each directory in order without mentally subtracting later features.

## Steps

1. `01-minimal-jsverb/` — minimal `xgoja.yaml`, one JavaScript verb, generated CLI binary, and smoke test.
2. `02-hello-web-server/` — copies Step 01 and adds the HTTP provider, `serve` command, and public Express routes.
3. `03-sqlite-cli-inbox/` — copies Step 02 and adds SQLite-backed CLI verbs for capture, list, and archive.
4. `04-api-client-server/` — copies Step 03, moves reusable JavaScript into `lib/`, adds public REST API routes, and changes CLI verbs to call the API with guarded fetch.
5. `05-embedded-retro-ui/` — copies Step 04 and adds embedded HTML/CSS/browser JS assets with a restrained monochrome retro UI.

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
```

Future steps will add a hello-world web server, a separate CLI verb, SQLite-backed inbox state, generated hostauth, device login, and programmatic capture.
