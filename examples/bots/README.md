# Bot CLI example repository

This directory is a realistic example repository for the `go-go-goja bots` command surface.

## Quick start

From the repository root:

```bash
GOWORK=off go run ./cmd/go-go-goja bots list --bot-repository ./examples/bots
```

You should see paths like:

```text
discord announce
discord banner
discord greet
issues list
math leaderboard
math multiply
meta ops status
nested relay
all-values echo-all
```

## Example commands

### 1. Structured output

```bash
GOWORK=off go run ./cmd/go-go-goja bots run discord greet --bot-repository ./examples/bots Manuel --excited
```

### 2. Text output

```bash
GOWORK=off go run ./cmd/go-go-goja bots run discord banner --bot-repository ./examples/bots Manuel
```

### 3. Async Promise settlement

```bash
GOWORK=off go run ./cmd/go-go-goja bots run math multiply --bot-repository ./examples/bots 6 7
```

### 4. Positional string-list expansion

```bash
GOWORK=off go run ./cmd/go-go-goja bots run math leaderboard --bot-repository ./examples/bots Alice Bob Charlie
```

### 5. Relative require from nested files

```bash
GOWORK=off go run ./cmd/go-go-goja bots run nested relay --bot-repository ./examples/bots hi there
```

### 6. Bound sections + context

```bash
GOWORK=off go run ./cmd/go-go-goja bots run issues list --bot-repository ./examples/bots acme/repo --state closed --labels bug --labels docs
```

### 7. Package metadata affecting command path

```bash
GOWORK=off go run ./cmd/go-go-goja bots run meta ops status --bot-repository ./examples/bots
```

### 8. `bind: all`

```bash
GOWORK=off go run ./cmd/go-go-goja bots run all-values echo-all --bot-repository ./examples/bots --repo acme/demo --dryRun --names one --names two
```

## Help examples

```bash
GOWORK=off go run ./cmd/go-go-goja bots help discord greet --bot-repository ./examples/bots
GOWORK=off go run ./cmd/go-go-goja bots help issues list --bot-repository ./examples/bots
```
