package audit_test

import (
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/internal/audittest"
)

func TestMemoryStoreContract(t *testing.T) {
	audittest.RunStoreContract(t, func(testing.TB) audittest.Harness {
		store := &audit.MemoryStore{}
		return audittest.Harness{Store: store, Snapshot: store.Snapshot}
	})
}
