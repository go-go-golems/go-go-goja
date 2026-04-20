---
Title: Real bot example repository smoke tests
Ticket: GOJA-18-BOT-CLI-VERBS
Status: active
Topics:
    - goja
    - javascript
    - cli
    - cobra
    - glazed
    - bots
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: examples/bots/README.md
      Note: Provides the repo-local quick-start commands mirrored by the playbook
    - Path: examples/bots/all-values.js
      Note: Exercises bind-all behavior in the real example repository
    - Path: examples/bots/discord.js
      Note: Exercises structured and text bot outputs in the real example repository
    - Path: examples/bots/issues.js
      Note: Exercises bound sections and context in the real example repository
    - Path: pkg/botcli/command_test.go
      Note: Automates the smoke coverage described by the playbook
ExternalSources: []
Summary: Repeatable smoke-test commands for the real example bot repository under `examples/bots`.
LastUpdated: 2026-04-20T13:00:00-04:00
WhatFor: Give operators and reviewers a copy/paste-ready sequence for validating the bot CLI against real example scripts.
WhenToUse: Use after changing `pkg/botcli`, `pkg/jsverbs`, or the example bot repository.
---


# Real bot example repository smoke tests

## Purpose

Validate the `go-go-goja bots` CLI against a realistic example repository instead of only minimal fixtures.

## Environment assumptions

- Current working directory is the `go-go-goja` repo root.
- Use `GOWORK=off` for commands in this repo.
- The real example repository lives at `./examples/bots`.

## Commands

### 1. List all example bots

```bash
GOWORK=off go run ./cmd/go-go-goja bots list --bot-repository ./examples/bots
```

Expected paths include:

- `all-values echo-all`
- `discord announce`
- `discord banner`
- `discord greet`
- `issues list`
- `math leaderboard`
- `math multiply`
- `meta ops status`
- `nested relay`

### 2. Structured output

```bash
GOWORK=off go run ./cmd/go-go-goja bots run discord greet --bot-repository ./examples/bots Manuel --excited
```

Success signal:
- JSON output includes `"greeting": "Hello, Manuel!"`

### 3. Text output

```bash
GOWORK=off go run ./cmd/go-go-goja bots run discord banner --bot-repository ./examples/bots Manuel
```

Success signal:
- stdout is exactly `*** Manuel ***`

### 4. Async Promise settlement

```bash
GOWORK=off go run ./cmd/go-go-goja bots run math multiply --bot-repository ./examples/bots 6 7
```

Success signal:
- JSON output includes `"product": 42`

### 5. Positional string-list expansion

```bash
GOWORK=off go run ./cmd/go-go-goja bots run math leaderboard --bot-repository ./examples/bots Alice Bob Charlie
```

Success signal:
- output contains rows or JSON objects with ranks and upper-cased names

### 6. Relative require

```bash
GOWORK=off go run ./cmd/go-go-goja bots run nested relay --bot-repository ./examples/bots hi there
```

Success signal:
- JSON output includes `"value": "hi:there"`

### 7. Bound sections + context

```bash
GOWORK=off go run ./cmd/go-go-goja bots run issues list --bot-repository ./examples/bots acme/repo --state closed --labels bug --labels docs
```

Success signal:
- JSON output includes:
  - `"repo": "acme/repo"`
  - `"state": "closed"`
  - `"labelCount": 2`

### 8. Package metadata path shaping

```bash
GOWORK=off go run ./cmd/go-go-goja bots run meta ops status --bot-repository ./examples/bots
```

Success signal:
- JSON output includes `"scope": "meta ops"`

### 9. `bind: all`

```bash
GOWORK=off go run ./cmd/go-go-goja bots run all-values echo-all --bot-repository ./examples/bots --repo acme/demo --dryRun --names one --names two
```

Success signal:
- JSON output includes:
  - `"repo": "acme/demo"`
  - `"dryRun": true`
  - `"count": 2`

### 10. Help output

```bash
GOWORK=off go run ./cmd/go-go-goja bots help issues list --bot-repository ./examples/bots
```

Success signal:
- help output includes `--state` and `--labels`

## Exit criteria

The playbook passes when:

1. the list command shows all expected example paths,
2. all run commands exit successfully,
3. text and structured outputs match their expected shapes,
4. async and relative `require()` cases work,
5. help output renders verb-specific flags.
