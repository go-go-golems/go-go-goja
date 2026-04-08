package replsession

import (
	"encoding/json"
	"strings"
	"time"
)

// EvalMode controls how source is executed inside a session.
type EvalMode string

const (
	EvalModeRaw          EvalMode = "raw"
	EvalModeInstrumented EvalMode = "instrumented"
)

// EvalPolicy controls code transformation and await behavior.
type EvalPolicy struct {
	Mode                  EvalMode `json:"mode"`
	CaptureLastExpression bool     `json:"captureLastExpression"`
	SupportTopLevelAwait  bool     `json:"supportTopLevelAwait"`
	TimeoutMS             int64    `json:"timeoutMs,omitempty"`
}

// ObservePolicy controls non-durable analysis and runtime introspection.
type ObservePolicy struct {
	StaticAnalysis  bool `json:"staticAnalysis"`
	RuntimeSnapshot bool `json:"runtimeSnapshot"`
	BindingTracking bool `json:"bindingTracking"`
	ConsoleCapture  bool `json:"consoleCapture"`
	JSDocExtraction bool `json:"jsdocExtraction"`
}

// PersistPolicy controls durable side effects to the session store.
type PersistPolicy struct {
	Enabled         bool `json:"enabled"`
	Sessions        bool `json:"sessions"`
	Evaluations     bool `json:"evaluations"`
	BindingVersions bool `json:"bindingVersions"`
	BindingDocs     bool `json:"bindingDocs"`
}

// SessionPolicy is the full behavior policy for one session.
type SessionPolicy struct {
	Eval    EvalPolicy    `json:"eval"`
	Observe ObservePolicy `json:"observe"`
	Persist PersistPolicy `json:"persist"`
}

// SessionOptions configures one live REPL session.
type SessionOptions struct {
	ID        string        `json:"id,omitempty"`
	CreatedAt time.Time     `json:"createdAt,omitempty"`
	Profile   string        `json:"profile,omitempty"`
	Policy    SessionPolicy `json:"policy"`
}

type sessionMetadata struct {
	Profile string        `json:"profile,omitempty"`
	Policy  SessionPolicy `json:"policy"`
}

// RawSessionOptions returns a near-straight-goja execution policy.
func RawSessionOptions() SessionOptions {
	return SessionOptions{
		Profile: "raw",
		Policy: SessionPolicy{
			Eval: EvalPolicy{
				Mode:      EvalModeRaw,
				TimeoutMS: 5000,
			},
		},
	}
}

// InteractiveSessionOptions returns REPL-friendly in-memory behavior.
func InteractiveSessionOptions() SessionOptions {
	return SessionOptions{
		Profile: "interactive",
		Policy: SessionPolicy{
			Eval: EvalPolicy{
				Mode:                  EvalModeInstrumented,
				CaptureLastExpression: true,
				SupportTopLevelAwait:  true,
				TimeoutMS:             5000,
			},
			Observe: ObservePolicy{
				StaticAnalysis:  true,
				RuntimeSnapshot: true,
				BindingTracking: true,
				ConsoleCapture:  true,
				JSDocExtraction: true,
			},
		},
	}
}

// PersistentSessionOptions returns the durable restore-aware session behavior.
func PersistentSessionOptions() SessionOptions {
	ops := InteractiveSessionOptions()
	ops.Profile = "persistent"
	ops.Policy.Persist = PersistPolicy{
		Enabled:         true,
		Sessions:        true,
		Evaluations:     true,
		BindingVersions: true,
		BindingDocs:     true,
	}
	return ops
}

// NormalizeSessionOptions applies defaults and sanitizes one options struct.
func NormalizeSessionOptions(opts SessionOptions) SessionOptions {
	normalized := opts
	normalized.Profile = strings.TrimSpace(normalized.Profile)
	if normalized.Profile == "" {
		normalized.Profile = "interactive"
	}
	normalized.Policy = NormalizeSessionPolicy(normalized.Policy)
	return normalized
}

// NormalizeSessionPolicy applies sensible defaults to partially populated policy values.
func NormalizeSessionPolicy(policy SessionPolicy) SessionPolicy {
	normalized := policy
	if normalized.Eval.Mode == "" {
		normalized.Eval.Mode = EvalModeInstrumented
	}
	if normalized.Eval.TimeoutMS < 0 {
		normalized.Eval.TimeoutMS = 0
	}
	return normalized
}

// Timeout returns the configured evaluation deadline.
func (p EvalPolicy) Timeout() time.Duration {
	if p.TimeoutMS <= 0 {
		return 0
	}
	return time.Duration(p.TimeoutMS) * time.Millisecond
}

// IsZero reports whether no explicit policy fields were set.
func (p SessionPolicy) IsZero() bool {
	return p == (SessionPolicy{})
}

// UsesInstrumentedExecution reports whether the session should use the rewrite pipeline.
func (p SessionPolicy) UsesInstrumentedExecution() bool {
	return NormalizeSessionPolicy(p).Eval.Mode == EvalModeInstrumented
}

// PersistenceEnabled reports whether durable writes are enabled at all.
func (p SessionPolicy) PersistenceEnabled() bool {
	return NormalizeSessionPolicy(p).Persist.Enabled
}

// SessionMetadataJSON serializes the profile/policy pair for durable storage.
func (o SessionOptions) SessionMetadataJSON() (json.RawMessage, error) {
	normalized := NormalizeSessionOptions(o)
	payload, err := json.Marshal(sessionMetadata{
		Profile: normalized.Profile,
		Policy:  normalized.Policy,
	})
	if err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}

// SessionOptionsFromMetadata reconstructs session options from session metadata JSON.
func SessionOptionsFromMetadata(raw json.RawMessage) (SessionOptions, bool, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "" || string(raw) == "{}" {
		return SessionOptions{}, false, nil
	}
	var metadata sessionMetadata
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return SessionOptions{}, false, err
	}
	if strings.TrimSpace(metadata.Profile) == "" && metadata.Policy.IsZero() {
		return SessionOptions{}, false, nil
	}
	return NormalizeSessionOptions(SessionOptions{
		Profile: metadata.Profile,
		Policy:  metadata.Policy,
	}), true, nil
}
