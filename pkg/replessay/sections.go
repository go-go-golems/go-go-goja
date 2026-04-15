package replessay

import (
	"net/http"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/pkg/errors"
)

const (
	ProfilesBootstrapPath      = "/api/essay/sections/profiles-change-behavior"
	ProfilesCreatePath         = "/api/essay/sections/profiles-change-behavior/session"
	profilesSnapshotPrefix     = "/api/essay/sections/profiles-change-behavior/session/"
	CodeFlowBootstrapPath      = "/api/essay/sections/what-happened-to-my-code"
	CodeFlowCreatePath         = "/api/essay/sections/what-happened-to-my-code/session"
	codeFlowSnapshotPrefix     = "/api/essay/sections/what-happened-to-my-code/session/"
	codeFlowEvaluatePathSuffix = "/evaluate"
)

type ProfileSectionResponse struct {
	Section         SectionSpec   `json:"section"`
	SelectedProfile string        `json:"selectedProfile"`
	Profiles        []ProfileSpec `json:"profiles"`
	RawRoutes       []RouteRef    `json:"rawRoutes"`
}

type ProfileSpec struct {
	ID         string                    `json:"id"`
	Title      string                    `json:"title"`
	Summary    string                    `json:"summary"`
	Policy     replsession.SessionPolicy `json:"policy"`
	Highlights []string                  `json:"highlights"`
}

type CodeFlowSectionResponse struct {
	Section        SectionSpec        `json:"section"`
	DefaultProfile string             `json:"defaultProfile"`
	StarterSource  string             `json:"starterSource"`
	Examples       []ExampleSourceRef `json:"examples"`
	RawRoutes      []RouteRef         `json:"rawRoutes"`
}

type ExampleSourceRef struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Source    string `json:"source"`
	Rationale string `json:"rationale"`
}

type createSessionRequest struct {
	Profile string `json:"profile"`
}

func buildProfileSectionResponse() ProfileSectionResponse {
	return ProfileSectionResponse{
		Section: SectionSpec{
			ID:      "profiles-change-behavior",
			Title:   "Profiles Change Behavior",
			Summary: "Raw, interactive, and persistent sessions are different execution contracts, not cosmetic labels.",
			Intro: []string{
				"A profile is a named bundle of session policy. When you change the profile, you are changing what the REPL is allowed to do with your code before, during, and after execution.",
				"This section compares the three built-in profiles, then lets the browser create a real session using the selected profile so you can confirm that the backend summary matches the documented contract.",
			},
			PrimaryAction: ActionSpec{
				Label:  "Create Selected Profile",
				Method: http.MethodPost,
				Path:   ProfilesCreatePath,
			},
			Panels: []PanelSpec{
				{
					ID:          "profile-comparison",
					Title:       "Profile Comparison",
					Kind:        "comparison-table",
					Description: "Side-by-side contract view for raw, interactive, and persistent profiles.",
				},
				{
					ID:          "live-profile-session",
					Title:       "Live Session",
					Kind:        "summary-card",
					Description: "Real backend session created using the currently selected profile.",
				},
			},
		},
		SelectedProfile: string(replapi.ProfileInteractive),
		Profiles:        buildProfileSpecs(),
		RawRoutes: []RouteRef{
			{Method: http.MethodGet, Path: ProfilesBootstrapPath, Purpose: "Article-scoped route describing the known profile presets and the comparison section."},
			{Method: http.MethodPost, Path: ProfilesCreatePath, Purpose: "Article-scoped route that creates one session using the requested profile override."},
			{Method: http.MethodGet, Path: profilesSnapshotPrefix + "{sessionID}", Purpose: "Article-scoped route that fetches one live profile-demo session."},
			{Method: http.MethodPost, Path: "/api/sessions", Purpose: "Underlying raw REPL create-session route. It still defaults to the app profile when called directly."},
		},
	}
}

func buildCodeFlowSectionResponse() CodeFlowSectionResponse {
	return CodeFlowSectionResponse{
		Section: SectionSpec{
			ID:      "what-happened-to-my-code",
			Title:   "What Happened To My Code?",
			Summary: "Instrumented sessions do not just execute source. They analyze it, rewrite it, execute it, and then report what changed.",
			Intro: []string{
				"This section focuses on the evaluation pipeline. The important question is not only what the code returned, but what the system learned and what transformations it applied along the way.",
				"The browser prepares one real evaluation session, submits source to the live API, and then renders the backend's rewrite, execution, and runtime reports in synchronized views.",
			},
			PrimaryAction: ActionSpec{
				Label:  "Evaluate Source",
				Method: http.MethodPost,
				Path:   codeFlowSnapshotPrefix + "{sessionID}" + codeFlowEvaluatePathSuffix,
			},
			Panels: []PanelSpec{
				{
					ID:          "source-transform",
					Title:       "Source Before and After",
					Kind:        "code-diff",
					Description: "Original user source compared with the transformed source the runtime actually executes.",
				},
				{
					ID:          "rewrite-operations",
					Title:       "Rewrite Operations",
					Kind:        "operation-list",
					Description: "Explicit step list describing which helpers and transformations were applied.",
				},
				{
					ID:          "execution-result",
					Title:       "Execution Result",
					Kind:        "result-summary",
					Description: "Runtime status, result preview, duration, await behavior, and console output for one evaluation.",
				},
			},
		},
		DefaultProfile: string(replapi.ProfileInteractive),
		StarterSource:  "const x = 1; x",
		Examples: []ExampleSourceRef{
			{
				ID:        "capture-last-expression",
				Label:     "Capture last expression",
				Source:    "const x = 1; x",
				Rationale: "Smallest useful example of declaration rewriting plus last-expression capture.",
			},
			{
				ID:        "top-level-await",
				Label:     "Top-level await",
				Source:    "await Promise.resolve(41 + 1)",
				Rationale: "Shows how instrumented sessions can support awaited values directly.",
			},
			{
				ID:        "global-side-effect",
				Label:     "Global side effect",
				Source:    "globalThis.answer = 42; answer",
				Rationale: "Makes runtime diffs and session-bound state changes visible.",
			},
		},
		RawRoutes: []RouteRef{
			{Method: http.MethodGet, Path: CodeFlowBootstrapPath, Purpose: "Article-scoped route describing the evaluation walkthrough section and starter examples."},
			{Method: http.MethodPost, Path: CodeFlowCreatePath, Purpose: "Article-scoped route that prepares one evaluation demo session, defaulting to the interactive profile."},
			{Method: http.MethodPost, Path: codeFlowSnapshotPrefix + "{sessionID}" + codeFlowEvaluatePathSuffix, Purpose: "Article-scoped route that runs one live evaluation for the selected demo session."},
			{Method: http.MethodPost, Path: "/api/sessions/{sessionID}/evaluate", Purpose: "Underlying raw REPL evaluation route used by the article wrapper."},
		},
	}
}

func buildProfileSpecs() []ProfileSpec {
	raw := replsession.RawSessionOptions()
	interactive := replsession.InteractiveSessionOptions()
	persistent := replsession.PersistentSessionOptions()
	return []ProfileSpec{
		{
			ID:      raw.Profile,
			Title:   "Raw",
			Summary: "The thinnest possible layer over goja. Minimal rewriting, minimal observation, and no durable history.",
			Policy:  raw.Policy,
			Highlights: []string{
				"Runs in raw mode without instrumented helper rewriting.",
				"Does not capture the last expression automatically.",
				"Turns off static analysis, runtime snapshots, and persistence.",
			},
		},
		{
			ID:      interactive.Profile,
			Title:   "Interactive",
			Summary: "Optimized for conversational exploration. It rewrites code to preserve useful REPL behavior and exposes rich observation data in memory.",
			Policy:  interactive.Policy,
			Highlights: []string{
				"Uses instrumented execution with helper insertion and last-expression capture.",
				"Enables static analysis, runtime snapshots, binding tracking, and console capture.",
				"Keeps state in memory but does not persist the session to SQLite.",
			},
		},
		{
			ID:      persistent.Profile,
			Title:   "Persistent",
			Summary: "Extends the interactive profile with durable storage so sessions, evaluations, bindings, and docs can survive process restarts.",
			Policy:  persistent.Policy,
			Highlights: []string{
				"Inherits the same interactive instrumentation and observation features.",
				"Adds durable storage for sessions, evaluations, binding versions, and binding docs.",
				"Is the profile that makes restore/history/export-style workflows possible.",
			},
		},
	}
}

func parseProfile(name string) (*replapi.Profile, error) {
	normalized := strings.TrimSpace(strings.ToLower(name))
	if normalized == "" {
		return nil, nil
	}
	profile := replapi.Profile(normalized)
	switch profile {
	case replapi.ProfileRaw, replapi.ProfileInteractive, replapi.ProfilePersistent:
		return &profile, nil
	default:
		return nil, errors.Errorf("unsupported profile %q", name)
	}
}
