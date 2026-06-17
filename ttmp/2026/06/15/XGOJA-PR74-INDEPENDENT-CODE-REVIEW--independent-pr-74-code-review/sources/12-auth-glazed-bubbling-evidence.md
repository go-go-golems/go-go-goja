---
Title: Auth Glazed Bubbling Evidence
Ticket: XGOJA-PR74-INDEPENDENT-CODE-REVIEW
Status: active
Topics:
  - review
  - evidence
  - auth
  - glazed
DocType: reference
Intent: short-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Evidence for the Glazed auth settings bubbling design guide.
LastUpdated: 2026-06-15T17:20:00-04:00
WhatFor: Line-anchored source excerpts and grep outputs for removing direct env lookups from auth config.
WhenToUse: Use with design/02-glazed-auth-settings-bubbling-implementation-guide.md.
---

# Auth Glazed Bubbling Evidence

## os.Getenv / LookupEnv / dsn-env inventory

```text
pkg/xgoja/hostauth/resolve_test.go:82:		Default: StoreConfig{Driver: "postgres", DSNEnv: "APP_AUTH_DATABASE_URL", ApplySchema: &applyDefault},
pkg/xgoja/hostauth/resolve_test.go:85:	}}, ResolveOptions{LookupEnv: func(key string) (string, bool) {
pkg/xgoja/hostauth/resolve_test.go:141:		{name: "missing dsn", cfg: Config{Stores: StoresConfig{Default: StoreConfig{Driver: "postgres"}}}, path: "auth.stores.session.dsn", want: "dsn or dsn-env is required"},
pkg/xgoja/hostauth/resolve_test.go:142:		{name: "dsn conflict", cfg: Config{Stores: StoresConfig{Default: StoreConfig{Driver: "postgres", DSN: "postgres://example", DSNEnv: "APP_AUTH_DATABASE_URL"}}}, path: "auth.stores.session", want: "set either dsn or dsn-env"},
pkg/xgoja/hostauth/resolve_test.go:143:		{name: "missing env", cfg: Config{Stores: StoresConfig{Default: StoreConfig{Driver: "postgres", DSNEnv: "APP_AUTH_DATABASE_URL"}}}, path: "auth.stores.session.dsn-env", want: "is not set"},
pkg/xgoja/hostauth/resolve_test.go:147:			_, err := ResolveConfig(tt.cfg, ResolveOptions{LookupEnv: func(string) (string, bool) { return "", false }})
pkg/xgoja/hostauth/builder_test.go:130:		Stores: StoresConfig{Default: StoreConfig{Driver: "sqlite", DSNEnv: "HOSTAUTH_TEST_DSN"}},
pkg/xgoja/hostauth/builder_test.go:141:			Stores: StoresConfig{Default: StoreConfig{Driver: "sqlite", DSNEnv: "HOSTAUTH_TEST_DSN", ApplySchema: &applySchema}},
pkg/xgoja/hostauth/builder_test.go:143:		LookupEnv: func(key string) (string, bool) {
examples/xgoja/21-generated-host-auth/cmd/host/main.go:65:	switch strings.ToLower(strings.TrimSpace(os.Getenv("XGOJA_AUTH_STORE"))) {
examples/xgoja/21-generated-host-auth/cmd/host/main.go:69:		dsn := strings.TrimSpace(os.Getenv("XGOJA_AUTH_SQLITE_DSN"))
examples/xgoja/21-generated-host-auth/cmd/host/main.go:71:			return hostauth.Config{}, fmt.Errorf("XGOJA_AUTH_SQLITE_DSN is required when XGOJA_AUTH_STORE=sqlite")
examples/xgoja/21-generated-host-auth/cmd/host/main.go:81:		return hostauth.Config{}, fmt.Errorf("unsupported XGOJA_AUTH_STORE %q (want memory or sqlite)", os.Getenv("XGOJA_AUTH_STORE"))
pkg/xgoja/hostauth/resolve.go:14:	LookupEnv func(string) (string, bool)
pkg/xgoja/hostauth/resolve.go:52:	stores, err := resolveStoresConfig(cfg.Stores, opts.LookupEnv)
pkg/xgoja/hostauth/resolve.go:191:		out.DSNEnv = ""
pkg/xgoja/hostauth/resolve.go:193:	if strings.TrimSpace(specific.DSNEnv) != "" {
pkg/xgoja/hostauth/resolve.go:194:		out.DSNEnv = specific.DSNEnv
pkg/xgoja/hostauth/resolve.go:216:	dsnEnv := strings.TrimSpace(cfg.DSNEnv)
pkg/xgoja/hostauth/resolve.go:218:		return "", configError(path, fmt.Errorf("set either dsn or dsn-env, not both"))
pkg/xgoja/hostauth/resolve.go:225:			return "", configError(path+".dsn-env", fmt.Errorf("environment lookup is not configured"))
pkg/xgoja/hostauth/resolve.go:229:			return "", configError(path+".dsn-env", fmt.Errorf("environment variable %q is not set", dsnEnv))
pkg/xgoja/hostauth/resolve.go:233:	return "", configError(path+".dsn", fmt.Errorf("dsn or dsn-env is required for non-memory stores"))
examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go:34:	issuer := flag.String("issuer", envOr("KEYCLOAK_ISSUER", "http://127.0.0.1:18080/realms/goja-demo"), "OIDC issuer URL")
examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go:35:	clientID := flag.String("client-id", envOr("KEYCLOAK_CLIENT_ID", "goja-app"), "OIDC client ID")
examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go:36:	clientSecret := flag.String("client-secret", os.Getenv("KEYCLOAK_CLIENT_SECRET"), "OIDC client secret, if configured")
examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go:37:	sessionDBDSN := flag.String("session-db-dsn", os.Getenv("SESSION_DB_DSN"), "Postgres DSN for persistent app sessions; empty uses in-memory sessions")
examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go:38:	auditDBDSN := flag.String("audit-db-dsn", os.Getenv("AUDIT_DB_DSN"), "Postgres DSN for persistent audit records; empty logs audit records")
examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go:39:	appDBDSN := flag.String("app-db-dsn", os.Getenv("APPAUTH_DB_DSN"), "Postgres DSN for persistent appauth users/resources; empty uses in-memory appauth")
examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go:40:	capabilityDBDSN := flag.String("capability-db-dsn", os.Getenv("CAPABILITY_DB_DSN"), "Postgres DSN for persistent capability tokens; empty uses in-memory capabilities")
examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go:409:	if value := os.Getenv(name); value != "" {
pkg/xgoja/hostauth/builder.go:19:	LookupEnv   func(string) (string, bool)
pkg/xgoja/hostauth/builder.go:51:	lookupEnv := b.options.LookupEnv
pkg/xgoja/hostauth/builder.go:53:		lookupEnv = os.LookupEnv
pkg/xgoja/hostauth/builder.go:55:	resolved, err := ResolveConfig(b.options.Config, ResolveOptions{LookupEnv: lookupEnv})
cmd/xgoja/doc/11-provider-runtime-config-and-host-services.md:401:See `examples/xgoja/21-generated-host-auth` for a runnable runtime-package host that uses memory stores by default and SQLite stores when `XGOJA_AUTH_STORE=sqlite` plus `XGOJA_AUTH_SQLITE_DSN` are provided.
examples/xgoja/21-generated-host-auth/README.md:74:To demonstrate persistent stores, set `XGOJA_AUTH_STORE=sqlite` and provide the
examples/xgoja/21-generated-host-auth/README.md:79:XGOJA_AUTH_STORE=sqlite \
examples/xgoja/21-generated-host-auth/README.md:80:XGOJA_AUTH_SQLITE_DSN=/tmp/xgoja-generated-host-auth.sqlite \
pkg/xgoja/hostauth/config.go:65:	DSNEnv      string `yaml:"dsn-env" json:"dsn-env"`
cmd/xgoja/doc/17-xgoja-v2-reference.md:431:      dsn-env: AUTH_DB_DSN
cmd/xgoja/doc/17-xgoja-v2-reference.md:436:      dsn-env: CAP_DB_DSN  # override inherited DSN source only
cmd/xgoja/doc/17-xgoja-v2-reference.md:439:`dsn` and `dsn-env` are mutually exclusive after inheritance, and an explicit
examples/xgoja/19-express-keycloak-auth-host/docker-compose.yml:22:      KEYCLOAK_ADMIN: admin
examples/xgoja/19-express-keycloak-auth-host/docker-compose.yml:23:      KEYCLOAK_ADMIN_PASSWORD: admin
examples/xgoja/19-express-keycloak-auth-host/docker-compose.yml:26:      - "${KEYCLOAK_PORT:-18080}:8080"
examples/xgoja/19-express-keycloak-auth-host/Makefile:6:KEYCLOAK_PORT ?= 18080
examples/xgoja/19-express-keycloak-auth-host/Makefile:8:ISSUER ?= http://127.0.0.1:$(KEYCLOAK_PORT)/realms/goja-demo
examples/xgoja/19-express-keycloak-auth-host/Makefile:9:SESSION_DB_DSN ?= postgres://goja:goja@127.0.0.1:$(POSTGRES_PORT)/goja_auth?sslmode=disable
examples/xgoja/19-express-keycloak-auth-host/Makefile:10:AUDIT_DB_DSN ?= $(SESSION_DB_DSN)
examples/xgoja/19-express-keycloak-auth-host/Makefile:11:APPAUTH_DB_DSN ?= $(SESSION_DB_DSN)
examples/xgoja/19-express-keycloak-auth-host/Makefile:12:CAPABILITY_DB_DSN ?= $(SESSION_DB_DSN)
examples/xgoja/19-express-keycloak-auth-host/Makefile:17:	KEYCLOAK_PORT=$(KEYCLOAK_PORT) POSTGRES_PORT=$(POSTGRES_PORT) docker compose -f $(EXAMPLE_DIR)/docker-compose.yml up -d
examples/xgoja/19-express-keycloak-auth-host/Makefile:31:	cd $(REPO_ROOT) && GOWORK=off go run $(HOST) --script $(SCRIPT) --listen $(LISTEN) --issuer $(ISSUER) --session-db-dsn '$(SESSION_DB_DSN)' --audit-db-dsn '$(AUDIT_DB_DSN)' --app-db-dsn '$(APPAUTH_DB_DSN)' --capability-db-dsn '$(CAPABILITY_DB_DSN)'
examples/xgoja/19-express-keycloak-auth-host/Makefile:37:	KEYCLOAK_PORT=$(KEYCLOAK_PORT) POSTGRES_PORT=$(POSTGRES_PORT) ISSUER=$(ISSUER) LISTEN=$(LISTEN) SESSION_DB_DSN='$(SESSION_DB_DSN)' AUDIT_DB_DSN='$(AUDIT_DB_DSN)' APPAUTH_DB_DSN='$(APPAUTH_DB_DSN)' CAPABILITY_DB_DSN='$(CAPABILITY_DB_DSN)' $(EXAMPLE_DIR)/scripts/smoke.sh
examples/xgoja/19-express-keycloak-auth-host/scripts/smoke.sh:8:KEYCLOAK_PORT="${KEYCLOAK_PORT:-18080}"
examples/xgoja/19-express-keycloak-auth-host/scripts/smoke.sh:10:ISSUER="${ISSUER:-http://127.0.0.1:${KEYCLOAK_PORT}/realms/goja-demo}"
examples/xgoja/19-express-keycloak-auth-host/scripts/smoke.sh:11:SESSION_DB_DSN="${SESSION_DB_DSN:-postgres://goja:goja@127.0.0.1:${POSTGRES_PORT}/goja_auth?sslmode=disable}"
examples/xgoja/19-express-keycloak-auth-host/scripts/smoke.sh:12:AUDIT_DB_DSN="${AUDIT_DB_DSN:-${SESSION_DB_DSN}}"
examples/xgoja/19-express-keycloak-auth-host/scripts/smoke.sh:13:APPAUTH_DB_DSN="${APPAUTH_DB_DSN:-${SESSION_DB_DSN}}"
examples/xgoja/19-express-keycloak-auth-host/scripts/smoke.sh:14:CAPABILITY_DB_DSN="${CAPABILITY_DB_DSN:-${SESSION_DB_DSN}}"
examples/xgoja/19-express-keycloak-auth-host/scripts/smoke.sh:86:if [[ "${SKIP_KEYCLOAK_UP:-0}" != "1" ]]; then
examples/xgoja/19-express-keycloak-auth-host/scripts/smoke.sh:88:  KEYCLOAK_PORT="${KEYCLOAK_PORT}" POSTGRES_PORT="${POSTGRES_PORT}" docker compose -f "${EXAMPLE_DIR}/docker-compose.yml" up -d
examples/xgoja/19-express-keycloak-auth-host/scripts/smoke.sh:101:  --session-db-dsn "${SESSION_DB_DSN}" \
examples/xgoja/19-express-keycloak-auth-host/scripts/smoke.sh:102:  --audit-db-dsn "${AUDIT_DB_DSN}" \
examples/xgoja/19-express-keycloak-auth-host/scripts/smoke.sh:103:  --app-db-dsn "${APPAUTH_DB_DSN}" \
examples/xgoja/19-express-keycloak-auth-host/scripts/smoke.sh:104:  --capability-db-dsn "${CAPABILITY_DB_DSN}" >"${HOST_LOG}" 2>&1 &
examples/xgoja/19-express-keycloak-auth-host/README.md:23:It uses only Python standard-library HTTP/form handling; no browser driver is required. Set `KEEP_KEYCLOAK=1` to leave the containers running after the smoke, `KEYCLOAK_PORT=18081` if port `18080` is already in use, or `POSTGRES_PORT=15433` if port `15432` is already in use.
examples/xgoja/21-generated-host-auth/Makefile:46:	cd $(REPO_ROOT) && XGOJA_AUTH_STORE=sqlite XGOJA_AUTH_SQLITE_DSN=$$db GOWORK=off go run ./examples/xgoja/21-generated-host-auth/cmd/host serve sites demo --http-listen $$addr >$$log 2>&1 & \
```

## go-go-goja line anchors

### pkg/xgoja/hostauth/config.go:23-66
```go
    23		StoreDriverPostgres StoreDriver = "postgres"
    24	)
    25	
    26	// Config is the generated-host auth infrastructure configuration. It is host
    27	// config, not JavaScript route config and not an authorization policy DSL.
    28	type Config struct {
    29		Mode    Mode          `yaml:"mode" json:"mode"`
    30		Session SessionConfig `yaml:"session" json:"session"`
    31		Stores  StoresConfig  `yaml:"stores" json:"stores"`
    32	}
    33	
    34	// SessionConfig controls server-side app session behavior.
    35	type SessionConfig struct {
    36		Cookie          CookieConfig `yaml:"cookie" json:"cookie"`
    37		IdleTimeout     string       `yaml:"idle-timeout" json:"idle-timeout"`
    38		AbsoluteTimeout string       `yaml:"absolute-timeout" json:"absolute-timeout"`
    39	}
    40	
    41	// CookieConfig controls the app session cookie. Empty Name delegates to
    42	// sessionauth.New's secure default cookie name.
    43	type CookieConfig struct {
    44		AllowInsecureHTTP bool   `yaml:"allow-insecure-http" json:"allow-insecure-http"`
    45		Name              string `yaml:"name" json:"name"`
    46		SameSite          string `yaml:"same-site" json:"same-site"`
    47		Path              string `yaml:"path" json:"path"`
    48	}
    49	
    50	// StoresConfig configures the persistent stores used by host-owned auth
    51	// infrastructure. Per-store blocks inherit from Default field-by-field.
    52	type StoresConfig struct {
    53		Default    StoreConfig `yaml:"default" json:"default"`
    54		Session    StoreConfig `yaml:"session" json:"session"`
    55		Audit      StoreConfig `yaml:"audit" json:"audit"`
    56		AppAuth    StoreConfig `yaml:"appauth" json:"appauth"`
    57		Capability StoreConfig `yaml:"capability" json:"capability"`
    58	}
    59	
    60	// StoreConfig configures one store. ApplySchema is a pointer so inheritance can
    61	// distinguish an omitted value from an explicit false override.
    62	type StoreConfig struct {
    63		Driver      string `yaml:"driver" json:"driver"`
    64		DSN         string `yaml:"dsn" json:"dsn"`
    65		DSNEnv      string `yaml:"dsn-env" json:"dsn-env"`
    66		ApplySchema *bool  `yaml:"apply-schema" json:"apply-schema"`
```

### pkg/xgoja/hostauth/resolve.go:39-56
```go
    39	func ResolveConfig(cfg Config, opts ResolveOptions) (ResolvedConfig, error) {
    40		mode, err := parseMode(cfg.Mode)
    41		if err != nil {
    42			return ResolvedConfig{}, configError("auth.mode", err)
    43		}
    44		if mode == ModeOIDC {
    45			return ResolvedConfig{}, configError("auth.mode", ErrOIDCNotImplemented)
    46		}
    47	
    48		session, err := resolveSessionConfig(cfg.Session)
    49		if err != nil {
    50			return ResolvedConfig{}, err
    51		}
    52		stores, err := resolveStoresConfig(cfg.Stores, opts.LookupEnv)
    53		if err != nil {
    54			return ResolvedConfig{}, err
    55		}
    56		return ResolvedConfig{Mode: mode, Session: session, Stores: stores}, nil
```

### pkg/xgoja/hostauth/resolve.go:176-233
```go
   176	
   177		dsn, err := resolveDSN(path, merged, lookupEnv)
   178		if err != nil {
   179			return ResolvedStoreConfig{}, err
   180		}
   181		return ResolvedStoreConfig{Name: name, Driver: driver, DSN: dsn, ApplySchema: applySchema}, nil
   182	}
   183	
   184	func inheritStoreConfig(specific StoreConfig, defaults StoreConfig) StoreConfig {
   185		out := defaults
   186		if strings.TrimSpace(specific.Driver) != "" {
   187			out.Driver = specific.Driver
   188		}
   189		if strings.TrimSpace(specific.DSN) != "" {
   190			out.DSN = specific.DSN
   191			out.DSNEnv = ""
   192		}
   193		if strings.TrimSpace(specific.DSNEnv) != "" {
   194			out.DSNEnv = specific.DSNEnv
   195			out.DSN = ""
   196		}
   197		if specific.ApplySchema != nil {
   198			out.ApplySchema = specific.ApplySchema
   199		}
   200		return out
   201	}
   202	
   203	func parseStoreDriver(value string) (StoreDriver, error) {
   204		switch normalized := StoreDriver(strings.ToLower(strings.TrimSpace(value))); normalized {
   205		case "":
   206			return StoreDriverMemory, nil
   207		case StoreDriverMemory, StoreDriverSQLite, StoreDriverPostgres:
   208			return normalized, nil
   209		default:
   210			return "", fmt.Errorf("unsupported store driver %q", value)
   211		}
   212	}
   213	
   214	func resolveDSN(path string, cfg StoreConfig, lookupEnv func(string) (string, bool)) (string, error) {
   215		dsn := strings.TrimSpace(cfg.DSN)
   216		dsnEnv := strings.TrimSpace(cfg.DSNEnv)
   217		if dsn != "" && dsnEnv != "" {
   218			return "", configError(path, fmt.Errorf("set either dsn or dsn-env, not both"))
   219		}
   220		if dsn != "" {
   221			return dsn, nil
   222		}
   223		if dsnEnv != "" {
   224			if lookupEnv == nil {
   225				return "", configError(path+".dsn-env", fmt.Errorf("environment lookup is not configured"))
   226			}
   227			value, ok := lookupEnv(dsnEnv)
   228			if !ok || strings.TrimSpace(value) == "" {
   229				return "", configError(path+".dsn-env", fmt.Errorf("environment variable %q is not set", dsnEnv))
   230			}
   231			return strings.TrimSpace(value), nil
   232		}
   233		return "", configError(path+".dsn", fmt.Errorf("dsn or dsn-env is required for non-memory stores"))
```

### pkg/xgoja/hostauth/builder.go:15-58
```go
    15	
    16	// BuilderOptions configures a generated-host auth service factory.
    17	type BuilderOptions struct {
    18		Config      Config
    19		LookupEnv   func(string) (string, bool)
    20		ActorLoader sessionauth.ActorLoader
    21		Now         func() time.Time
    22	}
    23	
    24	// Builder is the default hostauth ServiceFactory implementation.
    25	type Builder struct {
    26		options BuilderOptions
    27	}
    28	
    29	var (
    30		_ ServiceFactory = (*Builder)(nil)
    31	
    32		errServiceFactoryNil = errors.New("hostauth service factory is nil")
    33	)
    34	
    35	// NewServiceFactory returns a lazy generated-host auth service factory. The
    36	// factory resolves config and opens stores only when BuildHostAuthServices is
    37	// called, so command providers can discover the factory during command
    38	// construction without touching databases or env-dependent DSNs.
    39	func NewServiceFactory(opts BuilderOptions) *Builder {
    40		return &Builder{options: opts}
    41	}
    42	
    43	// BuildHostAuthServices builds concrete auth services for one command/runtime
    44	// execution. The vals argument is reserved for future Glazed-value overlays;
    45	// this first implementation resolves from BuilderOptions.Config and env refs.
    46	func (b *Builder) BuildHostAuthServices(ctx context.Context, vals *values.Values) (*Services, error) {
    47		_ = vals
    48		if b == nil {
    49			return nil, errNilBuilder()
    50		}
    51		lookupEnv := b.options.LookupEnv
    52		if lookupEnv == nil {
    53			lookupEnv = os.LookupEnv
    54		}
    55		resolved, err := ResolveConfig(b.options.Config, ResolveOptions{LookupEnv: lookupEnv})
    56		if err != nil {
    57			return nil, err
    58		}
```

### pkg/xgoja/providers/http/http.go:89-134
```go
    89	func (c *capability) GlazedConfigSections(providerapi.SectionRequest) ([]schema.Section, error) {
    90		section, err := httpConfigSection(schema.WithPrefix("http-"))
    91		if err != nil {
    92			return nil, err
    93		}
    94		return []schema.Section{section}, nil
    95	}
    96	
    97	func (c *capability) XGojaConfigSection(_ providerapi.SectionRequest, _ providerapi.ModuleDescriptor) (schema.Section, error) {
    98		return httpConfigSection()
    99	}
   100	
   101	func (c *capability) XGojaConfigFromGlazed(_ context.Context, req providerapi.XGojaConfigRequest) (*values.SectionValues, error) {
   102		out, err := values.NewSectionValues(req.ConfigSection)
   103		if err != nil {
   104			return nil, err
   105		}
   106		if req.GlazedValues == nil {
   107			return out, nil
   108		}
   109		for _, name := range []string{"enabled", "listen", "dev-errors", "reject-raw-routes"} {
   110			field, ok := req.GlazedValues.GetField("http", name)
   111			if !ok || !glazedFieldWasExplicit(field) {
   112				continue
   113			}
   114			definition, ok := req.ConfigSection.GetDefinitions().Get(name)
   115			if !ok {
   116				return nil, fmt.Errorf("internal http config field %q not found", name)
   117			}
   118			if err := out.Fields.UpdateWithLog(name, definition, field.Value, field.Log...); err != nil {
   119				return nil, err
   120			}
   121		}
   122		return out, nil
   123	}
   124	
   125	func httpConfigSection(options ...schema.SectionOption) (schema.Section, error) {
   126		options = append(options,
   127			schema.WithFields(
   128				fields.New("enabled", fields.TypeBool, fields.WithDefault(true), fields.WithHelp("Start the xgoja HTTP server for modules such as express")),
   129				fields.New("listen", fields.TypeString, fields.WithDefault("127.0.0.1:8787"), fields.WithHelp("HTTP listen address for xgoja-owned HTTP modules")),
   130				fields.New("dev-errors", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Return development JavaScript error details from the xgoja-owned HTTP host")),
   131				fields.New("reject-raw-routes", fields.TypeBool, fields.WithDefault(true), fields.WithHelp("Reject matched raw/unplanned routes; planned routes and static mounts are unaffected")),
   132			),
   133		)
   134		return schema.NewSection("http", "HTTP server", options...)
```

### pkg/xgoja/providers/http/serve.go:56-104
```go
    56	
    57	func newServeCommandSet(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
    58		jsverbSources, err := serveCommandJSVerbSources(ctx)
    59		if err != nil {
    60			return nil, err
    61		}
    62		if ctx.RuntimeFactory == nil {
    63			return nil, fmt.Errorf("http serve command requires runtime factory")
    64		}
    65		if _, _, err := hostauth.LookupServiceFactory(ctx.Host); err != nil {
    66			return nil, err
    67		}
    68	
    69		sections, err := providerutil.CollectGlazedConfigSections(ctx.SelectedModules, providerapi.SectionRequest{
    70			CommandProviderID: ctx.Name,
    71		}, nil)
    72		if err != nil {
    73			return nil, err
    74		}
    75		hotReloadSection, err := serveHotReloadSection()
    76		if err != nil {
    77			return nil, err
    78		}
    79		sections = append(sections, hotReloadSection)
    80	
    81		registries, err := jsverbSources.ScanAllJSVerbSources()
    82		if err != nil {
    83			return nil, err
    84		}
    85		commands := make([]cmds.Command, 0)
    86		for _, registry := range registries {
    87			if registry == nil {
    88				continue
    89			}
    90			for _, verb := range registry.Verbs() {
    91				verb := verb
    92				registry := registry
    93				cmd, err := registry.CommandForVerbWithInvoker(verb, func(runCtx context.Context, _ *jsverbs.Registry, verb *jsverbs.VerbSpec, parsedValues *values.Values) (interface{}, error) {
    94					return serveVerb(runCtx, ctx, registry, verb, parsedValues)
    95				})
    96				if err != nil {
    97					return nil, err
    98				}
    99				if len(sections) > 0 {
   100					if err := addSectionsToServeCommand(cmd.Description(), sections, "http serve runtime"); err != nil {
   101						return nil, err
   102					}
   103				}
   104				commands = append(commands, cmd)
```

### pkg/xgoja/providers/http/serve.go:110-170
```go
   110	func serveVerb(ctx context.Context, commandCtx providerapi.CommandSetContext, registry *jsverbs.Registry, verb *jsverbs.VerbSpec, parsedValues *values.Values) (interface{}, error) {
   111		if registry == nil {
   112			return nil, fmt.Errorf("jsverb registry is nil")
   113		}
   114		if verb == nil {
   115			return nil, fmt.Errorf("jsverb is nil")
   116		}
   117		hotReloadSettings, err := decodeServeHotReloadSettings(parsedValues)
   118		if err != nil {
   119			return nil, err
   120		}
   121		if hotReloadSettings.Enabled {
   122			return serveVerbHotReload(ctx, commandCtx, registry, verb, parsedValues, hotReloadSettings)
   123		}
   124		authServices, hasAuthFactory, err := buildServeAuthServices(ctx, commandCtx, parsedValues)
   125		if err != nil {
   126			return nil, err
   127		}
   128		if hasAuthFactory {
   129			defer func() { _ = authServices.Close(context.Background()) }()
   130		}
   131	
   132		var rt *engine.Runtime
   133		if hasAuthFactory {
   134			factory, ok := commandCtx.RuntimeFactory.(providerapi.RuntimeFactoryWithHostServices)
   135			if !ok || factory == nil {
   136				return nil, fmt.Errorf("http serve with hostauth requires runtime factory with per-runtime host services")
   137			}
   138			httpSettings, err := decodeHTTPServeSettings(parsedValues)
   139			if err != nil {
   140				return nil, err
   141			}
   142			includeGeneratedHost := true
   143			if externalHost, err := externalHostService(commandCtx.Host); err != nil {
   144				return nil, err
   145			} else if externalHost.Host != nil {
   146				includeGeneratedHost = false
   147			}
   148			runtimeServices, err := serveRuntimeServices(gojahttp.NewHost(hostOptionsWithAuth(httpSettings, authServices)), authServices, true, includeGeneratedHost)
   149			if err != nil {
   150				return nil, err
   151			}
   152			rt, err = factory.NewRuntimeFromSectionsWithHostServices(ctx, parsedValues, runtimeServices, require.WithLoader(registry.RequireLoader()))
   153			if err != nil {
   154				return nil, err
   155			}
   156		} else {
   157			rt, err = commandCtx.RuntimeFactory.NewRuntimeFromSections(ctx, parsedValues, require.WithLoader(registry.RequireLoader()))
   158			if err != nil {
   159				return nil, err
   160			}
   161		}
   162		defer func() { _ = rt.Close(context.Background()) }()
   163	
   164		if len(commandCtx.SelectedModules) > 0 {
   165			if err := providerutil.InitRuntimeFromSections(ctx, parsedValues, runtimeHandle{rt: rt}, commandCtx.SelectedModules); err != nil {
   166				return nil, err
   167			}
   168		}
   169		if _, err := registry.InvokeInRuntime(ctx, rt, verb, parsedValues); err != nil {
   170			return nil, err
```

### pkg/xgoja/providers/http/serve.go:406-421
```go
   406		services, err := factory.BuildHostAuthServices(ctx, parsedValues)
   407		if err != nil {
   408			return nil, false, err
   409		}
   410		if services == nil {
   411			return nil, false, fmt.Errorf("hostauth service factory returned nil services")
   412		}
   413		return services, true, nil
   414	}
   415	
   416	func hostOptionsWithAuth(cfg settings, authServices *hostauth.Services) gojahttp.HostOptions {
   417		opts := hostOptions(cfg)
   418		if authServices != nil {
   419			opts.Auth = authServices.AuthOptions
   420		}
   421		return opts
```

### pkg/xgoja/app/factory.go:147-223
```go
   147	func (f *RuntimeFactory) hostServicesForRuntime(ctx context.Context, vals *values.Values, descriptors []providerapi.ModuleDescriptor, runtimeServices providerapi.HostServices) (hostServicesForRuntime, error) {
   148		baseServices := f.services
   149		if runtimeServices != nil {
   150			baseServices = layeredHostServices{base: f.services, overlay: runtimeServices}
   151		}
   152		collector := newHostServiceCollector(baseServices)
   153		success := false
   154		defer func() {
   155			if !success {
   156				_ = closeHostServiceClosers(ctx, collector.closers)
   157			}
   158		}()
   159		seen := map[string]struct{}{}
   160		for _, descriptor := range descriptors {
   161			for _, capability := range descriptor.PackageCapabilities {
   162				hostContribution, ok := capability.(providerapi.HostServiceContributionCapability)
   163				if !ok {
   164					continue
   165				}
   166				key := descriptor.PackageID + "\x00" + capability.CapabilityID()
   167				if _, ok := seen[key]; ok {
   168					continue
   169				}
   170				seen[key] = struct{}{}
   171				if err := hostContribution.ContributeHostServices(ctx, providerapi.HostServiceContributionRequest{
   172					SectionRequest: providerapi.SectionRequest{PackageID: descriptor.PackageID, ModuleID: descriptor.ModuleID},
   173					Values:         vals,
   174					Modules:        descriptors,
   175				}, collector); err != nil {
   176					return hostServicesForRuntime{}, fmt.Errorf("contribute host services for %s capability %s: %w", descriptor.PackageID, capability.CapabilityID(), err)
   177				}
   178			}
   179		}
   180		success = true
   181		return collector.servicesForRuntime(), nil
   182	}
   183	
   184	func (f *RuntimeFactory) configForModuleInstance(ctx context.Context, instance RuntimeModulePlan, descriptor providerapi.ModuleDescriptor, vals *values.Values) (json.RawMessage, error) {
   185		config, err := json.Marshal(instance.Config)
   186		if err != nil {
   187			return nil, fmt.Errorf("marshal config for %s.%s: %w", instance.ProviderID(), instance.Name, err)
   188		}
   189		for _, capability := range descriptor.PackageCapabilities {
   190			xgojaConfig, ok := capability.(providerapi.XGojaConfigSectionCapability)
   191			if !ok {
   192				continue
   193			}
   194			sectionRequest := providerapi.SectionRequest{PackageID: descriptor.PackageID, ModuleID: descriptor.ModuleID}
   195			section, err := xgojaConfig.XGojaConfigSection(sectionRequest, descriptor)
   196			if err != nil {
   197				return nil, fmt.Errorf("xgoja config section for %s.%s capability %s: %w", instance.ProviderID(), instance.Name, capability.CapabilityID(), err)
   198			}
   199			staticValues, err := providerutil.ParseXGojaConfigMap(section, instance.Config)
   200			if err != nil {
   201				return nil, fmt.Errorf("parse xgoja config for %s.%s capability %s: %w", instance.ProviderID(), instance.Name, capability.CapabilityID(), err)
   202			}
   203			overrideValues, err := xgojaConfig.XGojaConfigFromGlazed(ctx, providerapi.XGojaConfigRequest{
   204				SectionRequest: sectionRequest,
   205				Descriptor:     descriptor,
   206				ConfigSection:  section,
   207				StaticConfig:   staticValues,
   208				GlazedValues:   vals,
   209			})
   210			if err != nil {
   211				return nil, fmt.Errorf("map glazed config for %s.%s capability %s: %w", instance.ProviderID(), instance.Name, capability.CapabilityID(), err)
   212			}
   213			mergedValues, err := providerutil.MergeSectionValues(section, staticValues, overrideValues)
   214			if err != nil {
   215				return nil, fmt.Errorf("merge xgoja config for %s.%s capability %s: %w", instance.ProviderID(), instance.Name, capability.CapabilityID(), err)
   216			}
   217			config, err = providerutil.SectionValuesToRawJSON(mergedValues)
   218			if err != nil {
   219				return nil, fmt.Errorf("encode xgoja config for %s.%s capability %s: %w", instance.ProviderID(), instance.Name, capability.CapabilityID(), err)
   220			}
   221		}
   222		return config, nil
   223	}
```

### pkg/xgoja/app/command_providers.go:54-103
```go
    54			ShortHelpSections: []string{schema.DefaultSlug},
    55			MiddlewaresFunc:   middlewaresFunc,
    56		}
    57		if set.ParserConfig != nil {
    58			parserConfig = *set.ParserConfig
    59			if parserConfig.MiddlewaresFunc == nil {
    60				parserConfig.MiddlewaresFunc = middlewaresFunc
    61			}
    62		}
    63		if err := glazedcli.AddCommandsToRootCommand(root, commands, nil, glazedcli.WithParserConfig(parserConfig)); err != nil {
    64			root.AddCommand(commandErrorStub(commandProviderUse(instance, mount), "Attach custom xgoja command provider", err))
    65		}
    66	}
    67	
    68	func (h *Host) newCommandSet(instance CommandPlan, provider providerapi.CommandSetProvider, mount string) (*providerapi.CommandSet, error) {
    69		config, err := json.Marshal(instance.Config)
    70		if err != nil {
    71			return nil, fmt.Errorf("marshal command provider config %s: %w", instance.ID, err)
    72		}
    73		selected, err := h.selectedModulesForCommandProvider(instance)
    74		if err != nil {
    75			return nil, err
    76		}
    77		sourceRegistry := h.SourceRegistry.ScopedWithRuntimeAliases(instance.Sources, moduleAliases(selected))
    78		set, err := provider.NewCommandSet(providerapi.CommandSetContext{
    79			Context:         context.Background(),
    80			PackageID:       instance.ProviderID(),
    81			Name:            instance.Name,
    82			Mount:           mount,
    83			Config:          config,
    84			Host:            h.Services,
    85			Providers:       h.Providers,
    86			RuntimeFactory:  h.Factory,
    87			SelectedModules: selected,
    88			Sources:         sourceRegistry,
    89		})
    90		if err != nil {
    91			return nil, fmt.Errorf("create command set %s.%s: %w", instance.ProviderID(), instance.Name, err)
    92		}
    93		if set == nil {
    94			return nil, fmt.Errorf("command provider %s.%s returned nil command set", instance.ProviderID(), instance.Name)
    95		}
    96		return set, nil
    97	}
    98	
    99	func (h *Host) selectedModulesForCommandProvider(instance CommandPlan) ([]providerapi.ModuleDescriptor, error) {
   100		descriptors, err := h.Factory.selectedModuleDescriptors()
   101		if err != nil {
   102			return nil, err
   103		}
```

### pkg/xgoja/app/middlewares.go:17-53
```go
    17	// config file loading, it preserves the historical default chain: Cobra flags,
    18	// positional arguments, and field defaults only.
    19	func MiddlewaresFromSpec(runtimePlan *RuntimePlan) cli.CobraMiddlewaresFunc {
    20		envPrefix := EffectiveEnvPrefix(runtimePlan)
    21		hasConfig := runtimePlan != nil && runtimePlan.App.ConfigFile != nil && runtimePlan.App.ConfigFile.Enabled
    22	
    23		if envPrefix == "" && !hasConfig {
    24			return cli.CobraCommandDefaultMiddlewares
    25		}
    26	
    27		return func(parsedCommandSections *values.Values, cmd *cobra.Command, args []string) ([]cmdsources.Middleware, error) {
    28			// The returned slice is ordered from highest to lowest precedence because
    29			// Glazed middlewares call next before applying their own source values.
    30			// Effective precedence is: defaults < config < env < args < cobra flags.
    31			middlewares := []cmdsources.Middleware{
    32				cmdsources.FromCobra(cmd, fields.WithSource("cobra")),
    33				cmdsources.FromArgs(args, fields.WithSource("arguments")),
    34			}
    35	
    36			if envPrefix != "" {
    37				middlewares = append(middlewares, cmdsources.FromEnv(envPrefix, fields.WithSource("env")))
    38			}
    39	
    40			if hasConfig {
    41				middlewares = append(middlewares,
    42					cmdsources.FromConfigPlanBuilder(
    43						func(_ context.Context, _ *values.Values) (*glazedconfig.Plan, error) {
    44							return buildConfigPlan(runtimePlan.App.ConfigFile, runtimePlan.AppName(), parsedCommandSections)
    45						},
    46						cmdsources.WithParseOptions(fields.WithSource("config")),
    47					),
    48				)
    49			}
    50	
    51			middlewares = append(middlewares,
    52				cmdsources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
    53			)
```

### pkg/xgoja/app/root.go:275-314
```go
   275		moduleSections, selectedModules, err := factory.sectionsForRuntime("jsverbs")
   276		if err != nil {
   277			return nil, err
   278		}
   279		commands := []cmds.Command{}
   280		if sourceRegistry == nil {
   281			return nil, fmt.Errorf("source registry is required")
   282		}
   283		jsverbSources := sourceRegistry.JSVerbs()
   284		for _, source := range jsverbSources.ListJSVerbSources() {
   285			registry, err := sourceRegistry.scanJSVerbSource(source.ID, sourceGraphRuntimeAliases(moduleAliases(selectedModules)))
   286			if err != nil {
   287				return nil, err
   288			}
   289			if registry == nil {
   290				continue
   291			}
   292			for _, verb := range registry.Verbs() {
   293				verb := verb
   294				registry := registry
   295				cmd, err := registry.CommandForVerbWithInvoker(verb, func(ctx context.Context, _ *jsverbs.Registry, verb *jsverbs.VerbSpec, parsedValues *values.Values) (interface{}, error) {
   296					rt, err := factory.NewRuntimeFromSections(ctx, parsedValues, require.WithLoader(registry.RequireLoader()))
   297					if err != nil {
   298						return nil, err
   299					}
   300					defer func() { _ = rt.Close(context.Background()) }()
   301					if len(selectedModules) > 0 {
   302						if err := initRuntimeFromSections(ctx, parsedValues, rt, selectedModules); err != nil {
   303							return nil, err
   304						}
   305					}
   306					return registry.InvokeInRuntime(ctx, rt, verb, parsedValues)
   307				})
   308				if err != nil {
   309					return nil, err
   310				}
   311				if len(moduleSections) > 0 {
   312					if err := addSectionsToCommandDescription(cmd.Description(), moduleSections, "jsverbs runtime"); err != nil {
   313						return nil, err
   314					}
```

### examples/xgoja/21-generated-host-auth/cmd/host/main.go:18-83
```go
    18			os.Exit(1)
    19		}
    20	
    21		var configureErr error
    22		bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{
    23			ConfigureServices: func(services *app.HostServices) {
    24				configureErr = services.SetHostService(hostauth.ServiceFactoryKey, hostauth.NewServiceFactory(hostauth.BuilderOptions{
    25					Config: authConfig,
    26				}))
    27			},
    28		})
    29		if err != nil {
    30			fmt.Fprintln(os.Stderr, err)
    31			os.Exit(1)
    32		}
    33		if configureErr != nil {
    34			fmt.Fprintln(os.Stderr, configureErr)
    35			os.Exit(1)
    36		}
    37	
    38		root := &cobra.Command{
    39			Use:          "generated-host-auth",
    40			Short:        "Serve the generated-host auth xgoja example",
    41			SilenceUsage: true,
    42		}
    43		bundle.AttachDefaultCommands(root)
    44		if err := root.Execute(); err != nil {
    45			fmt.Fprintln(os.Stderr, err)
    46			os.Exit(1)
    47		}
    48	}
    49	
    50	func authConfigFromEnv() (hostauth.Config, error) {
    51		cfg := hostauth.Config{
    52			Mode: hostauth.ModeDev,
    53			Session: hostauth.SessionConfig{
    54				Cookie: hostauth.CookieConfig{
    55					AllowInsecureHTTP: true,
    56				},
    57			},
    58			Stores: hostauth.StoresConfig{
    59				Default: hostauth.StoreConfig{
    60					Driver: string(hostauth.StoreDriverMemory),
    61				},
    62			},
    63		}
    64	
    65		switch strings.ToLower(strings.TrimSpace(os.Getenv("XGOJA_AUTH_STORE"))) {
    66		case "", "memory":
    67			return cfg, nil
    68		case "sqlite":
    69			dsn := strings.TrimSpace(os.Getenv("XGOJA_AUTH_SQLITE_DSN"))
    70			if dsn == "" {
    71				return hostauth.Config{}, fmt.Errorf("XGOJA_AUTH_SQLITE_DSN is required when XGOJA_AUTH_STORE=sqlite")
    72			}
    73			applySchema := true
    74			cfg.Stores.Default = hostauth.StoreConfig{
    75				Driver:      string(hostauth.StoreDriverSQLite),
    76				DSN:         dsn,
    77				ApplySchema: &applySchema,
    78			}
    79			return cfg, nil
    80		default:
    81			return hostauth.Config{}, fmt.Errorf("unsupported XGOJA_AUTH_STORE %q (want memory or sqlite)", os.Getenv("XGOJA_AUTH_STORE"))
    82		}
    83	}
```

### examples/xgoja/19-express-keycloak-auth-host/cmd/host/main.go:31-43
```go
    31	func main() {
    32		listen := flag.String("listen", "127.0.0.1:8790", "listen address")
    33		script := flag.String("script", "examples/xgoja/19-express-keycloak-auth-host/scripts/server.js", "JavaScript route script")
    34		issuer := flag.String("issuer", envOr("KEYCLOAK_ISSUER", "http://127.0.0.1:18080/realms/goja-demo"), "OIDC issuer URL")
    35		clientID := flag.String("client-id", envOr("KEYCLOAK_CLIENT_ID", "goja-app"), "OIDC client ID")
    36		clientSecret := flag.String("client-secret", os.Getenv("KEYCLOAK_CLIENT_SECRET"), "OIDC client secret, if configured")
    37		sessionDBDSN := flag.String("session-db-dsn", os.Getenv("SESSION_DB_DSN"), "Postgres DSN for persistent app sessions; empty uses in-memory sessions")
    38		auditDBDSN := flag.String("audit-db-dsn", os.Getenv("AUDIT_DB_DSN"), "Postgres DSN for persistent audit records; empty logs audit records")
    39		appDBDSN := flag.String("app-db-dsn", os.Getenv("APPAUTH_DB_DSN"), "Postgres DSN for persistent appauth users/resources; empty uses in-memory appauth")
    40		capabilityDBDSN := flag.String("capability-db-dsn", os.Getenv("CAPABILITY_DB_DSN"), "Postgres DSN for persistent capability tokens; empty uses in-memory capabilities")
    41		flag.Parse()
    42		if err := run(context.Background(), config{Listen: *listen, Script: *script, Issuer: *issuer, ClientID: *clientID, ClientSecret: *clientSecret, SessionDBDSN: *sessionDBDSN, AuditDBDSN: *auditDBDSN, AppDBDSN: *appDBDSN, CapabilityDBDSN: *capabilityDBDSN}); err != nil {
    43			log.Fatal(err)
```

## Geppetto / Pinocchio line anchors

### geppetto/pkg/sections/profile_sections.go:8-66
```go
     8	)
     9	
    10	type ProfileSettings struct {
    11		Profile           string   `glazed:"profile"`
    12		ProfileRegistries []string `glazed:"profile-registries"`
    13	}
    14	
    15	type ProfileSettingsSectionOption func(*profileSettingsSectionOptions)
    16	
    17	type profileSettingsSectionOptions struct {
    18		profileDefault           string
    19		profileRegistriesDefault []string
    20	}
    21	
    22	func WithProfileDefault(profile string) ProfileSettingsSectionOption {
    23		return func(o *profileSettingsSectionOptions) {
    24			o.profileDefault = strings.TrimSpace(profile)
    25		}
    26	}
    27	
    28	func WithProfileRegistriesDefault(entries ...string) ProfileSettingsSectionOption {
    29		return func(o *profileSettingsSectionOptions) {
    30			o.profileRegistriesDefault = o.profileRegistriesDefault[:0]
    31			for _, entry := range entries {
    32				if trimmed := strings.TrimSpace(entry); trimmed != "" {
    33					o.profileRegistriesDefault = append(o.profileRegistriesDefault, trimmed)
    34				}
    35			}
    36		}
    37	}
    38	
    39	const ProfileSettingsSectionSlug = "profile-settings"
    40	
    41	func NewProfileSettingsSection(opts ...ProfileSettingsSectionOption) (schema.Section, error) {
    42		var sectionOptions profileSettingsSectionOptions
    43		for _, opt := range opts {
    44			opt(&sectionOptions)
    45		}
    46	
    47		profileOptions := []fields.Option{
    48			fields.WithHelp("Load the profile"),
    49		}
    50		if sectionOptions.profileDefault != "" {
    51			profileOptions = append(profileOptions, fields.WithDefault(sectionOptions.profileDefault))
    52		}
    53	
    54		profileRegistriesOptions := []fields.Option{
    55			fields.WithHelp("Comma-separated profile registry sources (yaml/sqlite/sqlite-dsn)"),
    56		}
    57		if len(sectionOptions.profileRegistriesDefault) > 0 {
    58			profileRegistriesOptions = append(profileRegistriesOptions, fields.WithDefault(append([]string(nil), sectionOptions.profileRegistriesDefault...)))
    59		}
    60	
    61		return schema.NewSection(
    62			ProfileSettingsSectionSlug,
    63			"Profile settings",
    64			schema.WithFields(
    65				fields.New("profile", fields.TypeString, profileOptions...),
    66				fields.New("profile-registries", fields.TypeStringList, profileRegistriesOptions...),
```

### geppetto/cmd/examples/runner-glazed-registry-flags/main.go:39-78
```go
    39		Prompt string `glazed:"prompt"`
    40	}
    41	
    42	func newRegistryFlagsCommand() (*registryFlagsCommand, error) {
    43		profileSettingsSection, err := geppettosections.NewProfileSettingsSection(
    44			geppettosections.WithProfileDefault("openai-fast"),
    45			geppettosections.WithProfileRegistriesDefault(runnerexample.ExampleEngineProfileRegistryPath()),
    46		)
    47		if err != nil {
    48			return nil, errors.Wrap(err, "create profile settings section")
    49		}
    50	
    51		description := cmds.NewCommandDescription(
    52			"run",
    53			cmds.WithShort("Run inference via pkg/inference/runner with only registry selection exposed through Glazed"),
    54			cmds.WithArguments(
    55				fields.New("prompt", fields.TypeString, fields.WithHelp("Prompt to run"), fields.WithRequired(true)),
    56			),
    57			cmds.WithSections(profileSettingsSection),
    58		)
    59	
    60		return &registryFlagsCommand{CommandDescription: description}, nil
    61	}
    62	
    63	func (c *registryFlagsCommand) RunIntoWriter(ctx context.Context, parsedValues *values.Values, w io.Writer) error {
    64		s := &registryFlagsSettings{}
    65		if err := parsedValues.DecodeSectionInto(values.DefaultSlug, s); err != nil {
    66			return err
    67		}
    68		profileSettings := &geppettosections.ProfileSettings{}
    69		if err := parsedValues.DecodeSectionInto(geppettosections.ProfileSettingsSectionSlug, profileSettings); err != nil {
    70			return err
    71		}
    72	
    73		stepSettings, closeRegistry, err := runnerexample.ResolveInferenceSettingsFromRegistry(ctx, profileSettings.ProfileRegistries, profileSettings.Profile)
    74		if err != nil {
    75			return err
    76		}
    77		defer func() {
    78			if closeRegistry != nil {
```

### geppetto/cmd/examples/internal/bootstrap/middlewares.go:18-70
```go
    18	func AppBootstrapConfig() geppettobootstrap.AppBootstrapConfig {
    19		cfg := geppettobootstrap.AppBootstrapConfig{
    20			AppName:          "pinocchio",
    21			EnvPrefix:        "PINOCCHIO",
    22			ConfigFileMapper: geppettobootstrap.DefaultConfigFileMapper,
    23			NewProfileSection: func() (schema.Section, error) {
    24				return geppettosections.NewProfileSettingsSection()
    25			},
    26			BuildBaseSections: func() ([]schema.Section, error) {
    27				return geppettosections.CreateGeppettoSections()
    28			},
    29		}
    30		cfg.ConfigPlanBuilder = func(parsed *values.Values) (*glazedconfig.Plan, error) {
    31			explicit := ""
    32			if parsed != nil {
    33				commandSettings := &cli.CommandSettings{}
    34				if err := parsed.DecodeSectionInto(cli.CommandSettingsSlug, commandSettings); err == nil {
    35					explicit = strings.TrimSpace(commandSettings.ConfigFile)
    36				}
    37			}
    38	
    39			return glazedconfig.NewPlan(
    40				glazedconfig.WithLayerOrder(
    41					glazedconfig.LayerSystem,
    42					glazedconfig.LayerUser,
    43					glazedconfig.LayerExplicit,
    44				),
    45				glazedconfig.WithDedupePaths(),
    46			).Add(
    47				glazedconfig.SystemAppConfig(cfg.AppName).Named("system-app-config").Kind("app-config"),
    48				glazedconfig.HomeAppConfig(cfg.AppName).Named("home-app-config").Kind("app-config"),
    49				glazedconfig.XDGAppConfig(cfg.AppName).Named("xdg-app-config").Kind("app-config"),
    50				glazedconfig.ExplicitFile(explicit).Named("explicit-config-file").Kind("explicit-file"),
    51			), nil
    52		}
    53		return cfg
    54	}
    55	
    56	func GetCobraCommandMiddlewares(parsed *values.Values, cmd *cobra.Command, args []string) ([]sources.Middleware, error) {
    57		cfg := AppBootstrapConfig()
    58		if err := cfg.Validate(); err != nil {
    59			return nil, err
    60		}
    61	
    62		return []sources.Middleware{
    63			sources.FromCobra(cmd, fields.WithSource("cobra")),
    64			sources.FromArgs(args, fields.WithSource("arguments")),
    65			sources.FromEnv(cfg.EnvPrefix, fields.WithSource("env")),
    66			sources.FromConfigPlanBuilder(func(_ctx context.Context, parsedValues *values.Values) (*glazedconfig.Plan, error) {
    67				return cfg.ConfigPlanBuilder(parsedValues)
    68			},
    69				sources.WithConfigFileMapper(cfg.ConfigFileMapper),
    70				sources.WithParseOptions(fields.WithSource("config")),
```

### geppetto/pkg/cli/bootstrap/engine_settings.go:26-55
```go
    26	func ResolveBaseInferenceSettings(cfg AppBootstrapConfig, parsed *values.Values) (*aisettings.InferenceSettings, []string, error) {
    27		if err := cfg.Validate(); err != nil {
    28			return nil, nil, err
    29		}
    30	
    31		sections_, err := cfg.BuildBaseSections()
    32		if err != nil {
    33			return nil, nil, errors.Wrap(err, "create hidden base sections")
    34		}
    35		schema_ := schema.NewSchema(schema.WithSections(sections_...))
    36		parsedValues := values.New()
    37		configMiddleware, configFiles, err := resolveConfigMiddleware(cfg, parsed)
    38		if err != nil {
    39			return nil, nil, err
    40		}
    41		if err := sources.Execute(
    42			schema_,
    43			parsedValues,
    44			sources.FromEnv(cfg.normalizedEnvPrefix(), fields.WithSource("env")),
    45			configMiddleware,
    46			sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
    47		); err != nil {
    48			return nil, configFiles.Paths, errors.Wrap(err, "resolve hidden base inference settings")
    49		}
    50		stepSettings, err := aisettings.NewInferenceSettingsFromParsedValues(parsedValues)
    51		if err != nil {
    52			return nil, configFiles.Paths, errors.Wrap(err, "build inference settings from hidden parsed values")
    53		}
    54		return stepSettings, configFiles.Paths, nil
    55	}
```

### geppetto/pkg/cli/bootstrap/profile_runtime.go:23-52
```go
    23		cfg AppBootstrapConfig,
    24		parsed *values.Values,
    25	) (*ResolvedCLIProfileRuntime, error) {
    26		if err := cfg.Validate(); err != nil {
    27			return nil, err
    28		}
    29	
    30		profileSection, err := cfg.NewProfileSection()
    31		if err != nil {
    32			return nil, errors.Wrap(err, "create profile settings section")
    33		}
    34	
    35		schema_ := schema.NewSchema(schema.WithSections(profileSection))
    36		resolvedValues := values.New()
    37		configMiddleware, configFiles, err := resolveConfigMiddleware(cfg, parsed)
    38		if err != nil {
    39			return nil, err
    40		}
    41		if err := sources.Execute(
    42			schema_,
    43			resolvedValues,
    44			sources.FromEnv(cfg.normalizedEnvPrefix(), fields.WithSource("env")),
    45			configMiddleware,
    46			sources.FromDefaults(fields.WithSource(fields.SourceDefaults)),
    47		); err != nil {
    48			return nil, errors.Wrap(err, "resolve profile settings from config/env/defaults")
    49		}
    50		if parsed != nil {
    51			if err := resolvedValues.Merge(parsed); err != nil {
    52				return nil, errors.Wrap(err, "merge explicit profile settings")
```
