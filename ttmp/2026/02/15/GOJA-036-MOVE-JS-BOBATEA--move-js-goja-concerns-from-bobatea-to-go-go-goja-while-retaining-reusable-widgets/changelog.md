# Changelog

## 2026-02-15

- Initial workspace created


## 2026-02-15

Created exhaustive Bobatea/go-go-goja JS migration analysis, added boundary/matrix scripts, closed GOJA-024/028/033/034/035, and uploaded analysis PDF to reMarkable

### Related Files

- /home/manuel/workspaces/2026-02-15/use-bobatea-goja/go-go-goja/ttmp/2026/02/15/GOJA-036-MOVE-JS-BOBATEA--move-js-goja-concerns-from-bobatea-to-go-go-goja-while-retaining-reusable-widgets/analysis/01-bobatea-to-go-go-goja-js-goja-migration-deep-analysis.md — Primary 8+ page analysis output
- /home/manuel/workspaces/2026-02-15/use-bobatea-goja/go-go-goja/ttmp/2026/02/15/GOJA-036-MOVE-JS-BOBATEA--move-js-goja-concerns-from-bobatea-to-go-go-goja-while-retaining-reusable-widgets/scripts/analyze_bobatea_goja_boundary.sh — Coupling boundary experiment script
- /home/manuel/workspaces/2026-02-15/use-bobatea-goja/go-go-goja/ttmp/2026/02/15/GOJA-036-MOVE-JS-BOBATEA--move-js-goja-concerns-from-bobatea-to-go-go-goja-while-retaining-reusable-widgets/scripts/widget_reuse_matrix.sh — Widget reuse matrix experiment script


## 2026-02-15

Implemented phased migration: moved JS evaluator ownership to go-go-goja, added adapter + cmd/js-repl, retired Bobatea js-repl/evaluator, and integrated suggest/contextbar/contextpanel widgets into smalltalk-inspector REPL with passing cross-repo regressions.

### Related Files

- /home/manuel/workspaces/2026-02-15/use-bobatea-goja/bobatea/examples/js-repl/README.md — Retired Bobatea JS REPL example and pointed to go-go-goja
- /home/manuel/workspaces/2026-02-15/use-bobatea-goja/bobatea/go.mod — Dependency cleanup after removing JS evaluator package
- /home/manuel/workspaces/2026-02-15/use-bobatea-goja/go-go-goja/cmd/js-repl/main.go — New go-go-goja-owned JS REPL command
- /home/manuel/workspaces/2026-02-15/use-bobatea-goja/go-go-goja/cmd/smalltalk-inspector/app/repl_widgets.go — Inspector REPL widget integration
- /home/manuel/workspaces/2026-02-15/use-bobatea-goja/go-go-goja/pkg/repl/adapters/bobatea/javascript.go — Adapter exposing Bobatea REPL interfaces from go-go-goja
- /home/manuel/workspaces/2026-02-15/use-bobatea-goja/go-go-goja/pkg/repl/evaluators/javascript/evaluator.go — Moved JS evaluator ownership and added runtime reuse support

