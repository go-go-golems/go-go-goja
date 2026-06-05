# xgoja Geppetto host-services example

This example builds a generated xgoja binary that imports the Geppetto provider and an example Geppetto host-services contributor. The generated binary exposes an embedded JavaScript verb that uses:

- `--profile-registries` to load Geppetto profile registry sources,
- `--profile` to choose the default inference profile,
- `--turns-db` to open a provider-local SQLite turn store,
- `--event-log` to install a contributed JSONL event sink,
- a contributed Go tool named `wordCount`, and
- a contributed Go middleware factory named `addSystemPrompt`.

The important point is lifecycle: the generated command parses Glazed values first, xgoja collects host-service contributions before module setup, and then `require("geppetto")` sees a loader configured with the generated module set, store, tool registry, middleware factory, and event sink.

## Run the non-network smoke

```bash
make smoke
```

This validates the buildspec, builds `dist/geppetto-host-services`, and prints the generated verb help. It does not call an LLM.

## Run the Pinocchio profile smoke port

This example includes a deterministic port of `pinocchio/examples/js/runner-profile-smoke.js` as an embedded xgoja jsverb. It uses only `require("geppetto")`, so it can run in a generated xgoja binary once the Geppetto provider has profile registry support. The smoke resolves a profile, builds an agent, and constructs a session without running inference.

```bash
make pinocchio-smoke \
  BASIC_PROFILE_REGISTRIES="profiles/basic.yaml" \
  BASIC_PROFILE=assistant
```

Expected output shape:

```json
[
  {
    "apiType": "openai",
    "hasSessionNext": true,
    "migration": "pure-geppetto-session-construction",
    "model": "gpt-5-mini",
    "profile": "assistant",
    "registry": "workspace",
    "session": "xgoja-pinocchio-profile-smoke",
    "source": "pinocchio/examples/js/runner-profile-smoke.js"
  }
]
```

## Run the live Geppetto smoke

The live smoke needs a Geppetto profile registry and a profile that can call a model. On Manuel's machine the Pinocchio profile registry is usually available at `~/.config/pinocchio/profiles.yaml`.

```bash
make live-smoke \
  PROFILE_REGISTRIES="$HOME/.config/pinocchio/profiles.yaml" \
  PROFILE=gpt-5-nano \
  SESSION="xgoja-geppetto-host-services-example"
```

The command runs:

```bash
dist/geppetto-host-services verbs demo run "$SESSION" \
  --profile-registries "$PROFILE_REGISTRIES" \
  --profile "$PROFILE" \
  --turns-db dist/turns.db \
  --event-log dist/events.jsonl \
  --output json
```

Expected JSON shape:

```json
[
  {
    "latestText": "hosted",
    "listed": 1,
    "sessionId": "xgoja-geppetto-host-services-example",
    "systemText": "Answer with exactly the word: hosted",
    "text": "hosted",
    "toolCount": 4
  }
]
```

The exact model output can vary, but the smoke is designed to request exactly the word `hosted`.

## Verify persistence and events

The `live-smoke` target runs these checks after inference:

```bash
sqlite3 dist/turns.db 'select count(*) from geppetto_turns; select session_id, phase, length(payload) from geppetto_turns;'
wc -l dist/events.jsonl
head -5 dist/events.jsonl
```

A successful run should persist one final turn and write provider-call/text events to the JSONL file.

## Pinocchio script migration matrix

| Pinocchio script | xgoja port | Classification | Notes |
| --- | --- | --- | --- |
| `pinocchio/examples/js/runner-profile-smoke.js` | `verbs/pinocchio_profiles.js` → `verbs pinocchio profile-smoke` | pure Geppetto, deterministic session construction | Uses `require("geppetto")`, profile resolution, agent construction, and session construction without a live model call. |
| `pinocchio/examples/js/runner-profile-demo.js` | `verbs/pinocchio_profiles.js` → `verbs pinocchio profile-demo` | pure Geppetto live inference | Uses `require("geppetto")`, profile resolution, session execution, and a live inference call. |
| Scripts that call `require("pinocchio")` | not portable yet | Pinocchio-specific | These need a future Pinocchio provider package or a deliberate replacement module. xgoja should not import Pinocchio into core. |
| Scripts that depend on host Go tools/middleware/event sinks | portable with contributors | Geppetto + host services | These can migrate once the required Go tool registries, middleware factories, and event sinks are provided by selected host-service contributor packages. |

## Source files

- `xgoja.yaml` selects the Geppetto provider package and the host-services example provider package.
- `verbs/demo.js` is the embedded JavaScript verb for tool/middleware/store/event validation.
- `verbs/pinocchio_profiles.js` contains ports of the two current `pinocchio/examples/js` profile scripts.
- The contributor package lives in the Geppetto repository at `pkg/js/modules/geppetto/provider/hostservicesexample`.
