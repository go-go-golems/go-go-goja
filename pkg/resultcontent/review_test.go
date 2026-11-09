package resultcontent

import (
	"context"
	"strings"
	"testing"
)

// holderProvider has a value receiver and an interface field: its dynamic type
// is comparable, but a value holding a slice panics when used as a map key.
type holderProvider struct{ payload any }

func (holderProvider) ResultContent(context.Context) ([]Content, error) {
	return []Content{{MIMEType: "text/plain", Data: []byte("h")}}, nil
}

func TestExtractTypedGoCollections(t *testing.T) {
	vm := newVM(t)
	x := &fakeLook{name: "x"}
	if err := vm.Set("looks", map[string]*fakeLook{"b": {name: "b"}, "a": {name: "a"}}); err != nil {
		t.Fatal(err)
	}
	if err := vm.Set("list", []*fakeLook{x, x}); err != nil {
		t.Fatal(err)
	}
	if err := vm.Set("arr", [1]*fakeLook{{name: "arr"}}); err != nil {
		t.Fatal(err)
	}
	ext := extract(t, vm, run(t, vm, `({m: looks, l: list, a: arr})`), Limits{})
	want := `{m: {a: [content 1: look a], b: [content 2: look b]}, l: [[content 3: look x], [content 3: look x]], a: [[content 4: look arr]]}`
	if ext.Text != want {
		t.Fatalf("text =\n%s\nwant\n%s", ext.Text, want)
	}
	if len(ext.Providers) != 4 {
		t.Fatalf("providers = %d", len(ext.Providers))
	}
}

type goNode map[string]any

func TestExtractGoCyclesAndBytes(t *testing.T) {
	vm := newVM(t)
	cyclic := goNode{"img": &fakeLook{name: "c"}}
	cyclic["self"] = cyclic
	if err := vm.Set("cyclic", cyclic); err != nil {
		t.Fatal(err)
	}
	untyped := map[string]any{"img": &fakeLook{name: "u"}}
	untyped["self"] = untyped
	if err := vm.Set("untyped", untyped); err != nil {
		t.Fatal(err)
	}
	// Untyped Go map cycles are bounded by MaxDepth and keep one provider number.
	if ext := extract(t, vm, run(t, vm, `untyped`), Limits{}); !ext.Truncated || len(ext.Providers) != 1 {
		t.Fatalf("untyped cyclic Go map: %#v", ext)
	}
	if err := vm.Set("raw", []byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	ext := extract(t, vm, run(t, vm, `({c: cyclic, raw})`), Limits{})
	if !strings.Contains(ext.Text, "self: [Circular]") || len(ext.Providers) != 1 {
		t.Fatalf("cyclic Go map: %#v", ext)
	}
	if !strings.Contains(ext.Text, "raw: [3 bytes]") {
		t.Fatalf("byte slice: %q", ext.Text)
	}
}

func TestExportedTypedCollectionsAndCycles(t *testing.T) {
	a := &fakeLook{name: "a"}
	in := map[string]any{
		"typed": map[string]*fakeLook{"a": a},
		"list":  []*fakeLook{a, {name: "b"}},
	}
	out, providers := ExtractExported(in, Limits{})
	m := out.(map[string]any)
	list := m["list"].([]any)
	if len(providers) != 2 || m["typed"].(map[string]any)["a"] != "[content 1: look a]" || list[0] != "[content 1: look a]" || list[1] != "[content 2: look b]" {
		t.Fatalf("out = %#v providers=%d", out, len(providers))
	}
	cyclic := map[string]any{"img": a}
	cyclic["self"] = cyclic
	out, providers = ExtractExported(cyclic, Limits{})
	if out.(map[string]any)["self"] != "[Circular]" || len(providers) != 1 {
		t.Fatalf("cyclic = %#v", out)
	}
}

func TestNonComparableProviderValuesDoNotPanic(t *testing.T) {
	h := holderProvider{payload: []int{1}}
	out, providers := ExtractExported([]any{h, h, map[string]any{"x": holderProvider{payload: map[string]int{}}}}, Limits{})
	if len(providers) != 3 || out.([]any)[1] != "[content 2]" {
		t.Fatalf("out = %#v providers=%d", out, len(providers))
	}
	if _, ok := IdentityKey(h); ok {
		t.Fatal("non-comparable value has an identity key")
	}
	if k1, ok := IdentityKey(&fakeLook{}); !ok || k1 == nil {
		t.Fatal("pointer provider has no identity key")
	}
	if _, ok := IdentityKey(holderProvider{payload: 1}); !ok {
		t.Fatal("comparable value provider has no identity key")
	}

	vm := newVM(t)
	if err := vm.Set("holders", []any{h, h}); err != nil {
		t.Fatal(err)
	}
	ext := extract(t, vm, run(t, vm, `holders`), Limits{})
	if len(ext.Providers) != 2 {
		t.Fatalf("extract providers = %d", len(ext.Providers))
	}
}
