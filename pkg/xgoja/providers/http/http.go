package http

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/modules/express"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

const PackageID = "go-go-goja-http"

const HostServiceKey = "go-go-goja-http.host"

type ExternalHostService struct {
	Host       *gojahttp.Host
	OwnsListen bool
}

func Register(registry *providerapi.ProviderRegistry) error {
	capability := newHTTPCapability()
	return registry.Package(PackageID,
		providerapi.Module{
			Name:        "express",
			DefaultAs:   "express",
			Description: "Express-style HTTP route registration backed by gojahttp",
			TypeScript:  express.NewRegistrar(nil).TypeScriptModule(),
			NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
				cfg, err := decodeSettingsConfig(ctx.Config)
				if err != nil {
					return nil, err
				}
				return capability.newExpressLoader(ctx.Host, cfg)
			},
		},
		providerapi.WithPackageCapability(capability),
		providerapi.CommandSetProvider{
			Name:         "serve",
			DefaultMount: "serve",
			Description:  "Serve JavaScript verb-backed HTTP sites",
			NewCommandSet: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
				return newServeCommandSet(ctx)
			},
		},
	)
}

type settings struct {
	Enabled         bool   `glazed:"enabled"`
	Listen          string `glazed:"listen"`
	DevErrors       bool   `glazed:"dev-errors"`
	RejectRawRoutes bool   `glazed:"reject-raw-routes"`
}

type runtimeEntry struct {
	mu                 sync.Mutex
	settings           settings
	settingsConfigured bool
	host               *gojahttp.Host
}

type capability struct {
	mu      sync.Mutex
	entries map[*goja.Runtime]*runtimeEntry
}

func newHTTPCapability() *capability {
	return &capability{entries: map[*goja.Runtime]*runtimeEntry{}}
}

func (c *capability) CapabilityID() string { return "go-go-goja-http.config" }

func (c *capability) GlazedConfigSections(providerapi.SectionRequest) ([]schema.Section, error) {
	section, err := httpConfigSection(schema.WithPrefix("http-"))
	if err != nil {
		return nil, err
	}
	return []schema.Section{section}, nil
}

func (c *capability) XGojaConfigSection(_ providerapi.SectionRequest, _ providerapi.ModuleDescriptor) (schema.Section, error) {
	return httpConfigSection()
}

func (c *capability) XGojaConfigFromGlazed(_ context.Context, req providerapi.XGojaConfigRequest) (*values.SectionValues, error) {
	out, err := values.NewSectionValues(req.ConfigSection)
	if err != nil {
		return nil, err
	}
	if req.GlazedValues == nil {
		return out, nil
	}
	for _, name := range []string{"enabled", "listen", "dev-errors", "reject-raw-routes"} {
		field, ok := req.GlazedValues.GetField("http", name)
		if !ok || !glazedFieldWasExplicit(field) {
			continue
		}
		definition, ok := req.ConfigSection.GetDefinitions().Get(name)
		if !ok {
			return nil, fmt.Errorf("internal http config field %q not found", name)
		}
		if err := out.Fields.UpdateWithLog(name, definition, field.Value, field.Log...); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func httpConfigSection(options ...schema.SectionOption) (schema.Section, error) {
	options = append(options,
		schema.WithFields(
			fields.New("enabled", fields.TypeBool, fields.WithDefault(true), fields.WithHelp("Start the xgoja HTTP server for modules such as express")),
			fields.New("listen", fields.TypeString, fields.WithDefault("127.0.0.1:8787"), fields.WithHelp("HTTP listen address for xgoja-owned HTTP modules")),
			fields.New("dev-errors", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Return development JavaScript error details from the xgoja-owned HTTP host")),
			fields.New("reject-raw-routes", fields.TypeBool, fields.WithDefault(true), fields.WithHelp("Reject matched raw/unplanned routes; planned routes and static mounts are unaffected")),
		),
	)
	return schema.NewSection("http", "HTTP server", options...)
}

func glazedFieldWasExplicit(field *fields.FieldValue) bool {
	if field == nil || len(field.Log) == 0 {
		return true
	}
	for _, step := range field.Log {
		if step.Source != fields.SourceDefaults {
			return true
		}
	}
	return false
}

func (c *capability) InitRuntimeFromSections(ctx context.Context, vals *values.Values, handle providerapi.RuntimeInitializerHandle) error {
	_ = ctx
	if handle == nil || handle.EngineRuntime() == nil || handle.EngineRuntime().VM == nil {
		return fmt.Errorf("http provider runtime handle is nil")
	}
	runtime := handle.EngineRuntime()
	cfg := defaultSettings(false)
	if vals != nil {
		cfg.Enabled = true
		if err := vals.DecodeSectionInto("http", &cfg); err != nil {
			return err
		}
	}
	entry := c.entry(runtime.VM)
	entry.mu.Lock()
	entry.settings = normalizeSettings(cfg)
	entry.settingsConfigured = true
	entry.mu.Unlock()
	return runtime.AddCloser(func(context.Context) error {
		c.cleanupRuntime(runtime.VM)
		return nil
	})
}

func (c *capability) NewExpressLoader() require.ModuleLoader {
	loader, _ := c.newExpressLoader(nil, defaultSettings(true))
	return loader
}

func (c *capability) newExpressLoader(hostServices providerapi.HostServices, cfg settings) (require.ModuleLoader, error) {
	externalHost, err := externalHostService(hostServices)
	if err != nil {
		return nil, err
	}
	return func(vm *goja.Runtime, moduleObj *goja.Object) {
		entry := c.entry(vm)
		entry.mu.Lock()
		if !entry.settingsConfigured || !settingsEqual(cfg, defaultSettings(true)) {
			entry.settings = normalizeSettings(cfg)
			entry.settingsConfigured = true
		}
		if entry.host == nil {
			if externalHost.Host != nil {
				entry.host = externalHost.Host
			} else {
				entry.host = gojahttp.NewHost(hostOptions(entry.settings))
			}
		}
		host := entry.host
		entry.mu.Unlock()

		express.NewLoader(host)(vm, moduleObj)
	}, nil
}

func externalHostService(hostServices providerapi.HostServices) (ExternalHostService, error) {
	lookup, ok := hostServices.(providerapi.HostServiceLookup)
	if !ok || lookup == nil {
		return ExternalHostService{}, nil
	}
	raw, ok := lookup.HostService(HostServiceKey)
	if !ok {
		return ExternalHostService{}, nil
	}
	service, ok := raw.(ExternalHostService)
	if !ok {
		return ExternalHostService{}, fmt.Errorf("http host service %q must be ExternalHostService, got %T", HostServiceKey, raw)
	}
	if service.Host == nil {
		return ExternalHostService{}, fmt.Errorf("http host service %q has nil Host", HostServiceKey)
	}
	return service, nil
}

func (c *capability) entry(vm *goja.Runtime) *runtimeEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[vm]
	if !ok {
		entry = &runtimeEntry{settings: defaultSettings(true)}
		c.entries[vm] = entry
	}
	return entry
}

func (c *capability) cleanupRuntime(vm *goja.Runtime) {
	c.mu.Lock()
	delete(c.entries, vm)
	c.mu.Unlock()
}

func defaultSettings(enabled bool) settings {
	return settings{Enabled: enabled, Listen: "127.0.0.1:8787", RejectRawRoutes: true}
}

func hostOptions(cfg settings) gojahttp.HostOptions {
	return gojahttp.HostOptions{Dev: cfg.DevErrors, RejectRawRoutes: cfg.RejectRawRoutes}
}

func settingsEqual(a, b settings) bool {
	a = normalizeSettings(a)
	b = normalizeSettings(b)
	return a.Enabled == b.Enabled && a.Listen == b.Listen && a.DevErrors == b.DevErrors && a.RejectRawRoutes == b.RejectRawRoutes
}

func decodeSettingsConfig(data json.RawMessage) (settings, error) {
	cfg := defaultSettings(true)
	if len(data) == 0 || string(data) == "null" {
		return cfg, nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return settings{}, fmt.Errorf("decode http provider config: %w", err)
	}
	if value, ok := raw["enabled"]; ok {
		if err := json.Unmarshal(value, &cfg.Enabled); err != nil {
			return settings{}, fmt.Errorf("decode http provider config enabled: %w", err)
		}
	}
	if value, ok := raw["listen"]; ok {
		if err := json.Unmarshal(value, &cfg.Listen); err != nil {
			return settings{}, fmt.Errorf("decode http provider config listen: %w", err)
		}
	}
	if value, ok := raw["dev-errors"]; ok {
		if err := json.Unmarshal(value, &cfg.DevErrors); err != nil {
			return settings{}, fmt.Errorf("decode http provider config dev-errors: %w", err)
		}
	}
	if value, ok := raw["reject-raw-routes"]; ok {
		if err := json.Unmarshal(value, &cfg.RejectRawRoutes); err != nil {
			return settings{}, fmt.Errorf("decode http provider config reject-raw-routes: %w", err)
		}
	}
	return normalizeSettings(cfg), nil
}

func normalizeSettings(cfg settings) settings {
	cfg.Listen = strings.TrimSpace(cfg.Listen)
	if cfg.Listen == "" {
		cfg.Listen = "127.0.0.1:8787"
	}
	return cfg
}
