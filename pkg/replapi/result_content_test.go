package replapi

import (
	"context"
	"testing"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/go-go-golems/go-go-goja/pkg/resultcontent"
	"github.com/rs/zerolog"
)

type apiLook struct{}

func (apiLook) ResultContent(context.Context) ([]resultcontent.Content, error) {
	return []resultcontent.Content{{MIMEType: "image/png", Data: []byte{1, 2, 3}}}, nil
}

type apiLookInitializer struct{}

func (apiLookInitializer) ID() string { return "api-look" }

func (apiLookInitializer) InitRuntime(ctx *engine.RuntimeInitializationContext) error {
	return ctx.VM.Set("look", func() apiLook { return apiLook{} })
}

type prefixProcessor struct{}

func (prefixProcessor) Process(ctx context.Context, vm *goja.Runtime, value goja.Value) (replsession.ProcessedResult, error) {
	result, err := replsession.DefaultResultProcessor{}.Process(ctx, vm, value)
	result.Text = "processed:" + result.Text
	return result, err
}

func TestAppPassesResultProcessorAndContent(t *testing.T) {
	ctx := context.Background()
	factory, err := engine.NewRuntimeFactoryBuilder().
		UseModuleMiddleware(engine.MiddlewareSafe()).
		WithRuntimeInitializers(apiLookInitializer{}).
		Build()
	if err != nil {
		t.Fatalf("build factory: %v", err)
	}
	app, err := New(ctx, factory, zerolog.Nop(),
		WithResultProcessor(prefixProcessor{}),
		WithResultContentLimits(resultcontent.Limits{MaxItems: 1}),
		WithProfile(ProfileInteractive),
	)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	defer func() { _ = app.Close(ctx) }()
	session, err := app.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	resp, err := app.Evaluate(ctx, session.ID, `[look(), look()]`)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if resp.Cell.Execution.Result != "processed:[[content 1], [content 2]]" {
		t.Fatalf("result = %q", resp.Cell.Execution.Result)
	}
	if len(resp.Content) != 1 || len(resp.Cell.Execution.Content) != 2 || resp.Cell.Execution.Content[1].Index != -1 {
		t.Fatalf("content=%#v refs=%#v", resp.Content, resp.Cell.Execution.Content)
	}
}
