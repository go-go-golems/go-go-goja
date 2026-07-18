package hostauth

import (
	"context"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/sessionauth"
)

func TestEnabledUserActorLoaderRejectsDisabledUser(t *testing.T) {
	users := appauth.NewMemoryStore()
	users.AddUser(appauth.User{ID: "user:alice", Email: "alice@example.test"})
	loader := enabledUserActorLoader{users: users}
	session := &sessionauth.Session{UserID: "user:alice", CreatedAt: time.Now()}
	if _, err := loader.ActorForSession(context.Background(), session); err != nil {
		t.Fatalf("enabled user: %v", err)
	}
	if err := users.DisableUser(context.Background(), "user:alice", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := loader.ActorForSession(context.Background(), session); err != gojahttp.ErrUnauthenticated {
		t.Fatalf("disabled user error = %v, want %v", err, gojahttp.ErrUnauthenticated)
	}
}
