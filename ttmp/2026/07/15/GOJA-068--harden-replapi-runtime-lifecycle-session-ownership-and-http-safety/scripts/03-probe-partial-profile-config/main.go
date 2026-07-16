package main

import (
	"context"
	"fmt"

	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/rs/zerolog"
)

func main() {
	ctx := context.Background()
	factory, err := engine.NewRuntimeFactoryBuilder().Build()
	must(err)

	// This looks equivalent to replapi.RawConfig(), but currently is not.
	app, err := replapi.NewWithConfig(ctx, factory, zerolog.Nop(), replapi.Config{Profile: replapi.ProfileRaw})
	must(err)
	session, err := app.CreateSession(ctx)
	must(err)

	fmt.Printf("profile=%s eval_mode=%s timeout_ms=%d static_analysis=%t binding_tracking=%t\n",
		session.Profile,
		session.Policy.Eval.Mode,
		session.Policy.Eval.TimeoutMS,
		session.Policy.Observe.StaticAnalysis,
		session.Policy.Observe.BindingTracking,
	)
	_ = app.DeleteSession(ctx, session.ID)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
