package scriptresult

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/resultcontent"
)

type look struct {
	name  string
	calls *int
}

func (l *look) ResultContent(context.Context) ([]resultcontent.Content, error) {
	*l.calls++
	return []resultcontent.Content{{MIMEType: "image/png", Data: []byte("png:" + l.name), Caption: l.name}}, nil
}

func (l *look) ResultPlaceholder() string { return "look " + l.name }

type lookInit struct{ calls *int }

func (lookInit) ID() string { return "scriptresult-test-look" }

func (i lookInit) InitRuntime(ctx *engine.RuntimeInitializationContext) error {
	return ctx.VM.Set("look", func(name string) *look { return &look{name: name, calls: i.calls} })
}

func newRuntime(t *testing.T, calls *int) *engine.Runtime {
	t.Helper()
	factory, err := engine.NewRuntimeFactoryBuilder().UseModuleMiddleware(engine.MiddlewareSafe()).WithRuntimeInitializers(lookInit{calls: calls}).Build()
	if err != nil {
		t.Fatal(err)
	}
	rt, err := factory.NewRuntime()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rt.Close(context.Background()) })
	return rt
}

func eval(t *testing.T, rt *engine.Runtime, src string) goja.Value {
	t.Helper()
	ret, err := rt.Owner.Call(context.Background(), "test", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(src)
	})
	if err != nil {
		t.Fatalf("run %q: %v", src, err)
	}
	return ret.(goja.Value)
}

func TestBuildWithoutContentDirDoesNotRender(t *testing.T) {
	calls := 0
	rt := newRuntime(t, &calls)
	ctx := context.Background()
	report, err := Build(ctx, rt.Owner, eval(t, rt, `({mean: 1, img: look("a")})`), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !report.HasContent() || report.Text != "{mean: 1, img: [content 1: look a]}" || calls != 0 {
		t.Fatalf("report=%#v calls=%d", report, calls)
	}
	out := &bytes.Buffer{}
	if err := report.Fprint(out); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "{mean: 1, img: [content 1: look a]}\n# 1 content item(s) not written; pass --content-dir DIR to write them\n" {
		t.Fatalf("output = %q", got)
	}

	report, err = Build(ctx, rt.Owner, eval(t, rt, `({plain: 1})`), Options{ContentDir: t.TempDir()})
	if err != nil || report.HasContent() {
		t.Fatalf("plain value: report=%#v err=%v", report, err)
	}
	out.Reset()
	_ = report.Fprint(out)
	if out.Len() != 0 {
		t.Fatalf("plain value printed %q", out.String())
	}
}

func TestSettleAndBuildWithContentDir(t *testing.T) {
	calls := 0
	rt := newRuntime(t, &calls)
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "looks")
	value, err := Settle(ctx, rt.Owner, eval(t, rt, `(async () => { await null; return [look("a"), look("b")] })()`))
	if err != nil {
		t.Fatalf("settle: %v", err)
	}
	report, err := Build(ctx, rt.Owner, value, Options{ContentDir: dir, Limits: resultcontent.Limits{MaxItems: 1}})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(report.Files) != 1 || filepath.Base(report.Files[0].Path) != "content-1.png" || calls != 1 {
		t.Fatalf("files=%#v calls=%d", report.Files, calls)
	}
	data, err := os.ReadFile(report.Files[0].Path)
	if err != nil || string(data) != "png:a" {
		t.Fatalf("file = %q %v", data, err)
	}
	out := &bytes.Buffer{}
	_ = report.Fprint(out)
	text := out.String()
	if !strings.Contains(text, "[[content 1: look a], [content 2: look b]]") ||
		!strings.Contains(text, "# content 1 a: "+report.Files[0].Path) ||
		!strings.Contains(text, "# content 2 dropped: not rendered: item limit (1) reached") {
		t.Fatalf("output = %q", text)
	}
}

func TestSettleRejectionAndPlainValues(t *testing.T) {
	calls := 0
	rt := newRuntime(t, &calls)
	ctx := context.Background()
	if _, err := Settle(ctx, rt.Owner, eval(t, rt, `Promise.reject(new Error("nope"))`)); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("rejection: %v", err)
	}
	v := eval(t, rt, `globalThis.touched = false; ({ get x() { touched = true; return 1 } })`)
	if got, err := Settle(ctx, rt.Owner, v); err != nil || got != v {
		t.Fatalf("plain value: %v %v", got, err)
	}
	if eval(t, rt, `touched`).ToBoolean() {
		t.Fatal("Settle invoked a getter on a plain object")
	}
	if got, err := Settle(ctx, rt.Owner, nil); err != nil || got != nil {
		t.Fatalf("nil value: %v %v", got, err)
	}
}
