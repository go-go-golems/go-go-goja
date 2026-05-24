package engine

import "github.com/dop251/goja_nodejs/require"

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
