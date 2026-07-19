package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	stdhttp "net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/hotreload"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerutil"
)

type serveHotReloadSettings struct {
	Enabled    bool     `glazed:"hot-reload"`
	WatchRoots []string `glazed:"hot-reload-watch-root"`
	WatchExts  []string `glazed:"hot-reload-watch-ext"`
	SmokePath  string   `glazed:"hot-reload-smoke-path"`
	Poll       string   `glazed:"hot-reload-poll"`
	Debounce   string   `glazed:"hot-reload-debounce"`
	CloseGrace string   `glazed:"hot-reload-close-grace"`
	StatusPath string   `glazed:"hot-reload-status-path"`
}

const serveHotReloadSectionSlug = "http-serve"

func serveCommandJSVerbSources(ctx providerapi.CommandSetContext) (providerapi.JSVerbSourceSet, error) {
	if ctx.Sources == nil {
		return nil, fmt.Errorf("http serve command requires command-scoped runtime sources")
	}
	jsverbSources := ctx.Sources.JSVerbs()
	if jsverbSources == nil || len(jsverbSources.ListJSVerbSources()) == 0 {
		return nil, fmt.Errorf("http serve command requires configured jsverb sources")
	}
	return jsverbSources, nil
}

func newServeCommandSet(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
	jsverbSources, err := serveCommandJSVerbSources(ctx)
	if err != nil {
		return nil, err
	}
	if ctx.RuntimeFactory == nil {
		return nil, fmt.Errorf("http serve command requires runtime factory")
	}
	authFactory, hasAuthFactory, err := hostauth.LookupServiceFactory(ctx.Host)
	if err != nil {
		return nil, err
	}

	sections, err := providerutil.CollectGlazedConfigSections(ctx.SelectedModules, providerapi.SectionRequest{
		CommandProviderID: ctx.Name,
	}, nil)
	if err != nil {
		return nil, err
	}
	commandHTTPDefaults, err := decodeSettingsConfig(ctx.Config)
	if err != nil {
		return nil, fmt.Errorf("decode http serve command config: %w", err)
	}
	for i, section := range sections {
		if section.GetSlug() != "http" {
			continue
		}
		section, err = httpConfigSectionWithDefaults(commandHTTPDefaults, schema.WithPrefix("http-"))
		if err != nil {
			return nil, fmt.Errorf("http serve command config section: %w", err)
		}
		sections[i] = section
		break
	}
	if hasAuthFactory {
		authSection, err := serveAuthSection(authFactory)
		if err != nil {
			return nil, err
		}
		sections = append(sections, authSection)
	}
	hotReloadSection, err := serveHotReloadSection()
	if err != nil {
		return nil, err
	}
	sections = append(sections, hotReloadSection)

	registries, err := jsverbSources.ScanAllJSVerbSources()
	if err != nil {
		return nil, err
	}
	commands := make([]cmds.Command, 0)
	for _, registry := range registries {
		if registry == nil {
			continue
		}
		for _, verb := range registry.Verbs() {
			verb := verb
			registry := registry
			cmd, err := registry.CommandForVerbWithInvoker(verb, func(runCtx context.Context, _ *jsverbs.Registry, verb *jsverbs.VerbSpec, parsedValues *values.Values) (interface{}, error) {
				return serveVerb(runCtx, ctx, registry, verb, parsedValues)
			})
			if err != nil {
				return nil, err
			}
			if len(sections) > 0 {
				if err := addSectionsToServeCommand(cmd.Description(), sections, "http serve runtime"); err != nil {
					return nil, err
				}
			}
			commands = append(commands, cmd)
		}
	}
	return &providerapi.CommandSet{Commands: commands}, nil
}

func serveVerb(ctx context.Context, commandCtx providerapi.CommandSetContext, registry *jsverbs.Registry, verb *jsverbs.VerbSpec, parsedValues *values.Values) (interface{}, error) {
	if registry == nil {
		return nil, fmt.Errorf("jsverb registry is nil")
	}
	if verb == nil {
		return nil, fmt.Errorf("jsverb is nil")
	}
	hotReloadSettings, err := decodeServeHotReloadSettings(parsedValues)
	if err != nil {
		return nil, err
	}
	if hotReloadSettings.Enabled {
		return serveVerbHotReload(ctx, commandCtx, registry, verb, parsedValues, hotReloadSettings)
	}
	httpSettings, err := decodeHTTPServeSettings(parsedValues)
	if err != nil {
		return nil, err
	}
	if !httpSettings.Enabled {
		return nil, fmt.Errorf("http serve requires http.enabled=true")
	}
	authServices, hasAuthFactory, err := buildServeAuthServices(ctx, commandCtx, parsedValues)
	if err != nil {
		return nil, err
	}
	if hasAuthFactory {
		defer func() { _ = authServices.Close(context.Background()) }()
	}

	factory, ok := commandCtx.RuntimeFactory.(providerapi.RuntimeFactoryWithHostServices)
	if !ok || factory == nil {
		return nil, fmt.Errorf("http serve requires runtime factory with per-runtime host services")
	}
	includeGeneratedHost := true
	externalOwnsListen := true
	serveHost := gojahttp.NewHost(hostOptionsWithAuth(httpSettings, authServices))
	if externalHost, err := externalHostService(commandCtx.Host); err != nil {
		return nil, err
	} else if externalHost.Host != nil {
		if authServices != nil {
			externalHost.Host.SetAuthOptions(authServices.AuthOptions)
		}
		serveHost = externalHost.Host
		externalOwnsListen = externalHost.OwnsListen
		includeGeneratedHost = false
	}
	runtimeServices, err := serveRuntimeServices(serveHost, authServices, false, includeGeneratedHost)
	if err != nil {
		return nil, err
	}
	rt, err := factory.NewRuntimeFromSectionsWithHostServices(ctx, parsedValues, runtimeServices, require.WithLoader(registry.RequireLoader()))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rt.Close(context.Background()) }()

	if len(commandCtx.SelectedModules) > 0 {
		if err := providerutil.InitRuntimeFromSections(ctx, parsedValues, runtimeHandle{rt: rt}, commandCtx.SelectedModules); err != nil {
			return nil, err
		}
	}
	if _, err := registry.InvokeInRuntime(ctx, rt, verb, parsedValues); err != nil {
		return nil, err
	}

	if !externalOwnsListen {
		fmt.Fprintln(os.Stderr, "xgoja http serve: routes registered into external host; listener is owned by caller")
		return nil, waitForServeNoListen(ctx)
	}

	topHandler, err := buildServeHandler(serveHost, authServices)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", httpSettings.Listen)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", httpSettings.Listen, err)
	}
	server := &stdhttp.Server{Addr: httpSettings.Listen, Handler: topHandler, ReadHeaderTimeout: 5 * time.Second}
	fmt.Fprintf(os.Stderr, "xgoja http serve: runtime is alive on %s; press Ctrl-C to stop\n", httpSettings.Listen)
	return nil, serveHTTPServer(ctx, server, listener)
}

func serveVerbHotReload(ctx context.Context, commandCtx providerapi.CommandSetContext, registry *jsverbs.Registry, verb *jsverbs.VerbSpec, parsedValues *values.Values, hotReloadSettings serveHotReloadSettings) (interface{}, error) {
	factory, ok := commandCtx.RuntimeFactory.(providerapi.RuntimeFactoryWithHostServices)
	if !ok || factory == nil {
		return nil, fmt.Errorf("http serve hot reload requires runtime factory with per-runtime host services")
	}
	httpSettings, err := decodeHTTPServeSettings(parsedValues)
	if err != nil {
		return nil, err
	}
	authServices, hasAuthFactory, err := buildServeAuthServices(ctx, commandCtx, parsedValues)
	if err != nil {
		return nil, err
	}
	if hasAuthFactory {
		defer func() { _ = authServices.Close(context.Background()) }()
	}
	if !httpSettings.Enabled {
		return nil, fmt.Errorf("http serve hot reload requires http.enabled=true")
	}
	poll, err := parseServeHotReloadDuration("hot-reload-poll", hotReloadSettings.Poll)
	if err != nil {
		return nil, err
	}
	debounce, err := parseServeHotReloadDuration("hot-reload-debounce", hotReloadSettings.Debounce)
	if err != nil {
		return nil, err
	}
	closeGrace, err := parseServeHotReloadDuration("hot-reload-close-grace", hotReloadSettings.CloseGrace)
	if err != nil {
		return nil, err
	}

	jsverbSources, err := serveCommandJSVerbSources(commandCtx)
	if err != nil {
		return nil, err
	}
	verbPath := verb.FullPath()
	manager, err := hotreload.NewManager(hotreload.Options{
		HostOptions: hostOptionsWithAuth(httpSettings, authServices),
		CloseGrace:  closeGrace,
		Load: func(ctx context.Context, candidate hotreload.Candidate) (hotreload.Runtime, error) {
			activeRegistry, activeVerb, err := resolveServeHotReloadVerb(jsverbSources, registry, verb, verbPath)
			if err != nil {
				return nil, err
			}
			services, err := serveRuntimeServices(candidate.Host, authServices, false, true)
			if err != nil {
				return nil, err
			}
			rt, err := factory.NewRuntimeFromSectionsWithHostServices(ctx, parsedValues, services, require.WithLoader(activeRegistry.RequireLoader()))
			if err != nil {
				return nil, err
			}
			if len(commandCtx.SelectedModules) > 0 {
				if err := providerutil.InitRuntimeFromSections(ctx, parsedValues, runtimeHandle{rt: rt}, commandCtx.SelectedModules); err != nil {
					_ = rt.Close(ctx)
					return nil, err
				}
			}
			if _, err := activeRegistry.InvokeInRuntime(ctx, rt, activeVerb, parsedValues); err != nil {
				_ = rt.Close(ctx)
				return nil, err
			}
			return rt, nil
		},
		Smoke: serveHotReloadSmoke(hotReloadSettings.SmokePath),
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = manager.Close(context.Background()) }()

	serveCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	watchRoots := hotReloadSettings.WatchRoots
	if len(watchRoots) == 0 {
		watchRoots = defaultServeHotReloadWatchRoots(jsverbSources)
	}
	if sourceSetHasTypeScript(jsverbSources) {
		hotReloadSettings.WatchExts = appendTypeScriptWatchExtensions(hotReloadSettings.WatchExts)
	}
	if len(watchRoots) > 0 {
		if err := startServeHotReloadWatcher(serveCtx, manager, watchRoots, hotReloadSettings, poll, debounce); err != nil {
			return nil, err
		}
	}

	if _, err := manager.Reload(ctx); err != nil {
		return nil, fmt.Errorf("initial hot reload: %w", err)
	}
	topHandler, err := buildServeHandler(manager, authServices, func(mux *stdhttp.ServeMux) error {
		statusPath := normalizeServeHotReloadStatusPath(hotReloadSettings.StatusPath)
		if statusPath != "" {
			mux.HandleFunc(statusPath, func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(manager.Status())
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", httpSettings.Listen)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", httpSettings.Listen, err)
	}
	server := &stdhttp.Server{Addr: httpSettings.Listen, Handler: topHandler, ReadHeaderTimeout: 5 * time.Second}
	fmt.Fprintf(os.Stderr, "xgoja http serve: hot reload runtime is alive on %s; press Ctrl-C to stop\n", httpSettings.Listen)
	return nil, serveHTTPServer(serveCtx, server, listener)
}

func startServeHotReloadWatcher(ctx context.Context, manager *hotreload.Manager, watchRoots []string, hotReloadSettings serveHotReloadSettings, poll, debounce time.Duration) error {
	baselineReady := make(chan struct{})
	initialErr := make(chan error, 1)
	var baselineClosed atomic.Bool
	go func() {
		err := manager.Watch(ctx, hotreload.WatchOptions{
			Roots:        watchRoots,
			Extensions:   hotReloadSettings.WatchExts,
			PollInterval: poll,
			Debounce:     debounce,
			OnBaseline: func() {
				if baselineClosed.CompareAndSwap(false, true) {
					close(baselineReady)
				}
			},
			OnReload: func(snapshot *hotreload.Snapshot) {
				fmt.Fprintf(os.Stderr, "xgoja http serve: hot reloaded version %d (%d routes)\n", snapshot.Version, len(snapshot.Routes))
			},
			OnError: func(err error) {
				fmt.Fprintf(os.Stderr, "xgoja http serve: hot reload failed: %v\n", err)
			},
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			if baselineClosed.Load() {
				fmt.Fprintf(os.Stderr, "xgoja http serve: hot reload watcher stopped: %v\n", err)
				return
			}
			initialErr <- err
		}
	}()

	select {
	case <-baselineReady:
		return nil
	case err := <-initialErr:
		return fmt.Errorf("initialize hot reload watcher: %w", err)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func resolveServeHotReloadVerb(sources providerapi.JSVerbSourceSet, fallbackRegistry *jsverbs.Registry, fallbackVerb *jsverbs.VerbSpec, fullPath string) (*jsverbs.Registry, *jsverbs.VerbSpec, error) {
	fullPath = strings.TrimSpace(fullPath)
	if sources == nil {
		return fallbackRegistry, fallbackVerb, nil
	}
	registries, err := sources.ScanAllJSVerbSources()
	if err != nil {
		return nil, nil, err
	}
	for _, registry := range registries {
		if registry == nil {
			continue
		}
		if verb, ok := registry.Verb(fullPath); ok {
			return registry, verb, nil
		}
	}
	return nil, nil, fmt.Errorf("hot reload could not find jsverb %q after rescan", fullPath)
}

func serveHotReloadSmoke(path string) hotreload.SmokeFunc {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return func(_ context.Context, snapshot *hotreload.Snapshot) error {
		if snapshot == nil || snapshot.Host == nil {
			return fmt.Errorf("hot reload smoke requires candidate host")
		}
		recorder := httptest.NewRecorder()
		snapshot.Host.ServeHTTP(recorder, httptest.NewRequest(stdhttp.MethodGet, path, nil))
		if recorder.Code < 200 || recorder.Code >= 300 {
			return fmt.Errorf("hot reload smoke GET %s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
		return nil
	}
}

func decodeHTTPServeSettings(vals *values.Values) (settings, error) {
	cfg := settings{Enabled: true, Listen: "127.0.0.1:8787"}
	if vals != nil {
		if err := vals.DecodeSectionInto("http", &cfg); err != nil {
			return settings{}, err
		}
	}
	return normalizeSettings(cfg), nil
}

func buildServeAuthServices(ctx context.Context, commandCtx providerapi.CommandSetContext, parsedValues *values.Values) (*hostauth.Services, bool, error) {
	factory, ok, err := hostauth.LookupServiceFactory(commandCtx.Host)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	services, err := factory.BuildHostAuthServices(ctx, parsedValues)
	if err != nil {
		return nil, false, err
	}
	if services == nil {
		return nil, false, fmt.Errorf("hostauth service factory returned nil services")
	}
	return services, true, nil
}

func hostOptionsWithAuth(cfg settings, authServices *hostauth.Services) gojahttp.HostOptions {
	opts := hostOptions(cfg)
	if authServices != nil {
		opts.Auth = authServices.AuthOptions
	}
	return opts
}

type serveMuxMount func(*stdhttp.ServeMux) error

func buildServeHandler(appHost stdhttp.Handler, authServices *hostauth.Services, mounts ...serveMuxMount) (stdhttp.Handler, error) {
	if appHost == nil {
		return nil, fmt.Errorf("http serve app host is nil")
	}
	mux := stdhttp.NewServeMux()
	if authServices != nil {
		for _, route := range authServices.NativeHandlers {
			method := strings.ToUpper(strings.TrimSpace(route.Method))
			path := strings.TrimSpace(route.Path)
			if method == "" || path == "" || route.Handler == nil {
				return nil, fmt.Errorf("http serve native auth handler requires method, path, and handler")
			}
			if !strings.HasPrefix(path, "/") {
				return nil, fmt.Errorf("http serve native auth handler path %q must start with /", path)
			}
			if err := muxHandle(mux, method+" "+path, route.Handler); err != nil {
				return nil, err
			}
		}
	}
	for _, mount := range mounts {
		if mount == nil {
			continue
		}
		if err := mount(mux); err != nil {
			return nil, err
		}
	}
	mux.Handle("/", appHost)
	if authServices != nil {
		return gojahttp.RequestIdentityMiddleware(authServices.RequestIdentity, mux), nil
	}
	return mux, nil
}

func muxHandle(mux *stdhttp.ServeMux, pattern string, handler stdhttp.Handler) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("mount http serve handler %q: %v", pattern, recovered)
		}
	}()
	mux.Handle(pattern, handler)
	return nil
}

func serveRuntimeServices(host *gojahttp.Host, authServices *hostauth.Services, ownsListen bool, includeHost bool) (app.HostServices, error) {
	services := app.HostServices{}
	if includeHost {
		if err := services.SetHostService(HostServiceKey, ExternalHostService{Host: host, OwnsListen: ownsListen}); err != nil {
			return app.HostServices{}, err
		}
	}
	if authServices != nil {
		if err := services.SetHostService(hostauth.ServicesKey, authServices); err != nil {
			return app.HostServices{}, err
		}
	}
	return services, nil
}

func parseServeHotReloadDuration(name, raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s %q: %w", name, raw, err)
	}
	if d < 0 {
		return 0, fmt.Errorf("%s must not be negative", name)
	}
	return d, nil
}

func sourceSetHasTypeScript(sources providerapi.JSVerbSourceSet) bool {
	if sources == nil {
		return false
	}
	for _, source := range sources.ListJSVerbSources() {
		if source.TypeScript != nil && source.TypeScript.Enabled {
			return true
		}
	}
	return false
}

func appendTypeScriptWatchExtensions(exts []string) []string {
	out := append([]string(nil), exts...)
	seen := map[string]struct{}{}
	for _, ext := range out {
		seen[strings.ToLower(strings.TrimSpace(ext))] = struct{}{}
	}
	for _, ext := range []string{".ts", ".tsx", ".mts", ".cts"} {
		if _, ok := seen[ext]; ok {
			continue
		}
		out = append(out, ext)
		seen[ext] = struct{}{}
	}
	return out
}

func defaultServeHotReloadWatchRoots(sources providerapi.JSVerbSourceSet) []string {
	if sources == nil {
		return nil
	}
	out := []string{}
	seen := map[string]struct{}{}
	for _, source := range sources.ListJSVerbSources() {
		path := strings.TrimSpace(source.Path)
		if path == "" || source.Embed || source.Provider != "" || source.Source != "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out
}

func normalizeServeHotReloadStatusPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func serveAuthSection(factory hostauth.ServiceFactory) (schema.Section, error) {
	base := hostauth.Config{}
	if defaultsProvider, ok := factory.(hostauth.ConfigDefaultsProvider); ok {
		base = defaultsProvider.AuthConfigDefaults()
	}
	return hostauth.GlazedConfigSection(base)
}

func serveHotReloadSection() (schema.Section, error) {
	return schema.NewSection(
		serveHotReloadSectionSlug,
		"HTTP serve hot reload",
		schema.WithFields(
			fields.New("hot-reload", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Enable blue/green hot reload for this HTTP serve command")),
			fields.New("hot-reload-watch-root", fields.TypeStringList, fields.WithHelp("File or directory to poll for reload changes; repeatable")),
			fields.New("hot-reload-watch-ext", fields.TypeStringList, fields.WithDefault([]string{".js", ".json", ".md", ".yaml", ".yml"}), fields.WithHelp("File extension that triggers hot reload; repeatable")),
			fields.New("hot-reload-smoke-path", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Optional candidate HTTP path to GET before swapping the reloaded runtime live")),
			fields.New("hot-reload-poll", fields.TypeString, fields.WithDefault((500*time.Millisecond).String()), fields.WithHelp("Polling interval for hot reload file watching, parsed as a Go duration")),
			fields.New("hot-reload-debounce", fields.TypeString, fields.WithDefault((250*time.Millisecond).String()), fields.WithHelp("Debounce delay after a watched file change, parsed as a Go duration")),
			fields.New("hot-reload-close-grace", fields.TypeString, fields.WithDefault((2*time.Second).String()), fields.WithHelp("Delay before closing a retired runtime after a successful hot reload swap")),
			fields.New("hot-reload-status-path", fields.TypeString, fields.WithDefault("/__xgoja/status"), fields.WithHelp("Optional Go-owned status endpoint path for hot reload state; empty disables it")),
		),
	)
}

func decodeServeHotReloadSettings(vals *values.Values) (serveHotReloadSettings, error) {
	settings := serveHotReloadSettings{
		WatchExts:  []string{".js", ".json", ".md", ".yaml", ".yml"},
		Poll:       (500 * time.Millisecond).String(),
		Debounce:   (250 * time.Millisecond).String(),
		CloseGrace: (2 * time.Second).String(),
		StatusPath: "/__xgoja/status",
	}
	if vals == nil {
		return settings, nil
	}
	if err := vals.DecodeSectionInto(serveHotReloadSectionSlug, &settings); err != nil {
		return serveHotReloadSettings{}, err
	}
	return settings, nil
}

func waitForServeNoListen(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	serveCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-serveCtx.Done()
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func serveHTTPServer(ctx context.Context, server *stdhttp.Server, listener net.Listener) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if server == nil {
		return fmt.Errorf("http serve server is nil")
	}
	if listener == nil {
		return fmt.Errorf("http serve listener is nil")
	}
	serveCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Serve(listener)
	}()
	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
			return err
		}
	case <-serveCtx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func addSectionsToServeCommand(desc *cmds.CommandDescription, sections []schema.Section, source string) error {
	if desc == nil {
		return fmt.Errorf("command description is nil")
	}
	seen := map[string]string{}
	if desc.Schema != nil {
		desc.Schema.ForEach(func(slug string, _ schema.Section) {
			seen[slug] = "command schema"
		})
	}
	collected := []schema.Section{}
	if err := providerutil.AppendUniqueSections(&collected, seen, sections, source); err != nil {
		return err
	}
	desc.SetSections(collected...)
	return nil
}

type runtimeHandle struct {
	rt *engine.Runtime
}

func (h runtimeHandle) EngineRuntime() *engine.Runtime { return h.rt }

func (h runtimeHandle) Close(ctx context.Context) error {
	if h.rt == nil {
		return nil
	}
	return h.rt.Close(ctx)
}
