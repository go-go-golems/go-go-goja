package server

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/batch"
	jsdocexport "github.com/go-go-golems/go-go-goja/pkg/jsdoc/export"
	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/extract"
	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
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

	parsePath, err := s.scopedPathParser()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	br, err := batch.BuildStore(r.Context(), inputs, batch.BatchOptions{
		ContinueOnError: req.ContinueOnError,
		ParsePath:       parsePath,
	})
	if err != nil {
		if isPathPolicyError(err) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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

	parsePath, err := s.scopedPathParser()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	br, err := batch.BuildStore(r.Context(), inputs, batch.BatchOptions{
		ContinueOnError: req.Options.ContinueOnError,
		ParsePath:       parsePath,
	})
	if err != nil {
		if isPathPolicyError(err) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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
		if bi.Content != "" && bi.Path != "" {
			return nil, errors.Errorf("inputs[%d]: path and content cannot both be set", i)
		}
		if bi.Content != "" {
			name := bi.DisplayName
			if name == "" {
				name = "inline"
			}
			out = append(out, batch.InputFile{
				Content:     []byte(bi.Content),
				DisplayName: name,
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

	return filepath.ToSlash(clean), nil
}

func (s *Server) scopedPathParser() (func(path string) (*model.FileDoc, error), error) {
	scopedFS, err := extract.NewScopedFS(s.dir)
	if err != nil {
		return nil, errors.Wrap(err, "create scoped fs")
	}

	return func(path string) (*model.FileDoc, error) {
		return extract.ParseFSFile(scopedFS, path)
	}, nil
}

func isPathPolicyError(err error) bool {
	return errors.Is(err, extract.ErrEmptyPath) ||
		errors.Is(err, extract.ErrAbsolutePath) ||
		errors.Is(err, extract.ErrInvalidPath) ||
		errors.Is(err, extract.ErrPathTraversal) ||
		errors.Is(err, extract.ErrOutsideAllowedRoot)
}
