---
Title: "Provider runtime config and host-service contributions"
Slug: provider-runtime-config-and-host-services
Short: "How provider authors expose Glazed flags, map them into xgoja module config, and contribute host services before module setup."
Topics:
- xgoja
- provider-api
- configuration
- glazed
- host-services
Commands:
- xgoja
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

Generated xgoja commands have two configuration phases that provider authors must keep separate.

The first phase is public command parsing. A provider can expose Glazed sections that become command flags, config-file fields, and environment-derived values. These fields are user-facing and should use names that make sense at the generated CLI boundary.

The second phase is provider module setup. A selected module receives `providerapi.ModuleSetupContext.Config` while it creates the CommonJS loader that will later satisfy `require("name")`. These fields are internal setup values. They may have different names and shapes than the public flags.

Use this page when a provider needs parsed command/config/env values before `Module.NewModuleFactory` runs, or when a provider package needs to contribute Go services such as tool registries, middleware factories, stores, event sinks, clients, caches, or other runtime-owned objects.

Generated hosts are driven by a v2-native `app.RuntimePlan`. Provider-facing command code receives the selected module descriptors and a command-scoped `providerapi.SourceRegistry`; it should not inspect generated JSON directly or assume all application sources are visible.

## The runtime construction sequence

For generated commands, xgoja constructs a runtime in this order:

```text
xgoja.yaml module config
  -> provider internal schema.Section
  -> static values.SectionValues

Glazed command/config/env/flag values
  -> provider public schema.Section
  -> values.Values

provider mapping
  -> internal values.SectionValues override

xgoja merge
  -> final internal values.SectionValues
  -> json.RawMessage ModuleSetupContext.Config

host-service contribution
  -> provider-neutral service bag
  -> ModuleSetupContext.Host

module setup
  -> Module.NewModuleFactory(ModuleSetupContext{Config, Host, AddCloser})
  -> require.ModuleLoader
```

The important rule is timing: `NewRuntimeFromSections` receives parsed Glazed values and applies provider mappings before module setup. If a value must affect the shape or behavior of `require("my-module")`, it must be mapped before `NewModuleFactory` returns the loader.

For example, the HTTP provider maps the public `http` section into the `express` module's xgoja config so static `xgoja.yaml` values and command-line overrides share one setup path:

```yaml
runtime:
  modules:
    - provider: go-go-goja-http
      name: express
      config:
        listen: 127.0.0.1:8787
        dev-errors: false
        reject-raw-routes: true
```

This config controls host infrastructure. JavaScript still declares route intent with `.public()`, `.auth(...)`, `.csrf()`, and `.allow(...)`; it should not configure cookies, OIDC clients, SQL stores, or application authorization policy.

## Public Glazed sections

Implement `providerapi.GlazedConfigSectionCapability` when a provider wants to expose user-facing command/config/env values.

```go
type GlazedConfigSectionCapability interface {
    providerapi.PackageCapability
    GlazedConfigSections(providerapi.SectionRequest) ([]schema.Section, error)
}
```

Example:

```go
type capability struct{}

func (capability) CapabilityID() string { return "my-provider-config" }

func (capability) GlazedConfigSections(providerapi.SectionRequest) ([]schema.Section, error) {
    section, err := schema.NewSection("my-provider", "My provider",
        schema.WithFields(
            fields.New("profile", fields.TypeString, fields.WithHelp("Default profile slug")),
            fields.New("database", fields.TypeString, fields.WithHelp("Database path")),
        ),
    )
    if err != nil {
        return nil, err
    }
    return []schema.Section{section}, nil
}
```

These sections are collected for built-in generated commands such as `eval`, `run`, `tui`, and `jsverbs`, and for provider-owned command sets when those commands use the xgoja runtime factory. The parsed result is a `*values.Values` containing one `*values.SectionValues` per section slug.

Public sections are part of the generated command UX. Choose stable, readable names. Do not expose internal setup names just because they happen to exist in your Go struct.

## Internal xgoja config sections

Implement `providerapi.XGojaConfigSectionCapability` when a provider has setup config that should be parsed, type-checked, merged, and passed into `ModuleSetupContext.Config`.

```go
type XGojaConfigSectionCapability interface {
    providerapi.PackageCapability
    XGojaConfigSection(providerapi.SectionRequest, providerapi.ModuleDescriptor) (schema.Section, error)
    XGojaConfigFromGlazed(context.Context, providerapi.XGojaConfigRequest) (*values.SectionValues, error)
}
```

The internal section is used for two inputs:

1. static `xgoja.yaml` module config, and
2. provider-produced overrides derived from parsed public Glazed values.

Example internal section:

```go
func (capability) XGojaConfigSection(
    req providerapi.SectionRequest,
    descriptor providerapi.ModuleDescriptor,
) (schema.Section, error) {
    return schema.NewSection("my-provider-xgoja", "My provider xgoja config",
        schema.WithFields(
            fields.New("defaultProfile", fields.TypeString),
            fields.New("databasePath", fields.TypeString),
        ),
    )
}
```

This section is not exposed as flags by default. It is the provider's setup contract.

## Mapping public values into internal config

`XGojaConfigFromGlazed` maps parsed public values into an internal override for one selected module instance.

```go
func (capability) XGojaConfigFromGlazed(
    ctx context.Context,
    req providerapi.XGojaConfigRequest,
) (*values.SectionValues, error) {
    out, err := values.NewSectionValues(req.ConfigSection)
    if err != nil {
        return nil, err
    }
    if req.GlazedValues == nil {
        return out, nil
    }

    field, ok := req.GlazedValues.GetField("my-provider", "profile")
    if !ok {
        return out, nil
    }
    definition, ok := req.ConfigSection.GetDefinitions().Get("defaultProfile")
    if !ok {
        return nil, fmt.Errorf("internal config field defaultProfile not found")
    }
    if err := out.Fields.UpdateWithLog("defaultProfile", definition, field.Value, field.Log...); err != nil {
        return nil, err
    }
    return out, nil
}
```

Use `UpdateWithLog` when copying a public value into an internal field. That preserves Glazed provenance: defaults, config files, environment variables, positional arguments, and Cobra flags remain visible in `FieldValue.Log`.

The request includes a `ModuleDescriptor`. Use it when the same provider package can be selected more than once under different aliases. Avoid package-global config patches unless the setting is truly global.

## Runtime factory entry point

Generated command paths should create runtimes with parsed values:

```go
runtime, err := factory.NewRuntimeFromSections(ctx, parsedValues, requireOptions...)
```

`NewRuntime(ctx, ...)` still exists for static-config-only runtime creation. It delegates to `NewRuntimeFromSections(ctx, nil, ...)`.

## xgoja framework debug fields

Generated runtime-backed commands also include a small framework-owned Glazed section named `xgoja`. This section is reserved for xgoja runtime controls rather than provider-specific configuration. Provider authors should not define their own public section with the `xgoja` slug.

The current field is:

```text
--debug-panic-stack
```

When this boolean field is enabled, xgoja forwards the parsed value into:

```go
engine.WithRecoveredPanicStack(true)
```

That makes recovered runtimeowner panic errors include a Go `runtime/debug.Stack()` dump. Use it when diagnosing provider panics during generated command execution, for example:

```bash
my-generated-xgoja verbs demo run --debug-panic-stack
my-generated-xgoja eval 'require("my-module").boom()' --debug-panic-stack
```

Leave the field disabled for normal users. Stack traces are noisy and include local filesystem paths, so xgoja keeps concise recovered panic errors as the default.

## Host-service contributions

Config fields are enough for strings, numbers, booleans, paths, and other JSON-compatible setup values. They are not enough for Go objects such as registries, clients, stores, middleware factories, or event sinks.

Use `providerapi.HostServiceContributionCapability` when a selected provider package needs to contribute Go services before module setup.

```go
type HostServiceContributionCapability interface {
    providerapi.PackageCapability
    ContributeHostServices(
        context.Context,
        providerapi.HostServiceContributionRequest,
        providerapi.HostServiceSink,
    ) error
}
```

A contribution capability receives:

- the generated module set,
- parsed Glazed values,
- all selected module descriptors, and
- a sink for adding host services and closers.

Example:

```go
const MyServiceKey = "my-provider.host-options.v1"

type MyHostOptions struct {
    Client *Client
    Sinks  []EventSink
}

func (capability) ContributeHostServices(
    ctx context.Context,
    req providerapi.HostServiceContributionRequest,
    sink providerapi.HostServiceSink,
) error {
    client, err := NewClientFromValues(req.Values)
    if err != nil {
        return err
    }
    if err := sink.AddCloser(func(context.Context) error { return client.Close() }); err != nil {
        _ = client.Close()
        return err
    }
    return sink.AddHostService(MyServiceKey, MyHostOptions{Client: client})
}
```

xgoja treats service values as opaque. Provider packages own their keys and payload types. Prefer stable, versioned keys such as `my-provider.host-options.v1`.

## Host-supplied services from generated package hosts

Generated runtime packages can also receive services from the embedding Go application. The generated package's `Options` type includes:

```go
ConfigureServices func(*app.HostServices)
```

The callback runs before `NewRuntimeFactory` captures the provider-neutral service bag. Use `SetHostService` for singleton host-owned services and `AddHostService` for intentionally multi-valued keys:

```go
bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{
    ConfigureServices: func(services *app.HostServices) {
        _ = services.SetHostService(MyServiceKey, MyHostOptions{Client: client})
    },
})
```

This path is useful when a long-lived Go program owns infrastructure that should be visible during module setup: HTTP hosts, database handles, registries, middleware chains, event sinks, or application-specific clients. Validation errors from provider consumption surface when the bundle creates a runtime, because providers read the service bag inside `Module.NewModuleFactory`.

The built-in HTTP provider uses this pattern for Go-owned servers. Inject `httpprovider.ExternalHostService` under `httpprovider.HostServiceKey` when JavaScript should register Express routes into an existing `*gojahttp.Host`:

```go
jsHost := gojahttp.NewHost(gojahttp.HostOptions{Dev: true})
bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{
    ConfigureServices: func(services *app.HostServices) {
        _ = services.SetHostService(httpprovider.HostServiceKey, httpprovider.ExternalHostService{
            Host:       jsHost,
            OwnsListen: false,
        })
    },
})
```

With `OwnsListen: false`, Express route/static registration populates `jsHost` but the HTTP provider does not bind a TCP listener. The outer Go application remains responsible for mounting `jsHost` on its mux and starting `net/http.Server`.

### HTTP host sharing and mountable handlers

The HTTP provider owns the `gojahttp.Host` used by the `express` runtime module unless a host application injects an external host service. Other provider modules should not depend on the Express provider directly when they need to expose Go HTTP behavior. Instead, they should expose JavaScript objects that carry the shared `gojahttp` mountable-handler ABI.

A producer module attaches a Go handler to a JavaScript object:

```go
obj := vm.NewObject()
if err := gojahttp.AttachHTTPHandler(vm, obj, handler); err != nil {
    return nil, err
}
```

JavaScript can then compose that object with Express:

```js
const express = require("express")
const app = express.app()

app.mount("/ws", wsServer)
```

This pattern keeps ownership clear. The producer module owns the Go `http.Handler`; the HTTP provider owns or receives the `gojahttp.Host`; JavaScript decides where the handler is mounted. Use host services when a Go embedding application needs to provide the host itself. Use the mountable-handler ABI when a JavaScript application needs to connect a Go-backed handler object to that host.

`app.mount()` deliberately performs prefix matching only. Do not make provider modules depend on JavaScript route parameters for Go-owned transports. If the mounted handler needs parameters or wildcards, route inside the Go handler by inspecting `r.URL.Path` or by using a Go router such as `http.ServeMux`:

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /ws/rooms/{roomID}", func(w http.ResponseWriter, r *http.Request) {
    roomID := r.PathValue("roomID")
    serveRoomSocket(w, r, roomID)
})
```

This is especially important for WebSocket servers, where the Go handler normally owns upgrade behavior, origin checks, subprotocol negotiation, and transport-specific routing.

## Command-scoped source registry

Provider command sets receive `providerapi.CommandSetContext`. The important v2 fields are:

```go
type CommandSetContext struct {
    Context         context.Context
    PackageID       string
    Name            string
    Mount           string
    Config          json.RawMessage
    Host            HostServices
    Providers       *ProviderRegistry
    RuntimeFactory  RuntimeFactory
    SelectedModules []ModuleDescriptor
    Sources         SourceRegistry
}
```

`Host` is the same generated-host service bag that module setup receives through `ModuleSetupContext.Host`. Provider-owned commands should use it when they need generated-host resources such as embedded assets, host-injected clients, shared HTTP hosts, or other opaque provider-defined services. This keeps provider commands from parsing embedded runtime-plan JSON directly and gives command sets the same asset/service view as runtime modules.

This matters for plain generated binaries, `runtime-package` artifacts, and custom `template` artifacts. In all three cases, the generated host usually owns the embedded asset filesystem and any `ConfigureServices` injections. Passing that host bag into command-set construction means a provider command produced by a template-backed host can resolve the same assets and services as a module loaded by that host. Without it, templates that emit a provider-owned command have to invent a side channel, duplicate asset lookup code, or hard-code generated runtime-plan internals.

For example, a command provider can read an embedded asset during command-set construction:

```go
func newCommandSet(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
    resolver := ctx.Host.AssetResolver()
    assetFS, root, ok := resolver.ResolveAsset("fixtures")
    if !ok {
        return nil, fmt.Errorf("missing fixtures asset source")
    }
    data, err := fs.ReadFile(assetFS, path.Join(root, "message.txt"))
    if err != nil {
        return nil, err
    }
    return commandsUsingEmbeddedMessage(string(data)), nil
}
```

The `examples/xgoja/05-command-provider` smoke test includes a concrete `fixture asset` command that reads an embedded asset this way.

### Lazy generated-host auth services

Some host services should be discoverable during command construction but built only when a command actually runs. Generated-host Express auth uses that pattern. A runtime-package host injects a lightweight factory with `Options.ConfigureServices`:

```go
bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{
    ConfigureServices: func(services *app.HostServices) {
        _ = services.SetHostService(
            hostauth.ServiceFactoryKey,
            hostauth.NewServiceFactory(hostauth.BuilderOptions{Config: authConfig}),
        )
    },
})
```

The HTTP `serve` command discovers `hostauth.ServiceFactoryKey` from `CommandSetContext.Host` while constructing provider commands, but it calls the factory later, after Glazed values are parsed. The resulting `hostauth.Services` bundle supplies `gojahttp.AuthOptions`, stores, session manager, and closers. The provider passes an auth-enabled `gojahttp.Host` through the existing `go-go-goja-http.host` service and also exposes the concrete bundle as `hostauth.ServicesKey` for future modules/tools.

See `examples/xgoja/21-generated-host-auth` for a runnable runtime-package host that uses memory stores by default and SQLite stores when Glazed-backed `--auth-default-store-*` command settings are provided.

`Sources` is scoped to the command's `commands[].sources` list. A provider command should use it as the source of truth:

```go
func newCommandSet(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
    jsverbSources := ctx.Sources.JSVerbs()
    registries, err := jsverbSources.ScanAllJSVerbSources()
    if err != nil {
        return nil, err
    }
    // Build commands from the scoped registries only.
}
```

For HTTP `serve`, this means both the initial verb discovery and hot reload watch roots are limited to the source IDs declared on the `serve` command. If a provider needs help or asset sources, use `ctx.Sources.ListSourcesByKind(...)` or `ctx.Sources.SourceByID(...)` rather than scanning global runtime metadata.

## Reading host services during module setup

Provider modules read contributed services from `ModuleSetupContext.Host` by asserting `providerapi.HostServiceLookup`.

```go
func applyHostOptions(host providerapi.HostServices, opts *Options) error {
    lookup, ok := host.(providerapi.HostServiceLookup)
    if !ok || lookup == nil {
        return nil
    }
    for _, raw := range lookup.HostServiceValues(MyServiceKey) {
        contribution, ok := raw.(MyHostOptions)
        if !ok {
            return fmt.Errorf("expected MyHostOptions, got %T", raw)
        }
        opts.Client = contribution.Client
        opts.Sinks = append(opts.Sinks, contribution.Sinks...)
    }
    return nil
}
```

Use `HostServiceValues` for intentionally multi-valued keys. Use `HostService` only when a key is expected to have exactly one semantic value.

## Resource cleanup

There are two cleanup paths.

A host-service contributor that creates a runtime-owned resource should call `HostServiceSink.AddCloser` while contributing services. xgoja registers those closers with the engine runtime before JavaScript executes. Closers run when the runtime closes, and xgoja also attempts to close unregistered contribution resources if runtime construction fails before they are attached to the engine runtime.

A provider module that creates a resource during `NewModuleFactory` should use `ModuleSetupContext.AddCloser`:

```go
NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
    store, err := OpenStore(...)
    if err != nil {
        return nil, err
    }
    if ctx.AddCloser != nil {
        if err := ctx.AddCloser(func(context.Context) error { return store.Close() }); err != nil {
            _ = store.Close()
            return nil, err
        }
    }
    return NewLoader(Options{Store: store}), nil
}
```

Always close the resource immediately if closer registration fails. Do not leak partially constructed stores, files, sockets, or clients.

## Duplicate policy

Named services visible to JavaScript should reject duplicates by default. A duplicate Go tool name or middleware factory name usually indicates an ambiguous generated binary. Event sinks, log sinks, metrics sinks, and similar append-only destinations can append.

The Geppetto provider follows this policy:

- contributed Go tool names are strict;
- contributed Go middleware factory names are strict;
- default event sinks append.

## Geppetto example

The Geppetto provider demonstrates the full pattern.

Public flags:

```text
--profile-registries
--profile
--turns-dsn
--turns-db
```

Internal setup fields:

```text
defaultProfileRegistries
defaultProfile
turnsDSN
turnsDB
```

Host-service key:

```go
const HostOptionsServiceKey = "geppetto.provider.host-options.v1"
```

Host-service payload:

```go
type HostOptionsContribution struct {
    ToolRegistry        tools.ToolRegistry
    MiddlewareFactories map[string]geppettomodule.MiddlewareFactory
    DefaultEventSinks   []events.EventSink
    Configure           func(context.Context, Config, *geppettomodule.Options) error
}
```

See `examples/xgoja/12-geppetto-host-services` for a generated binary that uses profile flags, a SQLite turn store, a contributed `wordCount` Go tool, a contributed `addSystemPrompt` middleware factory, and a contributed JSONL event sink.

## Checklist

When adding provider runtime config or host services:

1. Define public Glazed fields for user-facing command/config/env input.
2. Define a separate internal xgoja config section for module setup.
3. Map public values into internal `SectionValues` with `UpdateWithLog`.
4. Keep mappings scoped to the selected module descriptor.
5. Use `NewRuntimeFromSections` in commands that have parsed Glazed values.
6. Use host-service contributions for Go objects that cannot be represented as JSON config.
7. Register closers for provider-created resources.
8. Reject duplicate JS-visible names unless the provider explicitly documents override semantics.
