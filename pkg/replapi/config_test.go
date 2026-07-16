package replapi

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/rs/zerolog"
)

func TestResolveCreateSessionOptionsUsesProfilePresetAndPreservesExplicitFields(t *testing.T) {
	t.Parallel()

	base, err := ConfigForProfile(ProfilePersistent)
	if err != nil {
		t.Fatalf("persistent config: %v", err)
	}
	createdAt := time.Date(2026, time.April, 8, 18, 0, 0, 0, time.UTC)
	rawProfile := ProfileRaw

	resolved, err := resolveCreateSessionOptions(base, SessionOverrides{
		ID:        "manual-id",
		CreatedAt: createdAt,
		Profile:   &rawProfile,
	})
	if err != nil {
		t.Fatalf("resolve options: %v", err)
	}

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

func TestResolveCreateSessionOptionsAppliesExplicitPolicyReplacement(t *testing.T) {
	t.Parallel()

	base, err := ConfigForProfile(ProfileInteractive)
	if err != nil {
		t.Fatalf("interactive config: %v", err)
	}
	override := replsession.RawSessionOptions().Policy

	resolved, err := resolveCreateSessionOptions(base, SessionOverrides{Policy: &override})
	if err != nil {
		t.Fatalf("resolve options: %v", err)
	}

	if !reflect.DeepEqual(resolved.Policy, override) {
		t.Fatalf("expected complete policy replacement\nwant: %#v\n got: %#v", override, resolved.Policy)
	}
	if resolved.Profile != string(ProfileInteractive) {
		t.Fatalf("policy replacement must not relabel profile, got %q", resolved.Profile)
	}
}

func TestNormalizeConfigAppliesCompleteProfilePreset(t *testing.T) {
	t.Parallel()

	for _, profile := range []Profile{ProfileRaw, ProfileInteractive, ProfilePersistent} {
		profile := profile
		t.Run(string(profile), func(t *testing.T) {
			t.Parallel()
			preset, err := ConfigForProfile(profile)
			if err != nil {
				t.Fatalf("profile preset: %v", err)
			}
			normalized, err := normalizeConfig(Config{Profile: profile})
			if err != nil {
				t.Fatalf("normalize config: %v", err)
			}
			if normalized.Profile != preset.Profile || normalized.AutoRestore != preset.AutoRestore {
				t.Fatalf("expected profile-level preset %#v, got %#v", preset, normalized)
			}
			if !reflect.DeepEqual(normalized.SessionOptions, preset.SessionOptions) {
				t.Fatalf("expected complete session preset\nwant: %#v\n got: %#v", preset.SessionOptions, normalized.SessionOptions)
			}
		})
	}
}

func TestNormalizeConfigPreservesCallerStore(t *testing.T) {
	t.Parallel()

	store := &repldb.Store{} // dummy; normalization only needs pointer identity
	normalized, err := normalizeConfig(Config{Store: store})
	if err != nil {
		t.Fatalf("normalize config: %v", err)
	}
	if normalized.Store != store {
		t.Fatal("expected caller store to be preserved after normalization")
	}
	if normalized.Profile != ProfilePersistent || !normalized.AutoRestore {
		t.Fatalf("expected empty config to use persistent defaults, got %#v", normalized)
	}
}

func TestConfigForProfileRejectsUnknownValue(t *testing.T) {
	t.Parallel()

	_, err := ConfigForProfile(Profile("typo"))
	if !errors.Is(err, ErrUnknownProfile) {
		t.Fatalf("expected ErrUnknownProfile, got %v", err)
	}
	var profileErr *UnknownProfileError
	if !errors.As(err, &profileErr) || profileErr.Value != "typo" {
		t.Fatalf("expected typed profile error for typo, got %#v", err)
	}
}

func TestNormalizeConfigRejectsProfileMismatch(t *testing.T) {
	t.Parallel()

	_, err := normalizeConfig(Config{
		Profile:        ProfileRaw,
		SessionOptions: replsession.InteractiveSessionOptions(),
	})
	if !errors.Is(err, ErrProfileMismatch) {
		t.Fatalf("expected ErrProfileMismatch, got %v", err)
	}
	var mismatch *ProfileMismatchError
	if !errors.As(err, &mismatch) || mismatch.AppProfile != ProfileRaw || mismatch.SessionProfile != ProfileInteractive {
		t.Fatalf("unexpected mismatch details: %#v", mismatch)
	}
}

func TestNormalizeConfigRejectsStructurallyInvalidPolicy(t *testing.T) {
	t.Parallel()

	t.Run("unsupported eval mode", func(t *testing.T) {
		config := InteractiveConfig()
		config.SessionOptions.Policy.Eval.Mode = replsession.EvalMode("mystery")
		_, err := normalizeConfig(config)
		if !errors.Is(err, ErrInvalidSessionPolicy) {
			t.Fatalf("expected ErrInvalidSessionPolicy, got %v", err)
		}
	})

	t.Run("persistence details while disabled", func(t *testing.T) {
		config := InteractiveConfig()
		config.SessionOptions.Policy.Persist.Evaluations = true
		_, err := normalizeConfig(config)
		if !errors.Is(err, ErrInvalidSessionPolicy) {
			t.Fatalf("expected ErrInvalidSessionPolicy, got %v", err)
		}
	})
}

func TestWithDefaultSessionPolicyFullyReplacesPreset(t *testing.T) {
	t.Parallel()

	app, err := New(context.Background(),
		newTestFactory(t),
		zerolog.Nop(),
		WithProfile(ProfileInteractive),
		WithDefaultSessionPolicy(replsession.SessionPolicy{}),
	)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	session, err := app.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = app.DeleteSession(context.Background(), session.ID) }()

	if session.Policy.Eval.Mode != replsession.EvalModeInstrumented {
		t.Fatalf("zero replacement policy should normalize eval mode, got %q", session.Policy.Eval.Mode)
	}
	if session.Policy.Observe != (replsession.ObservePolicy{}) {
		t.Fatalf("expected zero observation flags after replacement, got %#v", session.Policy.Observe)
	}
	if session.Policy.Persist != (replsession.PersistPolicy{}) {
		t.Fatalf("expected zero persistence flags after replacement, got %#v", session.Policy.Persist)
	}
}

// Promoted from the P0 red suite by Phase 1.
func TestHardeningPartialRawConfigUsesRawPreset(t *testing.T) {
	app, err := NewWithConfig(context.Background(), newTestFactory(t), zerolog.Nop(), Config{Profile: ProfileRaw})
	if err != nil {
		t.Fatalf("new raw app: %v", err)
	}
	session, err := app.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("create raw session: %v", err)
	}
	defer func() { _ = app.DeleteSession(context.Background(), session.ID) }()

	if session.Profile != string(ProfileRaw) {
		t.Fatalf("expected raw profile, got %q", session.Profile)
	}
	if session.Policy.Eval.Mode != replsession.EvalModeRaw {
		t.Fatalf("expected raw eval mode, got %q", session.Policy.Eval.Mode)
	}
	if session.Policy.Eval.TimeoutMS != replsession.RawSessionOptions().Policy.Eval.TimeoutMS {
		t.Fatalf("expected raw timeout %dms, got %dms", replsession.RawSessionOptions().Policy.Eval.TimeoutMS, session.Policy.Eval.TimeoutMS)
	}
}

// Promoted from the P0 red suite by Phase 1.
func TestHardeningUnknownAppProfileIsRejected(t *testing.T) {
	store := openTestStore(t)
	defer func() { _ = store.Close() }()

	_, err := New(context.Background(),
		newTestFactory(t),
		zerolog.Nop(),
		WithProfile(Profile("typo")),
		WithStore(store),
	)
	if !errors.Is(err, ErrUnknownProfile) {
		t.Fatalf("expected ErrUnknownProfile, got %v", err)
	}
}

// Promoted from the P0 red suite by Phase 1.
func TestHardeningUnknownSessionProfileIsRejected(t *testing.T) {
	store := openTestStore(t)
	defer func() { _ = store.Close() }()
	app, err := New(context.Background(),
		newTestFactory(t),
		zerolog.Nop(),
		WithProfile(ProfileInteractive),
		WithStore(store),
	)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	unknown := Profile("typo")
	_, err = app.CreateSessionWithOptions(context.Background(), SessionOverrides{Profile: &unknown})
	if !errors.Is(err, ErrUnknownProfile) {
		t.Fatalf("expected ErrUnknownProfile, got %v", err)
	}
}
