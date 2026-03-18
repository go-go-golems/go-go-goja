package host

import (
	"fmt"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/contract"
)

// ValidateManifest enforces the v1 plugin manifest rules.
func ValidateManifest(cfg Config, manifest *contract.ModuleManifest) error {
	cfg = cfg.withDefaults()
	if err := contract.ValidateManifest(manifest, contract.ManifestValidationOptions{
		NamespacePrefix: cfg.Namespace,
		AllowModules:    cfg.AllowModules,
	}); err != nil {
		return fmt.Errorf("host validate manifest: %w", err)
	}
	return nil
}
