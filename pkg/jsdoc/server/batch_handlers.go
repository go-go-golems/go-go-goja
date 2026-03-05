package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/batch"
	jsdocexport "github.com/go-go-golems/go-go-goja/pkg/jsdoc/export"
)

type batchInput struct {
	Path        string `json:"path,omitempty"`
	Content     string `json:"content,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

type batchExtractRequest struct {
	Inputs          []batchInput `json:"inputs"`
	ContinueOnError bool         `json:"continueOnError,omitempty"`
}

type batchExportOptions struct {
	Shape           string `json:"shape,omitempty"`
	Pretty          bool   `json:"pretty,omitempty"`
	Indent          string `json:"indent,omitempty"`
	TOCDepth        int    `json:"tocDepth,omitempty"`
	ContinueOnError bool   `json:"continueOnError,omitempty"`
}

type batchExportRequest struct {
	Inputs  []batchInput       `json:"inputs"`
	Format  string             `json:"format"`
	Options batchExportOptions `json:"options,omitempty"`
}

func (s *Server) handleBatchExtract(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req := batchExtractRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	inputs, err := s.inputsFromRequest(req.Inputs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	br, err := batch.BuildStore(r.Context(), inputs, batch.BatchOptions{ContinueOnError: req.ContinueOnError})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	writeJSON(w, br)
}

func (s *Server) handleBatchExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req := batchExportRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Format == "" {
		http.Error(w, "format is required", http.StatusBadRequest)
		return
	}
	inputs, err := s.inputsFromRequest(req.Inputs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	br, err := batch.BuildStore(r.Context(), inputs, batch.BatchOptions{ContinueOnError: req.Options.ContinueOnError})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(br.Errors) > 0 {
		w.Header().Set("X-JSDoc-Error-Count", strconv.Itoa(len(br.Errors)))
	}

	format := jsdocexport.Format(req.Format)
	switch format {
	case jsdocexport.FormatJSON:
		w.Header().Set("Content-Type", "application/json")
	case jsdocexport.FormatYAML:
		w.Header().Set("Content-Type", "application/yaml")
	case jsdocexport.FormatMarkdown:
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	case jsdocexport.FormatSQLite:
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="docs.sqlite"`)
	default:
		http.Error(w, "unknown format", http.StatusBadRequest)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")

	indent := req.Options.Indent
	if indent == "" && format == jsdocexport.FormatJSON && req.Options.Pretty {
		indent = "  "
	}

	opts := jsdocexport.Options{
		Format:   format,
		Shape:    jsdocexport.Shape(req.Options.Shape),
		Indent:   indent,
		TOCDepth: req.Options.TOCDepth,
	}

	if err := jsdocexport.Export(r.Context(), br.Store, w, opts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) inputsFromRequest(in []batchInput) ([]batch.InputFile, error) {
	if len(in) == 0 {
		return nil, errors.New("inputs is required")
	}
	out := make([]batch.InputFile, 0, len(in))
	for i, bi := range in {
		if bi.Content != "" {
			name := bi.DisplayName
			if name == "" {
				name = bi.Path
			}
			out = append(out, batch.InputFile{
				Content:     []byte(bi.Content),
				DisplayName: name,
				Path:        bi.Path,
			})
			continue
		}
		if bi.Path == "" {
			return nil, errors.Errorf("inputs[%d]: either path or content is required", i)
		}
		p, err := s.resolvePath(bi.Path)
		if err != nil {
			return nil, errors.Wrapf(err, "inputs[%d]", i)
		}
		out = append(out, batch.InputFile{Path: p, DisplayName: bi.DisplayName})
	}
	return out, nil
}

func (s *Server) resolvePath(p string) (string, error) {
	if filepath.IsAbs(p) {
		return "", errors.Errorf("absolute paths are not allowed")
	}
	clean := filepath.Clean(p)
	if clean == "." || clean == string(filepath.Separator) {
		return "", errors.Errorf("path is invalid")
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.Errorf("path traversal is not allowed")
	}

	rootAbs, err := filepath.Abs(s.dir)
	if err != nil {
		return "", errors.Wrap(err, "abs root")
	}
	full := filepath.Join(rootAbs, clean)
	fullAbs, err := filepath.Abs(full)
	if err != nil {
		return "", errors.Wrap(err, "abs path")
	}

	sep := string(os.PathSeparator)
	if fullAbs != rootAbs && !strings.HasPrefix(fullAbs, rootAbs+sep) {
		return "", errors.Errorf("path is outside allowed root")
	}
	return fullAbs, nil
}
