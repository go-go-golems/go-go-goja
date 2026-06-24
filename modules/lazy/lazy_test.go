package lazy_test

import (
	"strconv"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules/lazy"
)

// counter is a counting provider for laziness assertions.
type counter struct {
	calls map[string]int
}

func newCounter() *counter { return &counter{calls: map[string]int{}} }

func (c *counter) times(key string) int { return c.calls[key] }

// deviceProvider returns a provider that counts calls and serves toy values.
func deviceProvider(c *counter) func(string) any {
	return func(key string) any {
		c.calls[key]++
		switch key {
		case "name":
			return "atom-3r"
		case "wifis":
			return []string{"scapegoat", "freifunk", "eduroam"}
		case "battery":
			return 87
		case "uptime":
			return "3d 4h"
		}
		return nil
	}
}

func newVM(t *testing.T) *goja.Runtime {
	t.Helper()
	return goja.New()
}

// mustSet binds name to val, failing the test on error (errcheck-safe).
func mustSet(t *testing.T, vm *goja.Runtime, name string, val any) {
	t.Helper()
	if err := vm.Set(name, val); err != nil {
		t.Fatalf("vm.Set(%q): %v", name, err)
	}
}

func runJS(t *testing.T, vm *goja.Runtime, src string) goja.Value {
	t.Helper()
	v, err := vm.RunString(src)
	if err != nil {
		t.Fatalf("JS error: %v\nsrc: %s", err, src)
	}
	return v
}

// --- Map: laziness, caching, identity ---------------------------------------

func TestMap_ReadIsLazyAndCached(t *testing.T) {
	vm := newVM(t)
	c := newCounter()
	m := &lazy.Map{
		Keys:     []string{"name", "wifis", "battery", "uptime"},
		Provider: deviceProvider(c),
	}
	mustSet(t, vm, "device", m.ToValue(vm))

	// Reading name calls the provider exactly once for name, nothing else.
	if v := runJS(t, vm, `device.name`); v.String() != "atom-3r" {
		t.Fatalf("device.name = %v", v)
	}
	if c.times("name") != 1 {
		t.Fatalf("name provider called %d times, want 1", c.times("name"))
	}
	if c.times("wifis") != 0 {
		t.Fatalf("wifis should not have been materialised, got %d calls", c.times("wifis"))
	}

	// Second read is cached: provider not called again.
	runJS(t, vm, `device.name`)
	if c.times("name") != 1 {
		t.Fatalf("name provider called %d times on second read, want 1 (cache)", c.times("name"))
	}
}

func TestMap_IdentityStable(t *testing.T) {
	vm := newVM(t)
	c := newCounter()
	m := &lazy.Map{
		Keys:     []string{"wifis"},
		Provider: deviceProvider(c),
	}
	mustSet(t, vm, "device", m.ToValue(vm))

	got := runJS(t, vm, `device.wifis === device.wifis`).ToBoolean()
	if !got {
		t.Fatalf("cached value identity not stable (=== failed)")
	}
}

// --- The §6.3 regression: enumeration must NOT materialise ------------------

func TestMap_EnumerationIsFree(t *testing.T) {
	vm := newVM(t)
	c := newCounter()
	m := &lazy.Map{
		Keys:     []string{"name", "wifis", "battery", "uptime"},
		Provider: deviceProvider(c),
	}
	mustSet(t, vm, "device", m.ToValue(vm))

	// Object.keys enumerates without calling the provider.
	keys := runJS(t, vm, `JSON.stringify(Object.keys(device))`).String()
	want := `["name","wifis","battery","uptime"]`
	if keys != want {
		t.Fatalf("Object.keys = %s, want %s", keys, want)
	}
	for _, k := range m.Keys {
		if c.times(k) != 0 {
			t.Fatalf("enumeration materialised %q (%d calls); want 0", k, c.times(k))
		}
	}

	// for..in is free too.
	runJS(t, vm, `for (var k in device) { void k; }`)
	for _, k := range m.Keys {
		if c.times(k) != 0 {
			t.Fatalf("for..in materialised %q (%d calls); want 0", k, c.times(k))
		}
	}
}

func TestMap_HasIsFree(t *testing.T) {
	vm := newVM(t)
	c := newCounter()
	m := &lazy.Map{
		Keys:     []string{"wifis"},
		Provider: deviceProvider(c),
	}
	mustSet(t, vm, "device", m.ToValue(vm))

	got := runJS(t, vm, `'wifis' in device && !('nope' in device)`).ToBoolean()
	if !got {
		t.Fatalf("'wifis' in device / !('nope' in device) failed")
	}
	if c.times("wifis") != 0 {
		t.Fatalf("'in' operator materialised wifis (%d calls); want 0", c.times("wifis"))
	}
}

// --- Documented tradeoff: explicit descriptor value is undefined -----------

func TestMap_GetOwnPropertyDescriptorValueIsUndefined(t *testing.T) {
	vm := newVM(t)
	c := newCounter()
	m := &lazy.Map{
		Keys:     []string{"name"},
		Provider: deviceProvider(c),
	}
	mustSet(t, vm, "device", m.ToValue(vm))

	// The flag-only descriptor reports enumerable=true (so Object.keys keeps it)
	// but its .value is undefined — the real value comes from [[Get]]. This is
	// the documented §6 tradeoff: getOwnPropertyDescriptor(k).value is undefined
	// while device.k returns the lazy value.
	desc := runJS(t, vm,
		`({enum: Object.getOwnPropertyDescriptor(device,"name").enumerable, val: Object.getOwnPropertyDescriptor(device,"name").value})`).Export()
	d := desc.(map[string]any)
	if d["enum"] != true {
		t.Fatalf("descriptor.enumerable = %v, want true", d["enum"])
	}
	if d["val"] != nil {
		t.Fatalf("descriptor.value = %v, want nil (undefined)", d["val"])
	}
	// ...but reading the value works and is lazy:
	if v := runJS(t, vm, `device.name`); v.String() != "atom-3r" {
		t.Fatalf("device.name = %v", v)
	}
}

// --- JSON.stringify unavoidably materialises visited values ------------------

func TestMap_JSONStringifyMaterialises(t *testing.T) {
	vm := newVM(t)
	c := newCounter()
	m := &lazy.Map{
		Keys:     []string{"name", "battery"},
		Provider: deviceProvider(c),
	}
	mustSet(t, vm, "device", m.ToValue(vm))

	out := runJS(t, vm, `JSON.stringify(device)`).String()
	if out != `{"name":"atom-3r","battery":87}` {
		t.Fatalf("JSON.stringify(device) = %s", out)
	}
	for _, k := range m.Keys {
		if c.times(k) != 1 {
			t.Fatalf("JSON.stringify should materialise %q once, got %d", k, c.times(k))
		}
	}
}

// --- Object.assign / spread use [[Get]] and receive lazy values -------------

func TestMap_AssignReceivesLazyValues(t *testing.T) {
	vm := newVM(t)
	c := newCounter()
	m := &lazy.Map{
		Keys:     []string{"name", "battery"},
		Provider: deviceProvider(c),
	}
	mustSet(t, vm, "device", m.ToValue(vm))

	out := runJS(t, vm, `JSON.stringify({...device})`).String()
	if out != `{"name":"atom-3r","battery":87}` {
		t.Fatalf("spread = %s", out)
	}
	for _, k := range m.Keys {
		if c.times(k) != 1 {
			t.Fatalf("spread should materialise %q once via [[Get]], got %d", k, c.times(k))
		}
	}
}

// --- Mutable Map: Set / deleteProperty --------------------------------------

func TestMap_MutableSetAndDelete(t *testing.T) {
	vm := newVM(t)
	c := newCounter()
	m := &lazy.Map{
		Keys:     []string{"name"},
		Provider: deviceProvider(c),
		Mutable:  true,
	}
	mustSet(t, vm, "device", m.ToValue(vm))

	// Set a new key, then read it back.
	runJS(t, vm, `device.custom = 42`)
	if v := runJS(t, vm, `device.custom`); v.ToInteger() != 42 {
		t.Fatalf("device.custom = %v, want 42", v)
	}
	// custom was set, not fetched from the provider:
	if c.times("custom") != 0 {
		t.Fatalf("set value should not call provider, got %d", c.times("custom"))
	}
	// It now appears in Object.keys.
	keys := runJS(t, vm, `JSON.stringify(Object.keys(device))`).String()
	if keys != `["name","custom"]` {
		t.Fatalf("keys after set = %s", keys)
	}

	// Delete it.
	runJS(t, vm, `delete device.custom`)
	keys = runJS(t, vm, `JSON.stringify(Object.keys(device))`).String()
	if keys != `["name"]` {
		t.Fatalf("keys after delete = %s", keys)
	}
}

// --- Array: indexing + length + bounds + caching ---------------------------

func TestArray_IndexingLengthCaching(t *testing.T) {
	vm := newVM(t)
	calls := map[int]int{}
	a := &lazy.Array{
		Len: 1_000_000,
		Provider: func(i int) any {
			calls[i]++
			return "item-" + strconv.Itoa(i*i)
		},
	}
	mustSet(t, vm, "infinite", a.ToValue(vm))

	// length is cheap and correct, no provider calls.
	if v := runJS(t, vm, `infinite.length`); v.ToInteger() != 1_000_000 {
		t.Fatalf("length = %v", v)
	}
	if len(calls) != 0 {
		t.Fatalf("reading length materialised elements: %v", calls)
	}

	// Indexing is lazy and cached.
	out := runJS(t, vm, `[infinite[5], infinite[3], infinite[5]].join("|")`).String()
	if out != "item-25|item-9|item-25" {
		t.Fatalf("indexed values = %s", out)
	}
	if calls[5] != 1 || calls[3] != 1 {
		t.Fatalf("indexing caching wrong: calls=%v", calls)
	}

	// Out-of-bounds is absent.
	if v := runJS(t, vm, `infinite[1000001]`); !goja.IsUndefined(v) {
		t.Fatalf("out-of-bounds = %v, want undefined", v)
	}
	// `in` respects bounds.
	if v := runJS(t, vm, `5 in infinite && !(1000001 in infinite)`).ToBoolean(); !v {
		t.Fatalf("in/bounds check failed")
	}
}

// --- Record: dynamic keys, free enumeration, free has -----------------------

func TestRecord_DynamicKeys(t *testing.T) {
	vm := newVM(t)
	c := newCounter()
	store := map[string]string{"HOME": "/home/manuel", "EDITOR": "nvim"}
	r := &lazy.Record{
		Keys: func() []string {
			out := make([]string, 0, len(store))
			for k := range store {
				out = append(out, k)
			}
			return out
		},
		Has:      func(key string) bool { _, ok := store[key]; return ok },
		Provider: func(key string) any { c.calls[key]++; return store[key] },
	}
	mustSet(t, vm, "env", r.ToValue(vm))

	// Read is lazy.
	if v := runJS(t, vm, `env.EDITOR`); v.String() != "nvim" {
		t.Fatalf("env.EDITOR = %v", v)
	}
	if c.times("EDITOR") != 1 {
		t.Fatalf("EDITOR fetched %d times, want 1", c.times("EDITOR"))
	}

	// Object.keys uses Keys() and does NOT fetch values.
	n := runJS(t, vm, `Object.keys(env).length`).ToInteger()
	if n != int64(len(store)) {
		t.Fatalf("Object.keys(env).length = %d, want %d", n, len(store))
	}
	if c.times("HOME") != 0 {
		t.Fatalf("enumeration fetched HOME (%d times); want 0", c.times("HOME"))
	}

	// `in` is free.
	if v := runJS(t, vm, `'HOME' in env && !('NOPE' in env)`).ToBoolean(); !v {
		t.Fatalf("in/has check failed")
	}
	if c.times("HOME") != 0 {
		t.Fatalf("'in' fetched HOME; want 0")
	}
}

func TestRecord_Mutable(t *testing.T) {
	vm := newVM(t)
	store := map[string]string{"HOME": "/home/manuel"}
	r := &lazy.Record{
		Keys:     func() []string { return []string{"HOME"} },
		Has:      func(key string) bool { _, ok := store[key]; return ok },
		Provider: func(key string) any { return store[key] },
		Mutable:  true,
	}
	mustSet(t, vm, "env", r.ToValue(vm))

	runJS(t, vm, `env.NEW = "added"`)
	if v := runJS(t, vm, `env.NEW`); v.String() != "added" {
		t.Fatalf("env.NEW = %v", v)
	}
	keys := runJS(t, vm, `JSON.stringify(Object.keys(env).sort())`).String()
	if keys != `["HOME","NEW"]` {
		t.Fatalf("keys after set = %s", keys)
	}
}
