package uidsl

import (
	"strings"
	"testing"
)

func TestUIDSLAttributeCompatibility(t *testing.T) {
	html := renderJS(t, `ui.div({
		class: ["base", false, "extra", ""],
		style: { color: "red", "font-weight": "bold" },
		id: "node-1",
		"data-index": 7,
		"aria-label": "A <B>",
		hidden: true,
		draggable: false,
		value: "",
		title: null,
		role: undefined
	}, "hello")`)
	for _, want := range []string{
		`aria-label="A &lt;B&gt;"`,
		`class="base extra"`,
		`data-index="7"`,
		`hidden`,
		`id="node-1"`,
		`style="color:red;font-weight:bold"`,
		`value=""`,
		`>hello</div>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %s", want, html)
		}
	}
	for _, notWant := range []string{`draggable=`, `title=`, `role=`} {
		if strings.Contains(html, notWant) {
			t.Fatalf("unexpected %q in %s", notWant, html)
		}
	}
}

func TestUIDSLChildVsAttrsDisambiguation(t *testing.T) {
	cases := []struct {
		name   string
		script string
		want   string
	}{
		{"text", `ui.div("text")`, `<div>text</div>`},
		{"number", `ui.div(42)`, `<div>42</div>`},
		{"bool", `ui.div(true)`, `<div>true</div>`},
		{"node", `ui.div(ui.span("child"))`, `<div><span>child</span></div>`},
		{"array", `ui.div([ui.span("child")])`, `<div><span>child</span></div>`},
		{"fragment", `ui.div(ui.fragment(ui.span("child")))`, `<div><span>child</span></div>`},
		{"empty attrs", `ui.div({})`, `<div></div>`},
		{"attrs", `ui.div({ class: "x" })`, `<div class="x"></div>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := renderJS(t, tc.script); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
