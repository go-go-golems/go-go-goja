package app

import "testing"

func TestJSVerbSourceSetListIncludesTypeScriptDescriptor(t *testing.T) {
	set := newJSVerbSourceSet(nil, nil, []SourcePlan{{
		ID:         "local",
		Path:       "verbs",
		Extensions: []string{".ts"},
		TypeScript: &TypeScriptPlan{
			Enabled:      true,
			Bundle:       true,
			Target:       "es2015",
			Format:       "cjs",
			Platform:     "neutral",
			External:     []string{"express"},
			Define:       map[string]string{"process.env.NODE_ENV": "\"test\""},
			CheckCommand: []string{"tsc", "--noEmit"},
		},
	}}, nil)

	sources := set.ListJSVerbSources()
	if len(sources) != 1 {
		t.Fatalf("sources len = %d, want 1", len(sources))
	}
	if sources[0].TypeScript == nil || !sources[0].TypeScript.Enabled || sources[0].TypeScript.Target != "es2015" {
		t.Fatalf("typescript descriptor = %#v", sources[0].TypeScript)
	}
	sources[0].TypeScript.External[0] = "mutated"
	sources[0].TypeScript.Define["process.env.NODE_ENV"] = "mutated"

	again := set.ListJSVerbSources()
	if got := again[0].TypeScript.External[0]; got != "express" {
		t.Fatalf("external alias mutated source set: %q", got)
	}
	if got := again[0].TypeScript.Define["process.env.NODE_ENV"]; got != "\"test\"" {
		t.Fatalf("define mutated source set: %q", got)
	}
}
