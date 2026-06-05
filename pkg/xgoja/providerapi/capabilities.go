package providerapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
)

// SectionRequest describes why a module's configuration sections are being
// requested. Built-in commands should set CommandName; custom command providers
// should set CommandProviderID. PackageID and ModuleID identify the selected
// provider module when the request is module-specific.
type SectionRequest struct {
	CommandName       string
	CommandProviderID string
	PackageID         string
	ModuleID          string
}

// ModuleDescriptor is the app-facing description of a selected runtime module
// plus the package capabilities that can configure or initialize it.
type ModuleDescriptor struct {
	PackageID           string
	ModuleID            string
	As                  string
	Module              Module
	PackageCapabilities []PackageCapability
}

// PackageCapability is the common marker for optional provider capabilities.
type PackageCapability interface {
	CapabilityID() string
}

// GlazedConfigSectionCapability lets a provider expose public Glazed sections
// that can be attached to built-in commands or package-owned command providers.
// These sections are user-facing CLI/config/env inputs; they are not necessarily
// the same schema as a provider's internal xgoja module config.
type GlazedConfigSectionCapability interface {
	PackageCapability
	GlazedConfigSections(SectionRequest) ([]schema.Section, error)
}

// XGojaConfigRequest identifies one selected module instance whose internal
// xgoja config is being prepared before module setup. StaticConfig contains the
// values parsed from xgoja.yaml using ConfigSection. GlazedValues contains the
// public command/config/env values parsed from Glazed sections.
type XGojaConfigRequest struct {
	SectionRequest
	Descriptor    ModuleDescriptor
	ConfigSection schema.Section
	StaticConfig  *values.SectionValues
	GlazedValues  *values.Values
}

// XGojaConfigSectionCapability exposes a provider's internal xgoja module
// config section and maps parsed public Glazed values into an internal config
// override. The returned SectionValues should use ConfigSection and only include
// fields that should override static xgoja.yaml config for this module instance.
type XGojaConfigSectionCapability interface {
	PackageCapability
	XGojaConfigSection(SectionRequest, ModuleDescriptor) (schema.Section, error)
	XGojaConfigFromGlazed(context.Context, XGojaConfigRequest) (*values.SectionValues, error)
}

// HostServiceContributionRequest is passed to package capabilities before a
// runtime is constructed. Capabilities can inspect the selected runtime modules
// and parsed Glazed values, then add opaque host services for provider modules
// to consume during ModuleSetupContext setup.
type HostServiceContributionRequest struct {
	SectionRequest
	Values  *values.Values
	Modules []ModuleDescriptor
}

// HostServiceSink collects opaque provider-defined host services. The sink may
// accept multiple values for the same key; consumers should use
// HostServiceLookup.HostServiceValues when a key is intentionally multi-valued.
// Contributors that create runtime-owned resources should register cleanup with
// AddCloser; xgoja wires those closers into the engine runtime before JavaScript
// executes.
type HostServiceSink interface {
	AddHostService(key string, value any) error
	AddCloser(fn func(context.Context) error) error
}

// HostServiceContributionCapability lets a selected package contribute
// Go-backed host services before provider modules are set up. xgoja core does
// not interpret the service values; provider packages own keys and payload
// types.
type HostServiceContributionCapability interface {
	PackageCapability
	ContributeHostServices(context.Context, HostServiceContributionRequest, HostServiceSink) error
}

// RuntimeInitializerHandle is the runtime-facing handle passed to runtime
// initializers. It exposes the owned engine runtime so providers can access the
// Goja VM, event loop, runtime owner, closer registration, and other runtime-scoped
// services when installing runtime functionality.
type RuntimeInitializerHandle interface {
	EngineRuntime() *engine.Runtime
	Close(context.Context) error
}

// RuntimeInitializerCapability is used by built-in xgoja commands such as run,
// repl, jsverbs, and eventually eval. The runtime already exists; the module
// configures it from parsed Glazed sections.
type RuntimeInitializerCapability interface {
	PackageCapability
	InitRuntimeFromSections(context.Context, *values.Values, RuntimeInitializerHandle) error
}

type capabilityEntry struct {
	capability PackageCapability
}

// WithPackageCapability registers a package-level capability. Capabilities are
// package-scoped for now and are attached to selected module descriptors for
// every selected module from that package.
func WithPackageCapability(capability PackageCapability) Entry {
	return capabilityEntry{capability: capability}
}

func (e capabilityEntry) applyToPackage(pkg *Package) error {
	return pkg.addCapability(e.capability)
}

func normalizeCapabilityID(capability PackageCapability) (string, error) {
	if capability == nil {
		return "", fmt.Errorf("capability is nil")
	}
	id := strings.TrimSpace(capability.CapabilityID())
	if id == "" {
		return "", fmt.Errorf("capability id is required")
	}
	return id, nil
}
