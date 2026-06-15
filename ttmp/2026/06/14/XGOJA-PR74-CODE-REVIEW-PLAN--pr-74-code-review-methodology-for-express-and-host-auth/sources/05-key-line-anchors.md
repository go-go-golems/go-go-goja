---
Title: Key Line Anchors
Ticket: XGOJA-PR74-CODE-REVIEW-PLAN
Status: active
Topics:
    - review
    - goja
    - xgoja
    - auth
    - security
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Captured line anchors for review-critical symbols and functions."
LastUpdated: 2026-06-14T20:55:00-04:00
WhatFor: "Evidence captured while planning the PR 74 code review."
WhenToUse: "Use as supporting evidence for the PR 74 review methodology guide."
---

# Key line anchors for PR 74 review planning

## pkg/gojahttp/auth_plan.go
32:	ErrUnauthenticated = errors.New("unauthenticated")
35:	ErrCSRF            = errors.New("csrf invalid")
40:type RoutePlan struct {
68:type ResourceSpec struct {
82:// AuditSpec describes the host-owned audit event emitted for a planned route.
87:// Actor is the minimal host-owned authenticated principal exposed to planned
97:// route handlers after resolution and authorization.
98:type ResourceRef struct {
106:type AuthOptions struct {
118:type ResourceResolver interface {
134:type ResourceRequest struct {
182:func ValidateRoutePlan(plan RoutePlan) (RoutePlan, error) {
205:		return RoutePlan{}, fmt.Errorf("planned route %s %s must declare .public() or .auth(...) before .handle(...)", plan.Method, plan.Pattern)

## pkg/gojahttp/planned_dispatch.go
68:		if h.auth.Authenticator == nil {
69:			return envelope, http.StatusInternalServerError, fmt.Errorf("planned route %s %s requires authenticator", plan.Method, plan.Pattern)
72:		actor, err = h.auth.Authenticator.Authenticate(ctx, httpReq, req.Session, plan.Security)
77:			return envelope, http.StatusUnauthorized, ErrUnauthenticated
85:		if h.auth.CSRF == nil {
86:			return envelope, http.StatusInternalServerError, fmt.Errorf("planned route %s %s requires csrf protector", plan.Method, plan.Pattern)
88:		if err := h.auth.CSRF.VerifyCSRF(ctx, CSRFRequest{HTTPRequest: httpReq, Request: req, Session: req.Session, Actor: actor, Plan: *plan}); err != nil {
94:		if h.auth.Resources == nil {
109:			resource, err := h.auth.Resources.ResolveResource(ctx, ResourceRequest{HTTPRequest: httpReq, Request: req, Actor: actor, Spec: spec, ID: id, TenantID: tenantID})
127:		if h.auth.Authorizer == nil {
128:			return envelope, http.StatusInternalServerError, fmt.Errorf("planned route %s %s requires authorizer", plan.Method, plan.Pattern)
131:		decision, err := h.auth.Authorizer.Authorize(ctx, AuthorizationRequest{HTTPRequest: httpReq, Request: req, Actor: actor, Action: plan.Action, Resource: resource, Resources: envelope.Resources})
160:	if h.auth.Audit == nil || plan == nil || plan.Audit.Event == "" {
174:	_ = h.auth.Audit.RecordAudit(ctx, AuditEvent{
193:	case errors.Is(err, ErrUnauthenticated):
194:		return http.StatusUnauthorized

## pkg/gojahttp/host.go
14:type HostOptions struct {
19:	RejectRawRoutes bool
42:	auth            AuthOptions
47:func NewHost(opts HostOptions) *Host {
48:	return &Host{registry: NewRegistry(), dev: opts.Dev, renderer: opts.Renderer, sessions: NewSessionManager(opts.Sessions), auth: opts.Auth, rejectRawRoutes: opts.RejectRawRoutes}
213:		message = fmt.Sprintf("raw route %s %s rejected: register a planned route with .public() or auth", route.Method, route.Pattern)

## modules/express/auth_builders.go
14:	authSpecs     sync.Map // map[*goja.Object]*gojahttp.SecuritySpec
23:	s.authSpecs.Store(obj, spec)
39:func (s *builderStore) authSpec(vm *goja.Runtime, value goja.Value) (gojahttp.SecuritySpec, error) {
41:		return gojahttp.SecuritySpec{}, fmt.Errorf(".auth(...) expects value returned by express.user()")
44:	raw, ok := s.authSpecs.Load(obj)
46:		return gojahttp.SecuritySpec{}, fmt.Errorf(".auth(...) expects value returned by express.user(); got %s", valueString(value))
50:		return gojahttp.SecuritySpec{}, fmt.Errorf("internal auth spec has invalid type")
137:	_ = obj.Set("auth", func(value goja.Value) (goja.Value, error) {
138:		spec, err := b.store.authSpec(b.vm, value)
189:	_ = obj.Set("csrf", func(call goja.FunctionCall) goja.Value {
200:	_ = obj.Set("audit", func(event string) (goja.Value, error) {
203:			return nil, fmt.Errorf(".audit(event) requires a non-empty event")

## modules/express/express.go
192:				panic(vm.NewTypeError("app.%s(pattern, handler) was removed; use app.%s(pattern).public().handle(handler) or app.%s(pattern).auth(...).allow(...).handle(handler)", method, method, method))

## pkg/xgoja/hostauth/config.go
1:package hostauth
8:// Mode selects the generated-host authentication infrastructure shape.
17:// StoreDriver selects the persistence backend for a host auth store.
26:// Config is the generated-host auth infrastructure configuration. It is host
27:// config, not JavaScript route config and not an authorization policy DSL.
28:type Config struct {
35:type SessionConfig struct {
42:// sessionauth.New's secure default cookie name.
50:// StoresConfig configures the persistent stores used by host-owned auth
55:	Audit      StoreConfig `yaml:"audit" json:"audit"`
56:	AppAuth    StoreConfig `yaml:"appauth" json:"appauth"`
71:type ResolvedConfig struct {
77:type ResolvedSessionConfig struct {
83:type ResolvedCookieConfig struct {
90:type ResolvedStoresConfig struct {
97:type ResolvedStoreConfig struct {

## pkg/xgoja/hostauth/resolve.go
1:package hostauth
11:var ErrOIDCNotImplemented = errors.New("hostauth: auth.mode=oidc is not implemented yet")
17:type ConfigError struct {
39:func ResolveConfig(cfg Config, opts ResolveOptions) (ResolvedConfig, error) {
42:		return ResolvedConfig{}, configError("auth.mode", err)
45:		return ResolvedConfig{}, configError("auth.mode", ErrOIDCNotImplemented)
70:		return "", fmt.Errorf("unsupported auth mode %q", mode)
77:		return ResolvedSessionConfig{}, configError("auth.session.cookie.same-site", err)
79:	idleTimeout, err := parseOptionalDuration("auth.session.idle-timeout", cfg.IdleTimeout)
83:	absoluteTimeout, err := parseOptionalDuration("auth.session.absolute-timeout", cfg.AbsoluteTimeout)
92:		return ResolvedSessionConfig{}, configError("auth.session.cookie.path", fmt.Errorf("must start with /"))
111:		return http.SameSiteStrictMode, nil
146:	audit, err := resolveStoreConfig("audit", cfg.Audit, defaults, lookupEnv)
150:	appauth, err := resolveStoreConfig("appauth", cfg.AppAuth, defaults, lookupEnv)
158:	return ResolvedStoresConfig{Session: session, Audit: audit, AppAuth: appauth, Capability: capability}, nil
162:	path := "auth.stores." + name

## pkg/xgoja/hostauth/stores.go
1:package hostauth
12:	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
13:	appauthsql "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth/sqlstore"
14:	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
15:	auditsql "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit/sqlstore"
16:	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability"
17:	capabilitysql "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability/sqlstore"
18:	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
19:	sessionauthsql "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth/sqlstore"
23:type StoreBundle struct {
24:	Session    sessionauth.Store
25:	Audit      audit.Store
40:// BuildStores creates all host auth stores described by cfg. SQL DB handles are
42:func BuildStores(ctx context.Context, cfg ResolvedStoresConfig) (*StoreBundle, error) {
67:	auditStore, err := b.buildAuditStore(ctx, cfg.Audit)
79:	return &StoreBundle{Session: sessionStore, Audit: auditStore, AppAuth: appAuthStores, Capability: capabilityStore, Closers: append([]func(context.Context) error(nil), b.closers...)}, nil
82:func (b *storeBuilder) buildSessionStore(ctx context.Context, cfg ResolvedStoreConfig) (sessionauth.Store, error) {
85:		return sessionauth.NewMemoryStore(), nil
91:		store, err := sessionauthsql.New(sessionauthsql.Config{DB: db, Dialect: sessionDialect(cfg.Driver)})
106:func (b *storeBuilder) buildAuditStore(ctx context.Context, cfg ResolvedStoreConfig) (audit.Store, error) {
109:		return &audit.MemoryStore{}, nil
113:			return nil, fmt.Errorf("build audit store: %w", err)
115:		store, err := auditsql.New(auditsql.Config{DB: db, Dialect: auditDialect(cfg.Driver)})
117:			return nil, fmt.Errorf("build audit store: %w", err)
126:		return nil, fmt.Errorf("build audit store: unsupported driver %q", cfg.Driver)
133:		store := appauth.NewMemoryStore()
138:			return AppAuthStores{}, fmt.Errorf("build appauth store: %w", err)
140:		store, err := appauthsql.New(appauthsql.Config{DB: db, Dialect: appAuthDialect(cfg.Driver)})
142:			return AppAuthStores{}, fmt.Errorf("build appauth store: %w", err)
151:		return AppAuthStores{}, fmt.Errorf("build appauth store: unsupported driver %q", cfg.Driver)
210:func sessionDialect(driver StoreDriver) sessionauthsql.Dialect {
212:		return sessionauthsql.DialectSQLite
214:	return sessionauthsql.DialectPostgres
217:func auditDialect(driver StoreDriver) auditsql.Dialect {
219:		return auditsql.DialectSQLite
221:	return auditsql.DialectPostgres
224:func appAuthDialect(driver StoreDriver) appauthsql.Dialect {
226:		return appauthsql.DialectSQLite
228:	return appauthsql.DialectPostgres

## pkg/xgoja/hostauth/builder.go
1:package hostauth
11:	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
12:	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
13:	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
16:// BuilderOptions configures a generated-host auth service factory.
17:type BuilderOptions struct {
20:	ActorLoader sessionauth.ActorLoader
24:// Builder is the default hostauth ServiceFactory implementation.
25:type Builder struct {
32:	errServiceFactoryNil = errors.New("hostauth service factory is nil")
35:// NewServiceFactory returns a lazy generated-host auth service factory. The
39:func NewServiceFactory(opts BuilderOptions) *Builder {
43:// BuildHostAuthServices builds concrete auth services for one command/runtime
78:	auditSink := audit.Sink{Store: stores.Audit}
79:	authOptions := BuildAuthOptions(sessionManager, stores, auditSink)
82:		AuthOptions:    authOptions,
85:		AuditSink:      auditSink,
95:// BuildSessionManager maps resolved generated-host config into sessionauth.
96:func BuildSessionManager(cfg ResolvedSessionConfig, store sessionauth.Store, actorLoader sessionauth.ActorLoader, now func() time.Time) (*sessionauth.Manager, error) {
97:	return sessionauth.New(sessionauth.Config{
110:// BuildAuthOptions wires a session manager and built auth stores into
111:// gojahttp's host-owned auth interfaces.
112:func BuildAuthOptions(sessionManager *sessionauth.Manager, stores *StoreBundle, auditSink gojahttp.AuditSink) gojahttp.AuthOptions {
118:	if auditSink != nil {
119:		options.Audit = auditSink
123:			options.Resources = appauth.Resolver{Store: stores.AppAuth.Resources}
126:			options.Authorizer = appauth.Authorizer{Memberships: stores.AppAuth.Memberships}
132:func errNilBuilder() error { return &ConfigError{Path: "auth", Err: errServiceFactoryNil} }

## pkg/xgoja/providers/http/serve.go
27:	"github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth"
33:type serveHotReloadSettings struct {
65:	if _, _, err := hostauth.LookupServiceFactory(ctx.Host); err != nil {
124:	authServices, hasAuthFactory, err := buildServeAuthServices(ctx, commandCtx, parsedValues)
129:		defer func() { _ = authServices.Close(context.Background()) }()
136:			return nil, fmt.Errorf("http serve with hostauth requires runtime factory with per-runtime host services")
148:		runtimeServices, err := serveRuntimeServices(gojahttp.NewHost(hostOptionsWithAuth(httpSettings, authServices)), authServices, true, includeGeneratedHost)
186:	authServices, hasAuthFactory, err := buildServeAuthServices(ctx, commandCtx, parsedValues)
191:		defer func() { _ = authServices.Close(context.Background()) }()
215:		HostOptions: hostOptionsWithAuth(httpSettings, authServices),
222:			services, err := serveRuntimeServices(candidate.Host, authServices, false, true)
398:func buildServeAuthServices(ctx context.Context, commandCtx providerapi.CommandSetContext, parsedValues *values.Values) (*hostauth.Services, bool, error) {
399:	factory, ok, err := hostauth.LookupServiceFactory(commandCtx.Host)
411:		return nil, false, fmt.Errorf("hostauth service factory returned nil services")
416:func hostOptionsWithAuth(cfg settings, authServices *hostauth.Services) gojahttp.HostOptions {
418:	if authServices != nil {
419:		opts.Auth = authServices.AuthOptions
424:func serveRuntimeServices(host *gojahttp.Host, authServices *hostauth.Services, ownsListen bool, includeHost bool) (app.HostServices, error) {
431:	if authServices != nil {
432:		if err := services.SetHostService(hostauth.ServicesKey, authServices); err != nil {

## pkg/gojahttp/auth/sessionauth/sessionauth.go
1:// Package sessionauth provides reusable session-cookie authentication and CSRF
3:package sessionauth
38:type Session struct {
77:type Config struct {
91:type Manager struct {
107:		return nil, fmt.Errorf("sessionauth: store is required")
137:// AuthOptions returns the gojahttp auth fields implemented by this manager.
149:	csrf, err := RandomToken()
153:	session := Session{ID: id, UserID: userID, CSRFToken: csrf, CreatedAt: now, LastSeenAt: now, IdleExpiresAt: now.Add(m.idleTimeout), AbsoluteExpiresAt: now.Add(m.absoluteTimeout)}
175:		return authError(err)
203:		return nil, authError(err)
206:		return nil, authError(err)
226:		return authError(err)
286:func authError(err error) error {
289:		return gojahttp.ErrUnauthenticated
314:type SessionOption func(*Session)

## pkg/gojahttp/auth/appauth/appauth.go
1:// Package appauth provides small app-owned user, tenant, membership,
2:// resource, and authorization helpers for gojahttp planned routes. It is not a
5:package appauth
25:type User struct {
43:type Membership struct {
51:type Resource struct {
61:type UserStore interface {
68:type MembershipStore interface {
75:type ResourceStore interface {
84:		return nil, fmt.Errorf("appauth: resource store is required")

## pkg/gojahttp/auth/keycloakauth/keycloakauth.go
1:// Package keycloakauth provides an opinionated OIDC browser-login adapter
3:// and creates an opaque application session for planned route authentication.
4:package keycloakauth
16:	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
17:	"golang.org/x/oauth2"
21:type Config struct {
30:	SessionManager   *sessionauth.Manager
48:type UserSession struct {
57:type UserNormalizer interface {
62:type UserNormalizerFunc func(ctx context.Context, claims OIDCClaims) (UserSession, error)
85:	oauth2Config   oauth2.Config
87:	sessionManager *sessionauth.Manager
97:		return nil, fmt.Errorf("keycloakauth: issuer URL is required")
100:		return nil, fmt.Errorf("keycloakauth: client ID is required")
103:		return nil, fmt.Errorf("keycloakauth: redirect URL is required")
106:		return nil, fmt.Errorf("keycloakauth: session manager is required")
109:		return nil, fmt.Errorf("keycloakauth: user normalizer is required")
116:		return nil, fmt.Errorf("keycloakauth: discover provider: %w", err)
119:	oauth2Config := oauth2.Config{
127:		oauth2Config:   oauth2Config,
146:	state, err := sessionauth.RandomToken()
151:	nonce, err := sessionauth.RandomToken()
156:	verifier, err := sessionauth.RandomToken()
169:	url := h.oauth2Config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), oauth2.SetAuthURLParam("nonce", nonce))
179:		http.Error(w, "oidc error: "+errText, http.StatusUnauthorized)
190:		http.Error(w, "invalid oidc state", http.StatusUnauthorized)
193:	token, err := h.oauth2Config.Exchange(r.Context(), code, oauth2.VerifierOption(tx.PKCEVerifier))
195:		http.Error(w, "oidc token exchange failed", http.StatusUnauthorized)
200:		http.Error(w, "oidc response missing id_token", http.StatusUnauthorized)
205:		http.Error(w, "oidc id_token verification failed", http.StatusUnauthorized)
209:		http.Error(w, "oidc nonce mismatch", http.StatusUnauthorized)
214:		http.Error(w, "oidc claims invalid", http.StatusUnauthorized)
219:		http.Error(w, "user normalization failed", http.StatusUnauthorized)
223:		http.Error(w, "user normalization returned empty user id", http.StatusUnauthorized)
227:		sessionauth.WithEmail(userSession.Email, userSession.EmailVerified),
228:		sessionauth.WithTenantIDs(userSession.TenantIDs...),
229:		sessionauth.WithClaims(userSession.Claims),

