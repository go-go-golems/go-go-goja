package engine

import (
	"sort"
	"strings"

	"github.com/go-go-golems/go-go-goja/modules"
)

// ModuleSelector chooses which modules to register from a set of available names.
type ModuleSelector func(available []string) []string

// ModuleMiddleware wraps a selector. The standard pattern is f(next) returns a new
// selector. This gives explicit control flow: each middleware decides whether to
// call next, short-circuit, modify before/after, etc.
type ModuleMiddleware func(next ModuleSelector) ModuleSelector

// SelectAll is the identity selector — includes all available modules.
func SelectAll(available []string) []string { return available }

// MiddlewareSafe returns only data-safe modules. It is an override middleware:
// it does NOT call next, replacing the entire selection.
func MiddlewareSafe() ModuleMiddleware {
	return func(next ModuleSelector) ModuleSelector {
		return func(available []string) []string {
			_ = next
			return intersect(available, dataOnlyDefaultRegistryModuleNames)
		}
	}
}

// MiddlewareOnly returns only the named modules. It is an override middleware:
// it does NOT call next, replacing the entire selection.
func MiddlewareOnly(names ...string) ModuleMiddleware {
	trimmed := make([]string, 0, len(names))
	for _, name := range names {
		if t := strings.TrimSpace(name); t != "" {
			trimmed = append(trimmed, t)
		}
	}
	return func(next ModuleSelector) ModuleSelector {
		return func(available []string) []string {
			_ = next
			expanded := expandDefaultRegistryModuleNames(trimmed)
			return intersect(available, expanded)
		}
	}
}

// MiddlewareExclude calls next first, then removes the named modules from the
// result. It is a transform middleware.
func MiddlewareExclude(names ...string) ModuleMiddleware {
	trimmed := make([]string, 0, len(names))
	for _, name := range names {
		if t := strings.TrimSpace(name); t != "" {
			trimmed = append(trimmed, t)
		}
	}
	return func(next ModuleSelector) ModuleSelector {
		return func(available []string) []string {
			selected := next(available)
			expanded := expandDefaultRegistryModuleNames(trimmed)
			return filterOut(selected, expanded)
		}
	}
}

// MiddlewareAdd calls next first, then appends the named modules (if they exist
// in the available set). It is a transform middleware.
func MiddlewareAdd(names ...string) ModuleMiddleware {
	trimmed := make([]string, 0, len(names))
	for _, name := range names {
		if t := strings.TrimSpace(name); t != "" {
			trimmed = append(trimmed, t)
		}
	}
	return func(next ModuleSelector) ModuleSelector {
		return func(available []string) []string {
			selected := next(available)
			expanded := expandDefaultRegistryModuleNames(trimmed)
			// Only add names that are actually available
			valid := intersect(available, expanded)
			return appendUnique(selected, valid)
		}
	}
}

// MiddlewareCustom calls next first, then applies an arbitrary transformation.
// It is a transform middleware.
func MiddlewareCustom(fn func(selected []string) []string) ModuleMiddleware {
	return func(next ModuleSelector) ModuleSelector {
		return func(available []string) []string {
			return fn(next(available))
		}
	}
}

// Pipeline composes middlewares left-to-right: the first middleware in the list
// executes first, wrapping the subsequent ones.
func Pipeline(mws ...ModuleMiddleware) ModuleMiddleware {
	return func(next ModuleSelector) ModuleSelector {
		handler := next
		for i := len(mws) - 1; i >= 0; i-- {
			handler = mws[i](handler)
		}
		return handler
	}
}

// allRegisteredModuleNames returns all module names from the default registry.
func allRegisteredModuleNames() []string {
	mods := modules.ListDefaultModules()
	names := make([]string, 0, len(mods))
	for _, m := range mods {
		if m != nil {
			names = append(names, m.Name())
		}
	}
	return names
}

// intersect returns elements in a that are also in b.
func intersect(a, b []string) []string {
	set := make(map[string]struct{}, len(b))
	for _, v := range b {
		set[v] = struct{}{}
	}
	ret := make([]string, 0)
	seen := make(map[string]struct{})
	for _, v := range a {
		if _, ok := set[v]; !ok {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		ret = append(ret, v)
	}
	return ret
}

// filterOut returns elements in a that are NOT in b.
func filterOut(a, b []string) []string {
	set := make(map[string]struct{}, len(b))
	for _, v := range b {
		set[v] = struct{}{}
	}
	ret := make([]string, 0)
	seen := make(map[string]struct{})
	for _, v := range a {
		if _, ok := set[v]; ok {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		ret = append(ret, v)
	}
	return ret
}

// appendUnique appends elements from src to dst, skipping duplicates.
func appendUnique(dst, src []string) []string {
	seen := make(map[string]struct{}, len(dst))
	for _, v := range dst {
		seen[v] = struct{}{}
	}
	for _, v := range src {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		dst = append(dst, v)
	}
	return dst
}

// sortedUnique returns a sorted deduplicated copy of the slice.
func sortedUnique(src []string) []string {
	seen := make(map[string]struct{}, len(src))
	for _, v := range src {
		seen[v] = struct{}{}
	}
	ret := make([]string, 0, len(seen))
	for v := range seen {
		ret = append(ret, v)
	}
	sort.Strings(ret)
	return ret
}
