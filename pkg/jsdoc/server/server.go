// Package server provides the HTTP server for the JS doc browser.
// It serves the web UI, exposes a JSON API, and pushes SSE events
// when JS files change.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/extract"
	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/model"
	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/watch"
)

// Server is the HTTP doc browser server.
type Server struct {
	store *model.DocStore
	dir   string
	host  string
	port  int

	mu      sync.RWMutex
	clients map[chan string]struct{}
}

// New creates a new Server.
func New(store *model.DocStore, dir, host string, port int) *Server {
	return &Server{
		store:   store,
		dir:     dir,
		host:    host,
		port:    port,
		clients: make(map[chan string]struct{}),
	}
}

// Handler returns an http.Handler exposing the web UI, JSON API, and SSE stream.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// API routes.
	mux.HandleFunc("/api/store", s.handleStore)
	mux.HandleFunc("/api/package/", s.handlePackage)
	mux.HandleFunc("/api/symbol/", s.handleSymbol)
	mux.HandleFunc("/api/example/", s.handleExample)
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/batch/extract", s.handleBatchExtract)
	mux.HandleFunc("/api/batch/export", s.handleBatchExport)

	// SSE for live reload.
	mux.HandleFunc("/events", s.handleSSE)

	// Serve the single-page app for all other routes.
	mux.HandleFunc("/", s.handleUI)

	return mux
}

// Run begins serving and watching for file changes, until ctx is canceled.
func (s *Server) Run(ctx context.Context) error {
	if s.host == "" {
		s.host = "127.0.0.1"
	}
	if s.port <= 0 {
		s.port = 8080
	}

	w := watch.New(s.dir, s.onFileChange)
	go func() {
		if err := w.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "watcher error: %v\n", err)
		}
	}()
	defer w.Stop()

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	fmt.Printf("Doc browser running at http://%s\n", addr)

	srv := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		return errors.Wrap(err, "http server")
	}
}

// onFileChange is called by the watcher when a JS file changes.
func (s *Server) onFileChange(ev watch.Event) {
	fmt.Printf("File changed: %s (%s)\n", ev.Path, ev.Op)

	if ev.Op == "remove" || ev.Op == "rename" {
		s.mu.Lock()
		s.store.AddFile(&model.FileDoc{FilePath: ev.Path}) // empty doc removes old entries
		s.mu.Unlock()
	} else {
		src, err := os.ReadFile(ev.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse error: reading %s: %v\n", ev.Path, err)
			return
		}
		fd, err := extract.ParseSource(ev.Path, src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
			return
		}
		s.mu.Lock()
		s.store.AddFile(fd)
		s.mu.Unlock()
	}

	// Broadcast SSE reload event.
	s.broadcast("reload")
}

// broadcast sends a message to all connected SSE clients.
func (s *Server) broadcast(msg string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for ch := range s.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

// ---- HTTP handlers ----

func (s *Server) handleStore(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, s.store)
}

func (s *Server) handlePackage(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/package/")
	s.mu.RLock()
	pkg, ok := s.store.ByPackage[name]
	s.mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, pkg)
}

func (s *Server) handleSymbol(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/symbol/")
	s.mu.RLock()
	sym, ok := s.store.BySymbol[name]
	s.mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	// Enrich: attach related examples.
	type SymbolResponse struct {
		*model.SymbolDoc
		Examples []*model.Example `json:"examples"`
	}
	resp := SymbolResponse{SymbolDoc: sym}

	s.mu.RLock()
	for _, ex := range s.store.ByExample {
		for _, sn := range ex.Symbols {
			if sn == name {
				resp.Examples = append(resp.Examples, ex)
				break
			}
		}
	}
	s.mu.RUnlock()

	writeJSON(w, resp)
}

func (s *Server) handleExample(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/example/")
	s.mu.RLock()
	ex, ok := s.store.ByExample[id]
	s.mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, ex)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, map[string]interface{}{"symbols": nil, "examples": nil, "packages": nil})
		return
	}

	type SearchResult struct {
		Symbols  []*model.SymbolDoc `json:"symbols"`
		Examples []*model.Example   `json:"examples"`
		Packages []*model.Package   `json:"packages"`
	}

	var result SearchResult

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, sym := range s.store.BySymbol {
		if strings.Contains(strings.ToLower(sym.Name), q) ||
			strings.Contains(strings.ToLower(sym.Summary), q) ||
			containsAny(sym.Tags, q) ||
			containsAny(sym.Concepts, q) {
			result.Symbols = append(result.Symbols, sym)
		}
	}
	for _, ex := range s.store.ByExample {
		if strings.Contains(strings.ToLower(ex.ID), q) ||
			strings.Contains(strings.ToLower(ex.Title), q) ||
			containsAny(ex.Tags, q) ||
			containsAny(ex.Concepts, q) {
			result.Examples = append(result.Examples, ex)
		}
	}
	for _, pkg := range s.store.ByPackage {
		if strings.Contains(strings.ToLower(pkg.Name), q) ||
			strings.Contains(strings.ToLower(pkg.Title), q) ||
			strings.Contains(strings.ToLower(pkg.Description), q) {
			result.Packages = append(result.Packages, pkg)
		}
	}

	writeJSON(w, result)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan string, 4)
	s.mu.Lock()
	s.clients[ch] = struct{}{}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, ch)
		s.mu.Unlock()
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial heartbeat.
	fmt.Fprintf(w, "data: connected\n\n")
	flusher.Flush()

	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(uiHTML))
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func containsAny(slice []string, q string) bool {
	for _, s := range slice {
		if strings.Contains(strings.ToLower(s), q) {
			return true
		}
	}
	return false
}
