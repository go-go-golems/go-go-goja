# Changelog

## 2026-03-18

- Initial workspace created

## 2026-03-18

Created GOJA-12 for doc-aware `js-repl` autocomplete and help. The main design deliverable recommends reusing the runtime-scoped `docaccess` hub directly from the evaluator so plugin module/export/method docs can enrich completions, the help bar, and the help drawer without evaluating JavaScript or building a second documentation index.

## 2026-03-18

Validated GOJA-12 with `docmgr doctor` and uploaded the bundle to reMarkable at `/ai/2026/03/18/GOJA-12-JS-REPL-DOC-AWARE-HELP`, where the remote listing shows `GOJA-12 Doc-aware js-repl help`.

## 2026-03-18

Implemented the first GOJA-12 slice. The JavaScript evaluator now reads the runtime-scoped docs hub through a dedicated resolver, adds plugin manifest-backed completion candidates for `require()` aliases, prefers docs-derived help text in the help bar, and renders full plugin documentation bodies in the help drawer. The docs registrar is now installed for plugin-backed runtimes even when no additional Glazed/jsdoc sources are configured.
