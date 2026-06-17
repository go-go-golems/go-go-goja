---
Title: Current Auth Surface Evidence
Ticket: XGOJA-PROGRAMMATIC-AUTH-DESIGN
Status: active
Topics:
  - auth
  - api
  - evidence
DocType: reference
Intent: short-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Line-anchored evidence for current PR 74 auth surfaces used by the programmatic auth design.
LastUpdated: 2026-06-15T21:40:00-04:00
WhatFor: Ground the token/device login implementation guide in current code contracts.
WhenToUse: Use while implementing tokenauth/deviceauth to see where integrations land.
---

# Current Auth Surface Evidence

## pkg/gojahttp/auth_plan.go:100-180
```go
   100		Type     string         `json:"type"`
   101		ID       string         `json:"id"`
   102		TenantID string         `json:"tenantId,omitempty"`
   103		Claims   map[string]any `json:"claims,omitempty"`
   104	}
   105	
   106	type AuthOptions struct {
   107		Authenticator Authenticator
   108		Resources     ResourceResolver
   109		Authorizer    Authorizer
   110		CSRF          CSRFProtector
   111		Audit         AuditSink
   112	}
   113	
   114	type Authenticator interface {
   115		Authenticate(ctx context.Context, req *http.Request, session *SessionDTO, spec SecuritySpec) (*Actor, error)
   116	}
   117	
   118	type ResourceResolver interface {
   119		ResolveResource(ctx context.Context, req ResourceRequest) (*ResourceRef, error)
   120	}
   121	
   122	type Authorizer interface {
   123		Authorize(ctx context.Context, req AuthorizationRequest) (AuthorizationDecision, error)
   124	}
   125	
   126	type CSRFProtector interface {
   127		VerifyCSRF(ctx context.Context, req CSRFRequest) error
   128	}
   129	
   130	type AuditSink interface {
   131		RecordAudit(ctx context.Context, event AuditEvent) error
   132	}
   133	
   134	type ResourceRequest struct {
   135		HTTPRequest *http.Request
   136		Request     *RequestDTO
   137		Actor       *Actor
   138		Spec        ResourceSpec
   139		ID          string
   140		TenantID    string
   141	}
   142	
   143	type AuthorizationRequest struct {
   144		HTTPRequest *http.Request
   145		Request     *RequestDTO
   146		Actor       *Actor
   147		Action      string
   148		Resource    *ResourceRef
   149		Resources   map[string]*ResourceRef
   150	}
   151	
   152	type CSRFRequest struct {
   153		HTTPRequest *http.Request
   154		Request     *RequestDTO
   155		Session     *SessionDTO
   156		Actor       *Actor
   157		Plan        RoutePlan
   158	}
   159	
   160	type AuditEvent struct {
   161		HTTPRequest *http.Request           `json:"-"`
   162		Request     *RequestDTO             `json:"-"`
   163		Event       string                  `json:"event"`
   164		Outcome     string                  `json:"outcome"`
   165		Reason      string                  `json:"reason,omitempty"`
   166		StatusCode  int                     `json:"statusCode,omitempty"`
   167		RouteName   string                  `json:"routeName,omitempty"`
   168		Method      string                  `json:"method"`
   169		Pattern     string                  `json:"pattern"`
   170		Action      string                  `json:"action,omitempty"`
   171		Actor       *Actor                  `json:"actor,omitempty"`
   172		Resource    *ResourceRef            `json:"resource,omitempty"`
   173		Resources   map[string]*ResourceRef `json:"resources,omitempty"`
   174		Attributes  map[string]any          `json:"attributes,omitempty"`
   175	}
   176	
   177	type AuthorizationDecision struct {
   178		Allowed bool
   179		Reason  string
   180	}
```

## pkg/gojahttp/planned_dispatch.go:45-145
```go
    45			h.recordAudit(r.Context(), r, req, route.Plan, envelope, "failed", http.StatusInternalServerError, err)
    46			if !res.Sent() {
    47				if h.dev {
    48					http.Error(w, fmt.Sprintf("JavaScript handler error: %v", err), http.StatusInternalServerError)
    49				} else {
    50					http.Error(w, "internal server error", http.StatusInternalServerError)
    51				}
    52			}
    53			return
    54		}
    55		h.recordAudit(r.Context(), r, req, route.Plan, envelope, "completed", res.Status(), nil)
    56	}
    57	
    58	func (h *Host) buildSecureEnvelope(ctx context.Context, httpReq *http.Request, req *RequestDTO, plan *RoutePlan) (*secureEnvelope, int, error) {
    59		if plan == nil {
    60			return nil, http.StatusInternalServerError, fmt.Errorf("planned route is missing route plan")
    61		}
    62		envelope := &secureEnvelope{Plan: *plan, Request: req, Body: req.Body, Resources: map[string]*ResourceRef{}}
    63		var actor *Actor
    64		switch plan.Security.Mode {
    65		case SecurityModePublic:
    66			// No actor required.
    67		case SecurityModeUser:
    68			if h.auth.Authenticator == nil {
    69				return envelope, http.StatusInternalServerError, fmt.Errorf("planned route %s %s requires authenticator", plan.Method, plan.Pattern)
    70			}
    71			var err error
    72			actor, err = h.auth.Authenticator.Authenticate(ctx, httpReq, req.Session, plan.Security)
    73			if err != nil {
    74				return envelope, statusForAuthError(err), err
    75			}
    76			if actor == nil {
    77				return envelope, http.StatusUnauthorized, ErrUnauthenticated
    78			}
    79			envelope.Actor = actor
    80		default:
    81			return envelope, http.StatusInternalServerError, fmt.Errorf("unsupported planned route security mode %q", plan.Security.Mode)
    82		}
    83	
    84		if plan.CSRF.Required && isUnsafeMethod(httpReq.Method) {
    85			if h.auth.CSRF == nil {
    86				return envelope, http.StatusInternalServerError, fmt.Errorf("planned route %s %s requires csrf protector", plan.Method, plan.Pattern)
    87			}
    88			if err := h.auth.CSRF.VerifyCSRF(ctx, CSRFRequest{HTTPRequest: httpReq, Request: req, Session: req.Session, Actor: actor, Plan: *plan}); err != nil {
    89				return envelope, statusForAuthError(fmt.Errorf("%w: %v", ErrCSRF, err)), err
    90			}
    91		}
    92	
    93		if len(plan.Resources) > 0 {
    94			if h.auth.Resources == nil {
    95				return envelope, http.StatusInternalServerError, fmt.Errorf("planned route %s %s requires resource resolver", plan.Method, plan.Pattern)
    96			}
    97			for _, spec := range plan.Resources {
    98				id, err := resolveValueSource(req, spec.ID)
    99				if err != nil {
   100					return envelope, http.StatusBadRequest, err
   101				}
   102				tenantID := ""
   103				if spec.Tenant != nil {
   104					tenantID, err = resolveValueSource(req, *spec.Tenant)
   105					if err != nil {
   106						return envelope, http.StatusBadRequest, err
   107					}
   108				}
   109				resource, err := h.auth.Resources.ResolveResource(ctx, ResourceRequest{HTTPRequest: httpReq, Request: req, Actor: actor, Spec: spec, ID: id, TenantID: tenantID})
   110				if err != nil {
   111					return envelope, statusForAuthError(err), err
   112				}
   113				if resource == nil {
   114					return envelope, http.StatusNotFound, ErrNotFound
   115				}
   116				if resource.Name == "" {
   117					resource.Name = spec.Name
   118				}
   119				if resource.Type == "" {
   120					resource.Type = spec.Type
   121				}
   122				envelope.Resources[spec.Name] = resource
   123			}
   124		}
   125	
   126		if plan.Security.Mode != SecurityModePublic && plan.Action != "" {
   127			if h.auth.Authorizer == nil {
   128				return envelope, http.StatusInternalServerError, fmt.Errorf("planned route %s %s requires authorizer", plan.Method, plan.Pattern)
   129			}
   130			resource := firstPlannedResource(plan, envelope.Resources)
   131			decision, err := h.auth.Authorizer.Authorize(ctx, AuthorizationRequest{HTTPRequest: httpReq, Request: req, Actor: actor, Action: plan.Action, Resource: resource, Resources: envelope.Resources})
   132			if err != nil {
   133				return envelope, statusForAuthError(err), err
   134			}
   135			if !decision.Allowed {
   136				if decision.Reason != "" {
   137					return envelope, http.StatusForbidden, fmt.Errorf("%w: %s", ErrForbidden, decision.Reason)
   138				}
   139				return envelope, http.StatusForbidden, ErrForbidden
   140			}
   141		}
   142		return envelope, 0, nil
   143	}
   144	
   145	func (h *Host) writePlannedError(w http.ResponseWriter, res *Response, status int, err error) {
```

## modules/express/auth_builders.go:115-205
```go
   115	type routeBuilder struct {
   116		registrar *Registrar
   117		store     *builderStore
   118		vm        *goja.Runtime
   119		plan      gojahttp.RoutePlan
   120	}
   121	
   122	func newRouteBuilder(vm *goja.Runtime, registrar *Registrar, store *builderStore, method, pattern string) goja.Value {
   123		b := &routeBuilder{registrar: registrar, store: store, vm: vm, plan: gojahttp.RoutePlan{Method: method, Pattern: pattern}}
   124		return b.needsSecurityObject()
   125	}
   126	
   127	func (b *routeBuilder) needsSecurityObject() goja.Value {
   128		obj := b.vm.NewObject()
   129		_ = obj.Set("name", func(name string) goja.Value {
   130			b.plan.Name = strings.TrimSpace(name)
   131			return obj
   132		})
   133		_ = obj.Set("public", func() goja.Value {
   134			b.plan.Security = gojahttp.SecuritySpec{Mode: gojahttp.SecurityModePublic}
   135			return b.needsHandlerObject()
   136		})
   137		_ = obj.Set("auth", func(value goja.Value) (goja.Value, error) {
   138			spec, err := b.store.authSpec(b.vm, value)
   139			if err != nil {
   140				return nil, err
   141			}
   142			b.plan.Security = spec
   143			return b.needsPolicyObject(), nil
   144		})
   145		return obj
   146	}
   147	
   148	func (b *routeBuilder) needsPolicyObject() goja.Value {
   149		obj := b.vm.NewObject()
   150		_ = obj.Set("resource", func(value goja.Value) (goja.Value, error) {
   151			spec, err := b.store.resourceSpec(b.vm, value)
   152			if err != nil {
   153				return nil, err
   154			}
   155			b.plan.Resources = append(b.plan.Resources, spec)
   156			return obj, nil
   157		})
   158		b.attachCSRFMethod(obj)
   159		b.attachAuditMethod(obj)
   160		_ = obj.Set("allow", func(action string) (goja.Value, error) {
   161			action = strings.TrimSpace(action)
   162			if action == "" {
   163				return nil, fmt.Errorf(".allow(action) requires a non-empty action")
   164			}
   165			b.plan.Action = action
   166			return b.needsHandlerObject(), nil
   167		})
   168		return obj
   169	}
   170	
   171	func (b *routeBuilder) needsHandlerObject() goja.Value {
   172		obj := b.vm.NewObject()
   173		b.attachCSRFMethod(obj)
   174		b.attachAuditMethod(obj)
   175		_ = obj.Set("handle", func(handler goja.Value) error {
   176			fn, ok := goja.AssertFunction(handler)
   177			if !ok {
   178				return fmt.Errorf("planned route .handle(...) requires a function")
   179			}
   180			if err := b.registrar.start(b.vm); err != nil {
   181				return err
   182			}
   183			return b.registrar.host.RegisterPlanned(b.plan, fn)
   184		})
   185		return obj
   186	}
   187	
   188	func (b *routeBuilder) attachCSRFMethod(obj *goja.Object) {
   189		_ = obj.Set("csrf", func(call goja.FunctionCall) goja.Value {
   190			required := true
   191			if len(call.Arguments) > 0 && !goja.IsUndefined(call.Argument(0)) && !goja.IsNull(call.Argument(0)) {
   192				required = call.Argument(0).ToBoolean()
   193			}
   194			b.plan.CSRF.Required = required
   195			return obj
   196		})
   197	}
   198	
   199	func (b *routeBuilder) attachAuditMethod(obj *goja.Object) {
   200		_ = obj.Set("audit", func(event string) (goja.Value, error) {
   201			event = strings.TrimSpace(event)
   202			if event == "" {
   203				return nil, fmt.Errorf(".audit(event) requires a non-empty event")
   204			}
   205			b.plan.Audit.Event = event
```

## pkg/gojahttp/auth/sessionauth/sessionauth.go:52-128
```go
    52		Claims            map[string]any
    53	}
    54	
    55	// Store persists server-side sessions.
    56	type Store interface {
    57		Create(ctx context.Context, session Session) error
    58		Get(ctx context.Context, id string) (*Session, error)
    59		Touch(ctx context.Context, id string, now time.Time, idleExpiresAt time.Time) error
    60		Rotate(ctx context.Context, oldID string, next Session) error
    61		Revoke(ctx context.Context, id string) error
    62	}
    63	
    64	// ActorLoader projects an application session into the gojahttp actor shape.
    65	type ActorLoader interface {
    66		ActorForSession(ctx context.Context, session *Session) (*gojahttp.Actor, error)
    67	}
    68	
    69	// ActorLoaderFunc adapts a function into ActorLoader.
    70	type ActorLoaderFunc func(ctx context.Context, session *Session) (*gojahttp.Actor, error)
    71	
    72	func (f ActorLoaderFunc) ActorForSession(ctx context.Context, session *Session) (*gojahttp.Actor, error) {
    73		return f(ctx, session)
    74	}
    75	
    76	// Config controls Manager behavior.
    77	type Config struct {
    78		Store             Store
    79		ActorLoader       ActorLoader
    80		CookieName        string
    81		Path              string
    82		SameSite          http.SameSite
    83		IdleTimeout       time.Duration
    84		AbsoluteTimeout   time.Duration
    85		AllowInsecureHTTP bool
    86		Now               func() time.Time
    87	}
    88	
    89	// Manager implements gojahttp.Authenticator and gojahttp.CSRFProtector from a
    90	// server-side session store.
    91	type Manager struct {
    92		store             Store
    93		actorLoader       ActorLoader
    94		cookieName        string
    95		path              string
    96		sameSite          http.SameSite
    97		idleTimeout       time.Duration
    98		absoluteTimeout   time.Duration
    99		allowInsecureHTTP bool
   100		now               func() time.Time
   101	}
   102	
   103	// New creates a session manager with secure-by-default cookie settings. For
   104	// localhost HTTP demos, set AllowInsecureHTTP.
   105	func New(cfg Config) (*Manager, error) {
   106		if cfg.Store == nil {
   107			return nil, fmt.Errorf("sessionauth: store is required")
   108		}
   109		if cfg.ActorLoader == nil {
   110			cfg.ActorLoader = ActorLoaderFunc(DefaultActorForSession)
   111		}
   112		if cfg.CookieName == "" {
   113			if cfg.AllowInsecureHTTP {
   114				cfg.CookieName = InsecureCookieName
   115			} else {
   116				cfg.CookieName = SecureCookieName
   117			}
   118		}
   119		if cfg.Path == "" {
   120			cfg.Path = "/"
   121		}
   122		if cfg.SameSite == 0 {
   123			cfg.SameSite = http.SameSiteLaxMode
   124		}
   125		if cfg.IdleTimeout == 0 {
   126			cfg.IdleTimeout = 30 * time.Minute
   127		}
   128		if cfg.AbsoluteTimeout == 0 {
```

## pkg/gojahttp/auth/capability/capability.go:52-126
```go
    52		Claims       map[string]string
    53		ExpiresAt    time.Time
    54		TTL          time.Duration
    55		SingleUse    bool
    56		CreatedBy    string
    57	}
    58	
    59	// IssueResult returns the stored capability plus the raw token exactly once.
    60	type IssueResult struct {
    61		Capability Capability
    62		Token      string
    63	}
    64	
    65	// Store persists capabilities by token hash.
    66	type Store interface {
    67		Create(ctx context.Context, capability Capability) error
    68		Redeem(ctx context.Context, tokenHash []byte, purpose string, now time.Time) (*Capability, error)
    69		Revoke(ctx context.Context, id string, now time.Time) error
    70		ByID(ctx context.Context, id string) (*Capability, error)
    71	}
    72	
    73	// Service issues, redeems, and revokes capability tokens.
    74	type Service struct {
    75		Store Store
    76		Audit gojahttp.AuditSink
    77		Now   func() time.Time
    78	}
    79	
    80	func (s Service) Issue(ctx context.Context, spec IssueSpec) (IssueResult, error) {
    81		if s.Store == nil {
    82			return IssueResult{}, fmt.Errorf("capability: store is required")
    83		}
    84		now := s.now()
    85		if spec.Purpose == "" {
    86			return IssueResult{}, fmt.Errorf("capability: purpose is required")
    87		}
    88		if spec.SubjectID == "" && (spec.ResourceType == "" || spec.ResourceID == "") {
    89			return IssueResult{}, fmt.Errorf("capability: subject or resource is required")
    90		}
    91		if spec.ExpiresAt.IsZero() {
    92			if spec.TTL <= 0 {
    93				return IssueResult{}, fmt.Errorf("capability: expiry or TTL is required")
    94			}
    95			spec.ExpiresAt = now.Add(spec.TTL)
    96		}
    97		id, err := randomToken()
    98		if err != nil {
    99			return IssueResult{}, err
   100		}
   101		token, err := randomToken()
   102		if err != nil {
   103			return IssueResult{}, err
   104		}
   105		capability := Capability{ID: id, Purpose: spec.Purpose, SubjectID: spec.SubjectID, ResourceType: spec.ResourceType, ResourceID: spec.ResourceID, Claims: cloneStringMap(spec.Claims), TokenHash: HashToken(token), ExpiresAt: spec.ExpiresAt, SingleUse: spec.SingleUse, CreatedBy: spec.CreatedBy, CreatedAt: now}
   106		if err := s.Store.Create(ctx, capability); err != nil {
   107			return IssueResult{}, err
   108		}
   109		s.record(ctx, "capability.issued", "completed", capability, nil)
   110		return IssueResult{Capability: redactCapability(capability), Token: token}, nil
   111	}
   112	
   113	func (s Service) Redeem(ctx context.Context, purpose, token string) (*Capability, error) {
   114		if s.Store == nil {
   115			return nil, fmt.Errorf("capability: store is required")
   116		}
   117		capability, err := s.Store.Redeem(ctx, HashToken(token), purpose, s.now())
   118		if err != nil {
   119			s.record(ctx, "capability.redeemed", "denied", Capability{Purpose: purpose}, err)
   120			return nil, err
   121		}
   122		s.record(ctx, "capability.redeemed", "completed", *capability, nil)
   123		redacted := redactCapability(*capability)
   124		return &redacted, nil
   125	}
   126	
```

## pkg/gojahttp/auth/keycloakauth/keycloakauth.go:150-255
```go
   150		}
   151		nonce, err := sessionauth.RandomToken()
   152		if err != nil {
   153			http.Error(w, "create login nonce", http.StatusInternalServerError)
   154			return
   155		}
   156		verifier, err := sessionauth.RandomToken()
   157		if err != nil {
   158			http.Error(w, "create pkce verifier", http.StatusInternalServerError)
   159			return
   160		}
   161		redirectURL := strings.TrimSpace(r.URL.Query().Get("return_to"))
   162		if redirectURL == "" || !strings.HasPrefix(redirectURL, "/") || strings.HasPrefix(redirectURL, "//") {
   163			redirectURL = h.afterLoginURL
   164		}
   165		if err := h.transactions.Put(r.Context(), Transaction{State: state, Nonce: nonce, PKCEVerifier: verifier, CreatedAt: time.Now(), RedirectURL: redirectURL}); err != nil {
   166			http.Error(w, "store login transaction", http.StatusInternalServerError)
   167			return
   168		}
   169		url := h.oauth2Config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), oauth2.SetAuthURLParam("nonce", nonce))
   170		http.Redirect(w, r, url, http.StatusFound)
   171	}
   172	
   173	func (h *Handlers) handleCallback(w http.ResponseWriter, r *http.Request) {
   174		if r.Method != http.MethodGet {
   175			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
   176			return
   177		}
   178		if errText := r.URL.Query().Get("error"); errText != "" {
   179			http.Error(w, "oidc error: "+errText, http.StatusUnauthorized)
   180			return
   181		}
   182		state := r.URL.Query().Get("state")
   183		code := r.URL.Query().Get("code")
   184		if state == "" || code == "" {
   185			http.Error(w, "missing oidc callback state or code", http.StatusBadRequest)
   186			return
   187		}
   188		tx, err := h.transactions.Take(r.Context(), state)
   189		if err != nil {
   190			http.Error(w, "invalid oidc state", http.StatusUnauthorized)
   191			return
   192		}
   193		token, err := h.oauth2Config.Exchange(r.Context(), code, oauth2.VerifierOption(tx.PKCEVerifier))
   194		if err != nil {
   195			http.Error(w, "oidc token exchange failed", http.StatusUnauthorized)
   196			return
   197		}
   198		rawIDToken, ok := token.Extra("id_token").(string)
   199		if !ok || rawIDToken == "" {
   200			http.Error(w, "oidc response missing id_token", http.StatusUnauthorized)
   201			return
   202		}
   203		idToken, err := h.verifier.Verify(r.Context(), rawIDToken)
   204		if err != nil {
   205			http.Error(w, "oidc id_token verification failed", http.StatusUnauthorized)
   206			return
   207		}
   208		if idToken.Nonce != tx.Nonce {
   209			http.Error(w, "oidc nonce mismatch", http.StatusUnauthorized)
   210			return
   211		}
   212		claims, err := claimsFromIDToken(idToken)
   213		if err != nil {
   214			http.Error(w, "oidc claims invalid", http.StatusUnauthorized)
   215			return
   216		}
   217		userSession, err := h.normalizer.NormalizeOIDCUser(r.Context(), claims)
   218		if err != nil {
   219			http.Error(w, "user normalization failed", http.StatusUnauthorized)
   220			return
   221		}
   222		if userSession.UserID == "" {
   223			http.Error(w, "user normalization returned empty user id", http.StatusUnauthorized)
   224			return
   225		}
   226		session, err := h.sessionManager.NewSession(r.Context(), userSession.UserID,
   227			sessionauth.WithEmail(userSession.Email, userSession.EmailVerified),
   228			sessionauth.WithTenantIDs(userSession.TenantIDs...),
   229			sessionauth.WithClaims(userSession.Claims),
   230		)
   231		if err != nil {
   232			http.Error(w, "create app session", http.StatusInternalServerError)
   233			return
   234		}
   235		h.sessionManager.SetCookie(w, session.ID)
   236		http.Redirect(w, r, tx.RedirectURL, http.StatusFound)
   237	}
   238	
   239	func (h *Handlers) handleLogout(w http.ResponseWriter, r *http.Request) {
   240		if r.Method != http.MethodPost && r.Method != http.MethodGet {
   241			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
   242			return
   243		}
   244		_ = h.sessionManager.RevokeRequestSession(r.Context(), r)
   245		h.sessionManager.ClearCookie(w)
   246		if r.Method == http.MethodGet {
   247			http.Redirect(w, r, h.afterLogoutURL, http.StatusFound)
   248			return
   249		}
   250		w.WriteHeader(http.StatusNoContent)
   251	}
   252	
   253	func claimsFromIDToken(idToken *oidc.IDToken) (OIDCClaims, error) {
   254		var raw map[string]any
   255		if err := idToken.Claims(&raw); err != nil {
```

## pkg/xgoja/hostauth/services.go:1-60
```go
     1	package hostauth
     2	
     3	import (
     4		"context"
     5	
     6		"github.com/go-go-golems/glazed/pkg/cmds/values"
     7		"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
     8		"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
     9		"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
    10		"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability"
    11		"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
    12	)
    13	
    14	const (
    15		// ServicesKey stores concrete auth services built for a host/runtime/command.
    16		ServicesKey = "go-go-goja-auth.services"
    17		// ServiceFactoryKey stores a lazy auth service factory. Command providers use
    18		// this key during command construction, then build concrete services after
    19		// command values have been parsed.
    20		ServiceFactoryKey = "go-go-goja-auth.service-factory"
    21	)
    22	
    23	// ServiceFactory builds concrete host auth services at command execution time.
    24	type ServiceFactory interface {
    25		BuildHostAuthServices(ctx context.Context, vals *values.Values) (*Services, error)
    26	}
    27	
    28	// AppAuthStores groups the app-owned authorization data stores.
    29	type AppAuthStores struct {
    30		Users       appauth.UserStore
    31		Memberships appauth.MembershipStore
    32		Resources   appauth.ResourceStore
    33	}
    34	
    35	// Services contains concrete host-owned auth infrastructure. It is intentionally
    36	// a Go service payload for generated/custom hosts, not a JavaScript API.
    37	type Services struct {
    38		Config      ResolvedConfig
    39		AuthOptions gojahttp.AuthOptions
    40	
    41		SessionManager *sessionauth.Manager
    42		SessionStore   sessionauth.Store
    43	
    44		AuditSink  gojahttp.AuditSink
    45		AuditStore audit.Store
    46	
    47		AppAuth    AppAuthStores
    48		Capability capability.Store
    49	
    50		Closers []func(context.Context) error
    51	}
    52	
    53	// Close closes resources owned by the services bundle.
    54	func (s *Services) Close(ctx context.Context) error {
    55		if s == nil {
    56			return nil
    57		}
    58		return closeAll(ctx, s.Closers)
    59	}
```

## pkg/xgoja/hostauth/builder.go:80-125
```go
    80		services := &Services{
    81			Config:         resolved,
    82			AuthOptions:    authOptions,
    83			SessionManager: sessionManager,
    84			SessionStore:   stores.Session,
    85			AuditSink:      auditSink,
    86			AuditStore:     stores.Audit,
    87			AppAuth:        stores.AppAuth,
    88			Capability:     stores.Capability,
    89			Closers:        stores.Closers,
    90		}
    91		success = true
    92		return services, nil
    93	}
    94	
    95	// BuildSessionManager maps resolved generated-host config into sessionauth.
    96	func BuildSessionManager(cfg ResolvedSessionConfig, store sessionauth.Store, actorLoader sessionauth.ActorLoader, now func() time.Time) (*sessionauth.Manager, error) {
    97		return sessionauth.New(sessionauth.Config{
    98			Store:             store,
    99			ActorLoader:       actorLoader,
   100			CookieName:        cfg.Cookie.Name,
   101			Path:              cfg.Cookie.Path,
   102			SameSite:          cfg.Cookie.SameSite,
   103			IdleTimeout:       cfg.IdleTimeout,
   104			AbsoluteTimeout:   cfg.AbsoluteTimeout,
   105			AllowInsecureHTTP: cfg.Cookie.AllowInsecureHTTP,
   106			Now:               now,
   107		})
   108	}
   109	
   110	// BuildAuthOptions wires a session manager and built auth stores into
   111	// gojahttp's host-owned auth interfaces.
   112	func BuildAuthOptions(sessionManager *sessionauth.Manager, stores *StoreBundle, auditSink gojahttp.AuditSink) gojahttp.AuthOptions {
   113		var options gojahttp.AuthOptions
   114		if sessionManager != nil {
   115			options.Authenticator = sessionManager
   116			options.CSRF = sessionManager
   117		}
   118		if auditSink != nil {
   119			options.Audit = auditSink
   120		}
   121		if stores != nil {
   122			if stores.AppAuth.Resources != nil {
   123				options.Resources = appauth.Resolver{Store: stores.AppAuth.Resources}
   124			}
   125			if stores.AppAuth.Memberships != nil {
```

## pkg/xgoja/hostauth/stores.go:20-75
```go
    20	)
    21	
    22	// StoreBundle contains the concrete stores built from ResolvedStoresConfig.
    23	type StoreBundle struct {
    24		Session    sessionauth.Store
    25		Audit      audit.Store
    26		AppAuth    AppAuthStores
    27		Capability capability.Store
    28	
    29		Closers []func(context.Context) error
    30	}
    31	
    32	// Close closes all resources owned by the bundle.
    33	func (b *StoreBundle) Close(ctx context.Context) error {
    34		if b == nil {
    35			return nil
    36		}
    37		return closeAll(ctx, b.Closers)
    38	}
    39	
    40	// BuildStores creates all host auth stores described by cfg. SQL DB handles are
    41	// shared when store configs resolve to the same driver and DSN.
    42	func BuildStores(ctx context.Context, cfg ResolvedStoresConfig) (*StoreBundle, error) {
    43		builder := storeBuilder{dbs: map[sqlDBKey]*sql.DB{}}
    44		bundle, err := builder.build(ctx, cfg)
    45		if err != nil {
    46			_ = closeAll(ctx, builder.closers)
    47			return nil, err
    48		}
    49		return bundle, nil
    50	}
    51	
    52	type storeBuilder struct {
    53		dbs     map[sqlDBKey]*sql.DB
    54		closers []func(context.Context) error
    55	}
    56	
    57	type sqlDBKey struct {
    58		driver StoreDriver
    59		dsn    string
    60	}
    61	
    62	func (b *storeBuilder) build(ctx context.Context, cfg ResolvedStoresConfig) (*StoreBundle, error) {
    63		sessionStore, err := b.buildSessionStore(ctx, cfg.Session)
    64		if err != nil {
    65			return nil, err
    66		}
    67		auditStore, err := b.buildAuditStore(ctx, cfg.Audit)
    68		if err != nil {
    69			return nil, err
    70		}
    71		appAuthStores, err := b.buildAppAuthStores(ctx, cfg.AppAuth)
    72		if err != nil {
    73			return nil, err
    74		}
    75		capabilityStore, err := b.buildCapabilityStore(ctx, cfg.Capability)
```

## pkg/xgoja/providers/http/serve.go:380-430
```go
   380			snapshot.Host.ServeHTTP(recorder, httptest.NewRequest(stdhttp.MethodGet, path, nil))
   381			if recorder.Code < 200 || recorder.Code >= 300 {
   382				return fmt.Errorf("hot reload smoke GET %s status=%d body=%s", path, recorder.Code, recorder.Body.String())
   383			}
   384			return nil
   385		}
   386	}
   387	
   388	func decodeHTTPServeSettings(vals *values.Values) (settings, error) {
   389		cfg := settings{Enabled: true, Listen: "127.0.0.1:8787"}
   390		if vals != nil {
   391			if err := vals.DecodeSectionInto("http", &cfg); err != nil {
   392				return settings{}, err
   393			}
   394		}
   395		return normalizeSettings(cfg), nil
   396	}
   397	
   398	func buildServeAuthServices(ctx context.Context, commandCtx providerapi.CommandSetContext, parsedValues *values.Values) (*hostauth.Services, bool, error) {
   399		factory, ok, err := hostauth.LookupServiceFactory(commandCtx.Host)
   400		if err != nil {
   401			return nil, false, err
   402		}
   403		if !ok {
   404			return nil, false, nil
   405		}
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
   422	}
   423	
   424	func serveRuntimeServices(host *gojahttp.Host, authServices *hostauth.Services, ownsListen bool, includeHost bool) (app.HostServices, error) {
   425		services := app.HostServices{}
   426		if includeHost {
   427			if err := services.SetHostService(HostServiceKey, ExternalHostService{Host: host, OwnsListen: ownsListen}); err != nil {
   428				return app.HostServices{}, err
   429			}
   430		}
```
