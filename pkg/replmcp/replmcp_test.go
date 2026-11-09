package replmcp

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/go-go-golems/go-go-goja/pkg/resultcontent"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
)

var pngBytes = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 1, 2, 3}

type testLook struct{ label string }

func (l *testLook) ResultContent(context.Context) ([]resultcontent.Content, error) {
	return []resultcontent.Content{{MIMEType: "image/png", Data: pngBytes, Caption: l.label}}, nil
}

func (l *testLook) ResultPlaceholder() string { return "look " + l.label }

type lookInit struct{}

func (lookInit) ID() string { return "replmcp-test-look" }

func (lookInit) InitRuntime(ctx *engine.RuntimeInitializationContext) error {
	return ctx.VM.Set("look", func(label string) *testLook { return &testLook{label: label} })
}

func newTestApp(t *testing.T) *replapi.App {
	t.Helper()
	factory, err := engine.NewRuntimeFactoryBuilder().
		UseModuleMiddleware(engine.MiddlewareSafe()).
		WithRuntimeInitializers(lookInit{}).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	app, err := replapi.New(context.Background(), factory, zerolog.Nop(),
		replapi.WithProfile(replapi.ProfileInteractive),
		replapi.WithResultProcessor(replsession.DefaultResultProcessor{FullText: true}),
	)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	t.Cleanup(func() { _ = app.Close(context.Background()) })
	return app
}

func connect(t *testing.T, ev Evaluator, opts ...Option) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	server, err := NewServer(ev, opts...)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "replmcp-test", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() {
		_ = clientSession.Close()
		_ = serverSession.Wait()
	})
	return clientSession
}

func call(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: protocol error %v", name, err)
	}
	return result
}

func firstText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("empty result content")
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("first content is %T", result.Content[0])
	}
	return text.Text
}

func structured[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()
	var out T
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode structured content %s: %v", raw, err)
	}
	return out
}

func TestEvalReturnsTextInDefaultSession(t *testing.T) {
	cs := connect(t, newTestApp(t))
	if r := call(t, cs, "eval", map[string]any{"code": "const x = 20; console.log('hi', x); x"}); r.IsError || !strings.HasPrefix(firstText(t, r), "20") || !strings.Contains(firstText(t, r), `[log] "hi" 20`) {
		t.Fatalf("first eval: isError=%v text=%q", r.IsError, firstText(t, r))
	}
	r := call(t, cs, "eval", map[string]any{"code": "x * 2 + 2"})
	if r.IsError || firstText(t, r) != "42" {
		t.Fatalf("second eval: isError=%v text=%q", r.IsError, firstText(t, r))
	}
	r = call(t, cs, "eval", map[string]any{"code": "({a: [1, {b: 'two'}], s: 'x'.repeat(150)})"})
	if want := `{a: [1, {b: "two"}], s: "` + strings.Repeat("x", 150) + `"}`; firstText(t, r) != want {
		t.Fatalf("full text eval: %q", firstText(t, r))
	}
}

func TestEvalReturnsImageContent(t *testing.T) {
	cs := connect(t, newTestApp(t))
	r := call(t, cs, "eval", map[string]any{"code": `({mean: 0.5, after: look("after")})`})
	if r.IsError {
		t.Fatalf("eval failed: %q", firstText(t, r))
	}
	text := firstText(t, r)
	if !strings.Contains(text, "after: [content 1: look after]") || !strings.Contains(text, "content 1 after: attached (image/png, 11 bytes)") {
		t.Fatalf("text = %q", text)
	}
	if len(r.Content) != 2 {
		t.Fatalf("content = %#v", r.Content)
	}
	image, ok := r.Content[1].(*mcp.ImageContent)
	if !ok || image.MIMEType != "image/png" || string(image.Data) != string(pngBytes) {
		t.Fatalf("image content = %#v", r.Content[1])
	}
}

func TestEvalImageLimitsAndTextTruncation(t *testing.T) {
	cs := connect(t, newTestApp(t), WithMaxImages(1), WithMaxResultChars(200))
	r := call(t, cs, "eval", map[string]any{"code": `[look("a"), look("b")]`})
	if len(r.Content) != 2 || !strings.Contains(firstText(t, r), "content 2 b: dropped (more than 1 items)") {
		t.Fatalf("image limit: content=%d text=%q", len(r.Content), firstText(t, r))
	}
	r = call(t, cs, "eval", map[string]any{"code": `"y".repeat(1000)`})
	if text := firstText(t, r); len(text) > 260 || !strings.Contains(text, "[result truncated:") {
		t.Fatalf("truncation: len=%d text=%q", len(text), text)
	}
	r = call(t, newTestClientWithByteLimit(t), "eval", map[string]any{"code": `look("big")`})
	if len(r.Content) != 1 || !strings.Contains(firstText(t, r), "byte limit") {
		t.Fatalf("byte limit: content=%d text=%q", len(r.Content), firstText(t, r))
	}
}

func newTestClientWithByteLimit(t *testing.T) *mcp.ClientSession {
	return connect(t, newTestApp(t), WithMaxImageBytes(4))
}

func TestEvalErrorsAreToolErrors(t *testing.T) {
	cs := connect(t, newTestApp(t))
	r := call(t, cs, "eval", map[string]any{"code": `throw new Error("boom")`})
	if !r.IsError || !strings.Contains(firstText(t, r), "boom") || !strings.Contains(firstText(t, r), "runtime-error") {
		t.Fatalf("throw: isError=%v text=%q", r.IsError, firstText(t, r))
	}
	r = call(t, cs, "eval", map[string]any{"code": "1", "sessionId": "does-not-exist"})
	if !r.IsError || !strings.Contains(firstText(t, r), "does-not-exist") {
		t.Fatalf("unknown session: isError=%v text=%q", r.IsError, firstText(t, r))
	}
	r = call(t, cs, "eval", map[string]any{"code": "  "})
	if !r.IsError {
		t.Fatal("empty code should be a tool error")
	}
	r = call(t, cs, "history", map[string]any{"sessionId": "nope"})
	if !r.IsError {
		t.Fatal("history of unknown session should be a tool error")
	}
}

func TestSessionTools(t *testing.T) {
	cs := connect(t, newTestApp(t))
	created := structured[CreateSessionOutput](t, call(t, cs, "create_session", nil))
	if created.SessionID == "" {
		t.Fatal("empty session id")
	}
	if r := call(t, cs, "eval", map[string]any{"code": "const answer = 42; answer", "sessionId": created.SessionID}); r.IsError {
		t.Fatalf("eval: %q", firstText(t, r))
	}
	call(t, cs, "eval", map[string]any{"code": "1"}) // creates the default session

	sessions := structured[ListSessionsOutput](t, call(t, cs, "list_sessions", nil))
	if len(sessions.Sessions) != 2 {
		t.Fatalf("sessions = %#v", sessions)
	}
	defaults := 0
	for _, s := range sessions.Sessions {
		if s.Default {
			defaults++
			if s.SessionID == created.SessionID {
				t.Fatal("explicit session marked default")
			}
		}
	}
	if defaults != 1 {
		t.Fatalf("defaults = %d", defaults)
	}

	history := structured[HistoryOutput](t, call(t, cs, "history", map[string]any{"sessionId": created.SessionID}))
	if len(history.Cells) != 1 || !strings.Contains(history.Cells[0].Source, "answer = 42") || history.Cells[0].Result != "42" || history.Cells[0].Status != "ok" {
		t.Fatalf("history = %#v", history)
	}

	bindings := structured[BindingsOutput](t, call(t, cs, "bindings", map[string]any{"sessionId": created.SessionID}))
	found := false
	for _, b := range bindings.Bindings {
		if b.Name == "answer" && b.Preview == "42" {
			found = true
		}
	}
	if !found {
		t.Fatalf("bindings = %#v", bindings)
	}
}

func TestToolListAndAnnotations(t *testing.T) {
	extraCalled := false
	cs := connect(t, newTestApp(t), WithName("cv"), WithExtraTools(func(s *mcp.Server) {
		extraCalled = true
		mcp.AddTool(s, &mcp.Tool{Name: "ping"}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "pong"}}}, nil, nil
		})
	}))
	if !extraCalled {
		t.Fatal("extra tools hook not called")
	}
	tools, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	byName := map[string]*mcp.Tool{}
	for _, tool := range tools.Tools {
		byName[tool.Name] = tool
	}
	for _, name := range []string{"eval", "create_session", "list_sessions", "history", "bindings", "ping"} {
		if byName[name] == nil {
			t.Fatalf("missing tool %s in %v", name, byName)
		}
	}
	for _, name := range []string{"list_sessions", "history", "bindings"} {
		if byName[name].Annotations == nil || !byName[name].Annotations.ReadOnlyHint {
			t.Fatalf("%s is not read-only", name)
		}
	}
	if byName["eval"].Annotations != nil && byName["eval"].Annotations.ReadOnlyHint {
		t.Fatal("eval marked read-only")
	}
	if init := cs.InitializeResult(); init == nil || init.ServerInfo.Name != "cv" || !strings.Contains(init.Instructions, "eval") {
		t.Fatalf("initialize result = %#v", init)
	}
}

func TestEvalResultDoesNotDuplicateProcessedError(t *testing.T) {
	failed := func(result string) *replsession.EvaluateResponse {
		return &replsession.EvaluateResponse{Cell: &replsession.CellReport{Execution: replsession.ExecutionReport{
			Status: "runtime-error",
			Result: result,
			Error:  "promise rejected: E_RANGE: black must be < white",
		}}}
	}
	r := EvalResult(failed("error:\n  code: E_RANGE\n  message: black must be < white"), Limits{})
	text := r.Content[0].(*mcp.TextContent).Text
	if !r.IsError || strings.Contains(text, "promise rejected") || !strings.HasPrefix(text, "error:\n  code: E_RANGE") {
		t.Fatalf("processed error: isError=%v text=%q", r.IsError, text)
	}
	r = EvalResult(failed(""), Limits{})
	text = r.Content[0].(*mcp.TextContent).Text
	if !r.IsError || text != "error (runtime-error): promise rejected: E_RANGE: black must be < white" {
		t.Fatalf("unprocessed error: isError=%v text=%q", r.IsError, text)
	}
}

func TestUnknownSessionIsAHelpfulToolError(t *testing.T) {
	cs := connect(t, newTestApp(t))
	// Before any default session exists.
	r := call(t, cs, "eval", map[string]any{"code": "1", "sessionId": "probe"})
	text := firstText(t, r)
	if !r.IsError || !strings.Contains(text, `session "probe" does not exist`) || !strings.Contains(text, "No sessions exist yet") ||
		!strings.Contains(text, "omit sessionId") || !strings.Contains(text, "create_session") || strings.Contains(text, "replsession:") {
		t.Fatalf("no sessions yet: isError=%v text=%q", r.IsError, text)
	}

	call(t, cs, "eval", map[string]any{"code": "1"}) // creates the default session
	sessions := structured[ListSessionsOutput](t, call(t, cs, "list_sessions", nil))
	if len(sessions.Sessions) != 1 {
		t.Fatalf("an unknown id was remembered as a session: %#v", sessions)
	}
	defaultID := sessions.Sessions[0].SessionID
	for _, tool := range []string{"eval", "history", "bindings"} {
		args := map[string]any{"sessionId": "probe"}
		if tool == "eval" {
			args["code"] = "1"
		}
		r := call(t, cs, tool, args)
		text := firstText(t, r)
		if !r.IsError || !strings.Contains(text, `session "probe" does not exist`) || !strings.Contains(text, "only the default session ("+defaultID+")") ||
			!strings.Contains(text, "omit sessionId") || strings.Contains(text, "replsession:") {
			t.Fatalf("%s: isError=%v text=%q", tool, r.IsError, text)
		}
	}

	created := structured[CreateSessionOutput](t, call(t, cs, "create_session", nil))
	r = call(t, cs, "eval", map[string]any{"code": "1", "sessionId": "probe"})
	text = firstText(t, r)
	if !strings.Contains(text, "Existing sessions: "+defaultID+" (default), "+created.SessionID) {
		t.Fatalf("several sessions: %q", text)
	}
}

func TestSessionIDSchemaDescription(t *testing.T) {
	cs := connect(t, newTestApp(t))
	tools, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		if tool.Name != "eval" && tool.Name != "history" && tool.Name != "bindings" {
			continue
		}
		raw, _ := json.Marshal(tool.InputSchema)
		if !strings.Contains(string(raw), "omit it to use the default session") || !strings.Contains(string(raw), "only pass ids returned by create_session or list_sessions") {
			t.Fatalf("%s sessionId description: %s", tool.Name, raw)
		}
	}
}

func TestUnknownSessionWithPersistentStore(t *testing.T) {
	ctx := context.Background()
	store, err := repldb.Open(ctx, filepath.Join(t.TempDir(), "repl.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	factory, err := engine.NewRuntimeFactoryBuilder().UseModuleMiddleware(engine.MiddlewareSafe()).Build()
	if err != nil {
		t.Fatal(err)
	}
	app, err := replapi.New(ctx, factory, zerolog.Nop(), replapi.WithProfile(replapi.ProfilePersistent), replapi.WithStore(store))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = app.Close(ctx)
		_ = store.Close()
	})
	cs := connect(t, app)
	for _, tool := range []string{"eval", "history", "bindings"} {
		args := map[string]any{"sessionId": "probe"}
		if tool == "eval" {
			args["code"] = "1"
		}
		r := call(t, cs, tool, args)
		if text := firstText(t, r); !r.IsError || !strings.Contains(text, `session "probe" does not exist`) || strings.Contains(text, "repldb:") {
			t.Fatalf("%s: isError=%v text=%q", tool, r.IsError, text)
		}
	}
}
