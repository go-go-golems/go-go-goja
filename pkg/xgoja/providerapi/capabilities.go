package providerapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

// SectionContext describes why a module's configuration sections are being
// requested. Built-in commands should set CommandName and RuntimeProfile;
// custom command providers should set CommandProviderID and, when applicable,
// RuntimeProfile or ModuleSelector.
type SectionContext struct {
	CommandName       string
	CommandProviderID string
	RuntimeProfile    string
	PackageID         string
	ModuleID          string
}

// ModuleDescriptor is the app-facing description of a selected runtime module
// plus the package capabilities that can configure or initialize it.
type ModuleDescriptor struct {
	PackageID    string
	ModuleID     string
	As           string
	Module       Module
	Capabilities []ModuleCapability
}

// ModuleCapability is the common marker for optional provider capabilities.
type ModuleCapability interface {
	CapabilityID() string
}

// ConfigSectionCapability lets a module expose Glazed sections that can be
// attached to built-in commands or package-owned command providers.
type ConfigSectionCapability interface {
	ModuleCapability
	ConfigSections(SectionContext) ([]schema.Section, error)
}

// RuntimeHandle is the minimal handle passed to runtime initializers. It avoids
// making providerapi depend on the concrete app or engine runtime type.
type RuntimeHandle interface {
	Runtime() *goja.Runtime
	Close(context.Context) error
}

// RuntimeInitializerCapability is used by built-in xgoja commands such as run,
// repl, jsverbs, and eventually eval. The runtime already exists; the module
// configures it from parsed Glazed sections.
type RuntimeInitializerCapability interface {
	ModuleCapability
	InitRuntimeFromSections(context.Context, *values.Values, RuntimeHandle) error
}

// ComponentInitializerCapability is used by package-owned command providers
// that need initialized domain objects, not only JS runtime mutation.
type ComponentInitializerCapability interface {
	ModuleCapability
	InitComponentFromSections(context.Context, *values.Values) (InitializedModule, error)
}

// InitializedModule is a component created from parsed Glazed sections by a
// package-owned command provider.
type InitializedModule interface {
	ModuleID() string
	Close(context.Context) error
}

type capabilityEntry struct {
	capability ModuleCapability
}

// WithCapability registers a package-level module capability. Capabilities are
// package-scoped for now and are attached to selected module descriptors for
// every selected module from that package.
func WithCapability(capability ModuleCapability) Entry {
	return capabilityEntry{capability: capability}
}

func (e capabilityEntry) applyToPackage(pkg *Package) error {
	return pkg.addCapability(e.capability)
}

func normalizeCapabilityID(capability ModuleCapability) (string, error) {
	if capability == nil {
		return "", fmt.Errorf("capability is nil")
	}
	id := strings.TrimSpace(capability.CapabilityID())
	if id == "" {
		return "", fmt.Errorf("capability id is required")
	}
	return id, nil
}
