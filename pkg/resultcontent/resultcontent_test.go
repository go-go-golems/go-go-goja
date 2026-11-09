package resultcontent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

type fakeLook struct {
	name  string
	data  []byte
	calls int
	err   error
	panic bool
}

var _ Provider = &fakeLook{}
var _ Describer = &fakeLook{}

func (f *fakeLook) ResultContent(context.Context) ([]Content, error) {
	f.calls++
	if f.panic {
		panic("boom")
	}
	if f.err != nil {
		return nil, f.err
	}
	return []Content{{MIMEType: "image/png", Data: f.data, Caption: f.name}}, nil
}

func (f *fakeLook) ResultPlaceholder() string { return "look " + f.name }

type plainProvider struct{}

func (plainProvider) ResultContent(context.Context) ([]Content, error) {
	return []Content{{MIMEType: "text/plain", Data: []byte("hi")}}, nil
}

func newVM(t *testing.T) *goja.Runtime {
	t.Helper()
	vm := goja.New()
	if err := vm.Set("look", func(name string) *fakeLook { return &fakeLook{name: name, data: []byte(name)} }); err != nil {
		t.Fatal(err)
	}
	return vm
}

func run(t *testing.T, vm *goja.Runtime, src string) goja.Value {
	t.Helper()
	v, err := vm.RunString(src)
	if err != nil {
		t.Fatalf("run %q: %v", src, err)
	}
	return v
}

func extract(t *testing.T, vm *goja.Runtime, v goja.Value, limits Limits) Extraction {
	t.Helper()
	ext, err := Extract(vm, v, limits)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	return ext
}

func TestExtractTopLevelProvider(t *testing.T) {
	vm := newVM(t)
	ext := extract(t, vm, run(t, vm, `look("a")`), Limits{})
	if ext.Text != "[content 1: look a]" {
		t.Fatalf("text = %q", ext.Text)
	}
	if len(ext.Providers) != 1 {
		t.Fatalf("providers = %d", len(ext.Providers))
	}
}

func TestExtractNestedProvidersAndDedupe(t *testing.T) {
	vm := newVM(t)
	v := run(t, vm, `const b = look("b"); ({stats: {mean: 0.5, label: "x"}, imgs: [look("a"), b], again: b, "odd key": null, n: undefined})`)
	ext := extract(t, vm, v, Limits{})
	want := `{stats: {mean: 0.5, label: "x"}, imgs: [[content 1: look a], [content 2: look b]], again: [content 2: look b], "odd key": null, n: undefined}`
	if ext.Text != want {
		t.Fatalf("text =\n%s\nwant\n%s", ext.Text, want)
	}
	if len(ext.Providers) != 2 {
		t.Fatalf("providers = %d", len(ext.Providers))
	}
	if ext.Truncated {
		t.Fatal("unexpected truncation")
	}
}

func TestExtractProviderWithoutDescriber(t *testing.T) {
	vm := goja.New()
	if err := vm.Set("p", plainProvider{}); err != nil {
		t.Fatal(err)
	}
	ext := extract(t, vm, run(t, vm, `[p]`), Limits{})
	if ext.Text != "[[content 1]]" {
		t.Fatalf("text = %q", ext.Text)
	}
}

func TestExtractGoMapIsWalked(t *testing.T) {
	vm := newVM(t)
	v := vm.ToValue(map[string]any{"img": &fakeLook{name: "m"}})
	ext := extract(t, vm, v, Limits{})
	if ext.Text != "{img: [content 1: look m]}" || len(ext.Providers) != 1 {
		t.Fatalf("text = %q providers=%d", ext.Text, len(ext.Providers))
	}
}

func TestExtractNeverTouchesThenGettersOrProxies(t *testing.T) {
	vm := newVM(t)
	v := run(t, vm, `
globalThis.touched = [];
const thenable = { then(resolve) { touched.push("then-called"); resolve(1) }, img: look("t") };
Object.defineProperty(thenable, "lazy", { enumerable: true, get() { touched.push("getter"); return 1 } });
const proxy = new Proxy({}, {
  get() { touched.push("proxy-get"); return 1 },
  ownKeys() { touched.push("proxy-keys"); return [] },
  getPrototypeOf() { touched.push("proxy-proto"); return null },
  getOwnPropertyDescriptor() { touched.push("proxy-desc"); return undefined },
});
const thenGetter = {};
Object.defineProperty(thenGetter, "then", { enumerable: true, get() { touched.push("then-getter") } });
({ thenable, proxy, thenGetter, p: Promise.resolve(1), f() {} })
`)
	ext := extract(t, vm, v, Limits{})
	touched := run(t, vm, `touched.join(",")`).String()
	if touched != "" {
		t.Fatalf("walk touched user code: %s", touched)
	}
	for _, want := range []string{"img: [content 1: look t]", "lazy: [Getter]", "proxy: [Proxy]", "thenGetter: {}", "p: [Promise]", "f: [Function]"} {
		if !strings.Contains(ext.Text, want) {
			t.Fatalf("text %q missing %q", ext.Text, want)
		}
	}
	if strings.Contains(ext.Text, "then") && !strings.Contains(ext.Text, "thenable") {
		t.Fatalf("then key rendered: %q", ext.Text)
	}
}

func TestExtractCycleAndDepth(t *testing.T) {
	vm := newVM(t)
	ext := extract(t, vm, run(t, vm, `const o = {a: look("a")}; o.self = o; o`), Limits{})
	if ext.Text != "{a: [content 1: look a], self: [Circular]}" {
		t.Fatalf("text = %q", ext.Text)
	}

	ext = extract(t, vm, run(t, vm, `({a: {b: {c: {d: look("deep")}}}})`), Limits{MaxDepth: 2})
	if !ext.Truncated || len(ext.Providers) != 0 || ext.Text != "{a: {b: …}}" {
		t.Fatalf("depth-limited extraction = %#v", ext)
	}
}

func TestExtractPropertyAndTextLimits(t *testing.T) {
	vm := newVM(t)
	ext := extract(t, vm, run(t, vm, `const a = new Array(50).fill(1); a.push(look("late")); a`), Limits{MaxProperties: 10})
	if !ext.Truncated || len(ext.Providers) != 0 {
		t.Fatalf("property-limited extraction = %#v", ext)
	}
	ext = extract(t, vm, run(t, vm, `[look("x"), "`+strings.Repeat("y", 200)+`"]`), Limits{MaxTextChars: 40})
	if !ext.Truncated || len(ext.Providers) != 1 || !strings.HasSuffix(ext.Text, "…") || len(ext.Text) > 45 {
		t.Fatalf("text-limited extraction = %#v", ext)
	}
}

func TestExtractErrorsAndDates(t *testing.T) {
	vm := newVM(t)
	ext := extract(t, vm, run(t, vm, `({e: new TypeError("bad"), d: new Date(0)})`), Limits{})
	if ext.Text != `{e: [Error: bad], d: 1970-01-01T00:00:00Z}` {
		t.Fatalf("text = %q", ext.Text)
	}
}

func TestProviderOfIgnoresPlainValues(t *testing.T) {
	vm := newVM(t)
	for _, src := range []string{`1`, `"s"`, `({})`, `[1]`, `new Proxy({}, {})`} {
		if _, ok := ProviderOf(run(t, vm, src)); ok {
			t.Fatalf("%s reported as provider", src)
		}
	}
	if _, ok := ProviderOf(run(t, vm, `look("a")`)); !ok {
		t.Fatal("look not reported as provider")
	}
}

func TestRenderLimitsAndErrors(t *testing.T) {
	ctx := context.Background()
	a := &fakeLook{name: "a", data: make([]byte, 10)}
	big := &fakeLook{name: "big", data: make([]byte, 100)}
	failing := &fakeLook{name: "f", err: errors.New("nope")}
	panicking := &fakeLook{name: "p", panic: true}
	b := &fakeLook{name: "b", data: make([]byte, 10)}
	late := &fakeLook{name: "late", data: make([]byte, 1)}

	r := Render(ctx, []Provider{a, big, failing, panicking, b, late}, Limits{MaxItems: 2, MaxTotalBytes: 50})
	if len(r.Content) != 2 || r.Content[0].Caption != "a" || r.Content[1].Caption != "b" {
		t.Fatalf("content = %#v", r.Content)
	}
	if late.calls != 0 {
		t.Fatal("provider after item limit was called")
	}
	if len(r.Refs) != 6 {
		t.Fatalf("refs = %#v", r.Refs)
	}
	wantDropped := map[int]string{2: "byte limit", 3: "render error: nope", 4: "provider panicked", 6: "not rendered: item limit"}
	for _, ref := range r.Refs {
		want, dropped := wantDropped[ref.Provider]
		if dropped != (ref.Index < 0) {
			t.Fatalf("ref %#v: dropped mismatch", ref)
		}
		if dropped && !strings.Contains(ref.Dropped, want) {
			t.Fatalf("ref %#v: want reason containing %q", ref, want)
		}
	}
	if notes := r.Notes(); len(notes) != 4 || !strings.Contains(notes[0], "content 2 (image/png, 100 bytes) dropped") {
		t.Fatalf("notes = %#v", notes)
	}
}

func TestRenderCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a := &fakeLook{name: "a"}
	r := Render(ctx, []Provider{a}, Limits{})
	if a.calls != 0 || len(r.Content) != 0 || len(r.Refs) != 1 || r.Refs[0].Index != -1 {
		t.Fatalf("rendering = %#v calls=%d", r, a.calls)
	}
}
