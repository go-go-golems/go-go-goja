---
Title: Glazed Auth Settings Bubbling Implementation Guide
Ticket: XGOJA-PR74-INDEPENDENT-CODE-REVIEW
Status: active
Topics:
    - review
    - goja
    - xgoja
    - auth
    - security
    - testing
    - glazed
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/geppetto/cmd/examples/runner-glazed-registry-flags/main.go
      Note: |-
        Reference pattern for decoding Glazed profile values at command execution time.
        Reference decode parsed profile settings at command execution
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/geppetto/pkg/sections/profile_sections.go
      Note: |-
        Reference pattern for exposing profile/profile-registry settings as Glazed sections.
        Reference profile settings section pattern
    - Path: pkg/xgoja/app/middlewares.go
      Note: |-
        Generated roots already support environment/config/default input through Glazed middleware; auth should use this path only.
        Glazed env/config/default middleware path replacing os.Getenv
    - Path: pkg/xgoja/hostauth/builder.go
      Note: |-
        BuildHostAuthServices currently defaults to os.LookupEnv; it should consume parsed values and never inspect process environment directly.
        Service build entry point that should decode parsed Glazed values
    - Path: pkg/xgoja/hostauth/config.go
      Note: |-
        Host auth config currently contains DSNEnv and nested store/session config shapes that should be flattened into Glazed-facing fields.
        Config shape that should drop DSNEnv and expose Glazed auth fields
    - Path: pkg/xgoja/hostauth/resolve.go
      Note: |-
        Current ResolveOptions.LookupEnv and dsn-env handling should be removed in favor of already-parsed Glazed values.
        LookupEnv/dsn-env resolver to remove
    - Path: pkg/xgoja/providers/http/http.go
      Note: Existing HTTP Glazed/XGoja config capability is the model for exposing --http-* settings and mapping parsed values.
    - Path: pkg/xgoja/providers/http/serve.go
      Note: |-
        HTTP serve provider wraps jsverbs and is the correct place to attach auth Glazed sections to per-site serve commands.
        Serve jsverb command wrapper where auth section should be attached
ExternalSources: []
Summary: Design and implementation guide for moving xgoja host auth settings to Glazed flags/config/env parsing, removing direct os.Getenv/LookupEnv/DSNEnv handling, and bubbling auth settings onto the HTTP serve jsverb commands.
LastUpdated: 2026-06-15T17:25:00-04:00
WhatFor: Give an intern enough architecture context, file references, API contracts, and implementation steps to refactor host auth settings into Glazed-managed command sections.
WhenToUse: Use before implementing PR 74 follow-up work that removes direct environment reads and makes generated-host auth configurable through --auth-* flags, config files, and Glazed environment sources.
---


# Glazed Auth Settings Bubbling Implementation Guide

## Executive summary

The auth configuration added in PR 74 should follow the same rule as the rest of xgoja command configuration: **application code should consume already-parsed Glazed values, not call `os.Getenv` directly**. Environment variables may still be a user input source, but only through Glazed middleware such as `sources.FromEnv(...)`; after parsing, code should see ordinary `*values.Values` and decode sections from them.

Today the host auth path has two environment-specific escape hatches:

1. `hostauth.StoreConfig.DSNEnv` and `ResolveOptions.LookupEnv` in `pkg/xgoja/hostauth/resolve.go`.
2. Direct `os.Getenv` calls in generated-host and Keycloak examples to seed auth settings before command parsing.

Those should be removed. The replacement is a Glazed auth section that gets attached to the correct commands, especially the HTTP `serve` provider's per-jsverb commands. A generated binary should expose flags such as:

```text
--auth-mode dev
--auth-session-cookie-allow-insecure-http
--auth-default-store-driver sqlite
--auth-default-store-dsn /tmp/auth.sqlite
--auth-default-store-apply-schema
```

If a generated app has `app.name` or `app.envPrefix` configured, the same values can come from environment variables through existing xgoja middleware, for example:

```text
GENERATED_HOST_AUTH_AUTH_MODE=dev
GENERATED_HOST_AUTH_AUTH_DEFAULT_STORE_DRIVER=sqlite
GENERATED_HOST_AUTH_AUTH_DEFAULT_STORE_DSN=/tmp/auth.sqlite
```

The important distinction is that **Glazed reads those environment variables**, not `hostauth` or an example `main.go`.

## Goals and non-goals

### Goals

- Remove direct environment lookup from host auth config resolution.
- Remove `dsn-env` / `DSNEnv` from host auth config.
- Bubble auth settings to the HTTP `serve` jsverb commands using Glazed sections.
- Preserve lazy construction: databases and auth stores should still open only when the selected `serve <package> <verb>` command executes.
- Keep JavaScript route files focused on route intent (`.public()`, `.auth()`, `.allow()`, `.csrf()`), not infrastructure settings.
- Make the implementation understandable to a new intern by showing the command/value flow end to end.

### Non-goals

- Do not add a JavaScript auth configuration DSL.
- Do not move credentials into `xgoja.yaml` as secrets.
- Do not make `hostauth` inspect process environment directly.
- Do not require every xgoja command to build auth services. Auth services are needed by HTTP serving, not by plain static type generation or module listing.

## Current architecture map

### Main packages

| Area | Files | Role |
| --- | --- | --- |
| Host auth config | `pkg/xgoja/hostauth/config.go`, `resolve.go`, `builder.go`, `stores.go` | Defines auth mode/session/store config, resolves defaults, opens stores, builds `gojahttp.AuthOptions`. |
| HTTP provider public config | `pkg/xgoja/providers/http/http.go` | Exposes `--http-*` settings through `GlazedConfigSections` and maps Glazed values into internal module config. |
| HTTP serve command provider | `pkg/xgoja/providers/http/serve.go` | Builds one command per jsverb and currently appends selected module Glazed sections to those commands. This is where auth flags must appear for `serve` commands. |
| Runtime factory | `pkg/xgoja/app/factory.go` | Builds a runtime from parsed values; runs host-service contributors; maps Glazed values into module config. |
| Command attachment | `pkg/xgoja/app/root.go`, `command_providers.go` | Attaches built-in commands and provider command sets to the generated Cobra root. |
| Input parsing | `pkg/xgoja/app/middlewares.go` | Chooses Glazed middleware chain: Cobra flags, args, env, config, defaults. |
| JS verb command schema | `pkg/jsverbs/command.go`, `pkg/jsverbs/runtime.go` | Builds per-verb command descriptions and passes parsed sections into JavaScript invocation. |

### Current host auth config path

The current config type includes both direct DSNs and env-based DSN indirection:

```go
type StoreConfig struct {
    Driver      string `yaml:"driver" json:"driver"`
    DSN         string `yaml:"dsn" json:"dsn"`
    DSNEnv      string `yaml:"dsn-env" json:"dsn-env"`
    ApplySchema *bool  `yaml:"apply-schema" json:"apply-schema"`
}
```

`ResolveConfig` accepts `ResolveOptions{LookupEnv: ...}` and eventually calls `resolveDSN(...)`, which checks `dsn-env` and calls the lookup function. `Builder.BuildHostAuthServices` defaults `LookupEnv` to `os.LookupEnv`.

That has two problems:

- It creates a second configuration system outside Glazed.
- It makes auth behavior depend on process environment at service-build time rather than on the command's parsed `*values.Values`.

## Existing xgoja Glazed bubbling model

The xgoja runtime already has the pieces needed to do this correctly.

```mermaid
flowchart TD
  Root[Generated Cobra root] --> Middleware[Glazed middleware chain]
  Middleware --> Values[*values.Values]
  Values --> Command[Command Run / provider command invoker]
  Command --> RuntimeFactory[RuntimeFactory.NewRuntimeFromSections]
  Values --> RuntimeFactory
  RuntimeFactory --> ModuleConfig[XGojaConfigFromGlazed overlays module config]
  RuntimeFactory --> RuntimeInit[InitRuntimeFromSections]
  RuntimeFactory --> JS[JavaScript runtime]
```

The middleware chain in `pkg/xgoja/app/middlewares.go` is the only place that should read environment/config/default sources. Its effective precedence is documented in code:

```text
defaults < config < env < args < cobra flags
```

When an app defines `app.name` or `app.envPrefix`, `MiddlewaresFromSpec` includes `sources.FromEnv(prefix, fields.WithSource("env"))`. A field named `value` in section `fixture` with app name `env-fixture` is read from `ENV_FIXTURE_FIXTURE_VALUE`, as shown by `TestGeneratedRootReadsModuleSectionFromDerivedEnvPrefix`.

The HTTP provider is the current example for provider-owned public settings:

```go
func (c *capability) GlazedConfigSections(providerapi.SectionRequest) ([]schema.Section, error) {
    section, err := httpConfigSection(schema.WithPrefix("http-"))
    if err != nil {
        return nil, err
    }
    return []schema.Section{section}, nil
}
```

That produces user-facing flags such as `--http-listen` while internally keeping section slug `http` and field name `listen`.

## Geppetto / Pinocchio reference pattern

The Geppetto/Pinocchio profile flow is the right conceptual model.

### What Geppetto does

Geppetto exposes profile selection as a Glazed section:

```go
const ProfileSettingsSectionSlug = "profile-settings"

type ProfileSettings struct {
    Profile           string   `glazed:"profile"`
    ProfileRegistries []string `glazed:"profile-registries"`
}

func NewProfileSettingsSection(...) (schema.Section, error) {
    return schema.NewSection(
        ProfileSettingsSectionSlug,
        "Profile settings",
        schema.WithFields(
            fields.New("profile", fields.TypeString, ...),
            fields.New("profile-registries", fields.TypeStringList, ...),
        ),
    )
}
```

Commands attach that section and decode it at execution time:

```go
profileSettings := &geppettosections.ProfileSettings{}
if err := parsedValues.DecodeSectionInto(geppettosections.ProfileSettingsSectionSlug, profileSettings); err != nil {
    return err
}

stepSettings, closeRegistry, err := runnerexample.ResolveInferenceSettingsFromRegistry(
    ctx,
    profileSettings.ProfileRegistries,
    profileSettings.Profile,
)
```

The Pinocchio-flavored bootstrap adds environment and config support through Glazed middleware:

```go
return []sources.Middleware{
    sources.FromCobra(cmd, fields.WithSource("cobra")),
    sources.FromArgs(args, fields.WithSource("arguments")),
    sources.FromEnv(cfg.EnvPrefix, fields.WithSource("env")),
    sources.FromConfigPlanBuilder(...),
    sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
}, nil
```

### What xgoja auth should copy

Auth should copy the same shape:

1. Define a section with fields for public auth settings.
2. Attach that section to the commands that need those settings.
3. Decode from `*values.Values` at command execution time.
4. Build services from decoded settings.
5. Never call `os.Getenv` inside the auth package.

## Target architecture

### High-level flow for `serve` jsverb commands

```mermaid
sequenceDiagram
  participant User
  participant Cobra as Cobra/Glazed parser
  participant Serve as HTTP serve provider
  participant HostAuth as hostauth builder
  participant Host as gojahttp.Host
  participant Runtime as xgoja runtime
  participant JS as JS route file

  User->>Cobra: serve sites demo --auth-mode dev --auth-default-store-dsn file.db
  Cobra->>Cobra: parse flags/env/config/defaults into *values.Values
  Cobra->>Serve: invoke serveVerb(ctx, parsedValues)
  Serve->>HostAuth: Decode auth section from parsedValues
  HostAuth->>HostAuth: ResolveConfig(config) without env lookup
  HostAuth->>HostAuth: Open stores / apply schema only now
  HostAuth->>Host: NewHost(hostOptionsWithAuth(httpSettings, authServices))
  Serve->>Runtime: NewRuntimeFromSectionsWithHostServices(parsedValues, host services)
  Runtime->>JS: load route file
  JS->>Host: app.get(...).auth(...).allow(...).handle(...)
```

### Proposed settings sections

Use a single public section named `auth`, with `schema.WithPrefix("auth-")` so flags are discoverable and namespaced.

Recommended first-pass fields:

| Section | Field | CLI flag | Type | Maps to |
| --- | --- | --- | --- | --- |
| `auth` | `mode` | `--auth-mode` | choice/string | `Config.Mode` |
| `auth` | `session-cookie-allow-insecure-http` | `--auth-session-cookie-allow-insecure-http` | bool | `Session.Cookie.AllowInsecureHTTP` |
| `auth` | `session-cookie-name` | `--auth-session-cookie-name` | string | `Session.Cookie.Name` |
| `auth` | `session-cookie-same-site` | `--auth-session-cookie-same-site` | choice/string | `Session.Cookie.SameSite` |
| `auth` | `session-cookie-path` | `--auth-session-cookie-path` | string | `Session.Cookie.Path` |
| `auth` | `session-idle-timeout` | `--auth-session-idle-timeout` | string | `Session.IdleTimeout` |
| `auth` | `session-absolute-timeout` | `--auth-session-absolute-timeout` | string | `Session.AbsoluteTimeout` |
| `auth` | `default-store-driver` | `--auth-default-store-driver` | choice/string | `Stores.Default.Driver` |
| `auth` | `default-store-dsn` | `--auth-default-store-dsn` | string | `Stores.Default.DSN` |
| `auth` | `default-store-apply-schema` | `--auth-default-store-apply-schema` | bool | `Stores.Default.ApplySchema` |
| `auth` | `session-store-driver` | `--auth-session-store-driver` | choice/string | `Stores.Session.Driver` |
| `auth` | `session-store-dsn` | `--auth-session-store-dsn` | string | `Stores.Session.DSN` |
| `auth` | `session-store-apply-schema` | `--auth-session-store-apply-schema` | bool | `Stores.Session.ApplySchema` |
| `auth` | `audit-store-*` | `--auth-audit-store-*` | varies | `Stores.Audit` |
| `auth` | `appauth-store-*` | `--auth-appauth-store-*` | varies | `Stores.AppAuth` |
| `auth` | `capability-store-*` | `--auth-capability-store-*` | varies | `Stores.Capability` |

This is intentionally verbose. Auth settings are operational/security settings; explicit names are better than clever abbreviations.

### Environment variable names

If the generated app has:

```yaml
app:
  name: generated-host-auth
```

then xgoja derives prefix `GENERATED_HOST_AUTH`. With section `auth` and field `default-store-dsn`, Glazed's env middleware reads:

```text
GENERATED_HOST_AUTH_AUTH_DEFAULT_STORE_DSN=/tmp/auth.sqlite
```

If the app config sets:

```yaml
app:
  envPrefix: XGOJA_DEMO
```

then the variable becomes:

```text
XGOJA_DEMO_AUTH_DEFAULT_STORE_DSN=/tmp/auth.sqlite
```

This replaces custom variables like `XGOJA_AUTH_SQLITE_DSN` and removes the need for `dsn-env`.

## API design

### New hostauth Glazed settings type

Add a small file such as `pkg/xgoja/hostauth/glazed.go`.

```go
package hostauth

import (
    "github.com/go-go-golems/glazed/pkg/cmds/fields"
    "github.com/go-go-golems/glazed/pkg/cmds/schema"
    "github.com/go-go-golems/glazed/pkg/cmds/values"
)

const SectionSlug = "auth"

type GlazedSettings struct {
    Mode string `glazed:"mode"`

    SessionCookieAllowInsecureHTTP bool   `glazed:"session-cookie-allow-insecure-http"`
    SessionCookieName              string `glazed:"session-cookie-name"`
    SessionCookieSameSite          string `glazed:"session-cookie-same-site"`
    SessionCookiePath              string `glazed:"session-cookie-path"`
    SessionIdleTimeout             string `glazed:"session-idle-timeout"`
    SessionAbsoluteTimeout         string `glazed:"session-absolute-timeout"`

    DefaultStoreDriver      string `glazed:"default-store-driver"`
    DefaultStoreDSN         string `glazed:"default-store-dsn"`
    DefaultStoreApplySchema bool   `glazed:"default-store-apply-schema"`

    SessionStoreDriver      string `glazed:"session-store-driver"`
    SessionStoreDSN         string `glazed:"session-store-dsn"`
    SessionStoreApplySchema bool   `glazed:"session-store-apply-schema"`

    AuditStoreDriver      string `glazed:"audit-store-driver"`
    AuditStoreDSN         string `glazed:"audit-store-dsn"`
    AuditStoreApplySchema bool   `glazed:"audit-store-apply-schema"`

    AppAuthStoreDriver      string `glazed:"appauth-store-driver"`
    AppAuthStoreDSN         string `glazed:"appauth-store-dsn"`
    AppAuthStoreApplySchema bool   `glazed:"appauth-store-apply-schema"`

    CapabilityStoreDriver      string `glazed:"capability-store-driver"`
    CapabilityStoreDSN         string `glazed:"capability-store-dsn"`
    CapabilityStoreApplySchema bool   `glazed:"capability-store-apply-schema"`
}
```

A first implementation can avoid explicitness tracking by creating the section with defaults from a base config. Then the decoded settings are already the effective settings.

```go
func GlazedConfigSection(base Config, opts ...schema.SectionOption) (schema.Section, error) {
    defaults := FlattenConfig(base)
    opts = append(opts,
        schema.WithPrefix("auth-"),
        schema.WithFields(
            fields.New("mode", fields.TypeChoice,
                fields.WithChoices("none", "dev", "oidc"),
                fields.WithDefault(defaults.Mode),
                fields.WithHelp("Generated-host auth mode")),
            // ...session/store fields...
        ),
    )
    return schema.NewSection(SectionSlug, "Generated host auth", opts...)
}

func ConfigFromValues(vals *values.Values, base Config) (Config, error) {
    section, err := GlazedConfigSection(base) // no prefix needed for decode; slug/field names same
    if err != nil { return Config{}, err }
    effective := GlazedSettings{}
    if vals == nil {
        return base, nil
    }
    if err := vals.DecodeSectionInto(SectionSlug, &effective); err != nil {
        return Config{}, err
    }
    return effective.ToConfig(), nil
}
```

Because Glazed applies defaults as a middleware, `DecodeSectionInto` sees values from flags/env/config/defaults. If a default comes from `base`, the command help and runtime behavior align.

### Simplify hostauth config

Remove `DSNEnv` and `LookupEnv`.

Before:

```go
type ResolveOptions struct {
    LookupEnv func(string) (string, bool)
}

type StoreConfig struct {
    Driver      string
    DSN         string
    DSNEnv      string
    ApplySchema *bool
}
```

After:

```go
type ResolveOptions struct{}

type StoreConfig struct {
    Driver      string `yaml:"driver" json:"driver"`
    DSN         string `yaml:"dsn" json:"dsn"`
    ApplySchema *bool  `yaml:"apply-schema" json:"apply-schema"`
}
```

`resolveDSN` becomes a pure validation function:

```go
func resolveDSN(path string, cfg StoreConfig) (string, error) {
    dsn := strings.TrimSpace(cfg.DSN)
    if dsn == "" {
        return "", configError(path+".dsn", fmt.Errorf("dsn is required for non-memory stores"))
    }
    return dsn, nil
}
```

### Change service factory semantics

`BuilderOptions.LookupEnv` should be removed. `BuildHostAuthServices` should use parsed values.

Recommended shape:

```go
type BuilderOptions struct {
    Config      Config // base/default config, not environment lookup config
    ActorLoader sessionauth.ActorLoader
    Now         func() time.Time
}

type ConfigDefaultsProvider interface {
    AuthConfigDefaults() Config
}

func (b *Builder) AuthConfigDefaults() Config {
    if b == nil { return Config{} }
    return b.options.Config
}

func (b *Builder) BuildHostAuthServices(ctx context.Context, vals *values.Values) (*Services, error) {
    if b == nil { return nil, errNilBuilder() }

    cfg, err := ConfigFromValues(vals, b.options.Config)
    if err != nil { return nil, err }

    resolved, err := ResolveConfig(cfg, ResolveOptions{})
    if err != nil { return nil, err }
    if resolved.Mode == ModeNone {
        return &Services{Config: resolved}, nil
    }
    // existing BuildStores/BuildSessionManager path
}
```

This keeps the existing lazy build point while making the input source explicit.

## Where to attach auth sections

### HTTP serve provider command set

`pkg/xgoja/providers/http/serve.go` is the most important attachment point. It builds a command per JavaScript verb and already appends selected module Glazed sections:

```go
sections, err := providerutil.CollectGlazedConfigSections(ctx.SelectedModules, providerapi.SectionRequest{
    CommandProviderID: ctx.Name,
}, nil)
...
cmd, err := registry.CommandForVerbWithInvoker(verb, ...)
...
if len(sections) > 0 {
    if err := addSectionsToServeCommand(cmd.Description(), sections, "http serve runtime"); err != nil {
        return nil, err
    }
}
```

Add the auth section here, with defaults from the configured service factory if present:

```go
func newServeCommandSet(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
    // existing setup...
    sections, err := providerutil.CollectGlazedConfigSections(...)
    if err != nil { return nil, err }

    authSection, err := authSectionForServe(ctx.Host)
    if err != nil { return nil, err }
    if authSection != nil {
        sections = append(sections, authSection)
    }

    hotReloadSection, err := serveHotReloadSection()
    if err != nil { return nil, err }
    sections = append(sections, hotReloadSection)
    // build commands...
}

func authSectionForServe(host providerapi.HostServices) (schema.Section, error) {
    defaults := hostauth.Config{}
    factory, ok, err := hostauth.LookupServiceFactory(host)
    if err != nil { return nil, err }
    if ok {
        if provider, ok := factory.(hostauth.ConfigDefaultsProvider); ok {
            defaults = provider.AuthConfigDefaults()
        }
    }
    // Return the section even without a factory if the stock HTTP provider can build auth from flags.
    return hostauth.GlazedConfigSection(defaults, schema.WithPrefix("auth-"))
}
```

### Built-in jsverbs command

The built-in `jsverbs` command in `pkg/xgoja/app/root.go` also appends module sections to each command. If auth is intended only for the HTTP `serve` command, do not build services there. The auth section may still appear because the HTTP provider exposes package sections, but it will only matter when `serveVerb` builds auth services.

If that is confusing in command help, use `SectionRequest.CommandProviderID == "serve"` in `GlazedConfigSections` to return auth fields only for the HTTP serve provider. The current `GlazedConfigSectionCapability` receives `SectionRequest`, so this is possible.

Recommended behavior:

- Always expose `http` settings for all runtime commands, as today.
- Expose `auth` settings only when `SectionRequest.CommandProviderID == "serve"` or `CommandName == "jsverbs"` if you intentionally want auth available there.
- Do not expose auth settings for `types`, `modules`, or docs-only commands.

## How `serveVerb` should build auth services

Currently `serveVerb` calls `buildServeAuthServices`, which looks up a host-provided `hostauth.ServiceFactory` and passes `parsedValues`. That is already the right call shape; the factory simply ignores `parsedValues` today.

Implementation choices:

### Option A: Keep service factory required for auth

This is the smallest change.

- `ConfigureServices` installs a `hostauth.ServiceFactory` with base defaults and optional hooks.
- `newServeCommandSet` uses factory defaults to build the auth section.
- `buildServeAuthServices` calls `factory.BuildHostAuthServices(ctx, parsedValues)`.
- The factory decodes `auth` section from `parsedValues` and merges with base defaults.

Pros:

- Minimal disruption to current `hostauth.LookupServiceFactory` design.
- Supports custom actor loaders and future OIDC normalizers.
- Keeps auth disabled when no factory exists.

Cons:

- A generated host still needs `ConfigureServices` to opt into generated-host auth.
- `--auth-mode dev` does nothing if no factory was registered, unless you add a clear validation error.

### Option B: Make stock auth a first-class HTTP provider capability

This is cleaner long-term.

- The HTTP provider always exposes `auth` settings for `serve` commands.
- `buildServeAuthServices` decodes `auth` settings directly.
- If `auth.mode == none`, no auth services are built.
- If `auth.mode != none`, build stock services with `hostauth.NewServiceFactory(...)` internally.
- A host-provided factory remains optional for custom hooks and can overlay/replace stock behavior.

Pros:

- No custom `main.go` service injection needed for the common generated-host auth case.
- `--auth-mode dev` always works on HTTP serve commands.
- Matches user mental model: auth is an HTTP serve setting.

Cons:

- Requires a slightly larger refactor of `buildServeAuthServices`.
- Needs a merge rule if both parsed auth settings and a custom factory are present.

Recommended path: **implement Option A first**, then consider Option B once OIDC/custom hostauth hooks are stable. Option A solves the immediate `os.Getenv` and `dsn-env` problem without redesigning provider ownership.

## Detailed implementation plan

### Phase 1: Add Glazed auth section helpers

Files:

- Add `pkg/xgoja/hostauth/glazed.go`.
- Add `pkg/xgoja/hostauth/glazed_test.go`.

Tasks:

1. Define `const SectionSlug = "auth"`.
2. Define `GlazedSettings` flat struct with `glazed` tags.
3. Implement `FlattenConfig(Config) GlazedSettings`.
4. Implement `GlazedSettings.ToConfig() Config`.
5. Implement `GlazedConfigSection(base Config, opts ...schema.SectionOption)`.
6. Implement `ConfigFromValues(vals *values.Values, base Config)`.

Tests:

- Section slug is `auth`.
- Section with prefix produces expected field names and defaults.
- `ConfigFromValues` maps `--auth-default-store-dsn` equivalent field values into `Config.Stores.Default.DSN`.
- Store-specific settings override defaults.
- Bool apply-schema maps into `*bool`.

Pseudocode for bool pointer handling:

```go
func boolPtr(v bool) *bool { return &v }

func (s GlazedSettings) store(prefix storePrefix) StoreConfig {
    apply := boolPtr(prefix.ApplySchema)
    return StoreConfig{
        Driver: strings.TrimSpace(prefix.Driver),
        DSN: strings.TrimSpace(prefix.DSN),
        ApplySchema: apply,
    }
}
```

If you need to distinguish omitted `apply-schema` from explicit false, use Glazed field logs or build section defaults from base config. For the first pass, defaults-from-base is usually enough because parsed values represent effective values, not patches.

### Phase 2: Remove `DSNEnv` and `LookupEnv`

Files:

- `pkg/xgoja/hostauth/config.go`
- `pkg/xgoja/hostauth/resolve.go`
- `pkg/xgoja/hostauth/builder.go`
- Tests under `pkg/xgoja/hostauth/*_test.go`
- Docs mentioning `dsn-env`.

Changes:

1. Delete `StoreConfig.DSNEnv`.
2. Delete `ResolveOptions.LookupEnv`.
3. Delete `BuilderOptions.LookupEnv`.
4. Remove `os` import from `hostauth/builder.go`.
5. Change `ResolveConfig(cfg, ResolveOptions{})` to never read env.
6. Change `inheritStoreConfig` so explicit `DSN` just overrides inherited `DSN`; there is no mutual exclusion with env.
7. Change missing DSN error to `dsn is required for non-memory stores`.

Tests to update/remove:

- Remove tests for `dsn-env`, missing env, and env conflict.
- Add tests that DSN inheritance still works.
- Add tests that `ResolveConfig(ModeNone, postgres-without-dsn)` follows the chosen mode-none behavior. I recommend `mode=none` not validating stores at all.

### Phase 3: Decode parsed values in `BuildHostAuthServices`

File:

- `pkg/xgoja/hostauth/builder.go`

Change:

```go
func (b *Builder) BuildHostAuthServices(ctx context.Context, vals *values.Values) (*Services, error) {
    if b == nil { return nil, errNilBuilder() }

    cfg, err := ConfigFromValues(vals, b.options.Config)
    if err != nil { return nil, err }

    resolved, err := ResolveConfig(cfg, ResolveOptions{})
    if err != nil { return nil, err }
    // existing build path
}
```

Tests:

- A builder with base memory/dev config can be overridden to SQLite through `values.Values`.
- No test injects `LookupEnv`; instead construct `values.Values` with `auth.default-store-dsn`.
- If `vals == nil`, base config behavior remains unchanged.

### Phase 4: Attach auth section to HTTP serve jsverb commands

File:

- `pkg/xgoja/providers/http/serve.go`

Change `newServeCommandSet` so every generated `serve` jsverb command has an `auth` section in its command description.

Pseudocode:

```go
func newServeCommandSet(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
    // existing jsverb/runtime checks

    sections, err := providerutil.CollectGlazedConfigSections(ctx.SelectedModules, providerapi.SectionRequest{
        CommandProviderID: ctx.Name,
    }, nil)
    if err != nil { return nil, err }

    authDefaults := hostauth.Config{}
    if factory, ok, err := hostauth.LookupServiceFactory(ctx.Host); err != nil {
        return nil, err
    } else if ok {
        if defaults, ok := factory.(hostauth.ConfigDefaultsProvider); ok {
            authDefaults = defaults.AuthConfigDefaults()
        }
    }

    authSection, err := hostauth.GlazedConfigSection(authDefaults, schema.WithPrefix("auth-"))
    if err != nil { return nil, err }
    sections = append(sections, authSection)

    // existing hot reload section and command construction
}
```

Tests:

- `TestNewServeCommandSetBuildsVerbCommandsWithHTTPSection` should become `...WithHTTPAndAuthSections`.
- Assert command description has `auth` section.
- Assert `auth` section has fields such as `mode`, `default-store-dsn`, and `session-cookie-allow-insecure-http`.
- If factory defaults are supplied, assert auth section defaults reflect them.

### Phase 5: Remove direct env reads from examples

Files:

- `examples/xgoja/21-generated-host-auth/cmd/host/main.go`
- `examples/xgoja/21-generated-host-auth/Makefile`
- `examples/xgoja/21-generated-host-auth/README.md`
- `examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go`
- Keycloak example docs/scripts as needed.

For generated-host auth, replace `authConfigFromEnv()` with static defaults and Glazed flags.

Before:

```go
switch strings.ToLower(strings.TrimSpace(os.Getenv("XGOJA_AUTH_STORE"))) {
case "sqlite":
    dsn := strings.TrimSpace(os.Getenv("XGOJA_AUTH_SQLITE_DSN"))
    // ...
}
```

After:

```go
ConfigureServices: func(services *app.HostServices) {
    configureErr = services.SetHostService(
        hostauth.ServiceFactoryKey,
        hostauth.NewServiceFactory(hostauth.BuilderOptions{
            Config: hostauth.Config{
                Mode: hostauth.ModeDev,
                Session: hostauth.SessionConfig{
                    Cookie: hostauth.CookieConfig{AllowInsecureHTTP: true},
                },
                Stores: hostauth.StoresConfig{
                    Default: hostauth.StoreConfig{Driver: string(hostauth.StoreDriverMemory)},
                },
            },
        }),
    )
}
```

The Makefile uses flags instead of env:

```make
sqlite-smoke:
	cd $(REPO_ROOT) && GOWORK=off go run ./examples/xgoja/21-generated-host-auth/cmd/host \
	  serve sites demo \
	  --http-listen $$addr \
	  --auth-mode dev \
	  --auth-default-store-driver sqlite \
	  --auth-default-store-dsn $$db \
	  --auth-default-store-apply-schema
```

If you want the smoke to demonstrate env, set `app.envPrefix` and use Glazed env names:

```make
GENERATED_HOST_AUTH_AUTH_DEFAULT_STORE_DRIVER=sqlite \
GENERATED_HOST_AUTH_AUTH_DEFAULT_STORE_DSN=$$db \
GENERATED_HOST_AUTH_AUTH_DEFAULT_STORE_APPLY_SCHEMA=true \
  go run ... serve sites demo --http-listen $$addr
```

That is acceptable because the example is not calling `os.Getenv`; Glazed middleware is.

For the Keycloak example, either:

- Convert it to a Glazed command with explicit sections, or
- Defer it until generated-host `mode=oidc` exists and document that all OIDC/client/store values must become Glazed fields.

Do not keep `flag.String("x", os.Getenv("X"), ...)` defaults.

### Phase 6: Update docs

Files to update:

- `cmd/xgoja/doc/17-xgoja-v2-reference.md`
- `cmd/xgoja/doc/11-provider-runtime-config-and-host-services.md`
- `pkg/doc/31-express-auth-examples.md`
- `examples/xgoja/21-generated-host-auth/README.md`

Remove `dsn-env` examples. Replace with flags/config/env-through-Glazed examples.

Old:

```yaml
auth:
  stores:
    default:
      driver: postgres
      dsn-env: AUTH_DB_DSN
```

New public command config file shape:

```yaml
auth:
  mode: dev
  default-store-driver: postgres
  default-store-dsn: postgres://user:pass@localhost/app?sslmode=disable
  default-store-apply-schema: true
```

New CLI shape:

```bash
generated-host-auth serve sites demo \
  --auth-mode dev \
  --auth-default-store-driver sqlite \
  --auth-default-store-dsn ./auth.sqlite \
  --auth-default-store-apply-schema
```

New env shape through Glazed:

```bash
GENERATED_HOST_AUTH_AUTH_MODE=dev \
GENERATED_HOST_AUTH_AUTH_DEFAULT_STORE_DRIVER=sqlite \
GENERATED_HOST_AUTH_AUTH_DEFAULT_STORE_DSN=./auth.sqlite \
GENERATED_HOST_AUTH_AUTH_DEFAULT_STORE_APPLY_SCHEMA=true \
  generated-host-auth serve sites demo
```

## Decision records

### Decision: Use Glazed sections instead of `dsn-env`

- **Context:** Host auth currently supports both direct DSNs and environment indirection through `dsn-env`/`LookupEnv`.
- **Options considered:** Keep `dsn-env`; remove env support completely; use Glazed env/config/flag parsing.
- **Decision:** Remove `dsn-env` and use Glazed parsing only.
- **Rationale:** xgoja already has a consistent command input pipeline. Direct env lookup duplicates precedence rules, makes tests harder, and bypasses command help/config introspection.
- **Consequences:** Users must pass DSNs as flags, config fields, or Glazed-recognized env vars. Docs must clearly show the new env variable names.
- **Status:** proposed

### Decision: Attach auth settings to HTTP `serve` jsverb commands

- **Context:** Auth services are needed when HTTP `serve` builds a Go-owned host and loads JavaScript route declarations.
- **Options considered:** Attach auth fields to all xgoja commands; attach only to generated-host custom main; attach to HTTP provider `serve` commands.
- **Decision:** Attach auth fields to HTTP provider `serve` jsverb commands.
- **Rationale:** `serve.go` already wraps jsverbs, appends runtime sections, receives parsed values, and builds auth services before runtime creation.
- **Consequences:** `serve sites demo --auth-*` becomes the main user interface. Built-in non-serve commands should not build auth services.
- **Status:** proposed

### Decision: Keep JavaScript route plans separate from auth infrastructure config

- **Context:** Express route files declare `.auth(...)`, `.allow(...)`, `.csrf()`, and `.resource(...)`; hostauth config controls cookies/stores/modes.
- **Options considered:** Let JS configure stores/cookies; keep all infra config in Go/Glazed.
- **Decision:** Keep infrastructure config in Go/Glazed.
- **Rationale:** Store DSNs, cookie security, OIDC clients, and schema application are host-owned operational concerns, not JS route author concerns.
- **Consequences:** JS remains portable; generated binary operators configure auth through flags/env/config.
- **Status:** proposed

## Testing strategy

### Unit tests

Add or update tests in `pkg/xgoja/hostauth`:

```text
TestGlazedConfigSectionDefaultsFromBaseConfig
TestConfigFromValuesMapsAuthSectionToNestedConfig
TestResolveConfigNoLongerSupportsDSNEnv
TestBuilderBuildHostAuthServicesUsesParsedValues
TestModeNoneDoesNotValidateStores
```

Add or update tests in `pkg/xgoja/providers/http`:

```text
TestNewServeCommandSetBuildsVerbCommandsWithHTTPAndAuthSections
TestServeVerbUsesAuthFlagsForSQLiteStore
TestServeVerbHotReloadUsesAuthFlags
```

Update examples tests/smokes:

```text
make -C examples/xgoja/21-generated-host-auth smoke
```

### Grep checks

Use these during review:

```bash
rg -n 'DSNEnv|LookupEnv|dsn-env|os\.Getenv\("XGOJA_AUTH|os\.Getenv\(".*DSN|KEYCLOAK_.*os\.Getenv' \
  pkg/xgoja/hostauth pkg/xgoja/providers/http examples/xgoja cmd/xgoja/doc pkg/doc -S
```

Expected outcome after the auth refactor:

- No `DSNEnv` or `LookupEnv` in `pkg/xgoja/hostauth`.
- No `dsn-env` in docs/examples.
- No auth DSN/OIDC `os.Getenv` defaults in example `main.go` files.
- Some unrelated `os.Getenv` may remain elsewhere in the repository for non-auth concerns; do not expand the PR unless the ticket explicitly asks for full repository cleanup.

### Manual CLI checks

After implementing the section, generated serve command help should include auth flags:

```bash
go run ./examples/xgoja/21-generated-host-auth/cmd/host serve sites demo --help | rg 'auth-'
```

SQLite smoke through flags:

```bash
db=$(mktemp -u /tmp/xgoja-auth.XXXXXX.sqlite)
go run ./examples/xgoja/21-generated-host-auth/cmd/host serve sites demo \
  --http-listen 127.0.0.1:8787 \
  --auth-mode dev \
  --auth-default-store-driver sqlite \
  --auth-default-store-dsn "$db" \
  --auth-default-store-apply-schema
```

Env-through-Glazed smoke, if the app has an env prefix:

```bash
GENERATED_HOST_AUTH_AUTH_MODE=dev \
GENERATED_HOST_AUTH_AUTH_DEFAULT_STORE_DRIVER=sqlite \
GENERATED_HOST_AUTH_AUTH_DEFAULT_STORE_DSN="$db" \
GENERATED_HOST_AUTH_AUTH_DEFAULT_STORE_APPLY_SCHEMA=true \
  go run ./examples/xgoja/21-generated-host-auth/cmd/host serve sites demo --http-listen 127.0.0.1:8787
```

## Intern reading order

Read these files in this order before coding:

1. `pkg/xgoja/app/middlewares.go` — understand where env/config/default values enter xgoja.
2. `pkg/xgoja/providers/http/http.go` — study the existing `--http-*` section and `XGojaConfigFromGlazed` pattern.
3. `pkg/xgoja/providers/http/serve.go` — understand how provider command sets turn jsverbs into `serve` subcommands and receive parsed values.
4. `pkg/xgoja/app/root.go` — compare built-in jsverbs command section attachment with the HTTP serve provider wrapper.
5. `pkg/xgoja/app/factory.go` — understand how parsed values are passed into runtime construction, module config overlays, and host-service contributors.
6. `pkg/xgoja/hostauth/config.go`, `resolve.go`, `builder.go`, `stores.go` — understand current auth config, resolution, lazy store opening, and service construction.
7. `geppetto/pkg/sections/profile_sections.go` and `geppetto/cmd/examples/runner-glazed-registry-flags/main.go` — understand the analogous profile section / decode / resolve pattern.
8. `examples/xgoja/21-generated-host-auth` — update the generated-host example after the library path is in place.

## Review checklist for the implementation PR

- [ ] `hostauth` no longer imports `os`.
- [ ] `StoreConfig` no longer has `DSNEnv`.
- [ ] `ResolveOptions` no longer has `LookupEnv`.
- [ ] Docs no longer mention `dsn-env` except possibly in migration notes.
- [ ] HTTP serve jsverb commands expose `auth` Glazed section fields.
- [ ] `BuildHostAuthServices(ctx, vals)` uses parsed `vals`.
- [ ] SQLite generated-host smoke uses flags or Glazed env, not direct `os.Getenv`.
- [ ] Tests cover flag/env/config-style parsed values by constructing `values.Values`, not process env lookups inside hostauth.
- [ ] `go test ./pkg/xgoja/hostauth ./pkg/xgoja/providers/http ./pkg/xgoja/app -count=1` passes.
- [ ] `make -C examples/xgoja/21-generated-host-auth smoke` passes.

## Appendix: old-to-new mapping

| Old setting source | New source |
| --- | --- |
| `StoreConfig.DSNEnv: "AUTH_DB_DSN"` | `auth.default-store-dsn` parsed from flag/config/env by Glazed |
| `BuilderOptions.LookupEnv` | Removed |
| `os.LookupEnv` in `BuildHostAuthServices` | Removed |
| `XGOJA_AUTH_STORE=sqlite` custom example env | `--auth-default-store-driver sqlite` or `GENERATED_HOST_AUTH_AUTH_DEFAULT_STORE_DRIVER=sqlite` |
| `XGOJA_AUTH_SQLITE_DSN=/tmp/auth.sqlite` custom example env | `--auth-default-store-dsn /tmp/auth.sqlite` or `GENERATED_HOST_AUTH_AUTH_DEFAULT_STORE_DSN=/tmp/auth.sqlite` |
| `flag.String("session-db-dsn", os.Getenv("SESSION_DB_DSN"), ...)` | Glazed field `auth.session-store-dsn` with flag/config/env parsing |

## References

- Evidence: `sources/12-auth-glazed-bubbling-evidence.md`
- xgoja middleware: `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/app/middlewares.go`
- HTTP provider config: `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/http/http.go`
- HTTP serve provider: `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/providers/http/serve.go`
- hostauth config: `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/config.go`
- hostauth resolver: `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/resolve.go`
- Geppetto profile section: `/home/manuel/code/wesen/go-go-golems/geppetto/pkg/sections/profile_sections.go`
- Geppetto registry flags example: `/home/manuel/code/wesen/go-go-golems/geppetto/cmd/examples/runner-glazed-registry-flags/main.go`
