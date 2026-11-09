package replsession

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
)

func TestInstrumentedFinalParenthesizedExpression(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewService(newPersistenceTestFactory(t), zerolog.Nop(), WithDefaultSessionOptions(InteractiveSessionOptions()))
	defer func() { _ = service.Close(ctx) }()
	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	for src, want := range map[string]string{
		`({a: 1})`:              `{"result":{"a":1}}`,
		`const x = 2; ({a: x})`: `{"result":{"a":2}}`,
		"(\n  ({b: 3})\n);":     `{"result":{"b":3}}`,
		`(1) + (2)`:             `{"result":3}`,
	} {
		resp, err := service.Evaluate(ctx, session.ID, src)
		if err != nil {
			t.Fatalf("evaluate %q: %v", src, err)
		}
		if resp.Cell.Execution.Status != "ok" || resp.Cell.Execution.ResultJSON != want {
			t.Fatalf("evaluate %q: status=%q error=%q json=%s\n%s", src, resp.Cell.Execution.Status, resp.Cell.Execution.Error, resp.Cell.Execution.ResultJSON, resp.Cell.Rewrite.TransformedSource)
		}
	}
}
