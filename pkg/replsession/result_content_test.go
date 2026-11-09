package replsession

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/resultcontent"
	"github.com/rs/zerolog"
)

const testLookSecret = "SECRET-IMAGE-BYTES"

// testLook mimics a cv look: a Go value with exported fields (so a naive
// JSON.stringify or json.Marshal would leak its bytes) that implements Provider.
type testLook struct {
	Name string
	Data []byte
}

var _ resultcontent.Provider = &testLook{}
var _ resultcontent.Describer = &testLook{}

func (l *testLook) ResultContent(context.Context) ([]resultcontent.Content, error) {
	return []resultcontent.Content{{MIMEType: "image/png", Data: l.Data, Caption: l.Name}}, nil
}

func (l *testLook) ResultPlaceholder() string { return "look " + l.Name }

type lookInitializer struct{}

func (lookInitializer) ID() string { return "test-look" }

func (lookInitializer) InitRuntime(ctx *engine.RuntimeInitializationContext) error {
	return ctx.VM.Set("look", func(name string) *testLook {
		return &testLook{Name: name, Data: []byte(testLookSecret + ":" + name)}
	})
}

func newLookTestFactory(t *testing.T) *engine.RuntimeFactory {
	t.Helper()
	factory, err := engine.NewRuntimeFactoryBuilder().
		UseModuleMiddleware(engine.MiddlewareSafe()).
		WithRuntimeInitializers(lookInitializer{}).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	return factory
}

func evalOK(t *testing.T, service *Service, sessionID string, source string) *EvaluateResponse {
	t.Helper()
	resp, err := service.Evaluate(context.Background(), sessionID, source)
	if err != nil {
		t.Fatalf("evaluate %q: %v", source, err)
	}
	if resp.Cell.Execution.Status != "ok" {
		t.Fatalf("evaluate %q: status %q error %q", source, resp.Cell.Execution.Status, resp.Cell.Execution.Error)
	}
	return resp
}

func newLookSession(t *testing.T, opts SessionOptions, serviceOpts ...Option) (*Service, string) {
	t.Helper()
	serviceOpts = append([]Option{WithDefaultSessionOptions(opts)}, serviceOpts...)
	service := NewService(newLookTestFactory(t), zerolog.Nop(), serviceOpts...)
	t.Cleanup(func() { _ = service.Close(context.Background()) })
	session, err := service.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	return service, session.ID
}

func TestResultContentTopLevelProvider(t *testing.T) {
	t.Parallel()
	for _, opts := range []SessionOptions{InteractiveSessionOptions(), RawSessionOptions()} {
		service, id := newLookSession(t, opts)
		resp := evalOK(t, service, id, `look("a")`)
		if resp.Cell.Execution.Result != "[content 1: look a]" {
			t.Fatalf("%s: result = %q", opts.Profile, resp.Cell.Execution.Result)
		}
		if len(resp.Content) != 1 || resp.Content[0].MIMEType != "image/png" || string(resp.Content[0].Data) != testLookSecret+":a" {
			t.Fatalf("%s: content = %#v", opts.Profile, resp.Content)
		}
		refs := resp.Cell.Execution.Content
		if len(refs) != 1 || refs[0].Index != 0 || refs[0].Provider != 1 || refs[0].Caption != "a" || refs[0].Bytes != len(testLookSecret)+2 {
			t.Fatalf("%s: refs = %#v", opts.Profile, refs)
		}
		if strings.Contains(resp.Cell.Execution.ResultJSON, "Data") || !strings.Contains(resp.Cell.Execution.ResultJSON, "[content 1: look a]") {
			t.Fatalf("%s: result JSON = %s", opts.Profile, resp.Cell.Execution.ResultJSON)
		}
	}
}

func TestResultContentNestedProviders(t *testing.T) {
	t.Parallel()
	service, id := newLookSession(t, InteractiveSessionOptions())
	resp := evalOK(t, service, id, `const after = look("after"); ({stats: {meanL: 0.33}, looks: [look("before"), after], again: after})`)
	want := `{stats: {meanL: 0.33}, looks: [[content 1: look before], [content 2: look after]], again: [content 2: look after]}`
	if resp.Cell.Execution.Result != want {
		t.Fatalf("result =\n%s\nwant\n%s", resp.Cell.Execution.Result, want)
	}
	if len(resp.Content) != 2 || resp.Content[0].Caption != "before" || resp.Content[1].Caption != "after" {
		t.Fatalf("content = %#v", resp.Content)
	}
	var envelope map[string]any
	if err := json.Unmarshal([]byte(resp.Cell.Execution.ResultJSON), &envelope); err != nil {
		t.Fatalf("result JSON: %v", err)
	}
	result := envelope["result"].(map[string]any)
	looks := result["looks"].([]any)
	if looks[0] != "[content 1: look before]" || looks[1] != "[content 2: look after]" || result["again"] != "[content 2: look after]" {
		t.Fatalf("result JSON = %s", resp.Cell.Execution.ResultJSON)
	}
}

func TestResultContentPlainValuesUnchanged(t *testing.T) {
	t.Parallel()
	service, id := newLookSession(t, InteractiveSessionOptions())
	resp := evalOK(t, service, id, `({a: 1, b: [2]})`)
	if resp.Cell.Execution.Result != "{a, b}" || len(resp.Content) != 0 || resp.Cell.Execution.Content != nil {
		t.Fatalf("plain value changed: result=%q content=%#v refs=%#v", resp.Cell.Execution.Result, resp.Content, resp.Cell.Execution.Content)
	}
}

func TestResultContentThenableIsNotAwaitedOrCalled(t *testing.T) {
	t.Parallel()
	for _, opts := range []SessionOptions{InteractiveSessionOptions(), RawSessionOptions()} {
		service, id := newLookSession(t, opts)
		resp := evalOK(t, service, id, `globalThis.thenCalls = 0; ({ then(resolve) { thenCalls++; resolve(1) }, img: look("t") })`)
		if !strings.Contains(resp.Cell.Execution.Result, "img: [content 1: look t]") || len(resp.Content) != 1 {
			t.Fatalf("%s: result=%q content=%d", opts.Profile, resp.Cell.Execution.Result, len(resp.Content))
		}
		check := evalOK(t, service, id, `thenCalls`)
		if check.Cell.Execution.Result != "0" {
			t.Fatalf("%s: then was called %s times", opts.Profile, check.Cell.Execution.Result)
		}
	}
}

func TestResultContentAwaitedProvider(t *testing.T) {
	t.Parallel()
	service, id := newLookSession(t, InteractiveSessionOptions())
	resp := evalOK(t, service, id, `await Promise.resolve({img: look("late")})`)
	if !resp.Cell.Execution.Awaited || len(resp.Content) != 1 || resp.Cell.Execution.Result != "{img: [content 1: look late]}" {
		t.Fatalf("awaited result=%q content=%d", resp.Cell.Execution.Result, len(resp.Content))
	}
}

func TestResultContentLimitsReportDrops(t *testing.T) {
	t.Parallel()
	service, id := newLookSession(t, InteractiveSessionOptions(), WithResultContentLimits(resultcontent.Limits{MaxItems: 1}))
	resp := evalOK(t, service, id, `[look("a"), look("b")]`)
	refs := resp.Cell.Execution.Content
	if len(resp.Content) != 1 || len(refs) != 2 || refs[1].Index != -1 || !strings.Contains(refs[1].Dropped, "item limit") {
		t.Fatalf("content=%d refs=%#v", len(resp.Content), refs)
	}
}

func TestResultContentIsNotRetainedOrPersisted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := openPersistenceTestStore(t)
	defer func() { _ = store.Close() }()
	service := NewService(newLookTestFactory(t), zerolog.Nop(), WithPersistence(store))
	defer func() { _ = service.Close(ctx) }()
	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	resp := evalOK(t, service, session.ID, `const l = look("kept"); ({l, n: 1})`)
	if len(resp.Content) != 1 {
		t.Fatalf("content = %#v", resp.Content)
	}

	state, err := service.getSession(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, cell := range state.cells {
		if cell.content != nil {
			t.Fatal("committed cell retained content bytes")
		}
	}

	exported, err := store.ExportSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	raw, err := json.Marshal(exported)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "SECRET") || strings.Contains(string(raw), "U0VDUkVU") || strings.Contains(string(raw), `\"Data\"`) {
		t.Fatalf("persisted export leaks content bytes: %s", raw)
	}
	if len(exported.Evaluations) != 1 {
		t.Fatalf("evaluations = %d", len(exported.Evaluations))
	}
	var cell CellReport
	if err := json.Unmarshal(exported.Evaluations[0].ResultJSON, &cell); err != nil {
		t.Fatalf("decode cell: %v", err)
	}
	if len(cell.Execution.Content) != 1 || cell.Execution.Content[0].Caption != "kept" || !strings.Contains(cell.Execution.Result, "[content 1: look kept]") {
		t.Fatalf("persisted refs/placeholder missing: %#v", cell.Execution)
	}
}

type recordingProcessor struct {
	before int
}

var _ ResultProcessor = &recordingProcessor{}
var _ BeforeEvalHook = &recordingProcessor{}
var _ FailureProcessor = &recordingProcessor{}

func (p *recordingProcessor) BeforeEval(context.Context, *goja.Runtime) error {
	p.before++
	return nil
}

func (p *recordingProcessor) Process(ctx context.Context, vm *goja.Runtime, value goja.Value) (ProcessedResult, error) {
	result, err := DefaultResultProcessor{}.Process(ctx, vm, value)
	if err != nil {
		return result, err
	}
	result.Text = "custom: " + gojaValuePreview(value, vm)
	return result, nil
}

func (p *recordingProcessor) ProcessFailure(_ context.Context, vm *goja.Runtime, evalErr error) (ProcessedResult, error) {
	queued, _ := vm.Get("queued").Export().(*testLook)
	if queued == nil {
		return ProcessedResult{}, nil
	}
	return ProcessedResult{Text: "failed after queueing", Providers: []resultcontent.Provider{queued}}, nil
}

func TestCustomResultProcessorHooks(t *testing.T) {
	t.Parallel()
	processor := &recordingProcessor{}
	service, id := newLookSession(t, InteractiveSessionOptions(), WithResultProcessor(processor))

	resp := evalOK(t, service, id, `1 + 1`)
	if resp.Cell.Execution.Result != "custom: 2" || processor.before != 1 {
		t.Fatalf("result=%q before=%d", resp.Cell.Execution.Result, processor.before)
	}

	resp, err := service.Evaluate(context.Background(), id, `globalThis.queued = look("q"); throw new Error("later")`)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if resp.Cell.Execution.Status != "runtime-error" || resp.Cell.Execution.Result != "failed after queueing" || len(resp.Content) != 1 {
		t.Fatalf("failure: status=%q result=%q content=%d", resp.Cell.Execution.Status, resp.Cell.Execution.Result, len(resp.Content))
	}
	if processor.before != 2 {
		t.Fatalf("before = %d", processor.before)
	}
}

func TestDefaultResultProcessorFullText(t *testing.T) {
	t.Parallel()
	service, id := newLookSession(t, InteractiveSessionOptions(), WithResultProcessor(DefaultResultProcessor{FullText: true}))
	resp := evalOK(t, service, id, `({a: 1, b: [2, "three"]})`)
	if resp.Cell.Execution.Result != `{a: 1, b: [2, "three"]}` || len(resp.Content) != 0 {
		t.Fatalf("result = %q", resp.Cell.Execution.Result)
	}
}
