package providerapi

import (
	"fmt"
	"sort"
	"strings"
)

type Entry interface {
	applyToPackage(*Package) error
}

type Registry struct {
	packages map[string]*Package
	order    []string
}

type Package struct {
	ID          string
	Modules     map[string]Module
	VerbSources map[string]VerbSource
}

func NewRegistry() *Registry {
	return &Registry{packages: map[string]*Package{}}
}

func (r *Registry) Package(id string, entries ...Entry) error {
	if r == nil {
		return fmt.Errorf("provider registry is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("provider package id is required")
	}
	if _, ok := r.packages[id]; ok {
		return fmt.Errorf("duplicate provider package %q", id)
	}
	pkg := &Package{
		ID:          id,
		Modules:     map[string]Module{},
		VerbSources: map[string]VerbSource{},
	}
	for i, entry := range entries {
		if entry == nil {
			return fmt.Errorf("provider package %q entry %d is nil", id, i)
		}
		if err := entry.applyToPackage(pkg); err != nil {
			return fmt.Errorf("provider package %q entry %d: %w", id, i, err)
		}
	}
	r.packages[id] = pkg
	r.order = append(r.order, id)
	return nil
}

func (r *Registry) ResolveModule(packageID, moduleName string) (Module, bool) {
	if r == nil {
		return Module{}, false
	}
	pkg := r.packages[strings.TrimSpace(packageID)]
	if pkg == nil {
		return Module{}, false
	}
	mod, ok := pkg.Modules[strings.TrimSpace(moduleName)]
	return mod, ok
}

func (r *Registry) ResolveVerbSource(packageID, sourceName string) (VerbSource, bool) {
	if r == nil {
		return VerbSource{}, false
	}
	pkg := r.packages[strings.TrimSpace(packageID)]
	if pkg == nil {
		return VerbSource{}, false
	}
	source, ok := pkg.VerbSources[strings.TrimSpace(sourceName)]
	return source, ok
}

func (r *Registry) Packages() []Package {
	if r == nil {
		return nil
	}
	ids := append([]string(nil), r.order...)
	if len(ids) == 0 && len(r.packages) > 0 {
		ids = make([]string, 0, len(r.packages))
		for id := range r.packages {
			ids = append(ids, id)
		}
		sort.Strings(ids)
	}
	out := make([]Package, 0, len(ids))
	for _, id := range ids {
		pkg := r.packages[id]
		if pkg == nil {
			continue
		}
		out = append(out, pkg.clone())
	}
	return out
}

func (p *Package) addModule(module Module) error {
	name := strings.TrimSpace(module.Name)
	if name == "" {
		return fmt.Errorf("module name is required")
	}
	if module.New == nil {
		return fmt.Errorf("module %q factory is required", name)
	}
	if _, ok := p.Modules[name]; ok {
		return fmt.Errorf("duplicate module %q", name)
	}
	module.Name = name
	module.DefaultAs = strings.TrimSpace(module.DefaultAs)
	module.Description = strings.TrimSpace(module.Description)
	p.Modules[name] = module
	return nil
}

func (p *Package) addVerbSource(source VerbSource) error {
	name := strings.TrimSpace(source.Name)
	if name == "" {
		return fmt.Errorf("verb source name is required")
	}
	if _, ok := p.VerbSources[name]; ok {
		return fmt.Errorf("duplicate verb source %q", name)
	}
	source.Name = name
	source.Description = strings.TrimSpace(source.Description)
	source.Root = strings.TrimSpace(source.Root)
	p.VerbSources[name] = source
	return nil
}

func (p *Package) clone() Package {
	out := Package{
		ID:          p.ID,
		Modules:     map[string]Module{},
		VerbSources: map[string]VerbSource{},
	}
	for name, module := range p.Modules {
		out.Modules[name] = module
	}
	for name, source := range p.VerbSources {
		out.VerbSources[name] = source
	}
	return out
}
