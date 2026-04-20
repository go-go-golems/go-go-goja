# Run Individual JavaScript Bots as CLI Verbs

This workspace contains the research and design bundle for ticket `GOJA-18-BOT-CLI-VERBS`.

## Goal

Define how `go-go-goja` should expose individual JavaScript bots as command-line verbs through the following stable surface:

```text
go-go-goja bots list
go-go-goja bots run <verb>
go-go-goja bots help <verb>
```

## Main documents

- `index.md` — ticket overview and reading order
- `design-doc/01-bot-cli-verbs-architecture-and-implementation-guide.md` — primary intern-friendly design and implementation guide
- `reference/01-bot-cli-verbs-command-surface-and-api-reference.md` — quick reference for commands, APIs, and file map
- `reference/02-diary.md` — chronological work log and delivery record
- `tasks.md` — completed work plus proposed implementation phases
- `changelog.md` — concise summary of ticket milestones

## Structure

- `design-doc/` — long-form architecture and implementation writing
- `reference/` — quick reference and diary
- `playbooks/` — future validation runbooks if implementation proceeds
- `scripts/` — ticket-local helper scripts if needed later
- `sources/` — imported reference material if added later
- `various/` — scratch notes
- `archive/` — deprecated or preserved artifacts

## Suggested reading order

1. `index.md`
2. `design-doc/01-bot-cli-verbs-architecture-and-implementation-guide.md`
3. `reference/01-bot-cli-verbs-command-surface-and-api-reference.md`
4. `reference/02-diary.md`
5. `tasks.md`
