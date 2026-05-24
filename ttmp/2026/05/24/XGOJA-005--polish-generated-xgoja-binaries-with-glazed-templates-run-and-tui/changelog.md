# Changelog

## 2026-05-24

- Initial workspace created


## 2026-05-24

Created XGOJA-005 ticket, tasks, intern-facing implementation guide, and diary.

### Related Files

- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/24/XGOJA-005--polish-generated-xgoja-binaries-with-glazed-templates-run-and-tui/design-doc/01-generated-binary-polish-design-and-implementation-guide.md — Design guide
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/24/XGOJA-005--polish-generated-xgoja-binaries-with-glazed-templates-run-and-tui/reference/01-diary.md — Diary


## 2026-05-24

Uploaded generated binary polish guide to reMarkable at /ai/2026/05/24/XGOJA-005/XGOJA 005 Generated Binary Polish Guide.pdf.

### Related Files

- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/ttmp/2026/05/24/XGOJA-005--polish-generated-xgoja-binaries-with-glazed-templates-run-and-tui/design-doc/01-generated-binary-polish-design-and-implementation-guide.md — Uploaded guide


## 2026-05-24

Step 3: Converted generated main.go rendering from inline strings to an embedded Go template.

### Related Files

- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/cmd/xgoja/internal/generate/main.go — RenderMain now delegates to template renderer
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/cmd/xgoja/internal/generate/templates.go — Template data and execution helper
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/cmd/xgoja/internal/generate/templates/main.go.tmpl — Generated main.go template


## 2026-05-24

Step 4: Installed Glazed logging flags and generated runtime help system in xgoja app roots.

### Related Files

- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/framework.go — Root framework installer
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/host.go — AttachDefaultCommands installs root framework
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/root_test.go — Help/logging regression test
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/doc/doc.go — Generated runtime help docs


## 2026-05-24

Phase 2: Converted generated modules command to Glazed command plumbing while preserving jsverb Glazed mounting.

### Related Files

- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/glazed.go — shared Glazed-to-Cobra helper
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/host.go — attaches generated modules command through Glazed Cobra builder
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/root.go — modules command is now a GlazeCommand
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/root_test.go — updated modules command regression test for Glazed table output


## 2026-05-24

Phase 3: Added generated run command for JavaScript files using xgoja runtime profiles and script module roots.

### Related Files

- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/cmd/xgoja/internal/buildspec/spec.go — Builder command spec includes run
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/host.go — Attaches run command when enabled
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/root_test.go — Run command regression test
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/run.go — Generated runtime run command
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/spec.go — Runtime command spec includes run


## 2026-05-24

Phase 3: Added generated TUI REPL command backed by an xgoja runtime-profile evaluator.

### Related Files

- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/host.go — Attaches TUI command when enabled
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/root_test.go — TUI help regression test
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/tui.go — Generated runtime TUI command and evaluator adapter


## 2026-05-24

Phase 4: Documented generated run/tui commands and extended xgoja examples to smoke-test run scripts.

### Related Files

- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/cmd/xgoja/doc/02-buildspec.md — Builder buildspec reference now documents run and TUI command specs
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/examples/xgoja — Examples now enable and smoke-test generated run commands
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/doc/01-runtime-overview.md — Generated runtime help documents run and TUI


## 2026-05-24

Completed generated xgoja polish: templates, generated help/logging, Glazed modules, run, TUI, docs, examples, and validation.


## 2026-05-24

Renamed generated command schema: commands.eval is one-shot evaluation and commands.repl is the rich Bubble Tea REPL.

### Related Files

- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/cmd/xgoja/internal/buildspec/spec.go — Buildspec command schema rename
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/host.go — Generated command attachment follows eval/repl semantics
- /home/manuel/workspaces/2026-05-22/xgoja/go-go-goja/pkg/xgoja/app/spec.go — Generated runtime command schema rename

