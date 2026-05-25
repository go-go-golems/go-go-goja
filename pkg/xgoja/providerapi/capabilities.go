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

// ConfigSectionCapability lets a module expose Glazed sections that can be
// attached to built-in commands or package-owned command providers.
type ConfigSectionCapability interface {
	PackageCapability
	ConfigSections(SectionContext) ([]schema.Section, error)
}

// RuntimeHandle is the minimal handle passed to runtime initializers. It avoids
// making providerapi depend on the concrete app or engine runtime type.
type RuntimeHandle interface {
	Runtime() *goja.Runtime
	Close(context.Context) error
}

// RuntimeCloserRegistry is an optional extension implemented by runtime handles
// that can attach cleanup hooks to the underlying engine runtime.
type RuntimeCloserRegistry interface {
	AddCloser(func(context.Context) error) error
}

// RuntimeInitializerCapability is used by built-in xgoja commands such as run,
// repl, jsverbs, and eventually eval. The runtime already exists; the module
// configures it from parsed Glazed sections.
type RuntimeInitializerCapability interface {
	PackageCapability
	InitRuntimeFromSections(context.Context, *values.Values, RuntimeHandle) error
}

type capabilityEntry struct {
	capability PackageCapability
}

// WithPackageCapability registers a package-level module capability. Capabilities are
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
