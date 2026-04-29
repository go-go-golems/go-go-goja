package engine

import (
	"sort"
	"testing"
)

func TestMiddlewareSafe(t *testing.T) {
	available := []string{"crypto", "events", "fs", "os", "exec", "path", "time", "timer", "yaml"}
	mw := MiddlewareSafe()
	selected := mw(SelectAll)(available)

	want := []string{"crypto", "events", "path", "time", "timer"}
	if !slicesEqual(selected, want) {
		t.Fatalf("MiddlewareSafe() = %v, want %v", selected, want)
	}
}

func TestMiddlewareSafeIgnoresNext(t *testing.T) {
	// Safe is an override middleware — it should ignore whatever next returns.
	available := []string{"crypto", "events", "fs", "os"}
	mw := MiddlewareSafe()
	// next returns everything, but Safe should still only return safe modules.
	selected := mw(SelectAll)(available)
	want := []string{"crypto", "events"}
	if !slicesEqual(selected, want) {
		t.Fatalf("MiddlewareSafe() override failed: got %v, want %v", selected, want)
	}
}

func TestMiddlewareOnly(t *testing.T) {
	available := []string{"crypto", "events", "fs", "os", "exec", "path"}
	mw := MiddlewareOnly("fs", "path")
	selected := mw(SelectAll)(available)

	want := []string{"fs", "path"}
	if !slicesEqual(selected, want) {
		t.Fatalf("MiddlewareOnly() = %v, want %v", selected, want)
	}
}

func TestMiddlewareOnlyWithAliases(t *testing.T) {
	available := []string{"crypto", "node:crypto", "fs", "node:fs"}
	mw := MiddlewareOnly("crypto")
	selected := mw(SelectAll)(available)

	// expandDefaultRegistryModuleNames adds node:crypto as an alias.
	want := []string{"crypto", "node:crypto"}
	if !slicesEqual(selected, want) {
		t.Fatalf("MiddlewareOnly with alias = %v, want %v", selected, want)
	}
}

func TestMiddlewareOnlyWithDatabaseAlias(t *testing.T) {
	available := []string{"database", "db", "fs"}
	mw := MiddlewareOnly("db")
	selected := mw(SelectAll)(available)

	want := []string{"database", "db"}
	if !slicesEqual(selected, want) {
		t.Fatalf("MiddlewareOnly with database alias = %v, want %v", selected, want)
	}
}

func TestMiddlewareOnlyIgnoresUnknown(t *testing.T) {
	available := []string{"crypto", "fs"}
	mw := MiddlewareOnly("fs", "nonexistent")
	selected := mw(SelectAll)(available)

	want := []string{"fs"}
	if !slicesEqual(selected, want) {
		t.Fatalf("MiddlewareOnly unknown = %v, want %v", selected, want)
	}
}

func TestMiddlewareExclude(t *testing.T) {
	available := []string{"crypto", "events", "fs", "os", "exec", "path"}
	mw := MiddlewareExclude("fs", "exec")
	selected := mw(SelectAll)(available)

	want := []string{"crypto", "events", "os", "path"}
	if !slicesEqual(selected, want) {
		t.Fatalf("MiddlewareExclude() = %v, want %v", selected, want)
	}
}

func TestMiddlewareAdd(t *testing.T) {
	available := []string{"crypto", "events", "fs", "os", "path"}
	mw := MiddlewareAdd("fs")
	selected := mw(func(a []string) []string {
		// Simulate a prior filter that removed fs.
		return []string{"crypto", "events", "path"}
	})(available)

	want := []string{"crypto", "events", "path", "fs"}
	if !slicesEqual(selected, want) {
		t.Fatalf("MiddlewareAdd() = %v, want %v", selected, want)
	}
}

func TestMiddlewareAddSkipsUnavailable(t *testing.T) {
	available := []string{"crypto", "events"}
	mw := MiddlewareAdd("fs") // fs is not in available
	selected := mw(SelectAll)(available)

	want := []string{"crypto", "events"}
	if !slicesEqual(selected, want) {
		t.Fatalf("MiddlewareAdd unavailable = %v, want %v", selected, want)
	}
}

func TestMiddlewareCustom(t *testing.T) {
	available := []string{"fs", "os", "exec"}
	mw := MiddlewareCustom(func(selected []string) []string {
		sort.Strings(selected)
		return selected
	})
	selected := mw(SelectAll)(available)

	want := []string{"exec", "fs", "os"}
	if !slicesEqual(selected, want) {
		t.Fatalf("MiddlewareCustom() = %v, want %v", selected, want)
	}
}

func TestPipelineOrder(t *testing.T) {
	available := []string{"crypto", "events", "fs", "os", "exec", "yaml", "path", "time", "timer"}

	// Safe is an override middleware — it short-circuits everything after it.
	// So Pipeline(Safe, Add(fs), Exclude(yaml)) is equivalent to just Safe.
	mw := Pipeline(
		MiddlewareSafe(),
		MiddlewareAdd("fs"),
		MiddlewareExclude("yaml"),
	)
	selected := mw(SelectAll)(available)
	want := []string{"crypto", "events", "path", "time", "timer"}
	if !slicesEqual(selected, want) {
		t.Fatalf("Pipeline safe+add+exclude = %v, want %v", selected, want)
	}
}

func TestPipelineAddThenSafe(t *testing.T) {
	available := []string{"crypto", "events", "fs", "os", "exec", "path"}

	// Add fs → Safe
	// Add runs first, calls Safe (override → returns safe only).
	// Add appends fs to the safe-only result.
	mw := Pipeline(
		MiddlewareAdd("fs"),
		MiddlewareSafe(),
	)
	selected := mw(SelectAll)(available)
	want := []string{"crypto", "events", "path", "fs"}
	if !slicesEqual(selected, want) {
		t.Fatalf("Pipeline add+safe = %v, want %v", selected, want)
	}
}

func TestIntersect(t *testing.T) {
	a := []string{"a", "b", "c", "d"}
	b := []string{"b", "d", "e"}
	got := intersect(a, b)
	want := []string{"b", "d"}
	if !slicesEqual(got, want) {
		t.Fatalf("intersect = %v, want %v", got, want)
	}
}

func TestIntersectDeduplicates(t *testing.T) {
	a := []string{"a", "a", "b"}
	b := []string{"a", "b", "b"}
	got := intersect(a, b)
	want := []string{"a", "b"}
	if !slicesEqual(got, want) {
		t.Fatalf("intersect dedup = %v, want %v", got, want)
	}
}

func TestFilterOut(t *testing.T) {
	a := []string{"a", "b", "c", "d"}
	b := []string{"b", "d"}
	got := filterOut(a, b)
	want := []string{"a", "c"}
	if !slicesEqual(got, want) {
		t.Fatalf("filterOut = %v, want %v", got, want)
	}
}

func TestFilterOutDeduplicates(t *testing.T) {
	a := []string{"a", "a", "b", "c"}
	b := []string{"a"}
	got := filterOut(a, b)
	want := []string{"b", "c"}
	if !slicesEqual(got, want) {
		t.Fatalf("filterOut dedup = %v, want %v", got, want)
	}
}

func TestAppendUnique(t *testing.T) {
	dst := []string{"a", "b"}
	src := []string{"b", "c"}
	got := appendUnique(dst, src)
	want := []string{"a", "b", "c"}
	if !slicesEqual(got, want) {
		t.Fatalf("appendUnique = %v, want %v", got, want)
	}
}

func TestMiddlewareOnlyForSafePlusExtra(t *testing.T) {
	// To get safe modules + one extra, use MiddlewareOnly with explicit list.
	available := []string{"crypto", "events", "fs", "os", "exec", "path"}
	mw := MiddlewareOnly("crypto", "events", "path", "time", "timer", "fs")
	selected := mw(SelectAll)(available)
	want := []string{"crypto", "events", "fs", "path"}
	if !slicesEqual(selected, want) {
		t.Fatalf("MiddlewareOnly safe+fs = %v, want %v", selected, want)
	}
}

func TestSortedUnique(t *testing.T) {
	got := sortedUnique([]string{"b", "a", "b", "c", "a"})
	want := []string{"a", "b", "c"}
	if !slicesEqual(got, want) {
		t.Fatalf("sortedUnique = %v, want %v", got, want)
	}
}

func TestAllRegisteredModuleNames(t *testing.T) {
	names := allRegisteredModuleNames()
	if len(names) == 0 {
		t.Fatal("allRegisteredModuleNames() returned empty slice")
	}
	// Verify at least some known modules are present.
	has := func(name string) bool {
		for _, n := range names {
			if n == name {
				return true
			}
		}
		return false
	}
	if !has("fs") {
		t.Fatal("expected fs in registered module names")
	}
	if !has("crypto") {
		t.Fatal("expected crypto in registered module names")
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
