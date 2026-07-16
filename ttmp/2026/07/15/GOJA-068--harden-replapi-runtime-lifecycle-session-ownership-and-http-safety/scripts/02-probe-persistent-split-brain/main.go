package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/rs/zerolog"
)

func main() {
	ctx := context.Background()
	tmp, err := os.MkdirTemp("", "goja-068-split-brain-")
	must(err)
	defer func() { _ = os.RemoveAll(tmp) }()

	store, err := repldb.Open(ctx, filepath.Join(tmp, "repl.sqlite"))
	must(err)
	defer func() { _ = store.Close() }()

	factory, err := engine.NewRuntimeFactoryBuilder().Build()
	must(err)
	newApp := func() *replapi.App {
		app, err := replapi.New(context.Background(), factory, zerolog.Nop(), replapi.WithProfile(replapi.ProfilePersistent), replapi.WithStore(store))
		must(err)
		return app
	}

	appA := newApp()
	defer func() { _ = appA.Close(context.Background()) }()
	session, err := appA.CreateSession(ctx)
	must(err)
	_, err = appA.Evaluate(ctx, session.ID, `const owner = "seed"; owner`)
	must(err)

	appB := newApp()
	defer func() { _ = appB.Close(context.Background()) }()
	_, err = appB.Snapshot(ctx, session.ID) // Restore a second live VM at cell 1.
	must(err)

	responseA, errA := appA.Evaluate(ctx, session.ID, `owner = "A"; owner`)
	responseB, errB := appB.Evaluate(ctx, session.ID, `owner = "B"; owner`)

	fmt.Printf("appA: cell=%d status=%s error=%v\n", responseA.Cell.ID, responseA.Cell.Execution.Status, errA)
	if responseB != nil && responseB.Cell != nil {
		fmt.Printf("appB: cell=%d status=%s error=%v\n", responseB.Cell.ID, responseB.Cell.Execution.Status, errB)
	} else {
		fmt.Printf("appB: response=nil error=%v\n", errB)
	}

	history, err := appA.History(ctx, session.ID)
	must(err)
	fmt.Printf("durable cell ids:")
	for _, record := range history {
		fmt.Printf(" %d", record.CellID)
	}
	fmt.Println()

	_ = appA.DeleteSession(ctx, session.ID)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
