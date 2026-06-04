package engine

import (
	"context"

	"github.com/dop251/goja_nodejs/require"
)

type runtimeOptions struct {
	startupContext  context.Context
	lifetimeContext context.Context
}

// RuntimeOption configures creation of one runtime instance.
type RuntimeOption func(*runtimeOptions)

// WithStartupContext sets the context used while constructing a runtime and
// running runtime initializers. If omitted, context.Background is used.
func WithStartupContext(ctx context.Context) RuntimeOption {
	return func(o *runtimeOptions) {
		o.startupContext = ctx
	}
}

// WithLifetimeContext sets the parent context for runtime-owned resources. The
// runtime derives its own cancelable lifetime context from this parent and also
// cancels it during Runtime.Close.
func WithLifetimeContext(ctx context.Context) RuntimeOption {
	return func(o *runtimeOptions) {
		o.lifetimeContext = ctx
	}
}

func defaultRuntimeOptions() runtimeOptions {
	return runtimeOptions{
		startupContext:  context.Background(),
		lifetimeContext: context.Background(),
	}
}

type builderSettings struct {
	requireOptions                 []require.Option
	implicitDefaultRegistryModules bool
	dataOnlyDefaultRegistryModules bool
}

func defaultBuilderSettings() builderSettings {
	return builderSettings{
		implicitDefaultRegistryModules: true,
		dataOnlyDefaultRegistryModules: true,
	}
}

// Option configures engine builder behavior.
type Option func(*builderSettings)

// WithRequireOptions appends require registry options for module loading.
func WithRequireOptions(opts ...require.Option) Option {
	return func(s *builderSettings) {
		s.requireOptions = append(s.requireOptions, opts...)
	}
}

// WithImplicitDefaultRegistryModules controls whether a builder with no
// explicit modules and no module middleware falls back to all default-registry
// modules. It does not disable explicitly requested module middleware.
func WithImplicitDefaultRegistryModules(enabled bool) Option {
	return func(s *builderSettings) {
		s.implicitDefaultRegistryModules = enabled
	}
}

// WithDataOnlyDefaultRegistryModules controls whether every runtime receives
// the safe data-only default modules in addition to the builder-selected module
// set.
func WithDataOnlyDefaultRegistryModules(enabled bool) Option {
	return func(s *builderSettings) {
		s.dataOnlyDefaultRegistryModules = enabled
	}
}
