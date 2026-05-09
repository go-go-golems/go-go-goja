package uidsl

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func testUI(t *testing.T) *goja.Runtime {
	t.Helper()
	vm := goja.New()
	obj := vm.NewObject()
	exports := vm.NewObject()
	_ = obj.Set("exports", exports)
	Loader(vm, obj)
	_ = vm.Set("ui", exports)
	return vm
}

func renderJS(t *testing.T, script string) string {
	t.Helper()
	vm := testUI(t)
	value, err := vm.RunString(script)
	if err != nil {
		t.Fatal(err)
	}
	html, err := RenderAny(vm, value)
	if err != nil {
		t.Fatal(err)
	}
	return html
}

func TestCodeBlockEscapesSimplePre(t *testing.T) {
	html := renderJS(t, `ui.codeBlock("sql!!", "SELECT '<x>'")`)
	for _, want := range []string{
		`<pre class="ui-codeblock ui-codeblock--sql ui-codeblock--wrap">`,
		`<span class="ui-codeblock__token ui-codeblock__token--keyword">SELECT</span>`,
		`<span class="ui-codeblock__token ui-codeblock__token--string">&#39;&lt;x&gt;&#39;</span>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %s", want, html)
		}
	}
}

func TestCodeBlockFigureOptions(t *testing.T) {
	html := renderJS(t, `ui.codeBlock("", "line1\nline2", { title: "T <x>", lineNumbers: true, wrap: false, copy: true, maxHeight: "120px", class: "extra" })`)
	for _, want := range []string{
		`<figure class="ui-codeblock ui-codeblock--text ui-codeblock--nowrap ui-codeblock--line-numbers extra">`,
		`<figcaption class="ui-codeblock__caption">`,
		`<span class="ui-codeblock__title">T &lt;x&gt;</span>`,
		`<button class="ui-codeblock__copy" type="button">Copy</button>`,
		`<pre class="ui-codeblock__pre" style="max-height:120px;overflow:auto">`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %s", want, html)
		}
	}
}

func TestCodeBlockAliasesAndJSON(t *testing.T) {
	html := renderJS(t, `ui.fragment(
		ui.sql("SELECT 1"),
		ui.js("const x = 1"),
		ui.jsonBlock({ b: 2, a: "<x>" }),
		ui.jsonBlock('{"z":1}')
	)`)
	for _, want := range []string{
		`ui-codeblock--sql`,
		`language-javascript`,
		`<span class="ui-codeblock__token ui-codeblock__token--key">&#34;a&#34;</span>`,
		`<span class="ui-codeblock__token ui-codeblock__token--string">&#34;\u003cx\u003e&#34;</span>`,
		`<span class="ui-codeblock__token ui-codeblock__token--number">1</span>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %s", want, html)
		}
	}
}

func TestBadgeRendersToneAndEscapes(t *testing.T) {
	html := renderJS(t, `ui.badge("yes <ok>", { tone: "success", title: "A <B>", class: "extra" })`)
	for _, want := range []string{
		`<span class="ui-badge ui-badge--success ui-badge--value-yes-ok extra" title="A &lt;B&gt;">yes &lt;ok&gt;</span>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %s", want, html)
		}
	}
}

func TestBadgeUnknownToneDefaults(t *testing.T) {
	html := renderJS(t, `ui.badge("view", { tone: "weird" })`)
	if !strings.Contains(html, `ui-badge--default`) {
		t.Fatalf("expected default tone in %s", html)
	}
}

func TestTabsRenderSelectedDuplicateDisabledAndEscaped(t *testing.T) {
	html := renderJS(t, `ui.tabs("Record Tabs!", [
		{ id: "summary", label: "Summary <x>", content: "safe <content>" },
		{ id: "json", label: "JSON", content: ui.jsonBlock({ a: 1 }) },
		{ id: "json", label: "Duplicate", content: "second" },
		{ id: "disabled", label: "Disabled", content: "no", disabled: true }
	], { selected: "json", class: "extra" })`)
	for _, want := range []string{
		`<div class="ui-tabs extra" id="record-tabs">`,
		`#record-tabs-json:checked~.ui-tabs__panels>[data-tab="json"]{display:block;}`,
		`<input class="ui-tabs__radio" id="record-tabs-summary" name="record-tabs" type="radio">`,
		`<label class="ui-tabs__tab" for="record-tabs-summary">Summary &lt;x&gt;</label>`,
		`<input checked class="ui-tabs__radio" id="record-tabs-json" name="record-tabs" type="radio">`,
		`data-tab="json-2"`,
		`disabled`,
		`ui-tabs__tab--disabled`,
		`safe &lt;content&gt;`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %s", want, html)
		}
	}
}

func TestTabsInvalidSelectionFallsBack(t *testing.T) {
	html := renderJS(t, `ui.tabs("tabs", [
		{ id: "off", label: "Off", content: "off", disabled: true },
		{ id: "on", label: "On", content: "on" }
	], { selected: "missing" })`)
	if !strings.Contains(html, `<input checked class="ui-tabs__radio" id="tabs-on" name="tabs" type="radio">`) {
		t.Fatalf("expected first non-disabled tab selected in %s", html)
	}
}
