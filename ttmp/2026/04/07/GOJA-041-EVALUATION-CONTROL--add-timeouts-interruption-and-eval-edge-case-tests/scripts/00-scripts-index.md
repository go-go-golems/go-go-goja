---
Title: Scripts index
Ticket: GOJA-041-EVALUATION-CONTROL
Status: active
Topics:
    - goja
    - go
    - repl
    - analysis
    - documentation
DocType: reference
Intent: long-term
Summary: "Numbered experiments used to retrace the GOJA-041 interruption investigation."
LastUpdated: 2026-04-08T18:29:00-04:00
WhatFor: "Provide exact scripts and commands for replaying the interruption experiments."
WhenToUse: "Use when retracing how the interruption investigation moved from the eventloop-runtime mismatch clue to the working engine/runtimeowner interrupt design."
---

# GOJA-041 Scripts

These are the numbered experiments used during the GOJA-041 interruption investigation.

Run them in numeric order if you want to retrace the reasoning:

1. `01-goja-plain-runtime-interrupt/main.go`
   - Run from the upstream `goja` checkout:
   - `cd /home/manuel/code/others/goja && go run /home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/scripts/01-goja-plain-runtime-interrupt/main.go`
   - Expected result: plain `goja` interrupt succeeds even for an async IIFE with `while (true) {}`.

2. `02-goja-nodejs-eventloop-interrupt/main.go`
   - Run from the `go-go-goja` repo:
   - `cd /home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja && go run /home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/scripts/02-goja-nodejs-eventloop-interrupt/main.go`
   - Expected result: interrupt appears to fail and the script times out waiting.

3. `03-eventloop-same-vm-check/main.go`
   - Run from the `go-go-goja` repo:
   - `cd /home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja && go run /home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/scripts/03-eventloop-same-vm-check/main.go`
   - Expected result: `sameVM false`
   - This is the key clue: `eventloop.NewEventLoop()` creates its own runtime, so interrupting a different runtime instance cannot work.

4. `04-engine-runtimeowner-interrupt-sync-loop/main.go`
   - Run from the `go-go-goja` repo:
   - `cd /home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja && go run /home/manuel/workspaces/2026-04-03/js-repl-smailnail/go-go-goja/ttmp/2026/04/07/GOJA-041-EVALUATION-CONTROL--add-timeouts-interruption-and-eval-edge-case-tests/scripts/04-engine-runtimeowner-interrupt-sync-loop/main.go`
   - Expected result:
     - `sameVM true`
     - interrupt error is returned
     - `interruptIs true`
     - `postInterrupt value=2 err=<nil>`
   - This is the decisive experiment for the current repo architecture: the `engine.Factory -> Runtime -> Owner.Call -> rt.VM.RunString` path is interruptible and reusable after `ClearInterrupt()`.
