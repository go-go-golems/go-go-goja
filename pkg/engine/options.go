package engine

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
)

type runtimeOptions struct {
	startupContext  context.Context
	lifetimeContext context.Context
	bridgeOwner     BridgeOwnerFunc
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

// BridgeOwnerFunc returns the owner that native modules reach through
// runtimebridge.Lookup for vm. base is the owner the runtime would otherwise
// publish; implementations usually wrap it and delegate to it.
type BridgeOwnerFunc func(vm *goja.Runtime, base runtimebridge.RuntimeOwner) runtimebridge.RuntimeOwner

// WithRuntimeBridgeOwner replaces the owner that the runtime publishes through
// runtimebridge before any module registers or loads. Native modules that
// settle promises or schedule work look the owner up with runtimebridge.Lookup,
// so a host can route their Call and Post through its own scheduler, for
// example to turn every asynchronous settlement into a host transaction.
//
// Only runtimebridge lookups see the replacement. Runtime.Owner, and the Owner
// fields of RuntimeModuleRegistrationContext and RuntimeInitializationContext,
// stay the runtime's own owner, which the host uses to drive the VM. A nil
// function, or one that returns nil, keeps the default owner.
func WithRuntimeBridgeOwner(fn BridgeOwnerFunc) RuntimeOption {
	return func(o *runtimeOptions) {
		o.bridgeOwner = fn
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
	includePanicStack              bool
	skippedGlobals                 map[DefaultGlobal]bool
}

// DefaultGlobal names one of the globals NewRuntime installs on every runtime
// besides require.
type DefaultGlobal string

const (
	// GlobalConsole is goja_nodejs's console, which writes to the process's
	// standard output and error.
	GlobalConsole DefaultGlobal = "console"
	// GlobalConsoleTimers is console.time, console.timeLog and console.timeEnd.
	// They are installed on the console global, so skipping GlobalConsole also
	// skips them.
	GlobalConsoleTimers DefaultGlobal = "console-timers"
	// GlobalPerformance is performance.now, a wall-clock timer that starts when
	// the runtime is created.
	GlobalPerformance DefaultGlobal = "performance"
	// GlobalBuffer is goja_nodejs's Buffer.
	GlobalBuffer DefaultGlobal = "buffer"
	// GlobalURL is goja_nodejs's URL and URLSearchParams.
	GlobalURL DefaultGlobal = "url"
)

// DefaultGlobals lists every global WithoutDefaultGlobals accepts.
func DefaultGlobals() []DefaultGlobal {
	return []DefaultGlobal{GlobalConsole, GlobalConsoleTimers, GlobalPerformance, GlobalBuffer, GlobalURL}
}

func validateDefaultGlobals(skipped map[DefaultGlobal]bool) error {
	known := map[DefaultGlobal]bool{}
	for _, g := range DefaultGlobals() {
		known[g] = true
	}
	for g := range skipped {
		if !known[g] {
			return fmt.Errorf("unknown default global %q", g)
		}
	}
	return nil
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

// WithoutDefaultGlobals skips globals that NewRuntime would otherwise install
// on every runtime. Hosts that need deterministic runtimes, or that install
// their own console, use it to keep wall clocks and stdout out of scripts.
// Calling it several times accumulates. Build fails on an unknown name.
// require is always installed; a runtime initializer may replace it.
func WithoutDefaultGlobals(globals ...DefaultGlobal) Option {
	return func(s *builderSettings) {
		if s.skippedGlobals == nil {
			s.skippedGlobals = map[DefaultGlobal]bool{}
		}
		for _, g := range globals {
			s.skippedGlobals[g] = true
		}
	}
}

// WithRecoveredPanicStack controls whether runtimeowner recovered panic errors
// include a Go debug stack. Keep this disabled for normal end-user CLIs; enable
// it while debugging provider panics inside generated runtimes.
func WithRecoveredPanicStack(enabled bool) Option {
	return func(s *builderSettings) {
		s.includePanicStack = enabled
	}
}
