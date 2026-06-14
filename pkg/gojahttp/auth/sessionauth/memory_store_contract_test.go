package sessionauth_test

import (
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/internal/sessionauthtest"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
)

func TestMemoryStoreContract(t *testing.T) {
	sessionauthtest.RunStoreContract(t, func(testing.TB) sessionauth.Store {
		return sessionauth.NewMemoryStore()
	})
}
