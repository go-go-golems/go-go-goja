package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ──────────────────────────────────────────────
// Data model for profile manifests
// ──────────────────────────────────────────────

// profileManifest is the top-level structure loaded from a profile summary YAML.
type profileManifest struct {
	GeneratedAt string              `yaml:"generated_at"`
	Artifacts   []profileArtifact   `yaml:"artifacts"`
	Comparisons []profileComparison `yaml:"comparisons"`
}

// profileArtifact describes a single profile artifact file.
type profileArtifact struct {
	ID          string   `yaml:"id"`
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Phase       string   `yaml:"phase"`
	TaskID      string   `yaml:"task_id"`
	Benchmark   string   `yaml:"benchmark"`
	Kind        string   `yaml:"kind"` // flamegraph_svg, pprof_cpu, pprof_mem, top_report, summary
	RelPath     string   `yaml:"rel_path"`
	Mime        string   `yaml:"mime"`
	Bytes       int64    `yaml:"bytes"`
	GeneratedAt string   `yaml:"generated_at"`
	Tags        []string `yaml:"tags"`
}

// profileComparison describes a baseline/candidate comparison.
type profileComparison struct {
	ID                  string `yaml:"id"`
	Title               string `yaml:"title"`
	BaselineArtifactID  string `yaml:"baseline_artifact_id"`
	CandidateArtifactID string `yaml:"candidate_artifact_id"`
	DiffArtifactID      string `yaml:"diff_artifact_id"`
	SummaryArtifactID   string `yaml:"summary_artifact_id"`
}

// ──────────────────────────────────────────────
// Manifest loading and artifact lookup
// ──────────────────────────────────────────────

// loadProfileManifest loads a profile manifest YAML from disk.
func loadProfileManifest(path string) (*profileManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := &profileManifest{}
	if err := yaml.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("parse profile manifest: %w", err)
	}
	return m, nil
}

// findArtifact looks up an artifact by ID in the manifest.
func (m *profileManifest) findArtifact(id string) *profileArtifact {
	for i := range m.Artifacts {
		if m.Artifacts[i].ID == id {
			return &m.Artifacts[i]
		}
	}
	return nil
}

// ──────────────────────────────────────────────
// Safe file serving
// ──────────────────────────────────────────────

// safeResolvePath validates that an artifact's rel_path resolves to a regular file
// under repoRoot. Returns the cleaned absolute path or an error.
func safeResolvePath(repoRoot, relPath string) (string, error) {
	if relPath == "" {
		return "", fmt.Errorf("empty path")
	}
	abs := filepath.Join(repoRoot, relPath)
	clean := filepath.Clean(abs)
	// Ensure it stays under repo root
	rootClean := filepath.Clean(repoRoot) + string(filepath.Separator)
	if !strings.HasPrefix(clean, rootClean) && clean != filepath.Clean(repoRoot) {
		return "", fmt.Errorf("path escapes repo root")
	}
	info, err := os.Stat(clean)
	if err != nil {
		return "", fmt.Errorf("stat: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file")
	}
	return clean, nil
}

// detectMime returns a MIME type for an artifact based on its kind and path.
func detectMime(art *profileArtifact) string {
	if art.Mime != "" {
		return art.Mime
	}
	ext := filepath.Ext(art.RelPath)
	switch ext {
	case ".svg":
		return "image/svg+xml"
	case ".pprof":
		return "application/octet-stream"
	case ".txt":
		return "text/plain; charset=utf-8"
	case ".yaml", ".yml":
		return "text/yaml; charset=utf-8"
	default:
		mt := mime.TypeByExtension(ext)
		if mt != "" {
			return mt
		}
		return "application/octet-stream"
	}
}

// ──────────────────────────────────────────────
// HTTP handlers
// ──────────────────────────────────────────────

// handleProfiles serves the Profiles HTML fragment for a phase.
// GET /api/profiles/{phase}
func (a *perfWebApp) handleProfiles(w http.ResponseWriter, r *http.Request) {
	phaseID := strings.TrimPrefix(r.URL.Path, "/api/profiles/")
	_, ok := a.phases[phaseID]
	if !ok {
		http.Error(w, "unknown phase", http.StatusNotFound)
		return
	}

	manifest := a.loadPhaseProfiles(phaseID)
	data := buildProfilesViewData(phaseID, manifest)

	tmpl := template.Must(template.New("profiles").Parse(profilesFragmentTemplate))
	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}

// handleProfileArtifact serves a profile artifact file.
// GET /api/profile-artifact/{phase}/{artifactID}
func (a *perfWebApp) handleProfileArtifact(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/profile-artifact/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request: expected /api/profile-artifact/{phase}/{artifactID}", http.StatusBadRequest)
		return
	}
	phaseID, artifactID := parts[0], parts[1]

	if _, ok := a.phases[phaseID]; !ok {
		http.Error(w, "unknown phase", http.StatusNotFound)
		return
	}

	manifest := a.loadPhaseProfiles(phaseID)
	if manifest == nil {
		http.Error(w, "no profile manifest", http.StatusNotFound)
		return
	}

	art := manifest.findArtifact(artifactID)
	if art == nil {
		http.Error(w, "artifact not found in manifest", http.StatusNotFound)
		return
	}

	absPath, err := safeResolvePath(a.repoRoot, art.RelPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("cannot serve artifact: %v", err), http.StatusNotFound)
		return
	}

	f, err := os.Open(absPath)
	if err != nil {
		http.Error(w, "cannot open artifact", http.StatusInternalServerError)
		return
	}
	defer func() { _ = f.Close() }()

	w.Header().Set("Content-Type", detectMime(art))

	// For download variant, check query param
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(art.RelPath)))
	}

	_, _ = io.Copy(w, f)
}

// loadPhaseProfiles loads the profile manifest for a given phase.
// It looks for a profiles.yaml file in the phase output directory.
func (a *perfWebApp) loadPhaseProfiles(phaseID string) *profileManifest {
	cfg, ok := a.phases[phaseID]
	if !ok {
		return nil
	}
	// Look for profiles.yaml next to the output file
	dir := filepath.Dir(filepath.Join(a.repoRoot, cfg.OutputFile))
	manifestPath := filepath.Join(dir, "profiles.yaml")
	m, err := loadProfileManifest(manifestPath)
	if err != nil {
		return nil
	}
	return m
}

// ──────────────────────────────────────────────
// View data for Profiles fragment
// ──────────────────────────────────────────────

type profilesViewData struct {
	Phase         string
	HasManifest   bool
	GeneratedAt   string
	ArtifactCount int
	CompareCount  int
	Artifacts     []profileArtifactView
	Comparisons   []profileComparisonView
}

type profileArtifactView struct {
	ID          string
	Title       string
	Description string
	Kind        string
	KindLabel   string
	TaskID      string
	Benchmark   string
	Tags        []string
	Phase       string
	ViewURL     string
	DownloadURL string
	IsSVG       bool
}

type profileComparisonView struct {
	ID            string
	Title         string
	BaselineView  *profileArtifactView
	CandidateView *profileArtifactView
	DiffView      *profileArtifactView
	SummaryView   *profileArtifactView
}

func buildProfilesViewData(phaseID string, manifest *profileManifest) profilesViewData {
	data := profilesViewData{Phase: phaseID}
	if manifest == nil {
		return data
	}
	data.HasManifest = true
	data.GeneratedAt = manifest.GeneratedAt
	data.ArtifactCount = len(manifest.Artifacts)
	data.CompareCount = len(manifest.Comparisons)

	// Build artifact views
	artMap := make(map[string]*profileArtifactView)
	for _, a := range manifest.Artifacts {
		av := profileArtifactView{
			ID:          a.ID,
			Title:       a.Title,
			Description: a.Description,
			Kind:        a.Kind,
			KindLabel:   kindLabel(a.Kind),
			TaskID:      a.TaskID,
			Benchmark:   shortBench(a.Benchmark),
			Tags:        a.Tags,
			Phase:       phaseID,
			ViewURL:     fmt.Sprintf("/api/profile-artifact/%s/%s", phaseID, a.ID),
			DownloadURL: fmt.Sprintf("/api/profile-artifact/%s/%s?download=1", phaseID, a.ID),
			IsSVG:       strings.HasSuffix(a.RelPath, ".svg"),
		}
		data.Artifacts = append(data.Artifacts, av)
		artMap[a.ID] = &data.Artifacts[len(data.Artifacts)-1]
	}

	// Build comparison views
	for _, c := range manifest.Comparisons {
		cv := profileComparisonView{
			ID:    c.ID,
			Title: c.Title,
		}
		if v, ok := artMap[c.BaselineArtifactID]; ok {
			cv.BaselineView = v
		}
		if v, ok := artMap[c.CandidateArtifactID]; ok {
			cv.CandidateView = v
		}
		if v, ok := artMap[c.DiffArtifactID]; ok {
			cv.DiffView = v
		}
		if v, ok := artMap[c.SummaryArtifactID]; ok {
			cv.SummaryView = v
		}
		data.Comparisons = append(data.Comparisons, cv)
	}

	return data
}

func kindLabel(kind string) string {
	switch kind {
	case "flamegraph_svg":
		return "Flamegraph"
	case "pprof_cpu":
		return "CPU Profile"
	case "pprof_mem":
		return "Memory Profile"
	case "top_report":
		return "Top Report"
	case "summary":
		return "Summary"
	default:
		return kind
	}
}

// ──────────────────────────────────────────────
// Profiles HTML fragment template
// ──────────────────────────────────────────────

const profilesFragmentTemplate = `<div>
  {{if not .HasManifest}}
    <div class="text-muted text-center py-4">
      No profile manifest found.<br>
      <span class="small">Place a <code>profiles.yaml</code> file next to the phase output file.</span>
    </div>
  {{else}}
    <div class="summary-bar">
      <span>&#x1F4C5; <strong>{{.GeneratedAt}}</strong></span>
      <span>&#x1F4CA; <strong>{{.ArtifactCount}}</strong> artifact{{if ne .ArtifactCount 1}}s{{end}}</span>
      {{if gt .CompareCount 0}}
        <span>&#x2194; <strong>{{.CompareCount}}</strong> comparison{{if ne .CompareCount 1}}s{{end}}</span>
      {{end}}
    </div>

    {{if gt .CompareCount 0}}
      <h6 class="mt-3 mb-2">Comparisons</h6>
      {{range .Comparisons}}
        <div class="border rounded p-3 mb-3 bg-white">
          <strong>{{.Title}}</strong>
          <div class="d-flex gap-3 mt-2 flex-wrap">
            {{if .BaselineView}}
              <div class="profile-card">
                <div class="small fw-semibold text-muted mb-1">Baseline</div>
                <div class="fw-semibold">{{.BaselineView.Title}}</div>
                <div class="mt-1">
                  <a href="{{.BaselineView.ViewURL}}" target="_blank" class="btn btn-sm btn-outline-primary">View</a>
                  <a href="{{.BaselineView.DownloadURL}}" class="btn btn-sm btn-outline-secondary">Download</a>
                </div>
              </div>
            {{end}}
            {{if .CandidateView}}
              <div class="profile-card">
                <div class="small fw-semibold text-muted mb-1">Candidate</div>
                <div class="fw-semibold">{{.CandidateView.Title}}</div>
                <div class="mt-1">
                  <a href="{{.CandidateView.ViewURL}}" target="_blank" class="btn btn-sm btn-outline-primary">View</a>
                  <a href="{{.CandidateView.DownloadURL}}" class="btn btn-sm btn-outline-secondary">Download</a>
                </div>
              </div>
            {{end}}
            {{if .DiffView}}
              <div class="profile-card diff">
                <div class="small fw-semibold text-muted mb-1">Diff</div>
                <div class="fw-semibold">{{.DiffView.Title}}</div>
                <div class="mt-1">
                  <a href="{{.DiffView.ViewURL}}" target="_blank" class="btn btn-sm btn-outline-danger">View Diff</a>
                  <a href="{{.DiffView.DownloadURL}}" class="btn btn-sm btn-outline-secondary">Download</a>
                </div>
              </div>
            {{end}}
          </div>
        </div>
      {{end}}
    {{end}}

    {{if gt .ArtifactCount 0}}
      <h6 class="mt-3 mb-2">All Artifacts</h6>
      <div class="table-responsive">
        <table class="table table-sm table-hover align-middle">
          <thead class="table-light">
            <tr>
              <th>Title</th>
              <th>Type</th>
              <th>Benchmark</th>
              <th>Tags</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {{range .Artifacts}}
              <tr>
                <td>
                  <strong>{{.Title}}</strong>
                  {{if .Description}}<br><span class="small text-muted">{{.Description}}</span>{{end}}
                </td>
                <td><span class="badge bg-secondary">{{.KindLabel}}</span></td>
                <td><code class="small">{{.Benchmark}}</code></td>
                <td>{{range .Tags}}<span class="badge bg-light text-dark me-1">{{.}}</span>{{end}}</td>
                <td>
                  <a href="{{.ViewURL}}" target="_blank" class="btn btn-sm btn-outline-primary py-0 px-2">View</a>
                  <a href="{{.DownloadURL}}" class="btn btn-sm btn-outline-secondary py-0 px-2">&#x2193;</a>
                </td>
              </tr>
            {{end}}
          </tbody>
        </table>
      </div>
    {{end}}
  {{end}}
</div>`
