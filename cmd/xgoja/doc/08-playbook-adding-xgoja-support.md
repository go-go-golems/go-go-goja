---
Title: "Playbook: adding xgoja support to an existing repository"
Slug: playbook-adding-xgoja-support
Short: "A step-by-step guide for turning an existing Go repository into an xgoja package provider."
Topics:
- xgoja
- providers
- modules
- command-providers
- help-system
- runtime
- migration
Commands:
- xgoja
- xgoja build
- xgoja doctor
- xgoja list-modules
Flags:
- --xgoja-replace
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

Adding xgoja support to a repository means making the repository discoverable by generated xgoja binaries. The generated binary needs to know which Go packages provide JavaScript modules, which top-level modules should be exposed to JavaScript, which commands should be mounted, and which runtime resources must be started and stopped around command execution.

This guide describes the complete path. It starts with a repository that already contains useful Go code and ends with a generated binary that can load the repository's provider, expose its modules, optionally mount package-owned Glazed commands, and run a smoke test through the generated command path.

## 1. Choose the integration shape

The first decision is not where to put files. The first decision is what kind of JavaScript and CLI surface the repository should expose. A small package may only need one module. A larger application may need modules, runtime configuration, command providers, and live resources such as HTTP servers, hardware sessions, database handles, or event subscriptions.

| Shape | Use it when | Primary API |
| --- | --- | --- |
| Module-only provider | JavaScript should call Go-backed functions with `require(...)`. | `providerapi.ModuleDescriptor` |
| Command provider | The generated binary should mount repository-owned commands. | `providerapi.CommandSetProvider` returning Glazed commands |
| Runtime/session integration | The repository owns runtime resources that need flags, initialization, and cleanup. | package capabilities with config sections and runtime initializers |
| Help documentation provider | Generated binaries should include package-owned API references or tutorials. | `providerapi.HelpSource` |

Most production integrations use more than one shape. For example, a hardware-oriented repository may expose `require("device/ui")`, provide a `devices` command provider, and install a runtime initializer that opens and closes the physical device.

## 2. Upgrade to a compatible go-go-goja version

Start by moving the repository to a go-go-goja version that contains xgoja providers, command providers, package capabilities, `RuntimeServices`, and explicit startup/lifetime runtime contexts.

```bash
go get github.com/go-go-golems/go-go-goja@<version>
go mod tidy
```

During workspace development, `go.work` can point downstream repositories at a local go-go-goja checkout. Before release, run at least one validation pass without workspace replacement so you know the published dependency graph works:

```bash
GOWORK=off go test ./...
```

A breaking go-go-goja API change must be released before downstream repositories that run hooks with `GOWORK=off` can pass without local replacements.

## 3. Add a provider package

A provider package is the discovery boundary. xgoja generated binaries call its `Register` function to learn what the repository contributes.

Recommended locations are:

```text
pkg/xgoja/provider/provider.go
pkg/<repo>js/provider/provider.go
runtime/js/provider/provider.go
```

The exact path is less important than the exported registration shape:

```go
package provider

import (
    "github.com/dop251/goja_nodejs/require"
    "github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

const PackageID = "my-repo"

func Register(registry *providerapi.ProviderRegistry) error {
    return registry.Package(PackageID,
        moduleEntry("my-repo", "Main module for my-repo.", NewMainLoader),
    )
}

func moduleEntry(
    name string,
    description string,
    loader func() require.ModuleLoader,
) providerapi.ModuleDescriptor {
    return providerapi.ModuleDescriptor{
        PackageID:   PackageID,
        ModuleID:    name,
        ModuleName:  name,
        Description: description,
        Loader:      loader(),
    }
}
```

Keep `PackageID` stable. It appears in `xgoja.yaml`, generated build metadata, module listings, and command-provider declarations.

## 4. Make modules loader-friendly

A module should expose a loader that can be registered by either a normal runtime setup path or an xgoja provider. This keeps the module independent from any single registry.

```go
const ModuleName = "my-repo"

func Loader() require.ModuleLoader {
    return func(vm *goja.Runtime, module *goja.Object) {
        exports := module.Get("exports").(*goja.Object)
        _ = exports.Set("hello", func(name string) string {
            return "hello " + name
        })
    }
}

func Register(registry *require.Registry) {
    registry.RegisterNativeModule(ModuleName, Loader())
}
```

This pattern gives you two useful properties. The provider can pass `Loader()` into a `ModuleDescriptor`, while existing application code can continue to call `Register(registry)`.

## 5. Use RuntimeServices for asynchronous modules

A module that settles Promises, calls JavaScript callbacks, or touches `goja.Value` from Go goroutines must go through runtime services. The engine stores services for each VM.

```go
runtimeServices, ok := runtimebridge.Lookup(vm)
if !ok || runtimeServices.Owner == nil {
    panic(vm.NewGoError(fmt.Errorf("module requires runtime services")))
}
```

Do not use the old runtime context APIs:

| Old API | New API |
| --- | --- |
| `runtimebridge.Bindings` | `runtimebridge.RuntimeServices` |
| `runtimebridge.CurrentContext(vm)` | `runtimebridge.CurrentOwnerContext(vm)` |
| `runtimeowner.Runner` | `runtimeowner.RuntimeOwner` |
| `runtimeowner.NewRunner(...)` | `runtimeowner.NewRuntimeOwner(...)` |
| `factory.NewRuntime(ctx)` | `factory.NewRuntime(engine.WithStartupContext(ctx), engine.WithLifetimeContext(ctx))` |

A Promise-settlement pattern looks like this:

```go
promise, resolve, reject := vm.NewPromise()
callCtx := runtimebridge.CurrentOwnerContext(vm)

go func() {
    value, err := slowWork()
    if err != nil {
        _ = runtimeServices.PostWithCustomContext(callCtx, "module.reject", func(context.Context, *goja.Runtime) {
            _ = reject(vm.ToValue(err.Error()))
        })
        return
    }

    _ = runtimeServices.PostWithCustomContext(callCtx, "module.resolve", func(context.Context, *goja.Runtime) {
        _ = resolve(vm.ToValue(value))
    })
}()

return vm.ToValue(promise)
```

Choose helper methods by intent:

| Situation | Helper |
| --- | --- |
| Synchronous callback from JS-facing native code | `CallWithCurrentContext(vm, op, fn)` |
| Deferred callback that belongs to the current owner entry | `PostWithCurrentContext(vm, op, fn)` |
| Runtime-owned background event | `CallWithLifetimeContext(op, fn)` or `PostWithLifetimeContext(op, fn)` |
| HTTP request, Discord event, hardware event, or explicit operation | `CallWithCustomContext(ctx, op, fn)` or `PostWithCustomContext(ctx, op, fn)` |

## 6. Add configuration sections for runtime resources

If a package needs flags, expose a package capability that contributes Glazed sections. The provider owns the schema; the runtime initializer decodes typed settings from parsed values.

```go
type httpCapability struct{}

func (c *httpCapability) CapabilityID() string { return "my-repo.http" }

func (c *httpCapability) GlazedConfigSections(providerapi.SectionRequest) ([]schema.Section, error) {
    section, err := schema.NewSection(
        "my-http",
        "HTTP",
        schema.WithPrefix("http-"),
        schema.WithFields(
            fields.New("listen", fields.TypeString, fields.WithDefault("127.0.0.1:8080")),
        ),
    )
    if err != nil {
        return nil, err
    }
    return []schema.Section{section}, nil
}

func (c *httpCapability) InitRuntimeFromSections(
    ctx context.Context,
    vals *values.Values,
    handle providerapi.RuntimeInitializerHandle,
) error {
    var settings struct {
        Listen string `glazed:"listen"`
    }

    if vals != nil {
        if err := vals.DecodeSectionInto("my-http", &settings); err != nil {
            return err
        }
    }

    return nil
}
```

Register the capability at the package level:

```go
return registry.Package(PackageID,
    moduleEntry(...),
    providerapi.WithPackageCapability(&httpCapability{}),
)
```

Treat `vals == nil` as discovery mode. Discovery should describe sections and modules; it should not open hardware devices, start servers, or mutate external systems.

## 7. Register closers for owned resources

Runtime resources must have a cleanup path. If the initializer opens a server, device, database, browser, event subscription, or goroutine group, register a closer on the engine runtime exposed by the runtime handle.

```go
runtime := handle.EngineRuntime()
if runtime == nil {
    return fmt.Errorf("runtime is nil")
}
return runtime.AddCloser(func(ctx context.Context) error {
    return resource.Close()
})
```

`engine.Runtime.Close(ctx)` cancels runtime lifetime first, then waits briefly for active owner calls, interrupts active JavaScript if necessary, runs closers while runtime services are still registered, and then removes runtimebridge services. Closers should be bounded and should expect the lifetime context to already be canceled.

## 8. Add command providers when the repository owns commands

Command providers return Glazed commands. They do not return Cobra commands. xgoja handles the final Cobra mounting boundary.

```go
func Register(registry *providerapi.ProviderRegistry) error {
    return registry.Package(PackageID,
        moduleEntry(...),
        providerapi.CommandSetProvider{
            Name:         "verbs",
            DefaultMount: "my-repo",
            Description:  "My repository commands",
            NewCommandSet: newCommandSet,
        },
    )
}

func newCommandSet(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
    command, err := cmds.NewBareCommand(
        cmds.NewCommandDescription("hello", cmds.WithShort("Say hello")),
        func(ctx context.Context, vals *values.Values) error {
            return nil
        },
    )
    if err != nil {
        return nil, err
    }

    return &providerapi.CommandSet{
        Commands: []cmds.Command{command},
    }, nil
}
```

If command execution needs JavaScript, use the xgoja-level runtime factory:

```go
rt, err := ctx.RuntimeFactory.NewRuntime(ctx.Context)
if err != nil {
    return nil, err
}
defer rt.Close(context.Background())
```

This API intentionally differs from the lower-level engine API. `providerapi.RuntimeFactory.NewRuntime(ctx, ...)` creates an xgoja runtime from the generated binary's top-level module set. The engine factory uses explicit runtime options.

## 9. Register package-owned help docs

Provider packages can ship Glazed help pages for API references, tutorials, and troubleshooting. This is the right place for documentation that belongs to the package API rather than to one generated application.

Create or reuse a docs package that embeds Markdown help entries:

```go
package doc

import (
    "embed"
    "io/fs"

    "github.com/go-go-golems/glazed/pkg/help"
)

//go:embed topics/*.md tutorials/*.md
var docFS embed.FS

func FS() fs.FS { return docFS }

func AddDocToHelpSystem(helpSystem *help.HelpSystem) error {
    return helpSystem.LoadSectionsFromFS(docFS, ".")
}
```

Then register that filesystem from the xgoja provider:

```go
import helpdoc "example.com/my-repo/docs/help"

func Register(registry *providerapi.ProviderRegistry) error {
    return registry.Package(PackageID,
        providerapi.HelpSource{
            Name:        "runtime-api",
            Description: "JavaScript runtime API reference and tutorials",
            FS:          helpdoc.FS(),
            Root:        ".",
        },
    )
}
```

Generated binary authors opt into the docs from `xgoja.yaml`:

```yaml
help:
  sources:
    - id: my-repo-runtime-api
      package: my-repo
      source: runtime-api
```

After building, validate with:

```bash
./dist/myapp help my-repo-js-api-reference
```

For a complete smoke-tested reference, inspect `examples/xgoja/09-provider-shipped-help-docs`. It builds a generated binary with the Loupedeck provider and verifies `help loupedeck-js-api-reference`.

Keep slugs unique across all built-in, provider, and local help pages. If two selected docs use the same slug, Glazed help loading should fail rather than silently choosing one.

## 10. Add a generated xgoja example

A provider is not complete until a generated binary can load it. Add a small example under the repository:

```text
examples/xgoja/my-repo-provider/
  xgoja.yaml
  Makefile
  README.md
  scripts/smoke.js
```

A minimal `xgoja.yaml` looks like this:

```yaml
name: my-repo-provider

target:
  kind: xgoja
  output: dist/my-repo-provider

packages:
  - id: my-repo
    import: github.com/go-go-golems/my-repo/pkg/xgoja/provider
    replace: ../../..

  - id: go-go-goja-core
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/core

modules:
  - package: my-repo
    module: my-repo
  - package: go-go-goja-core
    module: fs

commands:
  eval:
    enabled: false
  run:
    enabled: true
  repl:
    enabled: false
  jsverbs:
    enabled: false
```

If the repository provides commands, mount the command provider:

```yaml
commandProviders:
  - package: my-repo
    name: verbs
    mount: my-repo
```

## 11. Make the smoke test exercise the generated path

A package-level test is useful, but it does not prove the generated xgoja path. The smoke test should prove that xgoja can build a binary, load the provider, list the selected modules, mount configured commands, and execute one successful command or script.

```make
WORKSPACE_ROOT := $(abspath ../../../../..)
XGOJA_ROOT := $(WORKSPACE_ROOT)/go-go-goja
BIN := $(CURDIR)/dist/my-repo-provider
XGOJA := cd $(XGOJA_ROOT) && GOWORK=off go run ./cmd/xgoja

.PHONY: smoke doctor list build run clean

smoke: doctor list build run

doctor:
	$(XGOJA) doctor -f $(CURDIR)/xgoja.yaml

list:
	$(XGOJA) list-modules -f $(CURDIR)/xgoja.yaml

build:
	$(XGOJA) build -f $(CURDIR)/xgoja.yaml \
		--output $(BIN) \
		--xgoja-replace $(XGOJA_ROOT) \
		--keep-work

run:
	$(BIN) run scripts/smoke.js

clean:
	rm -rf $(CURDIR)/dist
```

For hardware, browser, cloud, or live-network integrations, keep `make smoke` deterministic and CI-safe. Add a separate target such as `make hardware`, `make browser`, or `make live` for interactive validation.

## 12. Validate the repository

Run focused package tests first:

```bash
go test ./pkg/xgoja/provider -count=1
go test ./pkg/... -count=1
```

Then run the generated smoke:

```bash
make -C examples/xgoja/my-repo-provider smoke
```

Search for old runtime APIs before committing:

```bash
rg -n 'runtimebridge\.(Bindings|CurrentContext|OwnerRunner)|runtimeowner\.Runner|\bNewRunner\(|\.NewRuntime\((ctx|context\.Background\(\)|context\.TODO\(\))\)' --glob '*.go'
```

Expected matches should be zero, except for migration documentation or unrelated uses of the word `bindings` in parser or REPL code.

## 13. Commit in reviewable phases

Small commits make xgoja integrations easier to review. A useful sequence is:

1. module loader/provider registration;
2. runtime config sections and capabilities;
3. command provider;
4. generated smoke example;
5. documentation and ticket diary.

Each commit should have its own validation command in the commit message, PR description, diary, or changelog.

## Final checklist

```text
[ ] go-go-goja dependency updated
[ ] provider package added
[ ] modules expose Loader()
[ ] async modules use RuntimeServices helpers
[ ] runtime config sections use DecodeSectionInto
[ ] runtime resources register closers
[ ] command providers return Glazed commands
[ ] generated xgoja example added
[ ] make smoke builds and runs generated binary
[ ] hardware/live demos separated from CI-safe smoke
[ ] old runtime API search is clean
[ ] focused Go tests pass
[ ] README/example docs updated
[ ] commits are scoped and reviewable
```

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `xgoja doctor` cannot import the package | The `packages[].import` path or local `replace` path is wrong. | Check the generated example's relative path and run `go list` from the build workspace when needed. |
| Module appears in provider tests but not in generated binary | The module descriptor is not included in `Register`, or the module set omits it. | Check both provider registration and `modules`. |
| Command provider builds but command does not appear | The command provider is not listed in `commandProviders`, or it is mounted elsewhere. | Run `xgoja list-command-providers` or inspect generated help output. |
| Async Promise never settles | Background goroutine touched JS directly or posted with the wrong context. | Use `RuntimeServices.PostWithCustomContext` or `PostWithLifetimeContext` to settle on owner. |
| Hardware/live smoke is flaky in CI | The smoke depends on external state. | Keep `make smoke` deterministic; move live behavior to a separate target. |
| `GOWORK=off` fails but workspace tests pass | The downstream repo depends on a not-yet-published go-go-goja API. | Publish/tag go-go-goja or add an intentional temporary replace during local testing. |

## See also

- `xgoja help tutorial-using-xgoja-yaml`
- `xgoja help tutorial-providing-package-and-modules`
- `xgoja help tutorial-providing-commands`
- `xgoja help xgoja-v2-reference`
- `xgoja help migrating-runtime-context-api`
