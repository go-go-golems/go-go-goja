package replsession

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/rs/zerolog"
)

type hardeningFlakyPersistence struct {
	failCell  int
	persisted []int
}

func (*hardeningFlakyPersistence) CreateSession(context.Context, repldb.SessionRecord) error {
	return nil
}

func (*hardeningFlakyPersistence) DeleteSession(context.Context, string, time.Time) error {
	return nil
}

func (p *hardeningFlakyPersistence) PersistEvaluation(_ context.Context, record repldb.EvaluationRecord) error {
	if record.CellID == p.failCell {
		return errors.New("injected persistence failure")
	}
	p.persisted = append(p.persisted, record.CellID)
	return nil
}

// TestHardeningPersistenceFailureBlocksLaterEvaluation is the red regression
// for GOJA-068 P0.4. JavaScript has already mutated the VM when persistence of
// cell 2 fails, so the service must reject cell 3 until exact retry or recovery.
func TestHardeningPersistenceFailureBlocksLaterEvaluation(t *testing.T) {
	ctx := context.Background()
	store := &hardeningFlakyPersistence{failCell: 2}
	service := NewService(
		newPersistenceTestFactory(t),
		zerolog.Nop(),
		WithPersistence(store),
	)

	session, err := service.CreateSession(ctx)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = service.DeleteSession(context.Background(), session.ID) }()

	if _, err := service.Evaluate(ctx, session.ID, `let x = 1; x`); err != nil {
		t.Fatalf("persist cell 1: %v", err)
	}
	if response, err := service.Evaluate(ctx, session.ID, `x = 2; x`); err == nil {
		t.Fatalf("expected cell 2 commit failure, got response %#v", response)
	}

	response3, err3 := service.Evaluate(ctx, session.ID, `x + 1`)
	if err3 == nil {
		if response3 == nil || response3.Cell == nil {
			t.Error("expected degraded session error, got nil error and no cell report")
		} else {
			t.Errorf(
				"expected degraded session to reject cell 3 before execution; got cell=%d status=%s result=%s",
				response3.Cell.ID,
				response3.Cell.Execution.Status,
				response3.Cell.Execution.Result,
			)
		}
	}
	if response3 != nil {
		if response3.Cell == nil {
			t.Error("expected no cell 3 response, got a non-nil response with nil cell")
		} else {
			t.Errorf("expected no executed cell 3 response, got cell=%d", response3.Cell.ID)
		}
	}
	if want := []int{1}; !reflect.DeepEqual(store.persisted, want) {
		t.Errorf("expected only cell 1 to be durable after failure, got %v", store.persisted)
	}
}
