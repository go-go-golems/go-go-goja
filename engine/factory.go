package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

// FactoryBuilder composes explicit module and runtime initializer configuration
// before producing an immutable Factory.
type FactoryBuilder struct {
	settings builderSettings

	modules             []ModuleSpec
	runtimeInitializers []RuntimeInitializer
	built               bool
}

// Factory creates runtime instances from an immutable build plan.
type Factory struct {
	registry            *require.Registry
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

// WithModules appends static module registrations.
func (b *FactoryBuilder) WithModules(mods ...ModuleSpec) *FactoryBuilder {
	b.assertMutable()
	b.modules = append(b.modules, mods...)
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

	reg := require.NewRegistry(b.settings.requireOptions...)
	for _, mod := range modules_ {
		if err := mod.Register(reg); err != nil {
			return nil, fmt.Errorf("register module %q: %w", mod.ID(), err)
		}
	}

	b.built = true

	return &Factory{
		registry:            reg,
		runtimeInitializers: append([]RuntimeInitializer(nil), inits...),
	}, nil
}

// NewRuntime creates a new owned runtime instance from this factory's frozen
// composition plan.
func (f *Factory) NewRuntime(ctx context.Context) (*Runtime, error) {
	if f == nil {
		return nil, fmt.Errorf("factory is nil")
	}
	if f.registry == nil {
		return nil, fmt.Errorf("factory has no require registry")
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

	reqMod := f.registry.Enable(vm)
	console.Enable(vm)

	rt := &Runtime{
		VM:      vm,
		Require: reqMod,
		Loop:    loop,
		Owner:   owner,
	}

	initCtx := &RuntimeContext{
		VM:      vm,
		Require: reqMod,
		Loop:    loop,
		Owner:   owner,
	}
	for _, init := range f.runtimeInitializers {
		if err := init.InitRuntime(initCtx); err != nil {
			_ = rt.Close(ctx)
			return nil, fmt.Errorf("runtime initializer %q: %w", init.ID(), err)
		}
	}

	return rt, nil
}
