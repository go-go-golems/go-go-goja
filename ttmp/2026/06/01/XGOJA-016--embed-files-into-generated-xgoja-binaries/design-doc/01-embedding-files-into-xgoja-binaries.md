---
Title: Embedding files into xgoja binaries
Ticket: XGOJA-016
Status: active
Topics:
    - architecture
    - fs
    - goja
    - goja-nodejs
    - modules
    - providers
    - runtime
    - templates
    - xgoja
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/cmd/xgoja/internal/buildspec/spec.go
      Note: Build-time xgoja YAML schema; target location for the proposed assets field.
    - Path: go-go-goja/cmd/xgoja/internal/buildspec/validate.go
      Note: Validator pattern for jsverbs/help sources; target location for asset path and ID validation.
    - Path: go-go-goja/cmd/xgoja/internal/generate/generate.go
      Note: Generated workspace writer and current embedded jsverbs/help copy pipeline to generalize for assets.
    - Path: go-go-goja/cmd/xgoja/internal/generate/main.go
      Note: Embedded runtime spec rendering and generated path rewriting pattern for embedded sources.
    - Path: go-go-goja/cmd/xgoja/internal/generate/templates/main.go.tmpl
      Note: Generated Go entrypoint template where embeddedAssets go:embed should be emitted.
    - Path: go-go-goja/modules/fs/fs.go
      Note: Current JavaScript fs API surface and loader; target for backend-backed embedded asset reads.
    - Path: go-go-goja/modules/fs/fs_async.go
      Note: Async fs promise behavior that must be preserved when backend calls are introduced.
    - Path: go-go-goja/modules/fs/fs_sync.go
      Note: Current host OS sync operations; target for OS backend extraction.
    - Path: go-go-goja/pkg/xgoja/app/factory.go
      Note: Runtime profile module construction and ModuleContext plumbing for host asset services.
    - Path: go-go-goja/pkg/xgoja/app/host.go
      Note: App host owns embedded filesystems and attaches commands; target for EmbeddedAssets host options.
    - Path: go-go-goja/pkg/xgoja/app/root.go
      Note: NewRootCommand options and embedded jsverbs scanning precedent for runtime FS handles.
    - Path: go-go-goja/pkg/xgoja/providers/host/host.go
      Note: Guarded host fs provider; target for per-alias host and embedded-only fs configuration.
ExternalSources: []
Summary: Design and implementation guide for embedding arbitrary files into generated xgoja binaries and exposing them through separately configured fs module instances such as require("fs:assets") and require("fs:host").
LastUpdated: 2026-06-01T08:09:12.246661629-04:00
WhatFor: Use this when implementing XGOJA-016 or onboarding an engineer to xgoja generation, embedded source handling, runtime profiles, provider modules, and the current fs module.
WhenToUse: Before changing xgoja buildspec, generated templates, app host/runtime wiring, or modules/fs to support embedded assets.
---


# Embedding files into xgoja binaries

## Executive summary

xgoja already has two narrow embedding pipelines: local JavaScript verb sources and local Glazed help documents can be copied into the generated Go module and embedded into the final binary with `go:embed`. The new feature should generalize that pattern to arbitrary asset directories, then let selected runtime profiles expose those embedded assets through separately configured fs module instances such as `require("fs:assets")` and `require("fs:host")`.

The recommended design is a three-layer change:

1. **Buildspec and generation layer:** add top-level `assets:` entries, copy embedded asset directories into `xgoja_embed/assets/<sanitized-id>/`, rewrite the embedded runtime spec to those generated roots, and emit `var embeddedAssets embed.FS` in generated `main.go` when needed.
2. **xgoja app/runtime layer:** add `EmbeddedAssets fs.FS` to `app.Options`, `app.HostOptions`, and `app.Host`; build an `AssetStore` from the normalized embedded spec; and pass that store through `providerapi.ModuleContext.Host` when runtime modules are constructed.
3. **fs module layer:** refactor `modules/fs` from direct `os.*` calls into a backend-backed module. The same provider module (`name: fs`) can be selected multiple times in one runtime under different `as:` names, each with its own config and backend. The host backend remains available for `config.allow: true`; a new read-only embedded backend resolves configured virtual mount paths against the xgoja `AssetStore`; optional overlay behavior can remain a later enhancement.

The safest initial behavior is **separate read-only embedded assets and host filesystem instances**. JavaScript can call `require("fs:assets").readFileSync("/app/config.json", "utf8")`, `require("fs:assets").readdirSync("/app/templates")`, `existsSync(...)`, `statSync(...)`, and async equivalents. Host filesystem access remains available through a separately configured alias such as `require("fs:host")`, and only when the runtime profile explicitly includes an instance with `config.allow: true`. Write operations against embedded mounts should fail with an `EROFS`-style JavaScript error.

## Problem statement and scope

Generated xgoja binaries are intended to be self-contained application runtimes. Today, a generated binary can embed JavaScript verbs and help docs, but application code that runs inside Goja cannot read arbitrary bundled files through the normal Node-like `fs` API. That means generated tools still need external runtime directories for templates, configuration fixtures, web assets, seed files, snippets, or data files unless each use case invents a bespoke provider module.

The requested outcome is:

- a generated binary can include arbitrary files from the project at build time;
- those files do not need to exist on disk when the generated binary runs;
- JavaScript can read them through the existing fs module implementation when the runtime profile registers a configured fs instance for assets;
- runtime profiles that do not opt in cannot see the embedded assets;
- host filesystem access remains a separate, explicit capability, preferably registered under a different `as:` name such as `fs:host`.

Out of scope for the first implementation:

- a fully writable virtual filesystem;
- hot reloading embedded assets after build;
- glob-level include/exclude filters; unsupported filter fields are rejected until implemented;
- exposing embedded files through HTTP static serving directly; JavaScript or a provider can layer that later using `fs` reads.

## Current-state architecture

### Mental model: what xgoja builds

A generated xgoja binary is a normal Go CLI produced from `xgoja.yaml`. The `xgoja build` command loads and validates the spec, writes a generated Go workspace, runs `go mod tidy`, and runs `go build`.

```text
xgoja.yaml
   |
   v
cmd/xgoja/internal/buildspec.LoadFile
   |
   v
cmd/xgoja/internal/generate.WriteAll
   |      copies selected embedded sources
   |      renders go.mod, main.go, xgoja.gen.json
   v
generated workspace
   |
   v
go build -o target.output .
   |
   v
self-contained generated binary
```

Evidence:

- `cmd/xgoja/cmd_build.go:87-113` loads the spec and calls `generate.WriteAll`.
- `cmd/xgoja/cmd_build.go:121-132` runs `go mod tidy` and `go build`.
- `cmd/xgoja/internal/generate/generate.go:23-49` creates the generated directory, copies embedded sources, then writes `go.mod`, `main.go`, and `xgoja.gen.json`.
- `cmd/xgoja/internal/buildexec/buildexec.go:14-28` is the small wrapper around `go mod tidy` and `go build`.

### Current buildspec shape

`cmd/xgoja/internal/buildspec/spec.go` defines the YAML schema used at build time. It already has top-level `JSVerbs` and `Help`, both of which can represent local filesystem sources with `embed: true`.

Important fields:

- `Spec.Packages`, `Spec.Runtimes`, and `Spec.Commands` are the core generated runtime model (`spec.go:5-15`).
- `ModuleInstance.Config` is a free-form `map[string]any` that is passed to provider modules (`spec.go:45-50`).
- `JSVerbSourceSpec` has `id`, `path`, `embed`, `package`, and `source` (`spec.go:88-94`).
- `HelpSourceSpec` has the same basic shape (`spec.go:100-106`).

The runtime-side copy of the spec is in `pkg/xgoja/app/spec.go`. It mirrors the JSON payload that generated `main.go` embeds. Any new `assets` field needs to be added to both the build-time spec and runtime app spec.

### Current embedded source pipeline

xgoja already proves that build-time copying plus generated `go:embed` works.

For JavaScript verbs:

- `copyEmbeddedJSVerbs` resolves a source path relative to the spec file and copies it into the generated workspace (`generate.go:52-68`).
- `embeddedJSVerbRoots` rewrites local embedded sources into `xgoja_embed/jsverbs/<sanitized-id>` (`main.go:101-124`).
- `runtimeSpec` rewrites the embedded JSON spec so the generated binary points at the generated root, not the developer's original source directory (`main.go:51-74`).
- `main.go.tmpl` emits `//go:embed xgoja_embed/jsverbs/*` and passes `embeddedJSVerbs` into `app.NewRootCommand` when there is at least one embedded local jsverb source (`templates/main.go.tmpl:27-33`, `templates.go:85-99`).
- At runtime, `scanVerbSource` uses `jsverbs.ScanFS(embeddedJSVerbs, source.Path)` for embedded sources (`pkg/xgoja/app/root.go:293-301`).

For help docs:

- `copyEmbeddedHelpSources` copies local embedded help sources into `xgoja_embed/help/<sanitized-id>` (`generate.go:71-87`).
- `embeddedHelpRoots` computes collision-free generated roots (`main.go:128-152`).
- `loadConfiguredHelpSources` uses `helpSystem.LoadSectionsFromFS(opts.EmbeddedHelp, path)` for embedded help sources (`pkg/xgoja/app/framework.go:90-96`).

An investigation script in this ticket confirmed the current behavior:

```text
scripts/01-inspect-current-embedded-sources.sh
scripts/01-inspect-current-embedded-sources.out
```

The script dry-runs `examples/xgoja/07-embedded-jsverbs/xgoja.yaml` and shows that generated `main.go` contains:

```go
//go:embed xgoja_embed/jsverbs/*
var embeddedJSVerbs embed.FS
```

and that `xgoja.gen.json` rewrites the original `./verbs` path to `xgoja_embed/jsverbs/local`.

### Current runtime profile and provider module flow

A runtime profile selects which provider modules are available through `require()`. The generated app does not expose every compiled provider module to every command automatically.

```text
generated main.go
   |
   v
providerapi.Registry contains compiled provider packages
   |
   v
app.NewRootCommand -> app.NewHostWithOptions
   |
   v
Host.AttachDefaultCommands
   |
   v
command chooses runtime profile
   |
   v
RuntimeFactory.NewRuntime(profile)
   |
   v
for each module instance: provider module factory receives config JSON
   |
   v
reg.RegisterNativeModule(alias, loader)
   |
   v
JavaScript: require(alias)
```

Evidence:

- `pkg/xgoja/app/host.go:11-18` stores providers, spec, factory, embedded jsverbs/help, and output writer.
- `pkg/xgoja/app/host.go:41-62` attaches built-in commands and command providers.
- `pkg/xgoja/app/factory.go:54-82` constructs a runtime from the named profile.
- `pkg/xgoja/app/factory.go:62-69` resolves each `runtime.modules[]` entry against the provider registry.
- `pkg/xgoja/app/factory.go:29-47` marshals `ModuleInstance.Config` and registers the module loader under the runtime alias.
- `pkg/xgoja/app/root.go:117-154` shows the eval command creating a fresh runtime and executing JavaScript.
- `pkg/xgoja/app/run.go:81-120` shows the run command creating a fresh runtime and executing a script as a module.

One notable unused hook exists already: `providerapi.ModuleContext` has a `Host providerapi.HostServices` field (`pkg/xgoja/providerapi/module.go:12-18`), but `providerRuntimeModuleSpec.RegisterRuntimeModule` currently does not populate it (`pkg/xgoja/app/factory.go:37-42`). This is the natural channel for passing embedded asset services to provider module factories without global variables.

### Current fs module behavior

The existing `modules/fs` package implements a Node-like subset with async and sync functions. It is currently host-filesystem only.

Evidence:

- The fs module exposes read/write/list/stat/remove APIs in TypeScript metadata (`modules/fs/fs.go:27-64`).
- The loader requires runtime services for async promise resolution (`modules/fs/fs.go:82-87`).
- Sync read calls `readFileBytes`, which calls `os.ReadFile` (`modules/fs/fs.go:134-138`, `modules/fs/fs_sync.go:21-24`).
- Sync write calls `writeFileBytes`, which calls `os.WriteFile` (`modules/fs/fs.go:139-145`, `modules/fs/fs_sync.go:26-28`).
- Directory listing and stat call `os.ReadDir` and `os.Stat` (`modules/fs/fs_sync.go:42-59`).
- Async operations run the same functions from goroutines and resolve/reject promises back on the runtime owner (`modules/fs/fs_async.go:11-67`).

The host provider exposes this module only behind an explicit config guard:

- `pkg/xgoja/providers/host/host.go:35-45` registers `fs`, `node:fs`, `exec`, `database`, and `db` as guarded host-capability modules.
- `pkg/xgoja/providers/host/host.go:48-70` wraps the native fs module and requires `config.allow=true` before returning the loader.
- The description explicitly states that this module touches the host filesystem and does not sandbox paths (`host.go:51-59`).

That guard must not be weakened by embedded assets. Embedded-only reads should not imply host filesystem access.

## Gap analysis

The current codebase has most of the required infrastructure, but the pieces are specialized:

| Need | Current support | Gap |
| --- | --- | --- |
| Copy arbitrary local files into generated workspace | `copyDir` exists for jsverbs/help | No generic `assets` spec or copy function |
| Emit generated `go:embed` for arbitrary files | Template emits jsverbs/help FS variables | No `embeddedAssets embed.FS` variable |
| Embed normalized runtime metadata | `RenderEmbeddedSpec` rewrites jsverbs/help paths | No asset metadata in embedded JSON |
| Pass embedded FS to runtime app | `app.Options` and `HostOptions` accept jsverbs/help FS | No embedded assets field or asset store |
| Pass host services into modules | `ModuleContext.Host` exists | Runtime factory does not populate it |
| Let `fs` read from non-host backends | Not supported | fs module is hardwired to `os.*` functions |
| Keep host fs guarded | Host provider requires `allow: true` | New design must preserve this contract |

## Proposed buildspec API

### Top-level asset declarations

Add a top-level `assets:` list to `xgoja.yaml`.

Minimal shape:

```yaml
assets:
  - id: app-assets
    path: ./assets
    embed: true
```

Recommended full shape for initial implementation:

```yaml
assets:
  - id: app-assets
    path: ./assets
    embed: true
    description: "Templates and static data used by JavaScript commands"
```

Do not accept `include` or `exclude` fields in the initial schema. Until filtering is implemented, accepting those fields would let a buildspec claim a narrower asset set than the generator actually embeds.

Initial semantics:

- `id` is required and unique within `assets`.
- `path` is required for local asset sources.
- `embed: true` means the source directory must exist at build time and is copied into the generated workspace.
- `embed: false` may be accepted later for development-time runtime filesystem assets, but the core feature should start with `embed: true` because the requested outcome is a self-contained binary.
- Source paths resolve relative to the directory containing `xgoja.yaml`, using the same `resolveSourcePath` convention as current jsverb/help embedding.
- The generated root is `xgoja_embed/assets/<sanitized-id>`, with collision suffixes such as `_2` if sanitized IDs collide.

Build-time schema additions:

```go
// cmd/xgoja/internal/buildspec/spec.go

type Spec struct {
    // existing fields...
    Assets []AssetSourceSpec `yaml:"assets"`
}

type AssetSourceSpec struct {
    ID          string `yaml:"id" json:"id"`
    Path        string `yaml:"path" json:"path,omitempty"`
    Embed       bool   `yaml:"embed" json:"embed"`
    Description string `yaml:"description" json:"description,omitempty"`
}
```

Runtime schema additions:

```go
// pkg/xgoja/app/spec.go

type Spec struct {
    // existing fields...
    Assets []AssetSourceSpec `json:"assets,omitempty"`
}

type AssetSourceSpec struct {
    ID          string `json:"id"`
    Path        string `json:"path,omitempty"`
    Embed       bool   `json:"embed"`
    Description string `json:"description,omitempty"`
}
```

### Runtime fs module configuration

Assets should not be globally visible just because they are compiled into the binary. Visibility belongs to runtime module config, because runtime profiles are the xgoja policy boundary.

The preferred API is to register the same provider module more than once with different `as:` names. In xgoja, `as:` is the require name used for `reg.RegisterNativeModule(instance.Alias(), loader)`; it is not an additional alias layered on top of `name`. Therefore `name: fs` plus `as: fs:assets` registers `require("fs:assets")` only. It does not also register `require("fs")`. A plain `require("fs")` exists only if a runtime module instance omits `as:` or sets `as: fs`.

Recommended split-host-and-assets config:

```yaml
packages:
  - id: go-go-goja-host
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/host

assets:
  - id: app-assets
    path: ./assets
    embed: true

runtimes:
  main:
    modules:
      # Read-only embedded assets, available as require("fs:assets").
      - package: go-go-goja-host
        name: fs
        as: fs:assets
        config:
          embedded:
            allow: true
            mounts:
              - asset: app-assets
                mount: /app

      # Host filesystem, available as require("fs:host").
      # This remains explicitly gated by allow: true.
      - package: go-go-goja-host
        name: fs
        as: fs:host
        config:
          allow: true
```

Expected JavaScript:

```js
const assetsFS = require("fs:assets")
const hostFS = require("fs:host")

const text = assetsFS.readFileSync("/app/templates/welcome.txt", "utf8")
const names = assetsFS.readdirSync("/app/templates")
const exists = assetsFS.existsSync("/app/config/default.json")

// Host reads and writes are explicit and visually distinct in code review.
hostFS.writeFileSync("./out.txt", text, "utf8")
```

Alias semantics:

- `name` selects the provider module implementation from the provider registry.
- `as` selects the JavaScript `require()` name registered in this runtime.
- Each runtime module entry has its own `config`, so `fs:assets` and `fs:host` can close over different backends.
- Two module entries may use the same `package` and `name` as long as their resolved aliases differ.
- xgoja validation rejects duplicate aliases in the same runtime profile.
- `require("fs")` does not clash with `require("fs:assets")` or `require("fs:host")` unless the runtime also registers an instance whose alias is exactly `fs`.

Semantics:

- With `config.embedded.allow: true` and no `config.allow: true`, that fs instance is read-only and only sees configured embedded mounts.
- With `config.allow: true` and no embedded config, that fs instance behaves like the current guarded host fs module.
- Prefer separate aliases (`fs:assets`, `fs:host`) over a single overlay `fs` instance because call sites make the security boundary obvious.
- Overlay mode can be added later for compatibility, but it should not be the default API. If a runtime intentionally registers a combined `as: fs`, reads may consult the embedded mount table first and then host filesystem only when `config.allow: true`; writes under embedded mount paths should still fail with `EROFS`.
- `node:fs` can accept the same config shape if users want Node-style names, for example `as: node:fs:assets`.

Configuration structs:

```go
type FSConfig struct {
    Allow    bool           `json:"allow"` // existing host fs guard
    Embedded EmbeddedConfig `json:"embedded"`
}

type EmbeddedConfig struct {
    Allow      bool         `json:"allow"`
    Mounts     []AssetMount `json:"mounts"`
    Precedence string       `json:"precedence,omitempty"` // optional overlay-only knob; default "embedded-first"
}

type AssetMount struct {
    Asset string `json:"asset"`
    Mount string `json:"mount"`
    Root  string `json:"root,omitempty"` // optional subdirectory inside the asset
}
```

## Proposed runtime architecture

### New app-level asset store

Add an xgoja-owned host service that maps asset IDs to filesystem roots inside the generated `embed.FS`.

```go
// pkg/xgoja/app/assets.go

type AssetStore struct {
    fsys   fs.FS
    assets map[string]AssetSourceSpec
}

func NewAssetStore(fsys fs.FS, spec *Spec) (*AssetStore, error) {
    // validate duplicate IDs, required paths for embedded assets,
    // and that embedded fsys can open each configured root.
}

func (s *AssetStore) Resolve(id string) (fs.FS, string, bool) {
    // returns the embedded fs and root path for asset id.
}
```

Host service wrapper:

```go
// pkg/xgoja/app/host_services.go

type HostServices struct {
    Assets *AssetStore
}
```

Wire it through app construction:

```go
type Options struct {
    Providers       *providerapi.Registry
    SpecJSON        string
    Out             io.Writer
    EmbeddedJSVerbs fs.FS
    EmbeddedHelp    fs.FS
    EmbeddedAssets  fs.FS
}

type HostOptions struct {
    EmbeddedJSVerbs fs.FS
    EmbeddedHelp    fs.FS
    EmbeddedAssets  fs.FS
    Out             io.Writer
}

type Host struct {
    Providers       *providerapi.Registry
    Spec            *Spec
    Factory         *RuntimeFactory
    EmbeddedJSVerbs fs.FS
    EmbeddedHelp    fs.FS
    EmbeddedAssets  fs.FS
    Services        HostServices
    Out             io.Writer
}
```

Then populate `ModuleContext.Host`:

```go
type RuntimeFactory struct {
    providers *providerapi.Registry
    spec      *Spec
    services  providerapi.HostServices
}

type providerRuntimeModuleSpec struct {
    instance ModuleInstance
    module   providerapi.Module
    services providerapi.HostServices
}

loader, err := s.module.New(providerapi.ModuleContext{
    Context: ctx.Context,
    Name:    s.instance.Name,
    As:      s.instance.Alias(),
    Config:  config,
    Host:    s.services,
})
```

This keeps embedded assets out of global state and makes them available only to module factories selected by a runtime profile.

### New generated `main.go` shape

Generated code should follow the existing jsverbs/help pattern.

Current pattern:

```go
//go:embed xgoja_embed/jsverbs/*
var embeddedJSVerbs embed.FS

//go:embed xgoja_embed/help/*
var embeddedHelp embed.FS
```

Proposed addition:

```go
//go:embed all:xgoja_embed/assets/*
var embeddedAssets embed.FS
```

Generated root construction:

```go
root, err := app.NewRootCommand(app.Options{
    Providers:       registry,
    SpecJSON:        embeddedSpecJSON,
    EmbeddedJSVerbs: embeddedJSVerbs,
    EmbeddedHelp:    embeddedHelp,
    EmbeddedAssets:  embeddedAssets,
})
```

For adapter/cobra targets, pass the asset FS through `app.NewHostWithOptions` just like existing embedded jsverbs/help.

### New build generation helpers

Mirror the jsverbs/help helpers rather than inventing a new pattern.

```go
func copyEmbeddedAssets(dir string, spec *buildspec.Spec) error {
    roots := embeddedAssetRoots(spec)
    for i, source := range spec.Assets {
        root := roots[i]
        if root == "" { continue }
        src, err := resolveSourcePath(spec.BaseDir, source.Path)
        if err != nil { return fmt.Errorf("resolve embedded asset source %s: %w", source.ID, err) }
        dst := filepath.Join(dir, filepath.FromSlash(root))
        if err := copyDirWithOptions(dst, src, copyDirOptions{skipNodeModules: true}); err != nil {
            return fmt.Errorf("copy embedded asset source %s: %w", source.ID, err)
        }
    }
    return nil
}

func embeddedAssetRoots(spec *buildspec.Spec) map[int]string {
    // same collision-free sanitized-id logic as embeddedJSVerbRoots
    roots[i] = "xgoja_embed/assets/" + name
}
```

Update `WriteAll` ordering:

```go
if err := copyEmbeddedJSVerbs(dir, spec); err != nil { return err }
if err := copyEmbeddedHelpSources(dir, spec); err != nil { return err }
if err := copyEmbeddedAssets(dir, spec); err != nil { return err }
```

Update `runtimeSpec`:

```go
clone.Assets = append([]buildspec.AssetSourceSpec(nil), spec.Assets...)
roots := embeddedAssetRoots(spec)
for i := range clone.Assets {
    if root := roots[i]; root != "" {
        clone.Assets[i].Path = root
    }
}
```

## Proposed fs module architecture

### Refactor goal

The fs module should stop being hardwired to `os.ReadFile`, `os.WriteFile`, `os.ReadDir`, and `os.Stat`. Instead, the loader should close over a backend. The JavaScript API can stay the same.

Current shape:

```text
fs.Loader
  -> readFileSync JS function
       -> readFileBytes(path)
            -> os.ReadFile(path)
```

Proposed shape:

```text
fs.LoaderWithBackend(backend)
  -> readFileSync JS function
       -> backend.ReadFile(path)
            -> OS backend OR embedded backend OR overlay backend
```

### Backend interface sketch

Keep the first backend interface close to what the current module already supports.

```go
// modules/fs/backend.go

type Backend interface {
    ReadFile(path string) ([]byte, error)
    WriteFile(path string, data []byte, mode os.FileMode) error
    Exists(path string) bool
    Mkdir(path string, recursive bool, mode os.FileMode) error
    ReadDir(path string) ([]string, error)
    Stat(path string) (fileStats, error)
    Remove(path string) error
    AppendFile(path string, data []byte, mode os.FileMode) error
    Rename(oldPath, newPath string) error
    CopyFile(src, dst string) error
    RemoveAll(path string) error
}
```

The initial implementation can reduce this if desired, but matching the existing operations reduces churn in `fs.go`, `fs_sync.go`, and `fs_async.go`.

Backends:

1. `OSBackend`: wraps the current `os.*` operations.
2. `ReadOnlyFSBackend`: wraps `io/fs` for embedded assets and returns `EROFS` for mutating operations.
3. `OverlayBackend`: routes read operations through a virtual mount table and delegates host operations only when host access is enabled.

### Path model

Use slash-separated virtual paths for embedded mounts, regardless of host OS.

Rules:

- Normalize JavaScript paths with `path.Clean("/" + input)` from the Go `path` package, not `filepath`.
- Treat mount points as absolute virtual paths such as `/app`.
- Reject traversal that escapes the embedded asset root.
- Preserve the JavaScript-visible path in errors (`err.path`) so debugging is intuitive.

Example resolver pseudocode:

```go
func (m MountTable) Resolve(jsPath string) (fsys fs.FS, subpath string, ok bool) {
    clean := path.Clean("/" + strings.TrimSpace(jsPath))
    for _, mount := range m.sortedLongestPrefixFirst {
        if clean == mount.Mount || strings.HasPrefix(clean, mount.Mount + "/") {
            rel := strings.TrimPrefix(clean, mount.Mount)
            rel = strings.TrimPrefix(rel, "/")
            if rel == "" { rel = "." }
            return mount.FS, path.Join(mount.Root, rel), true
        }
    }
    return nil, "", false
}
```

Sort mounts by descending mount path length so `/app/templates` wins over `/app`.

### Error behavior

The existing fs error layer converts Go errors into JavaScript errors with `code`, `path`, and `syscall`. Preserve that shape.

Recommended mappings:

| Condition | Code | Syscall |
| --- | --- | --- |
| missing embedded file | `ENOENT` | `open`, `stat`, or `scandir` |
| write under embedded mount | `EROFS` | `open`, `write`, `rename`, `unlink`, or `rm` |
| path escapes mount root | `EACCES` | requested operation |
| unsupported operation | `ENOSYS` or `EROFS` | requested operation |

If the current `fsErrorCode` helper does not map `EROFS`, extend it with a typed sentinel error. Avoid string-matching Go errors.

### Provider integration

Refactor the host provider's fs module factory rather than bypassing it. The factory must treat every runtime module instance independently: the same provider module can be selected twice, and `ctx.Config` plus `ctx.As` determine the backend and JavaScript-visible require name for that specific instance.

Pseudocode:

```go
func fsModule(name string) providerapi.Module {
    return providerapi.Module{
        Name: name,       // provider module name, usually "fs" or "node:fs"
        DefaultAs: name,  // default require name when xgoja.yaml omits as:
        Description: "Guarded filesystem module with optional embedded read-only mounts.",
        ConfigSchema: fsConfigSchema,
        New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
            cfg := FSConfig{}
            if err := decodeConfig(ctx.Config, &cfg); err != nil { return nil, err }

            requireName := ctx.As
            if requireName == "" { requireName = ctx.Name }

            var backend fsmod.Backend

            switch {
            case cfg.Embedded.Allow && cfg.Allow:
                // Supported for compatibility, but prefer separate fs:assets and
                // fs:host instances in new xgoja.yaml files.
                store, err := assetStoreFromHost(ctx.Host)
                if err != nil { return nil, err }
                embedded, err := backendFromMounts(store, cfg.Embedded.Mounts)
                if err != nil { return nil, err }
                backend = fsmod.NewOverlayBackend(embedded, fsmod.OSBackend{})

            case cfg.Embedded.Allow:
                store, err := assetStoreFromHost(ctx.Host)
                if err != nil { return nil, err }
                backend, err = backendFromMounts(store, cfg.Embedded.Mounts)
                if err != nil { return nil, err }

            case cfg.Allow:
                backend = fsmod.OSBackend{}

            default:
                return nil, fmt.Errorf("%s module requires config.allow=true or config.embedded.allow=true", requireName)
            }

            return fsmod.New(fsmod.WithName(requireName), fsmod.WithBackend(backend)).Loader, nil
        },
    }
}
```

This preserves the old `allow: true` contract while adding an embedded-only path that does not grant host filesystem access. It also documents the important alias rule for implementation: if the runtime config says `as: fs:assets`, the module should be registered as `require("fs:assets")`; it should not additionally register `require("fs")`.

## Implementation plan for an intern

### Phase 0: Read and run the current examples

Goal: understand the existing pipeline before editing.

Commands:

```bash
cd /home/manuel/workspaces/2026-06-01/xgoja-embed-assets/go-go-goja
GOWORK=off go test ./cmd/xgoja/internal/buildspec ./cmd/xgoja/internal/generate ./pkg/xgoja/app ./modules/fs -count=1
make -C examples/xgoja/07-embedded-jsverbs smoke
```

Read first:

- `cmd/xgoja/internal/buildspec/spec.go`
- `cmd/xgoja/internal/buildspec/validate.go`
- `cmd/xgoja/internal/generate/generate.go`
- `cmd/xgoja/internal/generate/main.go`
- `cmd/xgoja/internal/generate/templates.go`
- `cmd/xgoja/internal/generate/templates/main.go.tmpl`
- `pkg/xgoja/app/spec.go`
- `pkg/xgoja/app/host.go`
- `pkg/xgoja/app/factory.go`
- `pkg/xgoja/app/root.go`
- `modules/fs/fs.go`
- `modules/fs/fs_sync.go`
- `modules/fs/fs_async.go`
- `pkg/xgoja/providers/host/host.go`

### Phase 1: Add buildspec and runtime spec fields

Files:

- `cmd/xgoja/internal/buildspec/spec.go`
- `pkg/xgoja/app/spec.go`

Tasks:

1. Add `Assets []AssetSourceSpec` to both `Spec` structs.
2. Add `AssetSourceSpec` to both packages.
3. Keep JSON tags on build-time specs because `RenderEmbeddedSpec` marshals buildspec values.
4. Add unit tests proving YAML loads and JSON renders.

Test sketch:

```go
func TestLoadSpecWithAssets(t *testing.T) {
    spec := loadFixture(`
name: assets
packages:
  - id: host
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/host
runtimes:
  main:
    modules:
      - package: host
        name: fs
        as: fs:assets
        config:
          embedded:
            allow: true
            mounts: [{asset: app, mount: /app}]
commands:
  eval: {enabled: true, runtime: main}
assets:
  - id: app
    path: ./assets
    embed: true
`)
    require.Equal(t, "app", spec.Assets[0].ID)
}
```

### Phase 2: Validate assets

File:

- `cmd/xgoja/internal/buildspec/validate.go`

Tasks:

1. Add `validateAssets(report, spec)` and call it from `Validate`.
2. Enforce unique non-empty IDs.
3. Require `path` for local asset sources.
4. For `embed: true`, require an existing directory with `requireExistingDir`.
5. For `embed: false`, either reject in phase 1 or mark OK as a runtime filesystem source. Prefer rejecting until runtime filesystem assets are explicitly designed.
6. Optionally validate runtime fs module config mount references in a later pass. This requires inspecting module config maps and is more invasive; do it only after core embedding works.

Validation pseudocode:

```go
func validateAssets(report *Report, spec *Spec) {
    ids := map[string]struct{}{}
    for i, source := range spec.Assets {
        path := fmt.Sprintf("assets[%d]", i)
        id := strings.TrimSpace(source.ID)
        if id == "" { report.AddError("asset-id", path+".id", "asset id is required") }
        if _, ok := ids[id]; ok { report.AddError("asset-id", path+".id", fmt.Sprintf("duplicate asset id %q", id)) }
        ids[id] = struct{}{}

        if strings.TrimSpace(source.Path) == "" { report.AddError("asset-path", path+".path", "asset path is required") }
        if source.Embed {
            if err := requireExistingDir(spec.BaseDir, source.Path); err != nil { report.AddError("asset-path", path+".path", err.Error()) }
        } else {
            report.AddError("asset-embed", path+".embed", "assets currently require embed: true")
        }
    }
}
```

### Phase 3: Generate embedded assets

Files:

- `cmd/xgoja/internal/generate/generate.go`
- `cmd/xgoja/internal/generate/main.go`
- `cmd/xgoja/internal/generate/templates.go`
- `cmd/xgoja/internal/generate/templates/main.go.tmpl`
- `cmd/xgoja/internal/generate/generate_test.go`

Tasks:

1. Add `copyEmbeddedAssets` using asset-specific copy rules that preserve dot directories and skip `node_modules`.
2. Add `embeddedAssetRoots` using the same collision-free sanitized ID logic as jsverbs/help.
3. Update `runtimeSpec` and `RenderEmbeddedSpec` to include `assets` and rewrite embedded asset paths.
4. Add `hasEmbeddedAssetSources`.
5. Extend `mainTemplateData` with `HasEmbeddedAssets`.
6. Update `HasEmbedded` to include jsverbs, help, or assets.
7. Emit `//go:embed all:xgoja_embed/assets/*` only when needed.
8. Pass `EmbeddedAssets` into `app.NewRootCommand` / `app.NewHostWithOptions`.

Template sketch:

```gotemplate
{{- if .HasEmbeddedAssets }}
//go:embed all:xgoja_embed/assets/*
var embeddedAssets embed.FS
{{ end }}
```

Unit tests:

- render main contains `var embeddedAssets embed.FS` only when assets exist;
- embedded spec rewrites `./assets` to `xgoja_embed/assets/app`;
- duplicate sanitized asset IDs produce `_2` suffixes;
- `WriteAll` copies nested asset files;
- no asset config produces no `embed` import if jsverbs/help are also absent.

### Phase 4: Add app asset store and host-service plumbing

Files:

- `pkg/xgoja/app/spec.go`
- `pkg/xgoja/app/host.go`
- `pkg/xgoja/app/root.go`
- `pkg/xgoja/app/factory.go`
- new `pkg/xgoja/app/assets.go`

Tasks:

1. Add `EmbeddedAssets fs.FS` to `Options`, `HostOptions`, and `Host`.
2. Add `AssetStore` and `HostServices` structs.
3. Build the asset store inside `NewHostWithOptions` after the spec is decoded.
4. Add the service to `RuntimeFactory`.
5. Populate `providerapi.ModuleContext.Host` in `providerRuntimeModuleSpec.RegisterRuntimeModule`.
6. Add tests using `fstest.MapFS` to prove a test provider module can see `ctx.Host`.

Asset store test sketch:

```go
func TestAssetStoreResolveEmbeddedAsset(t *testing.T) {
    fsys := fstest.MapFS{
        "xgoja_embed/assets/app/config.json": {Data: []byte(`{"ok":true}`)},
    }
    spec := &Spec{Assets: []AssetSourceSpec{{ID: "app", Path: "xgoja_embed/assets/app", Embed: true}}}
    store, err := NewAssetStore(fsys, spec)
    require.NoError(t, err)
    fsys2, root, ok := store.Resolve("app")
    require.True(t, ok)
    data, err := fs.ReadFile(fsys2, path.Join(root, "config.json"))
    require.NoError(t, err)
    require.JSONEq(t, `{"ok":true}`, string(data))
}
```

### Phase 5: Refactor modules/fs to use backends

Files:

- `modules/fs/fs.go`
- `modules/fs/fs_sync.go`
- `modules/fs/fs_async.go`
- `modules/fs/fs_errors.go`
- new `modules/fs/backend.go`
- new `modules/fs/backend_embed.go`
- `modules/fs/fs_test.go`

Tasks:

1. Introduce backend interface and OS backend.
2. Replace package-level function calls with backend method calls.
3. Keep `init()` registering `fs` and `node:fs` with OS backend for the non-xgoja default registry path.
4. Add a constructor such as `New(WithName("fs"), WithBackend(backend))` or `NewModule(name, backend)`.
5. Add read-only embedded backend tests with `fstest.MapFS`.
6. Add EROFS tests for mutating operations against embedded mounts.
7. Add async tests for embedded reads because async code crosses goroutines and runtime owner callbacks.

Minimal constructor sketch:

```go
type Option func(*m)

func WithName(name string) Option { return func(m *m) { m.name = name } }
func WithBackend(backend Backend) Option { return func(m *m) { m.backend = backend } }

func New(opts ...Option) modules.NativeModule {
    mod := &m{backend: OSBackend{}}
    for _, opt := range opts { opt(mod) }
    if mod.name == "" { mod.name = "fs" }
    return mod
}

func init() {
    modules.Register(New(WithName("fs")))
    modules.Register(New(WithName("node:fs")))
}
```

### Phase 6: Update host provider fs configuration

File:

- `pkg/xgoja/providers/host/host.go`

Tasks:

1. Replace `GuardConfig` use for fs with a richer `FSConfig`.
2. Preserve `allow: true` as host filesystem access.
3. Add embedded mount config.
4. Extract `AssetStore` from `ctx.Host` with a small interface to avoid importing all app internals if necessary.

Avoid import cycles. If `host` provider cannot import `pkg/xgoja/app`, define an interface in `providerapi`:

```go
// pkg/xgoja/providerapi/assets.go

type AssetResolver interface {
    ResolveAsset(id string) (fs.FS, string, bool)
}

type HostServices interface {
    AssetResolver() AssetResolver
}
```

Then app implements it, and the provider depends only on `providerapi`.

### Phase 7: Add an end-to-end xgoja example

New example:

```text
examples/xgoja/10-embedded-assets-fs/
  Makefile
  README.md
  xgoja.yaml
  assets/config/default.json
  assets/templates/welcome.txt
  scripts/read-assets.js
```

Example `xgoja.yaml`:

```yaml
name: embedded-assets-fs
target:
  kind: xgoja
  output: dist/embedded-assets-fs
packages:
  - id: host
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/host
assets:
  - id: app-assets
    path: ./assets
    embed: true
runtimes:
  main:
    modules:
      - package: host
        name: fs
        as: fs:assets
        config:
          embedded:
            allow: true
            mounts:
              - asset: app-assets
                mount: /app
commands:
  eval:
    enabled: true
    runtime: main
  run:
    enabled: true
    runtime: main
  repl:
    enabled: false
  jsverbs:
    enabled: false
```

Example smoke script:

```js
const assetsFS = require("fs:assets")
const config = JSON.parse(assetsFS.readFileSync("/app/config/default.json", "utf8"))
const template = assetsFS.readFileSync("/app/templates/welcome.txt", "utf8")
if (config.name !== "embedded-assets-fs") throw new Error("bad config")
if (!template.includes("welcome")) throw new Error("bad template")
console.log("embedded assets ok")
```

Prove self-contained:

```bash
make -C examples/xgoja/10-embedded-assets-fs smoke
make -C examples/xgoja/10-embedded-assets-fs prove-self-contained
```

The self-contained proof should copy the built binary to a temp directory, delete or avoid the source `assets/` directory, and run the script through `eval` or an embedded jsverb.

### Phase 8: Documentation updates

Files:

- `cmd/xgoja/doc/02-user-guide.md`
- `cmd/xgoja/doc/03-tutorial-using-xgoja-yaml.md`
- `cmd/xgoja/doc/06-buildspec-reference.md`
- `pkg/xgoja/doc/01-runtime-overview.md`
- new example README

Add:

- `assets:` reference section;
- fs module embedded mount config examples;
- warning that embedded mounts are read-only;
- warning that `allow: true` still grants host filesystem access and is not needed for embedded-only reads;
- troubleshooting entries for missing asset IDs, missing embedded FS, and `EROFS` writes.

## Testing and validation strategy

### Unit tests

Buildspec:

```bash
GOWORK=off go test ./cmd/xgoja/internal/buildspec -count=1
```

Cover:

- valid embedded asset path;
- missing asset path;
- duplicate asset ID;
- missing directory for `embed: true`;
- generated defaults do not break specs without assets.

Generate:

```bash
GOWORK=off go test ./cmd/xgoja/internal/generate -count=1
```

Cover:

- `RenderMain` includes `embeddedAssets` only when needed;
- `RenderEmbeddedSpec` rewrites asset paths;
- `WriteAll` copies files;
- sanitized ID collisions.

App:

```bash
GOWORK=off go test ./pkg/xgoja/app -count=1
```

Cover:

- `NewRootCommand` stores embedded assets in host services;
- `RuntimeFactory` passes `ModuleContext.Host` to provider modules;
- adapter/cobra paths still receive embedded assets.

fs:

```bash
GOWORK=off go test ./modules/fs -count=1
```

Cover:

- OS backend preserves existing behavior;
- embedded backend can read, readdir, stat, exists;
- writes under embedded mounts fail with `EROFS`;
- async embedded reads resolve with Buffer or string based on encoding;
- async embedded missing files reject with `ENOENT`.

Host provider:

```bash
GOWORK=off go test ./pkg/xgoja/providers/host -count=1
```

Cover:

- old `allow: true` still creates host fs loader;
- empty config still fails;
- `embedded.allow: true` without host allow creates embedded-only backend;
- unknown asset ID fails at module creation with a clear message.

### End-to-end smoke tests

```bash
GOWORK=off go test ./cmd/xgoja/internal/generate -run TestGenerated -count=1
make -C examples/xgoja/10-embedded-assets-fs smoke
make -C examples/xgoja/10-embedded-assets-fs prove-self-contained
```

Add the new example to any existing example smoke workflow if one exists.

## Risks and review focus

### Security boundary between embedded reads and host filesystem

The highest-risk mistake is accidentally making `config.embedded.allow: true` equivalent to host filesystem access. Reviewers should search for all code paths that instantiate `OSBackend` and confirm they require `cfg.Allow == true`.

### Path traversal and mount prefix bugs

Virtual paths must not allow `../` to escape an embedded root. Reviewers should inspect normalization code with cases such as:

- `/app/../secret`
- `/app/templates/../../config/default.json`
- `/application/file` when `/app` is mounted
- `/app2/file` when `/app` is mounted
- `app/file` without leading slash

### Import cycles

The host provider should not create a cycle by importing `pkg/xgoja/app`. Prefer a small `providerapi.AssetResolver` interface implemented by app host services.

### Backwards compatibility

Existing specs use:

```yaml
config:
  allow: true
```

for host `fs`. Keep that working unless a separate breaking-change ticket explicitly removes it.

### Binary size

Embedded assets increase binary size. The implementation preserves dot directories such as `.well-known` for web/static asset use cases and still skips `node_modules`. Do not add `include`/`exclude` fields until the generator enforces them; unsupported filter fields should be rejected rather than silently ignored.

### Async runtime behavior

The async fs implementation resolves promises by posting back through runtime services. Refactoring to a backend must preserve the current owner/lifetime context behavior (`modules/fs/fs_async.go:11-70`).

## Alternatives considered

### Alternative A: Expose assets as a new `assets` module only

This would avoid touching `modules/fs`, but it does not satisfy the request to expose files through the Goja fs module. It also forces JavaScript code to use a custom API instead of familiar `fs.readFileSync` and `fs.readdirSync`.

Verdict: useful as a future convenience module, but not sufficient.

### Alternative B: Extract embedded assets to a temp directory at startup

This would let the current `os.*` fs module read them without backend refactoring.

Problems:

- creates cleanup and lifecycle issues;
- turns read-only embedded data into mutable host files;
- leaks files on crashes;
- weakens the security boundary by making paths host paths;
- complicates self-contained behavior in restricted environments.

Verdict: reject for first implementation.

### Alternative C: Add `require.WithGlobalFolders` for embedded assets

The `run` command already uses module roots for JavaScript module resolution, but this solves `require("./helper")`, not `fs.readFileSync("/app/file")`. `goja_nodejs/require` loaders are module loaders, not general file APIs.

Verdict: not the right layer.

### Alternative D: Make provider packages own all embedded files

Provider-shipped `VerbSource` and `HelpSource` already support package-owned files. For arbitrary application assets, requiring a custom Go provider package for every generated app is too heavy. xgoja should support local project assets directly in `xgoja.yaml`.

Verdict: keep provider-shipped sources as a separate pattern; add local `assets` for app-level files.

## Open questions

1. Should `embed: false` assets be supported immediately for development-time runtime filesystem reads, or should the first implementation require `embed: true` only?
2. Should `assets` support provider-shipped asset sources in the same style as jsverbs/help, or is local project embedding enough for the first ticket?
3. Should embedded mounts be absolute-only (`/app`) or also support relative paths (`app/...`)? The recommended answer is absolute virtual mounts, with relative input normalized to absolute.
4. Should overlay mode be implemented at all in the first pass, or should the supported API be separate aliases only (`fs:assets`, `fs:host`)? The recommended first implementation is separate aliases only; if overlay is later added, host writes must still require existing `allow: true` and embedded mount writes must fail.
5. Should `fs.watch` or streaming APIs be added? Current fs module does not expose them, so no.

## File reference map

| File | Why it matters |
| --- | --- |
| `cmd/xgoja/internal/buildspec/spec.go` | Build-time YAML schema; add `assets` here. |
| `cmd/xgoja/internal/buildspec/validate.go` | Spec validation; add asset ID/path/embed checks here. |
| `cmd/xgoja/internal/buildspec/load.go` | Sets `BaseDir`, which asset path resolution should reuse. |
| `cmd/xgoja/internal/generate/generate.go` | Generated workspace writer and source copier; add asset copy here. |
| `cmd/xgoja/internal/generate/main.go` | Embedded spec renderer and generated root path rewriting; add asset root rewriting here. |
| `cmd/xgoja/internal/generate/templates.go` | Template data model; add embedded asset booleans and constructor arguments here. |
| `cmd/xgoja/internal/generate/templates/main.go.tmpl` | Generated Go entrypoint; add `//go:embed all:xgoja_embed/assets/*` here. |
| `pkg/xgoja/app/spec.go` | Runtime JSON spec; add `assets` here. |
| `pkg/xgoja/app/host.go` | App host owns embedded FS handles and runtime factory; add `EmbeddedAssets` and services here. |
| `pkg/xgoja/app/factory.go` | Runtime profile-to-module construction; populate `ModuleContext.Host` here. |
| `pkg/xgoja/app/root.go` | xgoja root options and embedded FS inputs; add `EmbeddedAssets` here. |
| `pkg/xgoja/providers/host/host.go` | Current guarded fs provider; extend config to create per-alias host, embedded-only, and optional overlay backends here. |
| `pkg/xgoja/providerapi/module.go` | `ModuleContext.Host` already exists; define or use host services interface here. |
| `modules/fs/fs.go` | JavaScript API surface and loader; refactor to backend-backed operations here. |
| `modules/fs/fs_sync.go` | Current OS sync operations; turn into OS backend implementation. |
| `modules/fs/fs_async.go` | Async promise behavior; keep runtime owner posting while swapping backend calls. |
| `modules/fs/fs_errors.go` | JavaScript error shape; extend for `EROFS`/embedded errors. |
| `examples/xgoja/07-embedded-jsverbs` | Existing self-contained embedding example to copy structurally. |
| `examples/xgoja/10-embedded-assets-fs` | Proposed new smoke example for this feature. |

## Quick API reference

### Buildspec

```yaml
assets:
  - id: app-assets
    path: ./assets
    embed: true

runtimes:
  main:
    modules:
      - package: go-go-goja-host
        name: fs
        as: fs:assets
        config:
          embedded:
            allow: true
            mounts:
              - asset: app-assets
                mount: /app
      - package: go-go-goja-host
        name: fs
        as: fs:host
        config:
          allow: true
```

### JavaScript

```js
const assetsFS = require("fs:assets")
const hostFS = require("fs:host")

assetsFS.existsSync("/app/config/default.json")
assetsFS.readdirSync("/app/templates")
assetsFS.statSync("/app/templates/welcome.txt")
assetsFS.readFileSync("/app/templates/welcome.txt", "utf8")
await assetsFS.readFile("/app/templates/welcome.txt", "utf8")

hostFS.writeFileSync("./out.txt", "host writes are explicit", "utf8")
```

### Generated Go

```go
//go:embed all:xgoja_embed/assets/*
var embeddedAssets embed.FS

root, err := app.NewRootCommand(app.Options{
    Providers: registry,
    SpecJSON: embeddedSpecJSON,
    EmbeddedAssets: embeddedAssets,
})
```

### Provider module factory

```go
func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
    // ctx.Config contains runtime module config.
    // ctx.Host provides app-owned asset resolver services.
}
```
