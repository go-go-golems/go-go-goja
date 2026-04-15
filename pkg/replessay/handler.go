package replessay

import (
	"encoding/json"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replhttp"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/pkg/errors"
)

const (
	MeetSessionPagePath       = "/essay/meet-a-session"
	MeetSessionBootstrapPath  = "/api/essay/sections/meet-a-session"
	MeetSessionCreatePath     = "/api/essay/sections/meet-a-session/session"
	meetSessionSnapshotPrefix = "/api/essay/sections/meet-a-session/session/"
	meetSessionStaticPrefix   = "/static/essay/"
)

type BootstrapResponse struct {
	Section     SectionSpec     `json:"section"`
	DefaultView DefaultViewSpec `json:"defaultView"`
	RawRoutes   []RouteRef      `json:"rawRoutes"`
}

type SectionSpec struct {
	ID            string      `json:"id"`
	Title         string      `json:"title"`
	Summary       string      `json:"summary"`
	Intro         []string    `json:"intro"`
	PrimaryAction ActionSpec  `json:"primaryAction"`
	Panels        []PanelSpec `json:"panels"`
}

type ActionSpec struct {
	Label  string `json:"label"`
	Method string `json:"method"`
	Path   string `json:"path"`
}

type PanelSpec struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Kind        string      `json:"kind"`
	Description string      `json:"description"`
	Fields      []FieldSpec `json:"fields,omitempty"`
}

type FieldSpec struct {
	Label       string `json:"label"`
	JSONPath    string `json:"jsonPath"`
	Description string `json:"description"`
}

type DefaultViewSpec struct {
	Profile string                    `json:"profile"`
	Policy  replsession.SessionPolicy `json:"policy"`
}

type RouteRef struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Purpose string `json:"purpose"`
}

// NewHandler serves the section-1 article shell plus article-scoped API routes.
func NewHandler(app *replapi.App) (http.Handler, error) {
	if app == nil {
		return nil, errors.New("replessay: app is nil")
	}

	rawAPI, err := replhttp.NewHandler(app)
	if err != nil {
		return nil, err
	}
	uiDistFS := resolveEssayUIDistFS()

	mux := http.NewServeMux()
	mux.Handle("/api/sessions", rawAPI)
	mux.Handle("/api/sessions/", rawAPI)
	if uiDistFS != nil {
		mux.Handle(meetSessionStaticPrefix, http.StripPrefix(meetSessionStaticPrefix, http.FileServer(http.FS(uiDistFS))))
	}
	mux.HandleFunc(strings.TrimSuffix(meetSessionStaticPrefix, "/"), func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, meetSessionStaticPrefix, http.StatusMovedPermanently)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, MeetSessionPagePath, http.StatusSeeOther)
	})
	mux.HandleFunc(MeetSessionPagePath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		renderMeetSessionPage(w, uiDistFS)
	})
	mux.HandleFunc(MeetSessionBootstrapPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, buildBootstrapResponse())
	})
	mux.HandleFunc(MeetSessionCreatePath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		summary, err := app.CreateSession(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"session": summary})
	})
	mux.HandleFunc(meetSessionSnapshotPrefix, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		sessionID := strings.TrimPrefix(r.URL.Path, meetSessionSnapshotPrefix)
		sessionID = strings.TrimSpace(strings.Trim(sessionID, "/"))
		if sessionID == "" {
			writeJSONErrorMessage(w, http.StatusNotFound, "session id missing")
			return
		}
		summary, err := app.Snapshot(r.Context(), sessionID)
		if err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"session": summary})
	})
	mux.HandleFunc(ProfilesBootstrapPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, buildProfileSectionResponse())
	})
	mux.HandleFunc(ProfilesCreatePath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		req, err := decodeCreateSessionRequest(r)
		if err != nil {
			writeJSONErrorMessage(w, http.StatusBadRequest, err.Error())
			return
		}
		profile, err := parseProfile(req.Profile)
		if err != nil {
			writeJSONErrorMessage(w, http.StatusBadRequest, err.Error())
			return
		}
		summary, err := app.CreateSessionWithOptions(r.Context(), replapi.SessionOptions{Profile: profile})
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"session": summary})
	})
	mux.HandleFunc(profilesSnapshotPrefix, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		sessionID := strings.TrimPrefix(r.URL.Path, profilesSnapshotPrefix)
		sessionID = strings.TrimSpace(strings.Trim(sessionID, "/"))
		if sessionID == "" {
			writeJSONErrorMessage(w, http.StatusNotFound, "session id missing")
			return
		}
		summary, err := app.Snapshot(r.Context(), sessionID)
		if err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"session": summary})
	})
	mux.HandleFunc(CodeFlowBootstrapPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, buildCodeFlowSectionResponse())
	})
	mux.HandleFunc(CodeFlowCreatePath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		req, err := decodeCreateSessionRequest(r)
		if err != nil {
			writeJSONErrorMessage(w, http.StatusBadRequest, err.Error())
			return
		}
		profile, err := parseProfile(req.Profile)
		if err != nil {
			writeJSONErrorMessage(w, http.StatusBadRequest, err.Error())
			return
		}
		if profile == nil {
			defaultProfile := replapi.ProfileInteractive
			profile = &defaultProfile
		}
		summary, err := app.CreateSessionWithOptions(r.Context(), replapi.SessionOptions{Profile: profile})
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"session": summary})
	})
	mux.HandleFunc(codeFlowSnapshotPrefix, func(w http.ResponseWriter, r *http.Request) {
		trimmed := strings.TrimPrefix(r.URL.Path, codeFlowSnapshotPrefix)
		trimmed = strings.Trim(trimmed, "/")
		if trimmed == "" {
			writeJSONErrorMessage(w, http.StatusNotFound, "session id missing")
			return
		}
		if strings.HasSuffix(trimmed, codeFlowEvaluatePathSuffix) {
			if r.Method != http.MethodPost {
				writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			sessionID := strings.TrimSpace(strings.TrimSuffix(trimmed, codeFlowEvaluatePathSuffix))
			if sessionID == "" {
				writeJSONErrorMessage(w, http.StatusNotFound, "session id missing")
				return
			}
			var req replsession.EvaluateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSONErrorMessage(w, http.StatusBadRequest, "invalid JSON body")
				return
			}
			resp, err := app.Evaluate(r.Context(), sessionID, req.Source)
			if err != nil {
				writeJSONError(w, statusForError(err), err)
				return
			}
			writeJSON(w, http.StatusOK, resp)
			return
		}
		if r.Method != http.MethodGet {
			writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		sessionID := strings.TrimSpace(trimmed)
		summary, err := app.Snapshot(r.Context(), sessionID)
		if err != nil {
			writeJSONError(w, statusForError(err), err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"session": summary})
	})
	mux.HandleFunc(PersistenceBootstrapPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, buildPersistenceSectionResponse())
	})
	mux.HandleFunc(TimeoutBootstrapPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, buildTimeoutSectionResponse())
	})

	return mux, nil
}

func decodeCreateSessionRequest(r *http.Request) (createSessionRequest, error) {
	var req createSessionRequest
	if r == nil || r.Body == nil {
		return req, nil
	}
	if r.ContentLength == 0 {
		return req, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return createSessionRequest{}, errors.New("invalid JSON body")
	}
	return req, nil
}

func buildBootstrapResponse() BootstrapResponse {
	defaults := replsession.PersistentSessionOptions()
	return BootstrapResponse{
		Section: SectionSpec{
			ID:      "meet-a-session",
			Title:   "Meet a Session",
			Summary: "Create one real REPL session, then use it to learn how identity, policy, and backend state fit together.",
			Intro: []string{
				"A session is the durable unit of state in the new REPL. It is not only a prompt. It carries an id, a profile, a policy, and a growing body of runtime and persistence data.",
				"In this section, the browser will trigger one real session creation request, then render the resulting SessionSummary in several synchronized forms: prose, summary table, policy table, and raw JSON.",
				"The intended lesson is architectural. You should leave this section understanding which fields matter first, which backend routes produce them, and which source files own the behavior you are seeing.",
			},
			PrimaryAction: ActionSpec{
				Label:  "Create Session",
				Method: http.MethodPost,
				Path:   MeetSessionCreatePath,
			},
			Panels: []PanelSpec{
				{
					ID:          "session-summary",
					Title:       "Session Summary",
					Kind:        "summary-card",
					Description: "Compact identity and count fields taken directly from SessionSummary.",
					Fields: []FieldSpec{
						{Label: "Session ID", JSONPath: "session.id", Description: "The durable session identifier returned by the backend."},
						{Label: "Profile", JSONPath: "session.profile", Description: "The active profile for this session."},
						{Label: "Created", JSONPath: "session.createdAt", Description: "Session creation timestamp in UTC."},
						{Label: "Cell Count", JSONPath: "session.cellCount", Description: "How many cells have been evaluated in this session."},
						{Label: "Binding Count", JSONPath: "session.bindingCount", Description: "How many bindings are currently tracked in the session."},
					},
				},
				{
					ID:          "policy-card",
					Title:       "Policy",
					Kind:        "policy-card",
					Description: "Human-readable view of eval, observe, and persist policy fields.",
				},
				{
					ID:          "session-json",
					Title:       "Raw Session JSON",
					Kind:        "json-inspector",
					Description: "Exact JSON payload returned by the backend for trust and debugging.",
				},
			},
		},
		DefaultView: DefaultViewSpec{
			Profile: defaults.Profile,
			Policy:  defaults.Policy,
		},
		RawRoutes: []RouteRef{
			{Method: http.MethodPost, Path: MeetSessionCreatePath, Purpose: "Article-scoped route that creates one session using the essay's default persistent profile."},
			{Method: http.MethodGet, Path: meetSessionSnapshotPrefix + "{sessionID}", Purpose: "Article-scoped route that fetches a fresh read-model for one existing session id."},
			{Method: http.MethodPost, Path: "/api/sessions", Purpose: "Underlying raw REPL create-session route exposed for trust, debugging, and future deeper sections."},
			{Method: http.MethodGet, Path: "/api/sessions/{sessionID}", Purpose: "Underlying raw REPL snapshot route for direct inspection outside the essay wrapper."},
		},
	}
}

func renderMeetSessionPage(w http.ResponseWriter, uiDistFS fs.FS) {
	if uiDistFS != nil {
		if indexHTML, err := fs.ReadFile(uiDistFS, "index.html"); err == nil && len(indexHTML) > 0 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(indexHTML)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = meetSessionFallbackPageTemplate.Execute(w, map[string]any{
		"BootstrapPath":  MeetSessionBootstrapPath,
		"CreatePath":     MeetSessionCreatePath,
		"SnapshotPrefix": meetSessionSnapshotPrefix,
	})
}

func resolveEssayUIDistFS() fs.FS {
	if explicit := strings.TrimSpace(os.Getenv("GOJA_REPL_ESSAY_WEB_DIST")); explicit != "" {
		if explicitFS := dirFSIfExists(explicit); explicitFS != nil {
			return explicitFS
		}
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	defaultDist := filepath.Join(repoRoot, "web", "dist", "public")
	return dirFSIfExists(defaultDist)
}

func dirFSIfExists(path string) fs.FS {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return nil
	}
	return os.DirFS(path)
}

func statusForError(err error) int {
	switch {
	case errors.Is(err, replsession.ErrSessionNotFound), errors.Is(err, repldb.ErrSessionNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, err error) {
	writeJSONErrorMessage(w, status, err.Error())
}

func writeJSONErrorMessage(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

var meetSessionFallbackPageTemplate = template.Must(template.New("meet-a-session-fallback").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>GOJA-043 REPL Essay</title>
  <style>
    :root {
      --bg: #f5f4ef;
      --surface: #fffdfa;
      --ink: #1f2421;
      --muted: #66685f;
      --border: #d8d2c6;
      --accent: #1a6f63;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: Georgia, "Iowan Old Style", "Palatino Linotype", serif;
      background: var(--bg);
      color: var(--ink);
      line-height: 1.55;
    }
    main {
      max-width: 760px;
      margin: 0 auto;
      padding: 36px 20px 52px;
    }
    .panel {
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: 14px;
      padding: 18px;
    }
    h1 { margin: 0 0 10px; font-size: 2rem; }
    p { margin: 0 0 12px; color: var(--muted); }
    code {
      font-family: "SFMono-Regular", Consolas, monospace;
      font-size: 0.92rem;
      color: var(--ink);
    }
    a { color: var(--accent); }
    .meta { margin-top: 12px; font-size: 0.92rem; }
    ul { margin: 10px 0 0 0; padding-left: 20px; }
    li { margin: 6px 0; }
    .root {
      margin-top: 12px;
      padding: 12px;
      border: 1px dashed var(--border);
      border-radius: 10px;
    }
  </style>
</head>
<body>
  <main>
    <section class="panel">
      <h1>Meet a Session</h1>
      <p>Frontend build assets were not found, so the server is showing a fallback shell.</p>
      <p class="meta">To load the React frontend, run <code>pnpm -C web build</code> from the repository root, then reload this page.</p>
      <ul>
        <li>Bootstrap endpoint: <a href="{{ .BootstrapPath }}"><code>{{ .BootstrapPath }}</code></a></li>
        <li>Create endpoint: <code>{{ .CreatePath }}</code></li>
        <li>Snapshot endpoint prefix: <code>{{ .SnapshotPrefix }}</code></li>
      </ul>
      <div class="root"><code>#root</code> mount placeholder</div>
    </section>
  </main>
</body>
</html>`))
