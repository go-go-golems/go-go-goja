# Changelog

## 2026-05-25

- Initial workspace created


## 2026-05-25

Created design and implementation guide for xgoja-owned Express provider and Discord outbound runtime API.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/25/XGOJA-011--xgoja-http-express-provider-and-discord-outbound-runtime-api/design/01-http-express-provider-and-discord-outbound-api.md — New intern-oriented design and implementation guide
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/ttmp/2026/05/25/XGOJA-011--xgoja-http-express-provider-and-discord-outbound-runtime-api/reference/01-diary.md — Started implementation diary and recorded cleanup


## 2026-05-25

Uploaded XGOJA-011 design bundle to reMarkable.


## 2026-05-25

Implemented go-go-goja HTTP provider with express module and http listen Glazed section.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/modules/express/express.go — Exported NewLoader for provider module use
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/providerapi/capabilities.go — Added optional RuntimeCloserRegistry
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/providers/http/http.go — New xgoja HTTP provider registering express and http section


## 2026-05-25

Wired discord-bot command provider to selected xgoja module sections and runtime initializers.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/discord-bot/pkg/xgoja/provider/provider.go — Command provider now aggregates selected module sections and initializes runtime capabilities


## 2026-05-25

Added top-level discord.channels.send outbound API for session-bound Discord sends.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/discord-bot/internal/bot/bot.go — Live bot construction attaches session outbound ops
- /home/manuel/workspaces/2026-05-24/add-js-providers/discord-bot/internal/jsdiscord/runtime.go — RuntimeState now exposes discord.channels.send


## 2026-05-25

Updated generated Discord xgoja example to mount express and expose GET / plus POST /say.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/discord-bot/examples/xgoja/discord-bot-provider/bots/fs-express-smoke/index.js — Sample bot registers real Express routes
- /home/manuel/workspaces/2026-05-24/add-js-providers/discord-bot/examples/xgoja/discord-bot-provider/xgoja.yaml — Example selects go-go-goja-http express module
- /home/manuel/workspaces/2026-05-24/add-js-providers/go-go-goja/pkg/xgoja/providers/http/http.go — Discovery runtimes keep HTTP disabled when parsed values are absent


## 2026-05-25

Added focused Discord provider tests and completed final validation.

### Related Files

- /home/manuel/workspaces/2026-05-24/add-js-providers/discord-bot/pkg/xgoja/provider/provider_test.go — Tests cover selected module sections and runtime initializer bridge


## 2026-05-25

Implemented xgoja HTTP/Express provider, Discord outbound runtime API, generated example, tests, live tmux validation, and reMarkable final upload.

