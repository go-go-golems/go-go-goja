---
Title: "xgoja legacy buildspec migration note"
Slug: buildspec-reference
Short: "Legacy v1 buildspec docs are archived; use xgoja/v2 reference for normal work."
Topics:
- xgoja
- legacy
- migration
Commands:
- xgoja
- xgoja migrate-spec
- help
IsTopLevel: true
IsTemplate: false
ShowPerDefault: false
SectionType: GeneralTopic
---

This page is intentionally retained as a compatibility pointer for older links to `xgoja help buildspec-reference`.

The old buildspec schema is no longer the normal xgoja configuration model. Normal commands now expect native v2 specs:

```yaml
schema: xgoja/v2
name: my-app

providers:
  - id: core
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/core

runtime:
  modules:
    - provider: core
      name: path
      as: path

commands:
  - id: eval
    type: builtin.eval
    name: eval

artifacts:
  - id: binary
    type: binary
    output: dist/my-app
```

Use these docs instead:

- `xgoja help user-guide` for the current v2 user guide.
- `xgoja help xgoja-v2-reference` for the complete v2 field reference.
- `xgoja help migrating-to-xgoja-v2` for migration details.

To convert an old v1 file:

```bash
xgoja migrate-spec -f xgoja.yaml --out xgoja.v2.yaml
xgoja doctor -f xgoja.v2.yaml
```

Legacy v1 parsing lives only in the migration command path.
