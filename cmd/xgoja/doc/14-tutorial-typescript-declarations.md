---
Title: "Tutorial: TypeScript declarations for xgoja runtimes"
Slug: tutorial-typescript-declarations
Short: "Generate and expose .d.ts files for the require() modules selected by xgoja.yaml."
Topics:
- xgoja
- tutorial
- typescript
- developer-experience
- modules
Commands:
- xgoja gen-dts
- xgoja build
- xgoja generate
Flags:
- --out
- --check
- --strict
- --xgoja-replace
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

xgoja runtimes expose Go-backed CommonJS modules through `require()`. JavaScript and TypeScript editors do not know those module shapes unless you provide a declaration file.

Use `xgoja gen-dts` to generate a `.d.ts` file for the exact module set selected by `xgoja.yaml`, including `as:` aliases.

## 1. Generate declarations for a project

From the directory containing `xgoja.yaml`:

```bash
xgoja gen-dts -f xgoja.yaml --out js/types/xgoja-modules.d.ts
```

If you are developing against a local go-go-goja checkout rather than a released module version, pass a replacement path:

```bash
xgoja gen-dts -f xgoja.yaml \
  --out js/types/xgoja-modules.d.ts \
  --xgoja-replace /path/to/go-go-goja
```

The command writes a normal TypeScript declaration file containing blocks such as:

```ts
declare module "path" {
  export function join(...parts: string[]): string;
}
```

If the xgoja spec selects a module with an alias:

```yaml
modules:
  - package: go-go-goja-host
    name: fs
    as: fs:assets
```

then the declaration uses the alias JavaScript authors actually import:

```ts
declare module "fs:assets" {
  // ...
}
```

## 2. Point IntelliJ or GoLand at the generated file

A common layout is:

```text
js/
  src/
  types/
    xgoja-modules.d.ts
  tsconfig.json
```

Reference the generated declarations from `tsconfig.json`:

```json
{
  "compilerOptions": {
    "types": [],
    "typeRoots": ["./types", "./node_modules/@types"]
  },
  "include": ["src", "types"]
}
```

Alternatively, if the project already has a `types` directory included by `tsconfig.json`, no extra configuration may be necessary. JetBrains IDEs generally pick up `.d.ts` files once they are inside an included source root.

## 3. Use strict mode in CI

`--strict` fails if any selected module has no TypeScript descriptor:

```bash
xgoja gen-dts -f xgoja.yaml \
  --out js/types/xgoja-modules.d.ts \
  --strict
```

Use this when the selected runtime API must be fully typed. Leave it off if some selected modules are intentionally untyped.

## 4. Check declarations for drift

Use `--check` to fail when the generated output differs from the checked-in file:

```bash
xgoja gen-dts -f xgoja.yaml \
  --out js/types/xgoja-modules.d.ts \
  --check
```

This is useful in CI after developers update provider modules, aliases, or TypeScript descriptors.

## 5. Inspect the generated sidecar

`xgoja gen-dts` supports arbitrary third-party provider packages by generating and running a small temporary Go sidecar program. Keep that sidecar for debugging with:

```bash
xgoja gen-dts -f xgoja.yaml \
  --out js/types/xgoja-modules.d.ts \
  --keep-work
```

The sidecar imports provider packages from `xgoja.yaml`, registers them, renders the selected declarations, and prints the `.d.ts` payload. This mirrors the way `xgoja build` makes provider imports real in generated Go code.

## 6. Get declarations from a generated binary

Generated xgoja roots also include a `types` command:

```bash
./dist/my-runtime types > js/types/xgoja-modules.d.ts
./dist/my-runtime types --out js/types/xgoja-modules.d.ts
./dist/my-runtime types --check js/types/xgoja-modules.d.ts
./dist/my-runtime types --strict
```

Use this when you have the generated binary but not the original xgoja toolchain invocation.

## 7. Get declarations from generated package mode

When `target.kind: package` or `target.kind: source` is used, the generated `Bundle` exposes declarations programmatically:

```go
bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{})
if err != nil {
    return err
}

dts, err := bundle.TypeScriptDeclarations()
if err != nil {
    return err
}
fmt.Print(dts)
```

You can also stream declarations directly:

```go
err := bundle.WriteTypeScriptDeclarations(w)
```

This lets embedding applications decide whether declarations should be written to disk, served by an HTTP endpoint, or exposed through their own command tree.
