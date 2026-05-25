package http

import (
	"context"
	"errors"
	"fmt"
	stdhttp "net/http"
	"strings"
	"sync"
	"time"

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

func Register(registry *providerapi.Registry) error {
	capability := newHTTPCapability()
	return registry.Package(PackageID,
		providerapi.Module{
			Name:        "express",
			DefaultAs:   "express",
			Description: "Express-style HTTP route registration backed by gojahttp",
			New: func(providerapi.ModuleContext) (require.ModuleLoader, error) {
				return capability.NewExpressLoader(), nil
			},
		},
		providerapi.WithPackageCapability(capability),
	)
}

type settings struct {
	Enabled bool   `glazed:"enabled"`
	Listen  string `glazed:"listen"`
}

type runtimeEntry struct {
	mu       sync.Mutex
	settings settings
	host     *gojahttp.Host
	server   *stdhttp.Server
}

type capability struct {
	mu      sync.Mutex
	entries map[*goja.Runtime]*runtimeEntry
}

func newHTTPCapability() *capability {
	return &capability{entries: map[*goja.Runtime]*runtimeEntry{}}
}

func (c *capability) CapabilityID() string { return "go-go-goja-http.config" }

func (c *capability) ConfigSections(providerapi.SectionContext) ([]schema.Section, error) {
	section, err := schema.NewSection(
		"http",
		"HTTP server",
		schema.WithPrefix("http-"),
		schema.WithFields(
			fields.New("enabled", fields.TypeBool, fields.WithDefault(true), fields.WithHelp("Start the xgoja HTTP server for modules such as express")),
			fields.New("listen", fields.TypeString, fields.WithDefault("127.0.0.1:8787"), fields.WithHelp("HTTP listen address for xgoja-owned HTTP modules")),
		),
	)
	if err != nil {
		return nil, err
	}
	return []schema.Section{section}, nil
}

func (c *capability) InitRuntimeFromSections(ctx context.Context, vals *values.Values, handle providerapi.RuntimeHandle) error {
	_ = ctx
	if handle == nil || handle.Runtime() == nil {
		return fmt.Errorf("http provider runtime handle is nil")
	}
	cfg := settings{Enabled: false, Listen: "127.0.0.1:8787"}
	if vals != nil {
		cfg.Enabled = true
		if err := vals.DecodeSectionInto("http", &cfg); err != nil {
			return err
		}
	}
	entry := c.entry(handle.Runtime())
	entry.mu.Lock()
	entry.settings = normalizeSettings(cfg)
	entry.mu.Unlock()
	if closer, ok := handle.(providerapi.RuntimeCloserRegistry); ok {
		return closer.AddCloser(func(ctx context.Context) error {
			return c.shutdownRuntime(ctx, handle.Runtime())
		})
	}
	return nil
}

func (c *capability) NewExpressLoader() require.ModuleLoader {
	return func(vm *goja.Runtime, moduleObj *goja.Object) {
		entry := c.entry(vm)
		entry.mu.Lock()
		if entry.host == nil {
			entry.host = gojahttp.NewHost(gojahttp.HostOptions{})
		}
		host := entry.host
		entry.mu.Unlock()

		if err := c.start(vm, entry); err != nil {
			panic(vm.NewGoError(err))
		}
		express.NewLoader(host)(vm, moduleObj)
	}
}

func (c *capability) entry(vm *goja.Runtime) *runtimeEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[vm]
	if !ok {
		entry = &runtimeEntry{settings: settings{Enabled: true, Listen: "127.0.0.1:8787"}}
		c.entries[vm] = entry
	}
	return entry
}

func (c *capability) start(vm *goja.Runtime, entry *runtimeEntry) error {
	entry.mu.Lock()
	defer entry.mu.Unlock()
	cfg := normalizeSettings(entry.settings)
	entry.settings = cfg
	if !cfg.Enabled {
		return nil
	}
	if entry.host == nil {
		entry.host = gojahttp.NewHost(gojahttp.HostOptions{})
	}
	if entry.server != nil {
		return nil
	}
	server := &stdhttp.Server{Addr: cfg.Listen, Handler: entry.host, ReadHeaderTimeout: 5 * time.Second}
	entry.server = server
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
			fmt.Printf("xgoja http server failed on %s: %v\n", cfg.Listen, err)
		}
	}()
	_ = vm
	return nil
}

func (c *capability) shutdownRuntime(ctx context.Context, vm *goja.Runtime) error {
	c.mu.Lock()
	entry := c.entries[vm]
	delete(c.entries, vm)
	c.mu.Unlock()
	if entry == nil {
		return nil
	}
	entry.mu.Lock()
	server := entry.server
	entry.server = nil
	entry.mu.Unlock()
	if server == nil {
		return nil
	}
	return server.Shutdown(ctx)
}

func normalizeSettings(cfg settings) settings {
	cfg.Listen = strings.TrimSpace(cfg.Listen)
	if cfg.Listen == "" {
		cfg.Listen = "127.0.0.1:8787"
	}
	return cfg
}
