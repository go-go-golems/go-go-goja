# Tasks

## Ticket setup

- [x] Create ticket `GOJA-JSVERBS-INVOKER`
- [x] Add a design doc for the upstream API change
- [x] Add an implementation diary and keep it updated while working

## Design

- [x] Confirm that `Registry.Commands()` currently couples schema generation to runtime-owning execution
- [x] Confirm that `CommandDescriptionForVerb(...)` and `InvokeInRuntime(...)` already expose the lower-level pieces host apps need
- [x] Choose a minimal upstream API that separates command generation from execution policy
- [x] Decide to keep `Commands()` as the default convenience path and add explicit injected-invoker variants rather than changing the default behavior

## Implementation

- [x] Add a reusable invoker function type to `pkg/jsverbs/command.go`
- [x] Add `Registry.CommandsWithInvoker(...)`
- [x] Add `Registry.CommandForVerb(...)`
- [x] Add `Registry.CommandForVerbWithInvoker(...)`
- [x] Update the generated command wrappers so they call the injected invoker when present and fall back to the existing runtime-owning path otherwise
- [x] Keep output-mode behavior unchanged for both structured and text commands

## Tests

- [x] Add a regression test proving `CommandForVerbWithInvoker(...)` uses the custom invoker for structured output verbs
- [x] Add a regression test proving `CommandsWithInvoker(...)` uses the custom invoker for text-output verbs
- [x] Add a regression test proving a nil invoker falls back to the existing `Registry.Commands()` behavior
- [x] Run targeted package validation for `pkg/jsverbs` and `cmd/jsverbs-example`

## Docs

- [x] Update jsverbs package docs to describe the new injected-invoker APIs
- [x] Record the design rationale and validation details in this ticket
- [x] Relate the key implementation and documentation files to the ticket docs

## Follow-up

- [ ] Adopt the new API in downstream host applications that currently need caller-owned runtime execution
