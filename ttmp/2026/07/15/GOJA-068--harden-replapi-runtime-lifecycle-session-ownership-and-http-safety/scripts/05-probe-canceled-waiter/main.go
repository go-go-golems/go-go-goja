package main

import (
	"context"
	"fmt"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/rs/zerolog"
)

func main() {
	factory, err := engine.NewRuntimeFactoryBuilder().Build()
	must(err)
	app, err := replapi.New(context.Background(), factory, zerolog.Nop(), replapi.WithProfile(replapi.ProfileInteractive))
	must(err)
	session, err := app.CreateSession(context.Background())
	must(err)

	locked := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- app.WithRuntime(context.Background(), session.ID, func(_ context.Context, _ *engine.Runtime) error {
			close(locked)
			time.Sleep(300 * time.Millisecond)
			return nil
		})
	}()
	<-locked

	waitCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, snapshotErr := app.Snapshot(waitCtx, session.ID)
	elapsed := time.Since(started)

	fmt.Printf("snapshot_error=%v context_error=%v elapsed=%s\n", snapshotErr, waitCtx.Err(), elapsed.Round(time.Millisecond))
	must(<-done)
	_ = app.DeleteSession(context.Background(), session.ID)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
