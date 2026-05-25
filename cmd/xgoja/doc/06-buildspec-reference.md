---
Title: "xgoja buildspec quick reference"
Slug: buildspec-reference
Short: "Quick pointers for xgoja.yaml fields; see user-guide for the full reference."
Topics:
- xgoja
- buildspec
- yaml
Commands:
- xgoja
- xgoja build
- xgoja doctor
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

Use `xgoja help user-guide` for the full buildspec reference.

The most common `xgoja.yaml` shape is:

```yaml
name: my-app
target:
  kind: xgoja
  output: dist/my-app
packages:
  - id: go-go-goja-core
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/core
runtimes:
  main:
    modules:
      - package: go-go-goja-core
        name: path
        as: path
commands:
  eval:
    enabled: true
    runtime: main
  run:
    enabled: true
    runtime: main
```

Validate and build with:

```bash
xgoja doctor -f xgoja.yaml
xgoja build -f xgoja.yaml --xgoja-replace /path/to/go-go-goja
```
