package replapi

import (
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
)

func TestResolveCreateSessionOptionsUsesProfilePresetAndPreservesExplicitFields(t *testing.T) {
	t.Parallel()

	base := ConfigForProfile(ProfilePersistent)
	createdAt := time.Date(2026, time.April, 8, 18, 0, 0, 0, time.UTC)
	rawProfile := ProfileRaw

	resolved := resolveCreateSessionOptions(base, SessionOverrides{
		ID:        "manual-id",
		CreatedAt: createdAt,
		Profile:   &rawProfile,
	})

	if resolved.ID != "manual-id" {
		t.Fatalf("expected explicit id to be preserved, got %q", resolved.ID)
	}
	if !resolved.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected explicit createdAt to be preserved, got %s", resolved.CreatedAt)
	}
	if resolved.Profile != string(ProfileRaw) {
		t.Fatalf("expected raw profile preset, got %q", resolved.Profile)
	}
	if resolved.Policy.Eval.Mode != replsession.EvalModeRaw {
		t.Fatalf("expected raw eval mode, got %q", resolved.Policy.Eval.Mode)
	}
	if resolved.Policy.PersistenceEnabled() {
		t.Fatal("expected raw override to disable persistence")
	}
}

func TestResolveCreateSessionOptionsAppliesExplicitPolicyOverride(t *testing.T) {
	t.Parallel()

	base := ConfigForProfile(ProfileInteractive)
	override := replsession.RawSessionOptions().Policy

	resolved := resolveCreateSessionOptions(base, SessionOverrides{
		Policy: &override,
	})

	if resolved.Policy.Eval.Mode != replsession.EvalModeRaw {
		t.Fatalf("expected policy override to win, got %q", resolved.Policy.Eval.Mode)
	}
}

func TestNormalizeConfigHonorsExplicitProfile(t *testing.T) {
	t.Parallel()

	// When Config.Profile is set to raw but SessionOptions is zero-valued,
	// normalizeConfig must propagate the profile before normalizing options.
	// Note: policy resolution happens later in resolveSessionOptions, not here.
	normalized := normalizeConfig(Config{Profile: ProfileRaw})
	if normalized.SessionOptions.Profile != string(ProfileRaw) {
		t.Fatalf("expected raw profile, got %q", normalized.SessionOptions.Profile)
	}
}

func TestNormalizeConfigPreservesCallerStore(t *testing.T) {
	t.Parallel()

	// When only Store is set and Profile is empty, normalizeConfig should
	// fill defaults from DefaultConfig() but keep the caller's Store.
	store := &repldb.Store{} // dummy, just need a non-nil pointer
	normalized := normalizeConfig(Config{Store: store})
	if normalized.Store != store {
		t.Fatal("expected caller store to be preserved after normalization")
	}
}
