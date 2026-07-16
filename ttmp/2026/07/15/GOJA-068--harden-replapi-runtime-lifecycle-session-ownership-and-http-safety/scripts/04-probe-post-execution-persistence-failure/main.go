package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/rs/zerolog"
)

type flakyPersistence struct {
	mu        sync.Mutex
	failCell  int
	persisted []int
}

func (*flakyPersistence) CreateSession(context.Context, repldb.SessionRecord) error { return nil }
func (*flakyPersistence) DeleteSession(context.Context, string, time.Time) error    { return nil }
func (p *flakyPersistence) PersistEvaluation(_ context.Context, record repldb.EvaluationRecord) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if record.CellID == p.failCell {
		return errors.New("injected persistence failure")
	}
	p.persisted = append(p.persisted, record.CellID)
	return nil
}

func main() {
	ctx := context.Background()
	factory, err := engine.NewRuntimeFactoryBuilder().Build()
	must(err)
	store := &flakyPersistence{failCell: 2}
	service := replsession.NewService(factory, zerolog.Nop(), replsession.WithPersistence(store))

	session, err := service.CreateSession(ctx)
	must(err)
	_, err = service.Evaluate(ctx, session.ID, `let x = 1; x`)
	must(err)

	response2, err2 := service.Evaluate(ctx, session.ID, `x = 2; x`)
	response3, err3 := service.Evaluate(ctx, session.ID, `x + 1`)
	snapshot, snapshotErr := service.Snapshot(ctx, session.ID)

	fmt.Printf("cell2_response_nil=%t cell2_error=%v\n", response2 == nil, err2)
	if response3 != nil {
		fmt.Printf("cell3_id=%d cell3_result=%s cell3_error=%v\n", response3.Cell.ID, response3.Cell.Execution.Result, err3)
	}
	fmt.Printf("snapshot_cell_count=%d snapshot_error=%v persisted_cell_ids=%v\n", snapshot.CellCount, snapshotErr, store.persisted)
	_ = service.DeleteSession(ctx, session.ID)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
