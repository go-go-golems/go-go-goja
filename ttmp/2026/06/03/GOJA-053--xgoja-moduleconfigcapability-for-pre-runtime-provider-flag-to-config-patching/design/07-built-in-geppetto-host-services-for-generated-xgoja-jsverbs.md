---
Title: Built-in Geppetto Host Services for Generated xgoja JavaScript Verbs
Ticket: GOJA-053
Status: active
Topics:
    - xgoja
    - geppetto
    - javascript
    - jsverbs
    - runtime
    - design
DocType: design
Intent: implementation-guide
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../geppetto/pkg/inference/middlewarecfg/registry.go
      Note: Middleware definition registry pattern available for host-provided middleware definitions.
    - Path: ../../../../../../../../geppetto/pkg/inference/tools/registry.go
      Note: Go ToolRegistry contract used by Geppetto agent().goTool and tool-loop execution.
    - Path: ../../../../../../../../geppetto/pkg/js/modules/geppetto/api_agent.go
      Note: Geppetto JavaScript agent builder APIs for inference, Go-backed middleware, Go tools, events, and persistence.
    - Path: ../../../../../../../../geppetto/pkg/js/modules/geppetto/api_event_emitters.go
      Note: Current EventEmitter-backed event sink path and runtime-manager dependency.
    - Path: ../../../../../../../../geppetto/pkg/js/modules/geppetto/provider/provider.go
      Note: Current Geppetto xgoja provider config, public Glazed flags, internal xgoja config mapping, and no-host default module setup.
    - Path: ../../../../../../../../geppetto/pkg/js/modules/geppetto/provider/sqlite_turn_store.go
      Note: Provider-local SQLite TurnStore used by generated xgoja binaries for turns-dsn and turns-db.
    - Path: ../../../../../../../geppetto/pkg/js/modules/geppetto/api_agent.go
      Note: JavaScript APIs for Go tools
    - Path: ../../../../../../../geppetto/pkg/js/modules/geppetto/provider/provider.go
      Note: Current Geppetto provider config
    - Path: ../../../../../../../geppetto/pkg/js/modules/geppetto/provider/sqlite_turn_store.go
      Note: Existing provider-local turn-store implementation used as first built-in host service
    - Path: pkg/xgoja/app/factory.go
      Note: xgoja runtime construction phase where host-service contributions should be collected before module setup
    - Path: pkg/xgoja/app/root.go
      Note: jsverbs command path that passes parsed Glazed values into runtime creation
    - Path: ttmp/2026/06/pkg/xgoja/app/factory.go
      Note: xgoja runtime construction path that maps parsed Glazed values into module setup config before module registration.
    - Path: ttmp/2026/06/pkg/xgoja/app/root.go
      Note: jsverbs command construction path that appends provider Glazed sections and passes parsed values into NewRuntimeFromSections.
ExternalSources: []
Summary: 'Intern-facing design and implementation guide for making generated xgoja JavaScript verbs run Pinocchio-style Geppetto scripts with built-in host services: tools, middleware, event sinks, profile selection, and durable turn storage.'
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Built-in Geppetto Host Services for Generated xgoja JavaScript Verbs

## 1. What this guide is for

Generated xgoja binaries can now run a Geppetto JavaScript verb that loads a Pinocchio profile registry, selects a profile, opens a SQLite turn store, performs an inference run, and reads the stored final turn back from `gp.turnStores.default()`. That is the first important milestone. It proves that the command-line values parsed by Glazed can reach provider module setup before `require("geppetto")` is installed.

The next milestone is broader. Pinocchio's `js` command is not only a profile loader and turn-store opener. It is also a host. It installs Go tools, Go middleware factories, event sinks, profile/runtime defaults, and other Go-owned services that JavaScript can use through the Geppetto API. If we want Pinocchio JavaScript scripts to run as generated xgoja verbs, generated xgoja must learn how to provide those host services without turning xgoja into a second Pinocchio configuration framework.

This document is an implementation guide for an intern. It explains the current architecture, identifies the missing pieces, and proposes a small host-service contribution model that lets generated xgoja binaries install built-in and custom Geppetto services. The goal is not to copy Pinocchio wholesale. The goal is to make the generated verb path capable enough that a Pinocchio script can become an xgoja verb with predictable behavior and explicit configuration.

## 2. The current path: command values become module setup config

The important sequence begins before JavaScript executes. xgoja builds Cobra/Glazed commands for each configured jsverb. When the user invokes one of those commands, Glazed parses command flags, config files, environment variables, defaults, and positional arguments into `values.Values`. xgoja then builds the Goja runtime and gives the selected provider modules a chance to map those parsed values into internal module config.

The current jsverbs command path is visible in `go-go-goja/pkg/xgoja/app/root.go`. The builder selects the jsverbs runtime profile, collects provider Glazed sections, builds each verb command, and invokes the verb inside a runtime created from the parsed values:

```text
root.go:233-276
buildVerbCommands
  -> factory.sectionsForRuntimeProfile("jsverbs", profile)
  -> registry.CommandForVerbWithInvoker(...)
  -> factory.NewRuntimeFromSections(ctx, profile, parsedValues, require.WithLoader(...))
  -> initRuntimeFromSections(...)
  -> registry.InvokeInRuntime(...)
```

The runtime factory then resolves selected modules and asks each provider whether it owns an internal xgoja config section. If it does, the provider receives the static `xgoja.yaml` config and the parsed Glazed values and returns a patch. The final merged config becomes the `json.RawMessage` passed to `Module.NewModuleFactory`.

```text
factory.go:73-155
NewRuntimeFromSections
  -> configForModuleInstance
    -> XGojaConfigSection
    -> ParseXGojaConfigMap(static config)
    -> XGojaConfigFromGlazed(parsed command values)
    -> MergeSectionValues
    -> SectionValuesToRawJSON
  -> providerRuntimeModuleRegistrar.RegisterRuntimeModule
  -> Module.NewModuleFactory(ModuleSetupContext{Config: finalConfig})
```

This matters because Geppetto must know about profile registries and turn stores before its CommonJS loader is created. The JS module object returned by `require("geppetto")` is shaped by `geppettomodule.Options`. If options are incomplete at module setup time, later JavaScript code can see missing stores, missing registries, missing tools, or missing event sinks.

## 3. What the Geppetto provider already does

The current Geppetto xgoja provider has four explicit internal config fields:

```go
// geppetto/pkg/js/modules/geppetto/provider/provider.go:22-27
type Config struct {
    DefaultProfileRegistries []string `json:"defaultProfileRegistries,omitempty"`
    DefaultProfile           string   `json:"defaultProfile,omitempty"`
    TurnsDSN                 string   `json:"turnsDSN,omitempty"`
    TurnsDB                  string   `json:"turnsDB,omitempty"`
}
```

The provider exposes public Glazed flags with Pinocchio-style names and maps them into those internal fields:

| Public flag | Internal xgoja config field | Purpose |
|---|---|---|
| `--profile-registries` | `defaultProfileRegistries` | Load one or more engine profile registry sources. |
| `--profile` | `defaultProfile` | Set the default profile for `gp.inferenceProfiles.resolve()`. |
| `--turns-dsn` | `turnsDSN` | Open a SQLite turn store by DSN. |
| `--turns-db` | `turnsDB` | Open a SQLite turn store from a file path. |

The mapping is provider-owned. The public names are friendly to users who know Pinocchio. The internal names remain precise and avoid reintroducing a broad nested `turns` object or an `enableStorage` gate. The relevant implementation is in `provider.go:177-240`.

The provider can also work without a host-specific `GeppettoOptions` implementation. During module setup it starts with empty `geppettomodule.Options`, optionally lets a host override or extend those options, and then applies profile and turn-store config:

```go
// provider.go:63-76
opts := geppettomodule.Options{}
if host, ok := ctx.Host.(HostServices); ok && host != nil {
    opts, err = host.GeppettoOptions(ctx.Context, cfg)
    ...
}
applyConfigRegistryOptions(ctx.Context, cfg, &opts)
applyConfigTurnStoreOptions(cfg, &opts)
return geppettomodule.NewLoader(opts), nil
```

That no-host behavior is what allowed the generated xgoja jsverb smoke test to run without Pinocchio. The generated binary imported the Geppetto provider, selected the Geppetto module, passed profile and turn-store flags, and got a usable `require("geppetto")` module.

## 4. What is still missing

The current generated xgoja path has profile selection and durable turn storage. It does not yet have a general way for generated binaries to install built-in or custom Geppetto host services.

Geppetto already has JavaScript APIs for those services:

| JavaScript API | Current Go option or runtime dependency | Evidence |
|---|---|---|
| `agent().goTool("name")` | `geppettomodule.Options.GoToolRegistry` | `api_agent.go:148-157` records selected tool names. |
| `agent().goMiddleware("name", opts)` | `geppettomodule.Options.GoMiddlewareFactories` | `api_agent.go:121-135` resolves named Go middleware factories. |
| `agent().events(sinkOrEmitter)` | `Options.DefaultEventSinks` or EventEmitter runtime manager | `api_agent.go:165-178` accepts event sinks or EventEmitter values. |
| `gp.turnStores.default()` | `Options.DefaultTurnStore` and `Options.TurnStores` | The provider now sets these from `turnsDSN` / `turnsDB`. |
| `gp.inferenceProfiles.resolve()` | `Options.EngineProfileRegistry` and default resolve input | The provider now sets these from `profile-registries` / `profile`. |

The missing part is not the JavaScript API. The missing part is the generated Go host that supplies the Go objects behind those APIs.

Pinocchio currently does that in its own `pinocchio js` command. It constructs a tool registry, resolves profile runtime state, opens a turn store, builds middleware factories, and creates a Geppetto runtime with all of those options. A generated xgoja binary should not import Pinocchio by default, but it should have an equivalent extension point: a provider package should be able to contribute Geppetto host services when selected in `xgoja.yaml`.

## 5. The design principle

The key design principle is separation of responsibility.

xgoja should orchestrate runtime construction. It should know how to collect selected modules, parse Glazed values, and pass host services into provider module setup. It should not know what a Geppetto tool registry is, how a Pinocchio profile is represented, or how a custom middleware is built.

The Geppetto provider should know how to interpret Geppetto-specific services. It should know that `GoToolRegistry` belongs in `geppettomodule.Options`, that `GoMiddlewareFactories` are keyed by JavaScript-visible names, and that event sinks belong in `DefaultEventSinks`.

Other provider packages should be able to contribute those services. A package that owns a domain-specific tool should not have to patch the Geppetto provider. It should register a capability that says: when the Geppetto module is selected, add this tool, this middleware factory, or this event sink to the Geppetto options.

This gives us three layers:

```text
Generated xgoja app
  - selects modules
  - parses Glazed values
  - aggregates host-service contributions
  - passes HostServices to provider ModuleSetupContext

Geppetto provider
  - maps profile and turn-store flags into internal config
  - reads Geppetto host-service contributions
  - creates geppettomodule.Options
  - returns require.ModuleLoader

Contributing packages
  - register tools, middleware factories, event sinks, or profile defaults
  - may expose their own Glazed sections if needed
  - do not modify xgoja core
```

## 6. Proposed API: host service contributions

The smallest general mechanism is a typed service bag plus a contribution capability. xgoja already passes `providerapi.HostServices` into module setup. Today that interface only exposes `AssetResolver()`. We can extend it without importing Geppetto into xgoja by adding a generic lookup method.

```go
// pkg/xgoja/providerapi/module.go

type HostServices interface {
    AssetResolver() AssetResolver
    HostService(key string) (any, bool)
}
```

The app-side `HostServices` implementation would keep the existing assets field and add a map:

```go
// pkg/xgoja/app/assets.go or a new host_services.go

type HostServices struct {
    Assets   *AssetStore
    Services map[string]any
}

func (s HostServices) HostService(key string) (any, bool) {
    if s.Services == nil || strings.TrimSpace(key) == "" {
        return nil, false
    }
    v, ok := s.Services[key]
    return v, ok
}
```

Then introduce a provider capability for contributing services during runtime construction:

```go
// pkg/xgoja/providerapi/capabilities.go

type HostServiceContributionRequest struct {
    SectionRequest
    RuntimeProfile string
    Values         *values.Values
    Modules        []ModuleDescriptor
}

type HostServiceSink interface {
    AddHostService(key string, value any) error
    MergeHostService(key string, merge func(existing any) (any, error)) error
}

type HostServiceContributionCapability interface {
    PackageCapability
    ContributeHostServices(context.Context, HostServiceContributionRequest, HostServiceSink) error
}
```

The contribution capability is intentionally generic. It does not mention Geppetto, Pinocchio, tools, middleware, or event sinks. A package can use it to contribute any host service keyed by a stable string. The Geppetto provider defines the key and the typed payload for Geppetto-specific options.

### The Geppetto-specific payload

In `geppetto/pkg/js/modules/geppetto/provider`, define a host-service payload. This lives in Geppetto, not xgoja, because it imports Geppetto types.

```go
package provider

const HostOptionsServiceKey = "geppetto.provider.host-options.v1"

type HostOptionsContribution struct {
    Tools               tools.ToolRegistry
    MiddlewareFactories map[string]geppettomodule.MiddlewareFactory
    DefaultEventSinks   []events.EventSink
    Configure           func(context.Context, Config, *geppettomodule.Options) error
}
```

The provider reads all contributions with that key, merges them, then applies normal config:

```go
opts := geppettomodule.Options{}
if host, ok := ctx.Host.(HostServices); ok && host != nil {
    opts, err = host.GeppettoOptions(ctx.Context, cfg)
}
applyHostOptionsContributions(ctx.Host, cfg, &opts)
applyConfigRegistryOptions(ctx.Context, cfg, &opts)
applyConfigTurnStoreOptions(cfg, &opts)
```

The important ordering is deliberate:

1. Start from host-provided options if the generated target has a custom host.
2. Apply service contributions from selected packages.
3. Apply profile and turn-store command config last, because command-line values should override defaults.

## 7. Why this is better than making xgoja understand Geppetto

A tempting shortcut is to add Geppetto-specific fields directly to xgoja's app host. That would make the first example easy, but it would create the wrong dependency direction. xgoja would need to import Geppetto types. Then every generated binary would carry Geppetto assumptions even when no Geppetto module is selected.

The host-service contribution design avoids that. xgoja only knows that selected providers may contribute keyed services. The Geppetto provider knows how to interpret the Geppetto key. A custom package that wants to add a Geppetto tool imports Geppetto and contributes a Geppetto payload. This keeps the core runtime generic while still allowing rich host behavior.

### Decision: Use keyed host-service contributions

- **Context:** Generated xgoja needs Go-backed host services, but xgoja core should not import Geppetto or Pinocchio.
- **Options considered:** Add Geppetto fields directly to `app.HostServices`; keep all services inside the Geppetto provider; introduce a generic contribution capability.
- **Decision:** Add a generic host-service contribution capability and a typed Geppetto payload keyed by a stable string.
- **Rationale:** This preserves xgoja's provider-neutral architecture and lets domain packages contribute services without modifying Geppetto or xgoja core for each new tool.
- **Consequences:** The implementation needs a merge API and clear collision rules. Provider authors must learn one additional capability.
- **Status:** proposed.

## 8. The example the intern should build

The example should prove three things in one generated xgoja binary:

1. A custom Go tool is available to a Geppetto agent.
2. A custom Go middleware runs during inference.
3. A custom event sink receives inference lifecycle events.

The example should also keep the already-proven profile and turn-store behavior:

```bash
./dist/geppetto-host-services verbs demo run \
  --profile-registries ~/.config/pinocchio/profiles.yaml \
  --profile gpt-5-nano \
  --turns-db /tmp/geppetto-host-services.db \
  --event-log /tmp/geppetto-host-services-events.jsonl \
  --output json
```

The JavaScript verb should look like this:

```js
function run(sessionId) {
  const gp = require("geppetto");
  const settings = gp.inferenceProfiles.resolve();
  const store = gp.turnStores.default();

  const agent = gp.agent()
    .name("generated-host-services-demo")
    .inference(settings)
    .goMiddleware("addSystemPrompt", { prompt: "Answer briefly." })
    .goTool("wordCount")
    .defaultStore()
    .build();

  const session = agent.session()
    .id(sessionId)
    .defaultStore()
    .metadata("demo", "host-services")
    .build();

  const result = session.next()
    .user("Count the words in: generated xgoja host services")
    .run();

  const latest = store.loadLatest({ sessionId, phase: "final" });

  return {
    text: result.text(),
    stored: latest !== null,
    sessionId,
  };
}

__verb__("run", {
  short: "Run a Geppetto host-services demo",
  fields: {
    sessionId: { argument: true },
  },
});
```

The Go contribution package should register a tool, a middleware factory, and an event sink. The implementation can live in a new example provider package first, before we generalize it for production use.

```go
package geppettoextras

func Register(registry *providerapi.ProviderRegistry) error {
    return registry.Package("geppetto-extras",
        providerapi.WithPackageCapability(hostServicesCapability{}),
        providerapi.WithPackageCapability(eventLogSectionCapability{}),
    )
}
```

The tool can be intentionally simple. It does not need to depend on a model choosing to call it for the smoke test; the test can verify the registry is present, and a later model-facing test can ask the model to call it.

```go
type WordCountInput struct {
    Text string `json:"text"`
}

func wordCount(_ context.Context, in WordCountInput) (map[string]any, error) {
    words := strings.Fields(in.Text)
    return map[string]any{"count": len(words)}, nil
}
```

The middleware should be deterministic. A system-prompt middleware is a good first example because Geppetto already has `middleware.NewSystemPromptMiddleware(prompt)`. A custom middleware can wrap that or implement the `middleware.Middleware` function directly.

```go
func addSystemPromptFactory(options map[string]any) (middleware.Middleware, error) {
    prompt, _ := options["prompt"].(string)
    if strings.TrimSpace(prompt) == "" {
        prompt = "Answer briefly."
    }
    return middleware.NewSystemPromptMiddleware(prompt), nil
}
```

The event sink should write JSON Lines to a path supplied by a Glazed section. That makes the evidence easy to inspect after the run.

```go
type jsonlEventSink struct {
    mu sync.Mutex
    w  *bufio.Writer
    f  *os.File
}

func (s *jsonlEventSink) PublishEvent(ev events.Event) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    return json.NewEncoder(s.w).Encode(encodeMinimalEvent(ev))
}
```

The example should produce three pieces of evidence:

```text
1. The jsverb JSON output says the turn was stored.
2. The SQLite turn table contains one final turn for the session.
3. The JSONL event log contains at least one inference lifecycle event.
```

## 9. How the contribution pipeline should run

The pipeline must run before module setup. That is the same timing requirement as the xgoja config patching pipeline. The Geppetto provider's `NewModuleFactory` can only build correct module options if host-service contributions have already been collected.

The runtime factory should do this once per runtime creation:

```go
func (f *RuntimeFactory) NewRuntimeFromSections(ctx context.Context, profile string, vals *values.Values, opts ...require.Option) (*JSRuntime, error) {
    runtime := f.runtimeSpec.Runtimes[profile]
    descriptors := selectedModuleDescriptors(profile)

    services := f.services
    contributed, err := f.collectHostServices(ctx, profile, vals, descriptors)
    if err != nil { return nil, err }
    services = mergeHostServices(services, contributed)

    for each module instance:
        config := configForModuleInstance(ctx, profile, instance, descriptor, vals)
        modules = append(modules, providerRuntimeModuleRegistrar{
            services: services,
            config: config,
        })

    return engine.NewRuntimeFactoryBuilder(...).WithModules(modules...).Build().NewRuntime(...)
}
```

The collection step should deduplicate package-level capabilities by package/capability ID, as section collection and runtime initialization already do. Host services are package-level contributions, not per-module patches. A selected runtime with two Geppetto modules should not install the same default event sink twice unless the contributing capability explicitly asks for per-module behavior.

## 10. Collision and merge rules

Host services need clear merge rules because multiple packages may contribute to the same Geppetto option.

| Service kind | Merge rule |
|---|---|
| Tool registry | Merge registries by tool name. Duplicate names are an error unless explicitly overridden. |
| Middleware factories | Merge by middleware name. Duplicate names are an error. |
| Default event sinks | Append. Each sink should be closed by the owner that created it. |
| Profile defaults | Command-line `--profile` and `--profile-registries` win over contribution defaults. |
| Turn store | Command-line `--turns-dsn` / `--turns-db` win over contribution defaults. |

The duplicate-name rule should be strict at first. Silent overwrites make generated binaries hard to debug. If users later need override semantics, add explicit configuration such as `allowOverride: true` on the contributing package.

### Decision: Strict duplicate handling for tools and middleware

- **Context:** Multiple selected packages can contribute Go tools and middleware factories.
- **Options considered:** First wins, last wins, or fail on duplicate names.
- **Decision:** Fail on duplicate names by default.
- **Rationale:** Generated binaries should be predictable. A duplicate tool name is more likely to be an accidental collision than an intentional override.
- **Consequences:** Users must rename conflicting tools or opt into future override support.
- **Status:** proposed.

## 11. Testing strategy

The implementation should have tests at three levels.

### Unit tests

Write unit tests for the host-service contribution collector. The tests should create fake capabilities and verify:

- A selected package can contribute a service.
- Duplicate tool names fail.
- Multiple event sinks append.
- Contributions are collected before provider `NewModuleFactory` runs.
- Capabilities are deduplicated by package and capability ID.

### Provider tests

Write Geppetto provider tests that build `geppettomodule.Options` from contributed services and then execute JS against `require("geppetto")`.

The provider test should prove:

- `agent().goTool("wordCount")` finds a host-provided tool registry.
- `agent().goMiddleware("addSystemPrompt")` finds a host-provided middleware factory.
- A default event sink receives events during `session.next().run()`.
- `gp.turnStores.default()` still works when `--turns-db` is supplied.

### Generated binary smoke test

Keep a generated xgoja smoke test as the final proof. It should build an example binary, run a jsverb, and inspect the output files. This test can live as a script first, then become a Go integration test if runtime cost is acceptable.

```bash
xgoja build -f examples/xgoja/10-geppetto-host-services/xgoja.yaml \
  --xgoja-replace "$REPO/go-go-goja" \
  --keep-work

./dist/geppetto-host-services verbs demo run "$SESSION" \
  --profile-registries "$HOME/.config/pinocchio/profiles.yaml" \
  --profile gpt-5-nano \
  --turns-db "$DB" \
  --event-log "$EVENTS" \
  --output json

sqlite3 "$DB" 'select count(*) from geppetto_turns'
test -s "$EVENTS"
```

## 12. Implementation phases

### Phase 1: Add generic host-service storage to xgoja

Add `HostService(key string) (any, bool)` to `providerapi.HostServices`. Update `app.HostServices` to store a service map. Preserve `AssetResolver()` exactly as it works today.

This phase should not import Geppetto. It should only create the generic mechanism.

### Phase 2: Add host-service contribution capability

Add `HostServiceContributionCapability` to `providerapi`. Implement collection in `RuntimeFactory.NewRuntimeFromSections` before provider module registrars are created. The collector should receive the parsed `values.Values`, the runtime profile, and selected module descriptors.

### Phase 3: Add Geppetto host-service payload and merge helpers

In the Geppetto provider package, define `HostOptionsServiceKey` and `HostOptionsContribution`. Add helper constructors so provider authors do not manually build service maps.

```go
provider.NewHostOptionsContribution(
    provider.WithToolRegistry(reg),
    provider.WithMiddlewareFactory("addSystemPrompt", addSystemPromptFactory),
    provider.WithDefaultEventSink(sink),
)
```

### Phase 4: Build the example contributor package

Create an example xgoja provider package that contributes:

- `wordCount` Go tool.
- `addSystemPrompt` Go middleware factory.
- JSONL event sink configured by `--event-log`.

The example should be small and deterministic. Its purpose is to teach the extension point, not to be a full Pinocchio clone.

### Phase 5: Run Pinocchio scripts as generated xgoja verbs

Pick two existing Pinocchio JavaScript scripts:

1. A profile-only script that resolves settings and runs a simple agent.
2. A persistence script that uses `gp.turnStores.default()`.

Port them into `examples/xgoja/10-geppetto-host-services/verbs`. The generated xgoja binary should run them with the same profile registry and profile flags that Pinocchio uses.

### Phase 6: Document the migration rule

Document the rule in user-facing help:

```text
If a Pinocchio JS script only uses require("geppetto"), profile selection, turn storage, tools, middleware, and event sinks, it should be runnable as a generated xgoja jsverb once the corresponding host-service contributor packages are selected in xgoja.yaml.
```

Scripts that depend on Pinocchio-specific globals or modules should either import the Pinocchio provider or be rewritten to use Geppetto APIs.

## 13. Open questions

### Should Geppetto public flags stay unprefixed?

The generated smoke used unprefixed `--profile`, `--profile-registries`, `--turns-db`, and `--turns-dsn`. This is convenient because it matches Pinocchio. It can collide with another selected provider that also exposes `--profile`. If collision becomes common, add a provider option that chooses between unprefixed and prefixed flags.

### Should the Geppetto provider SQLite schema align with Pinocchio's normalized schema?

The provider-local schema stores one YAML payload per final turn in `geppetto_turns`. Pinocchio stores a richer normalized turn/block schema. The provider-local schema is sufficient for generated xgoja and simple readback. If generated binaries need Pinocchio's export and analysis tools, align the schema or use Pinocchio's storage package through an optional contributor.

### Who closes provider-created resources?

The current SQLite store has `Close()`, and JavaScript can call `gp.turnStores.default().close()`. The runtime should eventually close provider-created resources automatically. The clean solution is for host-service contributions to register closers with the engine runtime, or for module setup to receive a closer hook. Do not rely on JavaScript to close production resources.

## 14. Summary for the intern

The central idea is simple: generated xgoja must be able to host Geppetto, not merely import it. Hosting means constructing the Go objects that the JavaScript API expects: tool registries, middleware factories, event sinks, profile registries, and turn stores.

Do not put Geppetto types in xgoja core. Add a generic host-service contribution mechanism to xgoja, then let the Geppetto provider define and consume a Geppetto-specific payload. That preserves xgoja's provider-neutral architecture and gives generated binaries enough power to run Pinocchio-style scripts as jsverbs.

The success condition is a generated binary that can run this command and produce inspectable evidence:

```bash
./dist/geppetto-host-services verbs demo run "$SESSION" \
  --profile-registries ~/.config/pinocchio/profiles.yaml \
  --profile gpt-5-nano \
  --turns-db /tmp/geppetto-host-services.db \
  --event-log /tmp/geppetto-host-services-events.jsonl \
  --output json
```

When that command returns a stored turn, a tool-capable agent, a middleware-modified run, and an event log, the generated xgoja verb path has crossed the threshold from "can import Geppetto" to "can host Geppetto."
