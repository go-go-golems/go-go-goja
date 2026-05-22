package providerapi

import "io/fs"

type VerbSource struct {
	Name        string
	Description string
	FS          fs.FS
	Root        string
}

func (s VerbSource) applyToPackage(pkg *Package) error {
	return pkg.addVerbSource(s)
}
