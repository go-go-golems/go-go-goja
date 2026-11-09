package resultcontent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type multiProvider struct{}

func (multiProvider) ResultContent(context.Context) ([]Content, error) {
	return []Content{
		{MIMEType: "image/jpeg", Data: []byte("jpeg")},
		{MIMEType: "application/json; charset=utf-8", Data: []byte(`{"a":1}`)},
	}, nil
}

func TestExtensionForMIME(t *testing.T) {
	for mimeType, want := range map[string]string{
		"image/png":                 ".png",
		"IMAGE/JPEG":                ".jpg",
		"text/plain; charset=utf-8": ".txt",
		"application/x-unknown-xyz": ".bin",
		"":                          ".bin",
	} {
		if got := ExtensionForMIME(mimeType); got != want {
			t.Fatalf("ExtensionForMIME(%q) = %q, want %q", mimeType, got, want)
		}
	}
}

func TestWriteFilesNamesAndSkipsDropped(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "out")
	a := &fakeLook{name: "a", data: []byte("png-a")}
	big := &fakeLook{name: "big", data: make([]byte, 100)}
	rendering, files, err := RenderToDir(ctx, []Provider{a, multiProvider{}, big}, Limits{MaxTotalBytes: 50}, dir, "")
	if err != nil {
		t.Fatalf("render to dir: %v", err)
	}
	if len(rendering.Dropped()) != 1 {
		t.Fatalf("dropped = %#v", rendering.Dropped())
	}
	var names []string
	for _, f := range files {
		names = append(names, filepath.Base(f.Path))
	}
	if strings.Join(names, ",") != "content-1.png,content-2-1.jpg,content-2-2.json" {
		t.Fatalf("names = %v", names)
	}
	data, err := os.ReadFile(files[0].Path)
	if err != nil || string(data) != "png-a" {
		t.Fatalf("content-1.png = %q, %v", data, err)
	}
	if files[2].Provider != 2 || files[2].Item != 2 || files[0].Caption != "a" {
		t.Fatalf("files = %#v", files)
	}
	if paths := PathsByProvider(files); len(paths[2]) != 2 || len(paths[1]) != 1 {
		t.Fatalf("paths by provider = %#v", paths)
	}
}

func TestWriteFilesPrefixAndEmpty(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if _, err := WriteFiles(ctx, Render(ctx, []Provider{&fakeLook{name: "x"}}, Limits{}), dir, "../x"); err == nil {
		t.Fatal("expected prefix with separator to fail")
	}
	files, err := WriteFiles(ctx, Rendering{}, filepath.Join(dir, "never-created"), "p")
	if err != nil || len(files) != 0 {
		t.Fatalf("empty rendering: files=%v err=%v", files, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "never-created")); !os.IsNotExist(err) {
		t.Fatal("directory created for an empty rendering")
	}
	files, err = WriteFiles(ctx, Render(ctx, []Provider{&fakeLook{name: "x", data: []byte("x")}}, Limits{}), dir, "cell-3")
	if err != nil || len(files) != 1 || filepath.Base(files[0].Path) != "cell-3-1.png" {
		t.Fatalf("prefixed: files=%v err=%v", files, err)
	}
	if _, err := WriteFiles(ctx, Rendering{}, " ", "p"); err == nil {
		t.Fatal("expected empty dir error")
	}
}

func TestExtractExported(t *testing.T) {
	a := &fakeLook{name: "a"}
	b := &fakeLook{name: "b"}
	in := map[string]any{
		"z":     a,
		"stats": map[string]any{"mean": 0.5},
		"looks": []any{b, a},
		"n":     int64(3),
	}
	out, providers := ExtractExported(in, Limits{})
	m := out.(map[string]any)
	if len(providers) != 2 {
		t.Fatalf("providers = %d", len(providers))
	}
	// keys sorted: looks, n, stats, z → b is 1, a is 2
	looks := m["looks"].([]any)
	if looks[0] != "[content 1: look b]" || looks[1] != "[content 2: look a]" || m["z"] != "[content 2: look a]" {
		t.Fatalf("out = %#v", out)
	}
	if m["stats"].(map[string]any)["mean"] != 0.5 || m["n"] != int64(3) {
		t.Fatalf("plain values changed: %#v", out)
	}
	if in["z"] != a {
		t.Fatal("input mutated")
	}
	if v, p := ExtractExported(a, Limits{}); v != "[content 1: look a]" || len(p) != 1 {
		t.Fatalf("top-level provider: %v %d", v, len(p))
	}
}

func TestMapExportedCustomReplacementAndLimits(t *testing.T) {
	a := &fakeLook{name: "a"}
	out, providers := MapExported([]any{a}, Limits{}, func(n int, p Provider) any { return "path-" + p.(*fakeLook).name })
	if out.([]any)[0] != "path-a" || len(providers) != 1 {
		t.Fatalf("out = %#v", out)
	}
	cyclic := map[string]any{"img": a}
	cyclic["self"] = cyclic
	out, providers = ExtractExported(cyclic, Limits{MaxDepth: 3})
	if len(providers) != 1 || out.(map[string]any)["self"] != "[Circular]" {
		t.Fatalf("cyclic: providers=%d out=%#v", len(providers), out)
	}
	nested := map[string]any{"a": map[string]any{"b": map[string]any{"c": map[string]any{"img": a}}}}
	out, _ = ExtractExported(nested, Limits{MaxDepth: 3})
	if deep := out.(map[string]any)["a"].(map[string]any)["b"].(map[string]any)["c"]; deep != "…" {
		t.Fatalf("depth limit kept container: %#v", deep)
	}
	many := make([]any, 20)
	for i := range many {
		many[i] = []any{a}
	}
	out, _ = ExtractExported(many, Limits{MaxProperties: 5})
	if out.([]any)[19] != "…" {
		t.Fatalf("property limit kept container: %#v", out.([]any)[19])
	}
}

func TestInlinePlaceholder(t *testing.T) {
	if got := InlinePlaceholder(&fakeLook{name: "a"}); got != "[content: look a]" {
		t.Fatalf("inline = %q", got)
	}
	if got := InlinePlaceholder(plainProvider{}); got != "[content]" {
		t.Fatalf("inline = %q", got)
	}
}
