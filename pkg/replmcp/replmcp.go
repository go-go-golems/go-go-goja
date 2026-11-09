// Package replmcp serves go-go-goja REPL sessions as an MCP server.
//
// The server exposes an "eval" tool plus session inspection tools over any
// Evaluator (normally *replapi.App). Result content produced through
// pkg/resultcontent (for example images returned by a module) is returned as
// MCP image content next to the text result.
package replmcp

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/go-go-golems/go-go-goja/pkg/replapi"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
)

// Evaluator is the session surface the MCP server needs. *replapi.App
// implements it. ListSessions and History may fail when the evaluator has no
// durable store; the server then falls back to live session state.
type Evaluator interface {
	CreateSession(ctx context.Context) (*replsession.SessionSummary, error)
	Evaluate(ctx context.Context, sessionID string, source string) (*replsession.EvaluateResponse, error)
	Snapshot(ctx context.Context, sessionID string) (*replsession.SessionSummary, error)
	ListSessions(ctx context.Context) ([]repldb.SessionRecord, error)
	History(ctx context.Context, sessionID string) ([]repldb.EvaluationRecord, error)
	Bindings(ctx context.Context, sessionID string) ([]replsession.BindingView, error)
}

var _ Evaluator = (*replapi.App)(nil)

const (
	// DefaultName is the MCP implementation name.
	DefaultName = "go-go-goja-repl"
	// DefaultVersion is the MCP implementation version.
	DefaultVersion = "dev"
	// DefaultMaxResultChars bounds the text of one tool result.
	DefaultMaxResultChars = 20000
	// DefaultMaxImages bounds the content items attached to one eval result.
	DefaultMaxImages = 8
	// DefaultMaxImageBytes bounds the size of one attached content item.
	DefaultMaxImageBytes = 4 << 20
)

// DefaultInstructions describes the server to MCP clients.
const DefaultInstructions = "Evaluate JavaScript in a persistent goja session with the eval tool. " +
	"Top-level declarations persist between calls in the same session, and top-level await is supported. " +
	"Values that carry images are returned as image content; the text result names them as [content N: ...]."

// Limits bounds one eval tool result. Zero or negative fields select defaults.
type Limits struct {
	MaxResultChars int
	MaxImages      int
	MaxImageBytes  int
}

func (l Limits) normalize() Limits {
	if l.MaxResultChars <= 0 {
		l.MaxResultChars = DefaultMaxResultChars
	}
	if l.MaxImages <= 0 {
		l.MaxImages = DefaultMaxImages
	}
	if l.MaxImageBytes <= 0 {
		l.MaxImageBytes = DefaultMaxImageBytes
	}
	return l
}

type config struct {
	name           string
	version        string
	instructions   string
	limits         Limits
	defaultSession string
	extraTools     []func(*mcp.Server)
	serverOptions  *mcp.ServerOptions
}

// Option configures NewServer.
type Option func(*config)

// WithName sets the MCP implementation name.
func WithName(name string) Option {
	return func(c *config) { c.name = strings.TrimSpace(name) }
}

// WithVersion sets the MCP implementation version.
func WithVersion(version string) Option {
	return func(c *config) { c.version = strings.TrimSpace(version) }
}

// WithInstructions replaces DefaultInstructions.
func WithInstructions(instructions string) Option {
	return func(c *config) { c.instructions = instructions }
}

// WithMaxResultChars bounds the text of one tool result (<= 0: default).
func WithMaxResultChars(n int) Option {
	return func(c *config) { c.limits.MaxResultChars = n }
}

// WithMaxImages bounds the content items attached to one eval result (<= 0: default).
func WithMaxImages(n int) Option {
	return func(c *config) { c.limits.MaxImages = n }
}

// WithMaxImageBytes bounds the size of one attached content item (<= 0: default).
func WithMaxImageBytes(n int) Option {
	return func(c *config) { c.limits.MaxImageBytes = n }
}

// WithLimits sets all eval result limits at once.
func WithLimits(limits Limits) Option {
	return func(c *config) { c.limits = limits }
}

// WithDefaultSession makes calls without a sessionId use this session. Without
// it, the server creates one implicit session on first use.
func WithDefaultSession(sessionID string) Option {
	return func(c *config) { c.defaultSession = strings.TrimSpace(sessionID) }
}

// WithExtraTools registers additional tools, resources or prompts after the
// built-in tools.
func WithExtraTools(register func(*mcp.Server)) Option {
	return func(c *config) {
		if register != nil {
			c.extraTools = append(c.extraTools, register)
		}
	}
}

// WithServerOptions passes options to mcp.NewServer. Instructions set through
// WithInstructions take precedence over opts.Instructions.
func WithServerOptions(opts *mcp.ServerOptions) Option {
	return func(c *config) { c.serverOptions = opts }
}

func newConfig(opts []Option) config {
	c := config{
		name:         DefaultName,
		version:      DefaultVersion,
		instructions: DefaultInstructions,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&c)
		}
	}
	if c.name == "" {
		c.name = DefaultName
	}
	if c.version == "" {
		c.version = DefaultVersion
	}
	c.limits = c.limits.normalize()
	return c
}

// NewServer builds an MCP server with the eval, create_session,
// list_sessions, history and bindings tools over ev.
func NewServer(ev Evaluator, opts ...Option) (*mcp.Server, error) {
	if ev == nil {
		return nil, errors.New("replmcp: evaluator is nil")
	}
	cfg := newConfig(opts)
	serverOpts := &mcp.ServerOptions{}
	if cfg.serverOptions != nil {
		copied := *cfg.serverOptions
		serverOpts = &copied
	}
	if cfg.instructions != "" {
		serverOpts.Instructions = cfg.instructions
	}
	server := mcp.NewServer(&mcp.Implementation{Name: cfg.name, Version: cfg.version}, serverOpts)
	h := &handlers{ev: ev, cfg: cfg, defaultSession: cfg.defaultSession, known: map[string]struct{}{}}
	if cfg.defaultSession != "" {
		h.known[cfg.defaultSession] = struct{}{}
	}
	h.register(server)
	for _, register := range cfg.extraTools {
		register(server)
	}
	return server, nil
}

// ServeStdio runs server over stdin/stdout until the client disconnects or
// ctx is canceled.
func ServeStdio(ctx context.Context, server *mcp.Server) error {
	if server == nil {
		return errors.New("replmcp: server is nil")
	}
	return server.Run(ctx, &mcp.StdioTransport{})
}

// NewStreamableHTTPHandler serves server over MCP streamable HTTP. All HTTP
// clients share the server (and so its implicit default session).
func NewStreamableHTTPHandler(server *mcp.Server, opts *mcp.StreamableHTTPOptions) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, opts)
}

type handlers struct {
	ev  Evaluator
	cfg config

	mu             sync.Mutex
	defaultSession string
	known          map[string]struct{}
	order          []string
}

func (h *handlers) remember(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.known[id]; ok {
		return
	}
	h.known[id] = struct{}{}
	h.order = append(h.order, id)
}

// resolveSession returns sessionID or the default session, creating the
// implicit default session on first use.
func (h *handlers) resolveSession(ctx context.Context, sessionID string) (string, error) {
	if id := strings.TrimSpace(sessionID); id != "" {
		return id, nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.defaultSession != "" {
		return h.defaultSession, nil
	}
	summary, err := h.ev.CreateSession(ctx)
	if err != nil {
		return "", errors.Wrap(err, "create default session")
	}
	h.defaultSession = summary.ID
	if _, ok := h.known[summary.ID]; !ok {
		h.known[summary.ID] = struct{}{}
		h.order = append(h.order, summary.ID)
	}
	return summary.ID, nil
}

func (h *handlers) currentDefault() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.defaultSession
}

func (h *handlers) knownSessions() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, 0, len(h.order)+1)
	if h.defaultSession != "" {
		out = append(out, h.defaultSession)
	}
	for _, id := range h.order {
		if id != h.defaultSession {
			out = append(out, id)
		}
	}
	return out
}
