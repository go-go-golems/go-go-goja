package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

// FactoryBuilder composes explicit module and runtime initializer configuration
// before producing an immutable Factory.
type FactoryBuilder struct {
	settings builderSettings

	modules                 []ModuleSpec
	runtimeModuleRegistrars []RuntimeModuleRegistrar
	runtimeInitializers     []RuntimeInitializer
	built                   bool
}

// Factory creates runtime instances from an immutable build plan.
type Factory struct {
	settings                builderSettings
	modules                 []ModuleSpec
	runtimeModuleRegistrars []RuntimeModuleRegistrar
	runtimeInitializers     []RuntimeInitializer
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

// WithModules appends static module registrations.
func (b *FactoryBuilder) WithModules(mods ...ModuleSpec) *FactoryBuilder {
	b.assertMutable()
	b.modules = append(b.modules, mods...)
	return b
}

// WithRuntimeModuleRegistrars appends runtime-scoped module registration hooks.
func (b *FactoryBuilder) WithRuntimeModuleRegistrars(registrars ...RuntimeModuleRegistrar) *FactoryBuilder {
	b.assertMutable()
	b.runtimeModuleRegistrars = append(b.runtimeModuleRegistrars, registrars...)
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

	modules_ := make([]ModuleSpec, 0, len(b.modules))
	for i, mod := range b.modules {
		if mod == nil {
			return nil, fmt.Errorf("module spec at index %d is nil", i)
		}
		modules_ = append(modules_, mod)
	}
	runtimeRegistrars := make([]RuntimeModuleRegistrar, 0, len(b.runtimeModuleRegistrars))
	for i, registrar := range b.runtimeModuleRegistrars {
		if registrar == nil {
			return nil, fmt.Errorf("runtime module registrar at index %d is nil", i)
		}
		runtimeRegistrars = append(runtimeRegistrars, registrar)
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
	if err := validateUniqueIDs(runtimeRegistrars, "runtime module registrar"); err != nil {
		return nil, err
	}
	if err := validateUniqueIDs(inits, "runtime initializer"); err != nil {
		return nil, err
	}

	b.built = true

	return &Factory{
		settings: builderSettings{
			requireOptions: append([]require.Option(nil), b.settings.requireOptions...),
		},
		modules:                 append([]ModuleSpec(nil), modules_...),
		runtimeModuleRegistrars: append([]RuntimeModuleRegistrar(nil), runtimeRegistrars...),
		runtimeInitializers:     append([]RuntimeInitializer(nil), inits...),
	}, nil
}

// NewRuntime creates a new owned runtime instance from this factory's frozen
// composition plan.
func (f *Factory) NewRuntime(ctx context.Context) (*Runtime, error) {
	if f == nil {
		return nil, fmt.Errorf("factory is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	vm := goja.New()
	loop := eventloop.NewEventLoop()
	go loop.Start()

	owner := runtimeowner.NewRunner(vm, loop, runtimeowner.Options{
		Name:          "go-go-goja-runtime",
		RecoverPanics: true,
	})
	// #nosec G118 -- the runtime owns this cancel func and calls it on close and on setup failures.
	runtimeCtx, runtimeCtxCancel := context.WithCancel(context.Background())
	runtimeValues := map[string]any{}

	rt := &Runtime{
		VM:               vm,
		Loop:             loop,
		Owner:            owner,
		Values:           runtimeValues,
		runtimeCtx:       runtimeCtx,
		runtimeCtxCancel: runtimeCtxCancel,
	}

	runtimebridge.Store(vm, runtimebridge.Bindings{
		Context: runtimeCtx,
		Loop:    loop,
		Owner:   owner,
	})

	reg := require.NewRegistry(f.settings.requireOptions...)
	for _, mod := range f.modules {
		if err := mod.Register(reg); err != nil {
			_ = rt.Close(ctx)
			return nil, fmt.Errorf("register module %q: %w", mod.ID(), err)
		}
	}
	moduleCtx := &RuntimeModuleContext{
		Context:   runtimeCtx,
		VM:        vm,
		Loop:      loop,
		Owner:     owner,
		AddCloser: rt.AddCloser,
		Values:    runtimeValues,
	}
	for _, registrar := range f.runtimeModuleRegistrars {
		if err := registrar.RegisterRuntimeModules(moduleCtx, reg); err != nil {
			_ = rt.Close(ctx)
			return nil, fmt.Errorf("runtime module registrar %q: %w", registrar.ID(), err)
		}
	}

	reqMod := reg.Enable(vm)
	console.Enable(vm)
	rt.Require = reqMod

	initCtx := &RuntimeContext{
		Context: runtimeCtx,
		VM:      vm,
		Require: reqMod,
		Loop:    loop,
		Owner:   owner,
		Values:  rt.Values,
	}
	for _, init := range f.runtimeInitializers {
		if err := init.InitRuntime(initCtx); err != nil {
			_ = rt.Close(ctx)
			return nil, fmt.Errorf("runtime initializer %q: %w", init.ID(), err)
		}
	}

	return rt, nil
}
