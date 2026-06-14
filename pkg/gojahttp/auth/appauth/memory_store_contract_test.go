package appauth_test

import (
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/internal/appauthtest"
)

func TestMemoryStoreContract(t *testing.T) {
	appauthtest.RunStoreContract(t, func(testing.TB) appauthtest.Harness {
		store := appauth.NewMemoryStore()
		return appauthtest.Harness{
			Users:       store,
			Memberships: store,
			Resources:   store,
			AddUser:     store.AddUser,
			AddMember:   store.AddMembership,
			AddResource: store.AddResource,
		}
	})
}
