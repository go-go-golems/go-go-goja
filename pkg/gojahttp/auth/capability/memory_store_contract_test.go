package capability_test

import (
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/capability"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/internal/capabilitytest"
)

func TestMemoryStoreContract(t *testing.T) {
	capabilitytest.RunStoreContract(t, func(testing.TB) capability.Store {
		return capability.NewMemoryStore()
	})
}
