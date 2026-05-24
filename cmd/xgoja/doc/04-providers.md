---
Title: "xgoja provider authoring"
Slug: providers
Short: "How to expose Goja bindings as xgoja provider packages."
Topics:
- xgoja
- goja
- providers
- modules
- js-bindings
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

Provider packages are the Go side of xgoja composition. A provider package is a normal Go package that exposes a registration function. Generated xgoja binaries import that package, call the registration function, and then make the provider's modules selectable from `xgoja.yaml` runtime profiles.

Use a provider when you have Go functionality that should be available to JavaScript through `require(...)`. Keep the provider small and explicit: registering a provider only makes modules available to the generated binary; each command still receives only the modules selected by its runtime profile.

## Provider package contract

A provider package exports a function that accepts `*providerapi.Registry` and returns an error. The default function name is `Register`, but `packages[].register` can name a different function.

```go
package myprovider

import (
    "github.com/dop251/goja"
    "github.com/dop251/goja_nodejs/require"
    "github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func Register(registry *providerapi.Registry) error {
    return registry.Package("my-provider",
        providerapi.Module{
            Name:        "hello",
            DefaultAs:   "hello",
            Description: "Small example module",
            New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
                return func(vm *goja.Runtime, module *goja.Object) {
                    exports := module.Get("exports").(*goja.Object)
                    _ = exports.Set("greet", func(name string) string {
                        return "hello " + name
                    })
                }, nil
            },
        },
    )
}
```

The first argument to `registry.Package` is the provider package ID. That ID is part of the public buildspec surface. It must match `packages[].id` in `xgoja.yaml`.

```yaml
packages:
  - id: my-provider
    import: github.com/example/project/pkg/myprovider
```

## Select modules with runtime profiles

A provider can register many modules. A runtime profile chooses which modules are visible to JavaScript for one command invocation.

```yaml
runtimes:
  safe:
    modules:
      - package: my-provider
        name: hello
        as: hello

commands:
  eval:
    enabled: true
    runtime: safe
```

JavaScript sees the alias from `as`:

```js
const hello = require("hello")
console.log(hello.greet("intern"))
```

If a provider registers a module but no runtime profile selects it, JavaScript cannot require it. This is intentional. Build-time package selection and runtime module selection are separate boundaries.

## Adapt an existing NativeModule

Many first-party go-go-goja modules already implement `modules.NativeModule` with a `Loader(*goja.Runtime, *goja.Object)` method. A provider can wrap that loader directly.

```go
func nativeModuleEntry(mod modules.NativeModule) providerapi.Module {
    return providerapi.Module{
        Name:        mod.Name(),
        DefaultAs:   mod.Name(),
        Description: mod.Doc(),
        New: func(providerapi.ModuleContext) (require.ModuleLoader, error) {
            return mod.Loader, nil
        },
    }
}
```

Import module packages for their `init()` registration before looking them up from the existing registry.

```go
import (
    "github.com/go-go-golems/go-go-goja/modules"
    _ "github.com/go-go-golems/go-go-goja/modules/path"
    _ "github.com/go-go-golems/go-go-goja/modules/yaml"
)

func Register(registry *providerapi.Registry) error {
    pathModule := modules.GetModule("path")
    yamlModule := modules.GetModule("yaml")
    return registry.Package("go-go-goja-core",
        nativeModuleEntry(pathModule),
        nativeModuleEntry(yamlModule),
    )
}
```

Prefer explicit lists over "register everything" helpers. Explicit lists make the provider reviewable and make dangerous host capabilities harder to expose by accident.

## Use config for module options

Each `runtimes.<profile>.modules[]` entry can include a `config` map. xgoja serializes this config and passes it to the provider module factory as `providerapi.ModuleContext.Config`.

```yaml
runtimes:
  host:
    modules:
      - package: go-go-goja-host
        name: exec
        as: exec
        config:
          allow: true
          allowedCommands:
            - echo
```

Decode config in the module factory and fail before registering the loader if the config is invalid or unsafe.

```go
type ExecConfig struct {
    Allow           bool     `json:"allow"`
    AllowedCommands []string `json:"allowedCommands,omitempty"`
}

New: func(ctx providerapi.ModuleContext) (require.ModuleLoader, error) {
    var cfg ExecConfig
    if err := json.Unmarshal(ctx.Config, &cfg); err != nil {
        return nil, fmt.Errorf("exec config: %w", err)
    }
    if !cfg.Allow {
        return nil, fmt.Errorf("exec module requires config.allow=true")
    }
    return newExecLoader(cfg), nil
},
```

Use config for static, serializable settings: allow flags, paths, module names, DSNs, API base URLs, and feature toggles. Use `ModuleContext.Host` only for live application services in target-mode integrations.

## Register provider-shipped JavaScript verbs

A provider can ship JavaScript verb files next to Go modules. Embed them and register a `providerapi.VerbSource`.

```go
//go:embed verbs/*.js
var verbsFS embed.FS

func Register(registry *providerapi.Registry) error {
    return registry.Package("my-provider",
        providerapi.Module{...},
        providerapi.VerbSource{
            Name:        "verbs",
            Description: "Provider bundled verbs",
            Root:        "verbs",
            FS:          verbsFS,
        },
    )
}
```

A buildspec can mount that source under the generated jsverbs command.

```yaml
commands:
  jsverbs:
    enabled: true
    runtime: main
    name: verbs

jsverbs:
  - id: provider-defaults
    package: my-provider
    source: verbs
```

Provider-shipped verbs are useful when a package has a stable Go module API plus higher-level JavaScript commands that should travel with it.

## Split safe and host-capability providers

Keep dangerous modules separate from safe helpers. A generated binary can compile both providers, but each command should select only the runtime profile it needs.

```yaml
packages:
  - id: go-go-goja-core
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/core
  - id: go-go-goja-host
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/host

runtimes:
  safe:
    modules:
      - package: go-go-goja-core
        name: path
        as: path
  host:
    modules:
      - package: go-go-goja-core
        name: path
        as: path
      - package: go-go-goja-host
        name: fs
        as: fs
        config:
          allow: true

commands:
  eval:
    enabled: true
    runtime: safe
  run:
    enabled: true
    runtime: host
```

In this shape, `eval` cannot `require("fs")`, while `run` can. See `examples/xgoja/multiple-runtimes/` for a runnable example.

## Validate a provider

A provider is not complete until it works in a generated binary. At minimum, add:

1. A provider unit test that calls `Register` and resolves the expected modules.
2. A generated xgoja example or generated-build test that imports the provider package.
3. A `run.js` or `eval` smoke that calls a deterministic API.
4. A negative test for guard behavior when the provider exposes host capabilities.

For local development, use `--xgoja-replace` so the generated module builds against your checkout.

```bash
xgoja doctor -f examples/xgoja/my-provider/xgoja.yaml
xgoja list-modules -f examples/xgoja/my-provider/xgoja.yaml
xgoja build -f examples/xgoja/my-provider/xgoja.yaml --xgoja-replace /path/to/go-go-goja --keep-work
./examples/xgoja/my-provider/dist/my-provider run examples/xgoja/my-provider/scripts/smoke.js
```

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `runtime main references unknown provider module pkg.name` | `packages[].id` does not match the ID used in `registry.Package(...)`, or the provider did not register the module name. | Make `packages[].id` exactly match the provider package ID and verify `Register` adds the module. |
| Generated build cannot import the provider | The generated module cannot resolve the provider import path. | Add package `version`, package `replace`, or pass `--xgoja-replace` for local go-go-goja changes. |
| `require("name")` fails at runtime | The module is compiled in but not selected by the command's runtime profile, or the `as` alias differs. | Check `xgoja list-modules -f xgoja.yaml` and the command's `runtime` field. |
| Host module returns a config error | The provider intentionally requires an allow/config switch. | Add the documented `config` block for that module. |
| Async module panics about runtime owner bindings | The loader assumes xgoja/engine runtime ownership but was used in a plain Goja runtime. | Test through generated xgoja or `engine.Runtime`, not raw `goja.New()`. |

## See also

- `xgoja help overview`
- `xgoja help buildspec`
- `xgoja help tutorial`
- `examples/xgoja/core-provider/`
- `examples/xgoja/host-provider/`
- `examples/xgoja/multiple-runtimes/`
