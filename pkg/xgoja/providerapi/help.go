package providerapi

import (
	"fmt"
	"io/fs"
	"strings"
)

// HelpSource registers a package-owned Glazed help documentation tree.
//
// Provider packages use HelpSource for API references, tutorials, and other
// user-facing docs that should be available from generated xgoja binaries when
// selected by the buildspec. FS must contain Glazed markdown files and Root is
// the directory passed to help.HelpSystem.LoadSectionsFromFS.
type HelpSource struct {
	Name        string
	Description string
	FS          fs.FS
	Root        string
}

func (s HelpSource) applyToPackage(pkg *Package) error {
	return pkg.addHelpSource(s)
}

func normalizeHelpSource(source HelpSource) (HelpSource, error) {
	name := strings.TrimSpace(source.Name)
	if name == "" {
		return HelpSource{}, fmt.Errorf("help source name is required")
	}
	if source.FS == nil {
		return HelpSource{}, fmt.Errorf("help source %q filesystem is required", name)
	}
	source.Name = name
	source.Description = strings.TrimSpace(source.Description)
	source.Root = strings.TrimSpace(source.Root)
	if source.Root == "" {
		source.Root = "."
	}
	return source, nil
}
