package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/dop251/goja_nodejs/url"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

type runtimebridgeOwner struct {
	owner runtimeowner.RuntimeOwner
}

func (o runtimebridgeOwner) Call(ctx context.Context, op string, fn func(context.Context, *goja.Runtime) (any, error)) (any, error) {
	return o.owner.Call(ctx, op, runtimeowner.CallFunc(fn))
}

func (o runtimebridgeOwner) Post(ctx context.Context, op string, fn func(context.Context, *goja.Runtime)) error {
	return o.owner.Post(ctx, op, runtimeowner.PostFunc(fn))
}

// FactoryBuilder composes explicit module and runtime initializer configuration
// before producing an immutable Factory.
type FactoryBuilder struct {
	settings builderSettings

	modules             []RuntimeModuleSpec
	moduleMiddlewares   []ModuleMiddleware
	runtimeInitializers []RuntimeInitializer
	built               bool
}

// Factory creates runtime instances from an immutable build plan.
type Factory struct {
	settings            builderSettings
	modules             []RuntimeModuleSpec
	runtimeInitializers []RuntimeInitializer
}

// NewBuilder starts a new explicit runtime composition flow.
func NewBuilder(opts ...Option) *FactoryBuilder {
	settings := defaultBuilderSettings()
	for _, opt := range opts {
		if opt != nil {
			opt(&settings)
		}
	}
	return &FactoryBuilder{
		settings: settings,
	}
}

func (b *FactoryBuilder) assertMutable() {
	if b == nil {
		panic("engine builder is nil")
	}
	if b.built {
		panic("engine builder is already built and immutable")
	}
}

// WithRequireOptions appends require options to the current builder.
func (b *FactoryBuilder) WithRequireOptions(opts ...require.Option) *FactoryBuilder {
	b.assertMutable()
	b.settings.requireOptions = append(b.settings.requireOptions, opts...)
	return b
}

// WithModules appends runtime-aware module registrations.
func (b *FactoryBuilder) WithModules(mods ...RuntimeModuleSpec) *FactoryBuilder {
	b.assertMutable()
	b.modules = append(b.modules, mods...)
	return b
}

// UseModuleMiddleware appends module-selection middlewares. A plain builder with
// no explicit modules enables all default-registry modules; when middlewares are
// present, the builder evaluates the pipeline at Build() time and converts the
// resulting module names into NativeModuleSpec registrations. This is the
// preferred way to control which default-registry modules are loaded.
//
// Middlewares are applied in order: the first middleware wraps the subsequent
// ones. Override middlewares (Safe, Only) replace the selection; transform
// middlewares (Exclude, Add, Custom) modify the result of the next handler.
func (b *FactoryBuilder) UseModuleMiddleware(mw ...ModuleMiddleware) *FactoryBuilder {
	b.assertMutable()
	b.moduleMiddlewares = append(b.moduleMiddlewares, mw...)
	return b
}

// WithRuntimeInitializers appends runtime initialization hooks executed for
// each created runtime instance.
func (b *FactoryBuilder) WithRuntimeInitializers(inits ...RuntimeInitializer) *FactoryBuilder {
	b.assertMutable()
	b.runtimeInitializers = append(b.runtimeInitializers, inits...)
	return b
}

func validateUniqueIDs[T interface{ ID() string }](entries []T, kind string) error {
	seen := map[string]int{}
	for i, entry := range entries {
		id := strings.TrimSpace(entry.ID())
		if id == "" {
			return fmt.Errorf("%s at index %d has empty ID", kind, i)
		}
		if j, ok := seen[id]; ok {
			return fmt.Errorf("duplicate %s ID %q at indexes %d and %d", kind, id, j, i)
		}
		seen[id] = i
	}
	return nil
}

// Build validates and freezes the composition into an immutable Factory.
func (b *FactoryBuilder) Build() (*Factory, error) {
	if b == nil {
		return nil, fmt.Errorf("engine builder is nil")
	}
	b.assertMutable()

	modules_ := make([]RuntimeModuleSpec, 0, len(b.modules))
	for i, mod := range b.modules {
		if mod == nil {
			return nil, fmt.Errorf("module spec at index %d is nil", i)
		}
		modules_ = append(modules_, mod)
	}

	// Evaluate module middleware pipeline and convert selected names to specs.
	// A plain NewBuilder().Build() preserves the historical default of exposing
	// all default-registry modules. Calling UseModuleMiddleware narrows or
	// transforms that selection; explicit WithModules(...) remains explicit and
	// does not auto-append the default registry.
	if len(b.moduleMiddlewares) > 0 || (len(b.modules) == 0 && b.settings.implicitDefaultRegistryModules) {
		selector := SelectAll
		for i := len(b.moduleMiddlewares) - 1; i >= 0; i-- {
			selector = b.moduleMiddlewares[i](selector)
		}
		selected := sortedUnique(selector(allRegisteredModuleNames()))
		for _, name := range selected {
			modules_ = append(modules_, defaultRegistryModule(name))
		}
	}

	inits := make([]RuntimeInitializer, 0, len(b.runtimeInitializers))
	for i, init := range b.runtimeInitializers {
		if init == nil {
			return nil, fmt.Errorf("runtime initializer at index %d is nil", i)
		}
		inits = append(inits, init)
	}

	if err := validateUniqueIDs(modules_, "module"); err != nil {
		return nil, err
	}
	if err := validateUniqueIDs(inits, "runtime initializer"); err != nil {
		return nil, err
	}

	b.built = true

	return &Factory{
		settings: builderSettings{
			requireOptions:                 append([]require.Option(nil), b.settings.requireOptions...),
			implicitDefaultRegistryModules: b.settings.implicitDefaultRegistryModules,
			dataOnlyDefaultRegistryModules: b.settings.dataOnlyDefaultRegistryModules,
		},
		modules:             append([]RuntimeModuleSpec(nil), modules_...),
		runtimeInitializers: append([]RuntimeInitializer(nil), inits...),
	}, nil
}

// NewRuntime creates a new owned runtime instance from this factory's frozen
// composition plan.
func (f *Factory) NewRuntime(opts ...RuntimeOption) (*Runtime, error) {
	if f == nil {
		return nil, fmt.Errorf("factory is nil")
	}
	settings := defaultRuntimeOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&settings)
		}
	}
	startupCtx := settings.startupContext
	if startupCtx == nil {
		startupCtx = context.Background()
	}
	lifetimeCtx := settings.lifetimeContext
	if lifetimeCtx == nil {
		lifetimeCtx = context.Background()
	}
	select {
	case <-startupCtx.Done():
		return nil, startupCtx.Err()
	default:
	}

	vm := goja.New()
	loop := eventloop.NewEventLoop()
	go loop.Start()

	owner := runtimeowner.NewRuntimeOwner(vm, loop, runtimeowner.Options{
		Name:          "go-go-goja-runtime",
		RecoverPanics: true,
	})
	// #nosec G118 -- the runtime owns this cancel func and calls it on close and on setup failures.
	runtimeCtx, runtimeCtxCancel := context.WithCancel(lifetimeCtx)
	runtimeValues := map[string]any{}

	rt := &Runtime{
		VM:               vm,
		Loop:             loop,
		Owner:            owner,
		Values:           runtimeValues,
		runtimeCtx:       runtimeCtx,
		runtimeCtxCancel: runtimeCtxCancel,
	}

	runtimebridge.Store(vm, runtimebridge.RuntimeServices{
		LifetimeContext: runtimeCtx,
		Loop:            loop,
		Owner:           runtimebridgeOwner{owner: owner},
	})

	reg := require.NewRegistry(f.settings.requireOptions...)
	moduleCtx := &RuntimeModuleContext{
		Context:   startupCtx,
		VM:        vm,
		Loop:      loop,
		Owner:     owner,
		AddCloser: rt.AddCloser,
		Values:    runtimeValues,
	}
	if f.settings.dataOnlyDefaultRegistryModules {
		if err := dataOnlyDefaultRegistryModules().RegisterRuntimeModule(moduleCtx, reg); err != nil {
			_ = rt.Close(startupCtx)
			return nil, fmt.Errorf("register data-only default modules: %w", err)
		}
	}
	for _, mod := range f.modules {
		if err := mod.RegisterRuntimeModule(moduleCtx, reg); err != nil {
			_ = rt.Close(startupCtx)
			return nil, fmt.Errorf("register module %q: %w", mod.ID(), err)
		}
	}

	reqMod := reg.Enable(vm)
	console.Enable(vm)
	buffer.Enable(vm)
	url.Enable(vm)
	if err := installPerformanceGlobals(vm); err != nil {
		_ = rt.Close(startupCtx)
		return nil, err
	}
	if err := installConsoleTimers(vm); err != nil {
		_ = rt.Close(startupCtx)
		return nil, err
	}
	rt.Require = reqMod

	initCtx := &RuntimeContext{
		Context: startupCtx,
		VM:      vm,
		Require: reqMod,
		Loop:    loop,
		Owner:   owner,
		Values:  rt.Values,
	}
	for _, init := range f.runtimeInitializers {
		if err := init.InitRuntime(initCtx); err != nil {
			_ = rt.Close(startupCtx)
			return nil, fmt.Errorf("runtime initializer %q: %w", init.ID(), err)
		}
	}

	return rt, nil
}
