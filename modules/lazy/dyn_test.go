package lazy_test

import (
	"strconv"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules/lazy"
)

// --- DynMap: laziness, caching, free enumeration, inheritance --------------

func TestDynMap_ReadIsLazyAndCached(t *testing.T) {
	vm := newVM(t)
	c := newCounter()
	m := &lazy.DynMap{
		Keys:     []string{"name", "wifis", "battery", "uptime"},
		Provider: deviceProvider(c),
	}
	mustSet(t, vm, "device", m.ToValue(vm))

	if v := runJS(t, vm, `device.name`); v.String() != "atom-3r" {
		t.Fatalf("device.name = %v", v)
	}
	if c.times("name") != 1 {
		t.Fatalf("name fetched %d times, want 1", c.times("name"))
	}
	if c.times("wifis") != 0 {
		t.Fatalf("wifis should not be materialised, got %d", c.times("wifis"))
	}
	// cached: second read does not call the provider
	runJS(t, vm, `device.name`)
	if c.times("name") != 1 {
		t.Fatalf("name fetched %d times on 2nd read, want 1 (cache)", c.times("name"))
	}
}

func TestDynMap_EnumerationIsFree(t *testing.T) {
	vm := newVM(t)
	c := newCounter()
	m := &lazy.DynMap{
		Keys:     []string{"name", "wifis", "battery", "uptime"},
		Provider: deviceProvider(c),
	}
	mustSet(t, vm, "device", m.ToValue(vm))

	keys := runJS(t, vm, `JSON.stringify(Object.keys(device))`).String()
	if keys != `["name","wifis","battery","uptime"]` {
		t.Fatalf("Object.keys = %s", keys)
	}
	for _, k := range m.Keys {
		if c.times(k) != 0 {
			t.Fatalf("enumeration materialised %q (%d calls); want 0", k, c.times(k))
		}
	}
	// for..in is free too
	runJS(t, vm, `for (var k in device) { void k; }`)
	for _, k := range m.Keys {
		if c.times(k) != 0 {
			t.Fatalf("for..in materialised %q (%d calls); want 0", k, c.times(k))
		}
	}
}

func TestDynMap_HasAndInherited(t *testing.T) {
	vm := newVM(t)
	c := newCounter()
	m := &lazy.DynMap{Keys: []string{"wifis"}, Provider: deviceProvider(c)}
	mustSet(t, vm, "device", m.ToValue(vm))

	// `in` is free (Has, not Get).
	if v := runJS(t, vm, `'wifis' in device && !('nope' in device)`).ToBoolean(); !v {
		t.Fatalf("'in' check failed")
	}
	if c.times("wifis") != 0 {
		t.Fatalf("'in' materialised wifis; want 0")
	}
	// inherited property: Get returns nil for "toString" -> delegates to prototype.
	if v := runJS(t, vm, `typeof device.toString === "function"`).ToBoolean(); !v {
		t.Fatalf("inherited toString not reachable")
	}
}

func TestDynMap_Mutable(t *testing.T) {
	vm := newVM(t)
	c := newCounter()
	m := &lazy.DynMap{Keys: []string{"name"}, Provider: deviceProvider(c), Mutable: true}
	mustSet(t, vm, "device", m.ToValue(vm))

	runJS(t, vm, `device.custom = 42`)
	if v := runJS(t, vm, `device.custom`); v.ToInteger() != 42 {
		t.Fatalf("device.custom = %v, want 42", v)
	}
	if c.times("custom") != 0 {
		t.Fatalf("set value should not call provider, got %d", c.times("custom"))
	}
	keys := runJS(t, vm, `JSON.stringify(Object.keys(device))`).String()
	if keys != `["name","custom"]` {
		t.Fatalf("keys after set = %s", keys)
	}
	runJS(t, vm, `delete device.custom`)
	keys = runJS(t, vm, `JSON.stringify(Object.keys(device))`).String()
	if keys != `["name"]` {
		t.Fatalf("keys after delete = %s", keys)
	}
}

func TestDynMap_ReadOnlyRejectsWrites(t *testing.T) {
	vm := newVM(t)
	c := newCounter()
	m := &lazy.DynMap{Keys: []string{"name"}, Provider: deviceProvider(c)} // Mutable=false
	mustSet(t, vm, "device", m.ToValue(vm))
	// non-strict assignment to a read-only dynamic object: the write is rejected
	// (Set returns false), so the value does not appear.
	runJS(t, vm, `device.x = 99`)
	if v := runJS(t, vm, `device.x`); !goja.IsUndefined(v) {
		t.Fatalf("read-only DynMap accepted write: device.x = %v", v)
	}
}

// --- DynArray --------------------------------------------------------------

func TestDynArray_IndexingLengthCaching(t *testing.T) {
	vm := newVM(t)
	calls := map[int]int{}
	a := &lazy.DynArray{
		Len: 1_000_000,
		Provider: func(i int) any {
			calls[i]++
			return "item-" + strconv.Itoa(i*i)
		},
	}
	mustSet(t, vm, "infinite", a.ToValue(vm))

	if v := runJS(t, vm, `infinite.length`); v.ToInteger() != 1_000_000 {
		t.Fatalf("length = %v", v)
	}
	if len(calls) != 0 {
		t.Fatalf("reading length materialised elements: %v", calls)
	}
	out := runJS(t, vm, `[infinite[5], infinite[3], infinite[5]].join("|")`).String()
	if out != "item-25|item-9|item-25" {
		t.Fatalf("indexed values = %s", out)
	}
	if calls[5] != 1 || calls[3] != 1 {
		t.Fatalf("indexing caching wrong: calls=%v", calls)
	}
}
