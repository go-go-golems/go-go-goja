---
Title: Current GojaHTTP Auth Surface Evidence
Ticket: XGOJA-GO-AUTH-API-DESIGN
Status: active
Topics:
  - goja
  - auth
  - architecture
  - rest-api
DocType: reference
Intent: short-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Line-anchored current code evidence for designing a Go-native planned auth API.
LastUpdated: 2026-06-15T21:55:00-04:00
WhatFor: Ground the Go-native planned auth guide in current gojahttp, express, and hostauth code.
WhenToUse: Use while implementing SecureContext, RegisterPlannedHTTP, Go route builders, and AuthKit.
---

# Current GojaHTTP Auth Surface Evidence

This file records the current code surfaces that the Go-native planned auth API should reuse or refactor. It intentionally captures product code only.

## pkg/gojahttp/auth_plan.go:1-220
```go
     1	package gojahttp
     2	
     3	import (
     4		"context"
     5		"errors"
     6		"fmt"
     7		"net/http"
     8		"strings"
     9		"time"
    10	)
    11	
    12	// SecurityMode describes the route-level security envelope that must run before
    13	// a planned JavaScript handler is invoked.
    14	type SecurityMode string
    15	
    16	const (
    17		SecurityModePublic SecurityMode = "public"
    18		SecurityModeUser   SecurityMode = "user"
    19	)
    20	
    21	// ValueSourceKind identifies where a route-plan value should be read from.
    22	type ValueSourceKind string
    23	
    24	const (
    25		ValueSourceParam   ValueSourceKind = "param"
    26		ValueSourceQuery   ValueSourceKind = "query"
    27		ValueSourceBody    ValueSourceKind = "body"
    28		ValueSourceLiteral ValueSourceKind = "literal"
    29	)
    30	
    31	var (
    32		ErrUnauthenticated = errors.New("unauthenticated")
    33		ErrForbidden       = errors.New("forbidden")
    34		ErrNotFound        = errors.New("not found")
    35		ErrCSRF            = errors.New("csrf invalid")
    36	)
    37	
    38	// RoutePlan is the Go-owned security contract compiled by the Express fluent
    39	// route builder at registration time.
    40	type RoutePlan struct {
    41		Name      string
    42		Method    string
    43		Pattern   string
    44		Security  SecuritySpec
    45		Resources []ResourceSpec
    46		Action    string
    47		CSRF      CSRFSpec
    48		Audit     AuditSpec
    49	}
    50	
    51	// SecuritySpec describes who may enter a planned route.
    52	type SecuritySpec struct {
    53		Mode           SecurityMode
    54		Required       bool
    55		MFAFreshWithin time.Duration
    56	}
    57	
    58	// ValueSource describes a typed value extraction from the HTTP adapter layer.
    59	// Resource resolvers receive the resolved value, not raw req.params maps.
    60	type ValueSource struct {
    61		Kind  ValueSourceKind
    62		Key   string
    63		Value string
    64	}
    65	
    66	// ResourceSpec describes which resource a route touches and how its identity is
    67	// extracted from the request adapter layer.
    68	type ResourceSpec struct {
    69		Name      string
    70		Type      string
    71		ID        ValueSource
    72		Tenant    *ValueSource
    73		MustExist bool
    74	}
    75	
    76	// CSRFSpec describes whether a planned route requires host-owned CSRF
    77	// verification before the JavaScript handler runs.
    78	type CSRFSpec struct {
    79		Required bool
    80	}
    81	
    82	// AuditSpec describes the host-owned audit event emitted for a planned route.
    83	type AuditSpec struct {
    84		Event string
    85	}
    86	
    87	// Actor is the minimal host-owned authenticated principal exposed to planned
    88	// route handlers.
    89	type Actor struct {
    90		ID        string         `json:"id"`
    91		Kind      string         `json:"kind"`
    92		TenantIDs []string       `json:"tenantIds,omitempty"`
    93		Claims    map[string]any `json:"claims,omitempty"`
    94	}
    95	
    96	// ResourceRef is the minimal host-owned resource handle exposed to planned
    97	// route handlers after resolution and authorization.
    98	type ResourceRef struct {
    99		Name     string         `json:"name"`
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
   181	
   182	func ValidateRoutePlan(plan RoutePlan) (RoutePlan, error) {
   183		plan.Method = strings.ToUpper(strings.TrimSpace(plan.Method))
   184		plan.Pattern = cleanPath(plan.Pattern)
   185		plan.Name = strings.TrimSpace(plan.Name)
   186		plan.Action = strings.TrimSpace(plan.Action)
   187		plan.Audit.Event = strings.TrimSpace(plan.Audit.Event)
   188	
   189		if plan.Method == "" {
   190			return RoutePlan{}, fmt.Errorf("planned route method is required")
   191		}
   192		if plan.Pattern == "" {
   193			return RoutePlan{}, fmt.Errorf("planned route pattern is required")
   194		}
   195	
   196		switch plan.Security.Mode {
   197		case SecurityModePublic:
   198			plan.Security.Required = false
   199		case SecurityModeUser:
   200			plan.Security.Required = true
   201			if plan.Action == "" {
   202				return RoutePlan{}, fmt.Errorf("planned user route %s %s requires .allow(action)", plan.Method, plan.Pattern)
   203			}
   204		default:
   205			return RoutePlan{}, fmt.Errorf("planned route %s %s must declare .public() or .auth(...) before .handle(...)", plan.Method, plan.Pattern)
   206		}
   207	
   208		pathParams := pathParamSet(plan.Pattern)
   209		for i := range plan.Resources {
   210			resource := &plan.Resources[i]
   211			resource.Name = strings.TrimSpace(resource.Name)
   212			resource.Type = strings.TrimSpace(resource.Type)
   213			if resource.Type == "" {
   214				return RoutePlan{}, fmt.Errorf("resource %d on %s %s requires a type", i+1, plan.Method, plan.Pattern)
   215			}
   216			if resource.Name == "" {
   217				resource.Name = resource.Type
   218			}
   219			if err := validateValueSource(resource.ID, pathParams, fmt.Sprintf("resource %q id", resource.Name)); err != nil {
   220				return RoutePlan{}, err
```

## pkg/gojahttp/planned_dispatch.go:1-180
```go
     1	package gojahttp
     2	
     3	import (
     4		"context"
     5		"errors"
     6		"fmt"
     7		"net/http"
     8	
     9		"github.com/dop251/goja"
    10	)
    11	
    12	type secureEnvelope struct {
    13		Plan      RoutePlan
    14		Request   *RequestDTO
    15		Actor     *Actor
    16		Resources map[string]*ResourceRef
    17		Body      any
    18	}
    19	
    20	func (h *Host) servePlannedRoute(w http.ResponseWriter, r *http.Request, route Route, req *RequestDTO) {
    21		res := NewResponse(w, h.renderer)
    22		envelope, status, err := h.buildSecureEnvelope(r.Context(), r, req, route.Plan)
    23		if err != nil {
    24			h.recordAudit(r.Context(), r, req, route.Plan, envelope, "denied", status, err)
    25			h.writePlannedError(w, res, status, err)
    26			return
    27		}
    28		h.recordAudit(r.Context(), r, req, route.Plan, envelope, "allowed", 0, nil)
    29		ret, err := h.owner.Call(r.Context(), "http-planned-handler", func(ctx context.Context, vm *goja.Runtime) (any, error) {
    30			result, err := route.Handler(goja.Undefined(), envelope.JSObject(vm), res.JSObject(vm))
    31			if err != nil {
    32				return nil, err
    33			}
    34			if promise, ok := result.Export().(*goja.Promise); ok {
    35				return promise, nil
    36			}
    37			return nil, h.finishHandlerResult(vm, res, result)
    38		})
    39		if err == nil {
    40			if promise, ok := ret.(*goja.Promise); ok {
    41				err = h.awaitAndFinishPromise(r.Context(), res, promise)
    42			}
    43		}
    44		if err != nil {
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
   146		if res.Sent() {
   147			return
   148		}
   149		if status == 0 {
   150			status = http.StatusInternalServerError
   151		}
   152		message := http.StatusText(status)
   153		if h.dev && err != nil && status >= 500 {
   154			message = err.Error()
   155		}
   156		http.Error(w, message, status)
   157	}
   158	
   159	func (h *Host) recordAudit(ctx context.Context, httpReq *http.Request, req *RequestDTO, plan *RoutePlan, envelope *secureEnvelope, outcome string, status int, err error) {
   160		if h.auth.Audit == nil || plan == nil || plan.Audit.Event == "" {
   161			return
   162		}
   163		var actor *Actor
   164		resources := map[string]*ResourceRef{}
   165		if envelope != nil {
   166			actor = envelope.Actor
   167			resources = copyResourceRefs(envelope.Resources)
   168		}
   169		resource := firstPlannedResource(plan, resources)
   170		reason := ""
   171		if err != nil {
   172			reason = err.Error()
   173		}
   174		_ = h.auth.Audit.RecordAudit(ctx, AuditEvent{
   175			HTTPRequest: httpReq,
   176			Request:     req,
   177			Event:       plan.Audit.Event,
   178			Outcome:     outcome,
   179			Reason:      reason,
   180			StatusCode:  status,
```

## pkg/gojahttp/planned_dispatch.go:258-318
```go
   258	func copyResourceRefs(resources map[string]*ResourceRef) map[string]*ResourceRef {
   259		out := make(map[string]*ResourceRef, len(resources))
   260		for name, resource := range resources {
   261			out[name] = resource
   262		}
   263		return out
   264	}
   265	
   266	func (e *secureEnvelope) JSObject(vm *goja.Runtime) *goja.Object {
   267		obj := vm.NewObject()
   268		_ = obj.Set("request", e.Request.Map())
   269		_ = obj.Set("actor", actorJSMap(e.Actor))
   270		_ = obj.Set("body", e.Body)
   271		_ = obj.Set("params", e.Request.Params)
   272		_ = obj.Set("resources", resourceJSMap(e.Resources))
   273		_ = obj.Set("action", e.Plan.Action)
   274		_ = obj.Set("routeName", e.Plan.Name)
   275		_ = obj.Set("resource", func(name string) goja.Value {
   276			resource := e.Resources[name]
   277			if resource == nil {
   278				return goja.Null()
   279			}
   280			return vm.ToValue(resourceRefJSMap(resource))
   281		})
   282		return obj
   283	}
   284	
   285	func actorJSMap(actor *Actor) map[string]any {
   286		if actor == nil {
   287			return nil
   288		}
   289		return map[string]any{
   290			"id":        actor.ID,
   291			"kind":      actor.Kind,
   292			"tenantIds": actor.TenantIDs,
   293			"claims":    actor.Claims,
   294		}
   295	}
   296	
   297	func resourceJSMap(resources map[string]*ResourceRef) map[string]any {
   298		out := make(map[string]any, len(resources))
   299		for name, resource := range resources {
   300			out[name] = resourceRefJSMap(resource)
   301		}
   302		return out
   303	}
   304	
   305	func resourceRefJSMap(resource *ResourceRef) map[string]any {
   306		if resource == nil {
   307			return nil
   308		}
   309		return map[string]any{
   310			"name":     resource.Name,
   311			"type":     resource.Type,
   312			"id":       resource.ID,
   313			"tenantId": resource.TenantID,
   314			"claims":   resource.Claims,
   315		}
   316	}
```

## pkg/gojahttp/host.go:1-120
```go
     1	package gojahttp
     2	
     3	import (
     4		"context"
     5		"fmt"
     6		"net/http"
     7		"strings"
     8		"time"
     9	
    10		"github.com/dop251/goja"
    11		"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
    12	)
    13	
    14	type HostOptions struct {
    15		Dev             bool
    16		Renderer        Renderer
    17		Sessions        SessionOptions
    18		Auth            AuthOptions
    19		RejectRawRoutes bool
    20	}
    21	
    22	type StaticMount struct {
    23		Prefix          string
    24		Handler         http.Handler
    25		ExcludePrefixes []string
    26	}
    27	
    28	// MountOptions configures a Go http.Handler mounted into Host prefix dispatch.
    29	// Generic handler mounts preserve the request path by default. Static asset
    30	// helpers opt into StripPrefix to preserve their historical behavior.
    31	type MountOptions struct {
    32		StripPrefix     bool
    33		ExcludePrefixes []string
    34	}
    35	
    36	type Host struct {
    37		registry        *Registry
    38		dev             bool
    39		renderer        Renderer
    40		owner           runtimeowner.RuntimeOwner
    41		sessions        *SessionManager
    42		auth            AuthOptions
    43		rejectRawRoutes bool
    44		static          []StaticMount
    45	}
    46	
    47	func NewHost(opts HostOptions) *Host {
    48		return &Host{registry: NewRegistry(), dev: opts.Dev, renderer: opts.Renderer, sessions: NewSessionManager(opts.Sessions), auth: opts.Auth, rejectRawRoutes: opts.RejectRawRoutes}
    49	}
    50	
    51	func (h *Host) SetRuntime(owner runtimeowner.RuntimeOwner) { h.owner = owner }
    52	func (h *Host) Register(method, pattern string, handler goja.Callable) {
    53		h.registry.Add(method, pattern, handler)
    54	}
    55	func (h *Host) RegisterPlanned(plan RoutePlan, handler goja.Callable) error {
    56		plan, err := ValidateRoutePlan(plan)
    57		if err != nil {
    58			return err
    59		}
    60		h.registry.AddPlanned(plan, handler)
    61		return nil
    62	}
    63	func (h *Host) Routes() []RouteDescriptor {
    64		if h == nil || h.registry == nil {
    65			return nil
    66		}
    67		return h.registry.Routes()
    68	}
    69	func (h *Host) RegisterStatic(prefix, dir string) {
    70		h.RegisterStaticHandler(prefix, http.FileServer(http.Dir(dir)))
    71	}
    72	
    73	func (h *Host) RegisterStaticHandler(prefix string, handler http.Handler) {
    74		h.RegisterStaticHandlerWithOptions(prefix, handler, nil)
    75	}
    76	
    77	func (h *Host) RegisterStaticHandlerWithOptions(prefix string, handler http.Handler, excludePrefixes []string) {
    78		h.RegisterHandlerWithOptions(prefix, handler, MountOptions{StripPrefix: true, ExcludePrefixes: excludePrefixes})
    79	}
    80	
    81	// RegisterHandler mounts handler under prefix using prefix matching while
    82	// preserving the original request path.
    83	func (h *Host) RegisterHandler(prefix string, handler http.Handler) {
    84		h.RegisterHandlerWithOptions(prefix, handler, MountOptions{})
    85	}
    86	
    87	// RegisterHandlerWithOptions mounts handler under prefix using prefix matching.
    88	// When StripPrefix is true, the mounted handler sees the path with prefix
    89	// removed. ExcludePrefixes skip this mount and allow later mounts/routes to try.
    90	func (h *Host) RegisterHandlerWithOptions(prefix string, handler http.Handler, opts MountOptions) {
    91		prefix = cleanPath(prefix)
    92		excludes := make([]string, 0, len(opts.ExcludePrefixes))
    93		for _, exclude := range opts.ExcludePrefixes {
    94			exclude = cleanPath(exclude)
    95			if exclude != "" {
    96				excludes = append(excludes, exclude)
    97			}
    98		}
    99		mounted := handler
   100		if opts.StripPrefix {
   101			mounted = stripMountPrefix(prefix, handler)
   102		}
   103		h.static = append(h.static, StaticMount{Prefix: prefix, Handler: mounted, ExcludePrefixes: excludes})
   104	}
   105	
   106	func stripMountPrefix(prefix string, handler http.Handler) http.Handler {
   107		if prefix == "/" {
   108			return handler
   109		}
   110		return http.StripPrefix(prefix, handler)
   111	}
   112	
   113	func staticMountMatches(prefix, requestPath string) bool {
   114		prefix = cleanPath(prefix)
   115		requestPath = cleanPath(requestPath)
   116		if prefix == "/" {
   117			return true
   118		}
   119		return requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/")
   120	}
```

## pkg/gojahttp/host.go:120-220
```go
   120	}
   121	
   122	func staticMountExcluded(excludePrefixes []string, requestPath string) bool {
   123		for _, exclude := range excludePrefixes {
   124			if staticMountMatches(exclude, requestPath) {
   125				return true
   126			}
   127		}
   128		return false
   129	}
   130	
   131	func (h *Host) ServeHTTP(w http.ResponseWriter, r *http.Request) {
   132		started := time.Now()
   133		requestID := ensureRequestID(w, r)
   134		logger := requestLogger(r, requestID)
   135		logger.Info().Str("event", "http_request_started").Msg("http request started")
   136		loggingWriter, wrappedWriter := newAccessLogResponseWriter(w)
   137		defer logRequestDone(logger, loggingWriter, started)
   138		w = wrappedWriter
   139	
   140		for _, mount := range h.static {
   141			if staticMountMatches(mount.Prefix, r.URL.Path) {
   142				if staticMountExcluded(mount.ExcludePrefixes, r.URL.Path) {
   143					continue
   144				}
   145				mount.Handler.ServeHTTP(w, r)
   146				return
   147			}
   148		}
   149		if h.owner == nil {
   150			http.Error(w, "runtime not initialized", http.StatusInternalServerError)
   151			return
   152		}
   153		route, params, ok := h.registry.Match(r.Method, r.URL.Path)
   154		if !ok && r.Method == http.MethodHead {
   155			route, params, ok = h.registry.Match(http.MethodGet, r.URL.Path)
   156			if ok {
   157				w = headResponseWriter{ResponseWriter: w}
   158			}
   159		}
   160		if !ok {
   161			http.NotFound(w, r)
   162			return
   163		}
   164		if route.Plan == nil && h.rejectRawRoutes {
   165			h.writeRawRouteRejected(w, route)
   166			return
   167		}
   168		session, err := h.sessions.Session(w, r)
   169		if err != nil {
   170			http.Error(w, err.Error(), http.StatusInternalServerError)
   171			return
   172		}
   173		req, err := NewRequestDTO(r, params, session)
   174		if err != nil {
   175			http.Error(w, err.Error(), http.StatusBadRequest)
   176			return
   177		}
   178		if route.Plan != nil {
   179			h.servePlannedRoute(w, r, route, req)
   180			return
   181		}
   182		res := NewResponse(w, h.renderer)
   183		ret, err := h.owner.Call(r.Context(), "http-handler", func(ctx context.Context, vm *goja.Runtime) (any, error) {
   184			result, err := route.Handler(goja.Undefined(), vm.ToValue(req.Map()), res.JSObject(vm))
   185			if err != nil {
   186				return nil, err
   187			}
   188			if promise, ok := result.Export().(*goja.Promise); ok {
   189				return promise, nil
   190			}
   191			return nil, h.finishHandlerResult(vm, res, result)
   192		})
   193		if err == nil {
   194			if promise, ok := ret.(*goja.Promise); ok {
   195				err = h.awaitAndFinishPromise(r.Context(), res, promise)
   196			}
   197		}
   198		if err != nil {
   199			logger.Error().Err(err).Str("event", "http_handler_error").Msg("http handler error")
   200		}
   201		if err != nil && !res.Sent() {
   202			if h.dev {
   203				http.Error(w, fmt.Sprintf("JavaScript handler error: %v", err), http.StatusInternalServerError)
   204			} else {
   205				http.Error(w, "internal server error", http.StatusInternalServerError)
   206			}
   207		}
   208	}
   209	
   210	func (h *Host) writeRawRouteRejected(w http.ResponseWriter, route Route) {
   211		message := "raw routes disabled"
   212		if h.dev {
   213			message = fmt.Sprintf("raw route %s %s rejected: register a planned route with .public() or auth", route.Method, route.Pattern)
   214		}
   215		http.Error(w, message, http.StatusInternalServerError)
   216	}
   217	
   218	func (h *Host) finishHandlerResult(vm *goja.Runtime, res *Response, result goja.Value) error {
   219		if !res.Sent() && !goja.IsUndefined(result) && !goja.IsNull(result) {
   220			if _, ok := result.Export().(string); ok {
```

## pkg/gojahttp/route_registry.go:1-90
```go
     1	package gojahttp
     2	
     3	import (
     4		"strings"
     5		"sync"
     6	
     7		"github.com/dop251/goja"
     8	)
     9	
    10	type Route struct {
    11		Method  string
    12		Pattern string
    13		Handler goja.Callable
    14		Plan    *RoutePlan
    15	}
    16	
    17	type RouteDescriptor struct {
    18		Method       string       `json:"method"`
    19		Pattern      string       `json:"pattern"`
    20		Planned      bool         `json:"planned"`
    21		SecurityMode SecurityMode `json:"securityMode,omitempty"`
    22		Action       string       `json:"action,omitempty"`
    23		Name         string       `json:"name,omitempty"`
    24		CSRFRequired bool         `json:"csrfRequired,omitempty"`
    25		AuditEvent   string       `json:"auditEvent,omitempty"`
    26	}
    27	
    28	type Registry struct {
    29		mu     sync.RWMutex
    30		routes []Route
    31	}
    32	
    33	func NewRegistry() *Registry { return &Registry{} }
    34	
    35	func (r *Registry) Add(method, pattern string, handler goja.Callable) {
    36		r.mu.Lock()
    37		defer r.mu.Unlock()
    38		r.routes = append(r.routes, Route{Method: strings.ToUpper(method), Pattern: cleanPath(pattern), Handler: handler})
    39	}
    40	
    41	func (r *Registry) AddPlanned(plan RoutePlan, handler goja.Callable) {
    42		r.mu.Lock()
    43		defer r.mu.Unlock()
    44		plan.Method = strings.ToUpper(plan.Method)
    45		plan.Pattern = cleanPath(plan.Pattern)
    46		r.routes = append(r.routes, Route{Method: plan.Method, Pattern: plan.Pattern, Handler: handler, Plan: &plan})
    47	}
    48	
    49	func (r *Registry) Routes() []RouteDescriptor {
    50		if r == nil {
    51			return nil
    52		}
    53		r.mu.RLock()
    54		defer r.mu.RUnlock()
    55		out := make([]RouteDescriptor, 0, len(r.routes))
    56		for _, route := range r.routes {
    57			descriptor := RouteDescriptor{Method: route.Method, Pattern: route.Pattern}
    58			if route.Plan != nil {
    59				descriptor.Planned = true
    60				descriptor.SecurityMode = route.Plan.Security.Mode
    61				descriptor.Action = route.Plan.Action
    62				descriptor.Name = route.Plan.Name
    63				descriptor.CSRFRequired = route.Plan.CSRF.Required
    64				descriptor.AuditEvent = route.Plan.Audit.Event
    65			}
    66			out = append(out, descriptor)
    67		}
    68		return out
    69	}
    70	
    71	func (r *Registry) Match(method, path string) (Route, map[string]string, bool) {
    72		r.mu.RLock()
    73		defer r.mu.RUnlock()
    74		method = strings.ToUpper(method)
    75		path = cleanPath(path)
    76		for _, route := range r.routes {
    77			if route.Method != method && route.Method != "ALL" {
    78				continue
    79			}
    80			params, ok := matchPattern(route.Pattern, path)
    81			if ok {
    82				return route, params, true
    83			}
    84		}
    85		return Route{}, nil, false
    86	}
    87	
    88	func cleanPath(p string) string {
    89		if p == "" {
    90			return "/"
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

## pkg/gojahttp/auth/sessionauth/sessionauth.go:80-235
```go
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
   129			cfg.AbsoluteTimeout = 12 * time.Hour
   130		}
   131		if cfg.Now == nil {
   132			cfg.Now = time.Now
   133		}
   134		return &Manager{store: cfg.Store, actorLoader: cfg.ActorLoader, cookieName: cfg.CookieName, path: cfg.Path, sameSite: cfg.SameSite, idleTimeout: cfg.IdleTimeout, absoluteTimeout: cfg.AbsoluteTimeout, allowInsecureHTTP: cfg.AllowInsecureHTTP, now: cfg.Now}, nil
   135	}
   136	
   137	// AuthOptions returns the gojahttp auth fields implemented by this manager.
   138	func (m *Manager) AuthOptions() gojahttp.AuthOptions {
   139		return gojahttp.AuthOptions{Authenticator: m, CSRF: m}
   140	}
   141	
   142	// NewSession creates, stores, and returns a new server-side session.
   143	func (m *Manager) NewSession(ctx context.Context, userID string, opts ...SessionOption) (*Session, error) {
   144		now := m.now()
   145		id, err := RandomToken()
   146		if err != nil {
   147			return nil, err
   148		}
   149		csrf, err := RandomToken()
   150		if err != nil {
   151			return nil, err
   152		}
   153		session := Session{ID: id, UserID: userID, CSRFToken: csrf, CreatedAt: now, LastSeenAt: now, IdleExpiresAt: now.Add(m.idleTimeout), AbsoluteExpiresAt: now.Add(m.absoluteTimeout)}
   154		for _, opt := range opts {
   155			opt(&session)
   156		}
   157		if err := m.store.Create(ctx, session); err != nil {
   158			return nil, err
   159		}
   160		return &session, nil
   161	}
   162	
   163	// SetCookie writes the manager's session cookie.
   164	func (m *Manager) SetCookie(w http.ResponseWriter, sessionID string) {
   165		m.setCookie(w, sessionID, int(m.absoluteTimeout.Seconds()))
   166	}
   167	
   168	// ClearCookie clears the manager's session cookie.
   169	func (m *Manager) ClearCookie(w http.ResponseWriter) { m.setCookie(w, "", -1) }
   170	
   171	// RevokeRequestSession revokes the session referenced by the request cookie.
   172	func (m *Manager) RevokeRequestSession(ctx context.Context, r *http.Request) error {
   173		id, err := m.sessionIDFromRequest(r)
   174		if err != nil {
   175			return authError(err)
   176		}
   177		return m.store.Revoke(ctx, id)
   178	}
   179	
   180	// SessionFromRequest loads and validates a session from the request cookie.
   181	func (m *Manager) SessionFromRequest(ctx context.Context, r *http.Request) (*Session, error) {
   182		id, err := m.sessionIDFromRequest(r)
   183		if err != nil {
   184			return nil, err
   185		}
   186		session, err := m.store.Get(ctx, id)
   187		if err != nil {
   188			return nil, err
   189		}
   190		if session == nil {
   191			return nil, ErrInvalidCookie
   192		}
   193		if err := validateSession(session, m.now()); err != nil {
   194			return nil, err
   195		}
   196		return session, nil
   197	}
   198	
   199	// Authenticate implements gojahttp.Authenticator.
   200	func (m *Manager) Authenticate(ctx context.Context, req *http.Request, _ *gojahttp.SessionDTO, spec gojahttp.SecuritySpec) (*gojahttp.Actor, error) {
   201		session, err := m.SessionFromRequest(ctx, req)
   202		if err != nil {
   203			return nil, authError(err)
   204		}
   205		if err := validateMFAFreshness(session, spec, m.now()); err != nil {
   206			return nil, authError(err)
   207		}
   208		actor, err := m.actorLoader.ActorForSession(ctx, session)
   209		if err != nil {
   210			return nil, err
   211		}
   212		if actor == nil {
   213			return nil, ErrNoActor
   214		}
   215		now := m.now()
   216		if err := m.store.Touch(ctx, session.ID, now, now.Add(m.idleTimeout)); err != nil {
   217			return nil, err
   218		}
   219		return actor, nil
   220	}
   221	
   222	// VerifyCSRF implements gojahttp.CSRFProtector.
   223	func (m *Manager) VerifyCSRF(ctx context.Context, req gojahttp.CSRFRequest) error {
   224		session, err := m.SessionFromRequest(ctx, req.HTTPRequest)
   225		if err != nil {
   226			return authError(err)
   227		}
   228		if req.Actor != nil && req.Actor.ID != session.UserID {
   229			return errors.New("session actor mismatch")
   230		}
   231		if !constantTimeEqual(req.HTTPRequest.Header.Get(CSRFHeaderName), session.CSRFToken) {
   232			return errors.New("missing or invalid X-CSRF-Token")
   233		}
   234		return nil
   235	}
```

## pkg/gojahttp/auth/appauth/appauth.go:80-180
```go
    80	type Resolver struct{ Store ResourceStore }
    81	
    82	func (r Resolver) ResolveResource(ctx context.Context, req gojahttp.ResourceRequest) (*gojahttp.ResourceRef, error) {
    83		if r.Store == nil {
    84			return nil, fmt.Errorf("appauth: resource store is required")
    85		}
    86		resource, err := r.Store.GetResource(ctx, req.Spec.Type, req.ID)
    87		if err != nil {
    88			return nil, err
    89		}
    90		if resource == nil {
    91			return nil, gojahttp.ErrNotFound
    92		}
    93		if req.TenantID != "" && resource.TenantID != req.TenantID {
    94			return nil, gojahttp.ErrNotFound
    95		}
    96		name := req.Spec.Name
    97		if name == "" {
    98			name = resource.Name
    99		}
   100		if name == "" {
   101			name = resource.Type
   102		}
   103		return &gojahttp.ResourceRef{Name: name, Type: resource.Type, ID: resource.ID, TenantID: resource.TenantID, Claims: cloneClaims(resource.Claims)}, nil
   104	}
   105	
   106	// Authorizer implements a small deny-by-default action switch suitable as a
   107	// starting point for monolith apps and demos.
   108	type Authorizer struct{ Memberships MembershipStore }
   109	
   110	func (a Authorizer) Authorize(ctx context.Context, req gojahttp.AuthorizationRequest) (gojahttp.AuthorizationDecision, error) {
   111		if req.Actor == nil {
   112			return deny("missing actor"), nil
   113		}
   114		switch req.Action {
   115		case ActionUserSelfRead:
   116			return allow(), nil
   117		case ActionUserSelfUpdate:
   118			if req.Resource == nil || req.Resource.Type != "user" {
   119				return deny("missing user resource"), nil
   120			}
   121			if req.Resource.ID == req.Actor.ID {
   122				return allow(), nil
   123			}
   124			return deny("cannot update another user"), nil
   125		case ActionProjectRead:
   126			if req.Resource == nil || req.Resource.Type != "project" {
   127				return deny("missing project resource"), nil
   128			}
   129			return a.memberDecision(ctx, req.Actor.ID, req.Resource.TenantID)
   130		case ActionProjectUpdate:
   131			if req.Resource == nil || req.Resource.Type != "project" {
   132				return deny("missing project resource"), nil
   133			}
   134			return a.roleDecision(ctx, req.Actor.ID, req.Resource.TenantID, "admin", "editor")
   135		case ActionOrgInvite:
   136			if req.Resource == nil {
   137				return deny("missing organization resource"), nil
   138			}
   139			tenantID := req.Resource.TenantID
   140			if req.Resource.Type == "org" && tenantID == "" {
   141				tenantID = req.Resource.ID
   142			}
   143			return a.roleDecision(ctx, req.Actor.ID, tenantID, "admin")
   144		default:
   145			return deny("unknown action"), nil
   146		}
   147	}
   148	
   149	func (a Authorizer) memberDecision(ctx context.Context, userID, tenantID string) (gojahttp.AuthorizationDecision, error) {
   150		if a.Memberships == nil {
   151			return deny("membership store is required"), nil
   152		}
   153		ok, err := a.Memberships.IsMember(ctx, userID, tenantID)
   154		if err != nil {
   155			return deny("membership lookup failed"), err
   156		}
   157		if ok {
   158			return allow(), nil
   159		}
   160		return deny("tenant membership required"), nil
   161	}
   162	
   163	func (a Authorizer) roleDecision(ctx context.Context, userID, tenantID string, roles ...string) (gojahttp.AuthorizationDecision, error) {
   164		if a.Memberships == nil {
   165			return deny("membership store is required"), nil
   166		}
   167		ok, err := a.Memberships.HasRole(ctx, userID, tenantID, roles...)
   168		if err != nil {
   169			return deny("role lookup failed"), err
   170		}
   171		if ok {
   172			return allow(), nil
   173		}
   174		return deny("required tenant role missing"), nil
   175	}
   176	
   177	func allow() gojahttp.AuthorizationDecision { return gojahttp.AuthorizationDecision{Allowed: true} }
   178	func deny(reason string) gojahttp.AuthorizationDecision {
   179		return gojahttp.AuthorizationDecision{Allowed: false, Reason: reason}
   180	}
```

## pkg/gojahttp/auth/audit/audit.go:40-105
```go
    40		Attributes   map[string]any `json:"attributes,omitempty"`
    41		CreatedAt    time.Time      `json:"createdAt"`
    42	}
    43	
    44	// Store persists normalized audit records.
    45	type Store interface {
    46		InsertAuditRecord(ctx context.Context, record Record) error
    47	}
    48	
    49	// Normalizer maps gojahttp.AuditEvent values into Records.
    50	type Normalizer struct {
    51		Now    func() time.Time
    52		IPHash func(ip string) string
    53	}
    54	
    55	func (n Normalizer) Normalize(event gojahttp.AuditEvent) Record {
    56		now := time.Now
    57		if n.Now != nil {
    58			now = n.Now
    59		}
    60		ipHasher := hashIP
    61		if n.IPHash != nil {
    62			ipHasher = n.IPHash
    63		}
    64		record := Record{
    65			Event:      event.Event,
    66			Outcome:    event.Outcome,
    67			Reason:     event.Reason,
    68			StatusCode: event.StatusCode,
    69			RouteName:  event.RouteName,
    70			Method:     event.Method,
    71			Pattern:    event.Pattern,
    72			Action:     event.Action,
    73			Attributes: RedactMap(event.Attributes),
    74			CreatedAt:  now(),
    75		}
    76		if event.Actor != nil {
    77			record.ActorID = event.Actor.ID
    78			record.ActorKind = event.Actor.Kind
    79		}
    80		if event.Resource != nil {
    81			record.ResourceType = event.Resource.Type
    82			record.ResourceID = event.Resource.ID
    83			record.TenantID = event.Resource.TenantID
    84		}
    85		if record.TenantID == "" {
    86			for _, resource := range event.Resources {
    87				if resource != nil && resource.TenantID != "" {
    88					record.TenantID = resource.TenantID
    89					break
    90				}
    91			}
    92		}
    93		if event.HTTPRequest != nil {
    94			record.RequestID = firstHeader(event.HTTPRequest, "X-Request-Id", "X-Correlation-Id")
    95			record.UserAgent = event.HTTPRequest.UserAgent()
    96			if ip := clientIP(event.HTTPRequest); ip != "" {
    97				record.IPHash = ipHasher(ip)
    98			}
    99		}
   100		return record
   101	}
   102	
   103	// Sink records audit events into a Store after normalization.
   104	type Sink struct {
   105		Store      Store
```

## pkg/xgoja/hostauth/builder.go:1-125
```go
     1	package hostauth
     2	
     3	import (
     4		"context"
     5		"errors"
     6		"time"
     7	
     8		"github.com/go-go-golems/glazed/pkg/cmds/values"
     9		"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
    10		"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
    11		"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
    12		"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
    13	)
    14	
    15	// BuilderOptions configures a generated-host auth service factory.
    16	type BuilderOptions struct {
    17		Config      Config
    18		ActorLoader sessionauth.ActorLoader
    19		Now         func() time.Time
    20	}
    21	
    22	// Builder is the default hostauth ServiceFactory implementation.
    23	type Builder struct {
    24		options BuilderOptions
    25	}
    26	
    27	var (
    28		_ ServiceFactory = (*Builder)(nil)
    29	
    30		errServiceFactoryNil = errors.New("hostauth service factory is nil")
    31	)
    32	
    33	// NewServiceFactory returns a lazy generated-host auth service factory. The
    34	// factory resolves config and opens stores only when BuildHostAuthServices is
    35	// called, so command providers can discover the factory during command
    36	// construction without touching databases or env-dependent DSNs.
    37	func NewServiceFactory(opts BuilderOptions) *Builder {
    38		return &Builder{options: opts}
    39	}
    40	
    41	// BuildHostAuthServices builds concrete auth services for one command/runtime
    42	// execution. The vals argument is reserved for future Glazed-value overlays;
    43	// this first implementation resolves from BuilderOptions.Config and env refs.
    44	func (b *Builder) AuthConfigDefaults() Config {
    45		if b == nil {
    46			return Config{}
    47		}
    48		return b.options.Config
    49	}
    50	
    51	func (b *Builder) BuildHostAuthServices(ctx context.Context, vals *values.Values) (*Services, error) {
    52		if b == nil {
    53			return nil, errNilBuilder()
    54		}
    55		cfg, err := ConfigFromValues(vals, b.options.Config)
    56		if err != nil {
    57			return nil, err
    58		}
    59		resolved, err := ResolveConfig(cfg, ResolveOptions{})
    60		if err != nil {
    61			return nil, err
    62		}
    63		if resolved.Mode == ModeNone {
    64			return &Services{Config: resolved}, nil
    65		}
    66	
    67		stores, err := BuildStores(ctx, resolved.Stores)
    68		if err != nil {
    69			return nil, err
    70		}
    71		success := false
    72		defer func() {
    73			if !success {
    74				_ = stores.Close(ctx)
    75			}
    76		}()
    77	
    78		sessionManager, err := BuildSessionManager(resolved.Session, stores.Session, b.options.ActorLoader, b.options.Now)
    79		if err != nil {
    80			return nil, err
    81		}
    82		auditSink := audit.Sink{Store: stores.Audit}
    83		authOptions := BuildAuthOptions(sessionManager, stores, auditSink)
    84		services := &Services{
    85			Config:         resolved,
    86			AuthOptions:    authOptions,
    87			SessionManager: sessionManager,
    88			SessionStore:   stores.Session,
    89			AuditSink:      auditSink,
    90			AuditStore:     stores.Audit,
    91			AppAuth:        stores.AppAuth,
    92			Capability:     stores.Capability,
    93			Closers:        stores.Closers,
    94		}
    95		success = true
    96		return services, nil
    97	}
    98	
    99	// BuildSessionManager maps resolved generated-host config into sessionauth.
   100	func BuildSessionManager(cfg ResolvedSessionConfig, store sessionauth.Store, actorLoader sessionauth.ActorLoader, now func() time.Time) (*sessionauth.Manager, error) {
   101		return sessionauth.New(sessionauth.Config{
   102			Store:             store,
   103			ActorLoader:       actorLoader,
   104			CookieName:        cfg.Cookie.Name,
   105			Path:              cfg.Cookie.Path,
   106			SameSite:          cfg.Cookie.SameSite,
   107			IdleTimeout:       cfg.IdleTimeout,
   108			AbsoluteTimeout:   cfg.AbsoluteTimeout,
   109			AllowInsecureHTTP: cfg.Cookie.AllowInsecureHTTP,
   110			Now:               now,
   111		})
   112	}
   113	
   114	// BuildAuthOptions wires a session manager and built auth stores into
   115	// gojahttp's host-owned auth interfaces.
   116	func BuildAuthOptions(sessionManager *sessionauth.Manager, stores *StoreBundle, auditSink gojahttp.AuditSink) gojahttp.AuthOptions {
   117		var options gojahttp.AuthOptions
   118		if sessionManager != nil {
   119			options.Authenticator = sessionManager
   120			options.CSRF = sessionManager
   121		}
   122		if auditSink != nil {
   123			options.Audit = auditSink
   124		}
   125		if stores != nil {
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

## pkg/xgoja/providers/http/http.go:130-230
```go
   130				fields.New("dev-errors", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Return development JavaScript error details from the xgoja-owned HTTP host")),
   131				fields.New("reject-raw-routes", fields.TypeBool, fields.WithDefault(true), fields.WithHelp("Reject matched raw/unplanned routes; planned routes and static mounts are unaffected")),
   132			),
   133		)
   134		return schema.NewSection("http", "HTTP server", options...)
   135	}
   136	
   137	func glazedFieldWasExplicit(field *fields.FieldValue) bool {
   138		if field == nil || len(field.Log) == 0 {
   139			return true
   140		}
   141		for _, step := range field.Log {
   142			if step.Source != fields.SourceDefaults {
   143				return true
   144			}
   145		}
   146		return false
   147	}
   148	
   149	func (c *capability) InitRuntimeFromSections(ctx context.Context, vals *values.Values, handle providerapi.RuntimeInitializerHandle) error {
   150		_ = ctx
   151		if handle == nil || handle.EngineRuntime() == nil || handle.EngineRuntime().VM == nil {
   152			return fmt.Errorf("http provider runtime handle is nil")
   153		}
   154		runtime := handle.EngineRuntime()
   155		cfg := defaultSettings(false)
   156		if vals != nil {
   157			cfg.Enabled = true
   158			if err := vals.DecodeSectionInto("http", &cfg); err != nil {
   159				return err
   160			}
   161		}
   162		entry := c.entry(runtime.VM)
   163		entry.mu.Lock()
   164		entry.settings = normalizeSettings(cfg)
   165		entry.settingsConfigured = true
   166		entry.mu.Unlock()
   167		return runtime.AddCloser(func(ctx context.Context) error {
   168			return c.shutdownRuntime(ctx, runtime.VM)
   169		})
   170	}
   171	
   172	func (c *capability) NewExpressLoader() require.ModuleLoader {
   173		loader, _ := c.newExpressLoader(nil, defaultSettings(true))
   174		return loader
   175	}
   176	
   177	func (c *capability) newExpressLoader(hostServices providerapi.HostServices, cfg settings) (require.ModuleLoader, error) {
   178		externalHost, err := externalHostService(hostServices)
   179		if err != nil {
   180			return nil, err
   181		}
   182		return func(vm *goja.Runtime, moduleObj *goja.Object) {
   183			entry := c.entry(vm)
   184			entry.mu.Lock()
   185			if !entry.settingsConfigured || !settingsEqual(cfg, defaultSettings(true)) {
   186				entry.settings = normalizeSettings(cfg)
   187				entry.settingsConfigured = true
   188			}
   189			if entry.host == nil {
   190				if externalHost.Host != nil {
   191					entry.host = externalHost.Host
   192					entry.external = true
   193					entry.ownsListen = externalHost.OwnsListen
   194				} else {
   195					entry.host = gojahttp.NewHost(hostOptions(entry.settings))
   196					entry.external = false
   197					entry.ownsListen = true
   198				}
   199			}
   200			host := entry.host
   201			entry.mu.Unlock()
   202	
   203			express.NewLoader(host, express.WithOnUse(func(vm *goja.Runtime) error {
   204				return c.start(vm, entry)
   205			}))(vm, moduleObj)
   206		}, nil
   207	}
   208	
   209	func externalHostService(hostServices providerapi.HostServices) (ExternalHostService, error) {
   210		lookup, ok := hostServices.(providerapi.HostServiceLookup)
   211		if !ok || lookup == nil {
   212			return ExternalHostService{}, nil
   213		}
   214		raw, ok := lookup.HostService(HostServiceKey)
   215		if !ok {
   216			return ExternalHostService{}, nil
   217		}
   218		service, ok := raw.(ExternalHostService)
   219		if !ok {
   220			return ExternalHostService{}, fmt.Errorf("http host service %q must be ExternalHostService, got %T", HostServiceKey, raw)
   221		}
   222		if service.Host == nil {
   223			return ExternalHostService{}, fmt.Errorf("http host service %q has nil Host", HostServiceKey)
   224		}
   225		return service, nil
   226	}
   227	
   228	func (c *capability) entry(vm *goja.Runtime) *runtimeEntry {
   229		c.mu.Lock()
   230		defer c.mu.Unlock()
```
