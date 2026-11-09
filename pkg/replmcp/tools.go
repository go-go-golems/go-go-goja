package replmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
	"github.com/go-go-golems/go-go-goja/pkg/resultcontent"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
)

// EvalInput is the input of the eval tool.
type EvalInput struct {
	Code      string `json:"code" jsonschema:"JavaScript source to evaluate. The value of the last expression is the result; top-level await is supported."`
	SessionID string `json:"sessionId,omitempty" jsonschema:"Optional; omit it to use the default session (where the host's globals live); only pass ids returned by create_session or list_sessions."`
}

// SessionInput selects a session; empty means the default session.
type SessionInput struct {
	SessionID string `json:"sessionId,omitempty" jsonschema:"Optional; omit it to use the default session (where the host's globals live); only pass ids returned by create_session or list_sessions."`
}

// CreateSessionOutput is the output of create_session.
type CreateSessionOutput struct {
	SessionID string `json:"sessionId"`
}

// SessionInfo describes one session in list_sessions.
type SessionInfo struct {
	SessionID string     `json:"sessionId"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	Default   bool       `json:"default,omitempty"`
	Durable   bool       `json:"durable,omitempty"`
}

// ListSessionsOutput is the output of list_sessions.
type ListSessionsOutput struct {
	Sessions []SessionInfo `json:"sessions"`
}

// HistoryCell is one previous cell in history.
type HistoryCell struct {
	CellID int    `json:"cellId"`
	Status string `json:"status,omitempty"`
	Source string `json:"source"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// HistoryOutput is the output of history.
type HistoryOutput struct {
	SessionID string        `json:"sessionId"`
	Cells     []HistoryCell `json:"cells"`
	// Previews is set when sources/results are 100-character previews from
	// live state because no durable history is available.
	Previews bool `json:"previews,omitempty"`
}

// BindingInfo is one persisted global in bindings.
type BindingInfo struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Type    string `json:"type,omitempty"`
	Preview string `json:"preview,omitempty"`
}

// BindingsOutput is the output of bindings.
type BindingsOutput struct {
	SessionID string        `json:"sessionId"`
	Bindings  []BindingInfo `json:"bindings"`
}

func readOnly(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true, Title: title}
}

func (h *handlers) register(server *mcp.Server) {
	notDestructive := false
	mcp.AddTool(server, &mcp.Tool{
		Name:        "eval",
		Description: "Evaluate JavaScript in a session and return the result text, console output, errors, and any image content the result carries.",
		Annotations: &mcp.ToolAnnotations{Title: "Evaluate JavaScript", DestructiveHint: &notDestructive},
	}, h.eval)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_session",
		Description: "Create a new, empty JavaScript session and return its id.",
		Annotations: &mcp.ToolAnnotations{Title: "Create session", DestructiveHint: &notDestructive},
	}, h.createSession)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_sessions",
		Description: "List sessions known to this server (and durable sessions when a store is configured).",
		Annotations: readOnly("List sessions"),
	}, h.listSessions)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "history",
		Description: "Return the previously evaluated cells of a session (source, status, result text).",
		Annotations: readOnly("Session history"),
	}, h.history)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "bindings",
		Description: "Return the global names persisted in a session with their kinds and value previews.",
		Annotations: readOnly("Session bindings"),
	}, h.bindings)
}

func (h *handlers) eval(ctx context.Context, _ *mcp.CallToolRequest, in EvalInput) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(in.Code) == "" {
		return nil, nil, errors.New("code is empty")
	}
	sessionID, err := h.resolveSession(ctx, in.SessionID)
	if err != nil {
		return nil, nil, err
	}
	resp, err := h.ev.Evaluate(ctx, sessionID, in.Code)
	if err != nil {
		if resp == nil || resp.Cell == nil {
			if isSessionNotFound(err) {
				return nil, nil, h.unknownSessionError(ctx, sessionID)
			}
			return nil, nil, errors.Wrapf(err, "evaluate in session %s", sessionID)
		}
		// A commit failure still carries the executed cell; report both.
		result := EvalResult(resp, h.cfg.limits)
		result.IsError = true
		result.Content = append([]mcp.Content{&mcp.TextContent{Text: "session error: " + err.Error()}}, result.Content...)
		return result, nil, nil
	}
	h.remember(sessionID)
	return EvalResult(resp, h.cfg.limits), nil, nil
}

// EvalResult converts an evaluation response into an MCP tool result: one text
// item (result, console, error, content notes) followed by the attached
// content items. It is exported for servers that add their own eval-like
// tools through WithExtraTools.
func EvalResult(resp *replsession.EvaluateResponse, limits Limits) *mcp.CallToolResult {
	limits = limits.normalize()
	maxChars, maxImages, maxImageBytes := limits.MaxResultChars, limits.MaxImages, limits.MaxImageBytes
	result := &mcp.CallToolResult{Content: []mcp.Content{}}
	if resp == nil || resp.Cell == nil {
		result.IsError = true
		result.Content = append(result.Content, &mcp.TextContent{Text: "no evaluation result"})
		return result
	}
	exec := resp.Cell.Execution
	sections := []string{}
	resultText := exec.Result
	if resultText == "" && exec.Error == "" {
		resultText = "undefined"
	}
	if resultText != "" {
		sections = append(sections, resultText)
	}
	if len(exec.Console) > 0 {
		var console strings.Builder
		console.WriteString("console:")
		for _, event := range exec.Console {
			fmt.Fprintf(&console, "\n[%s] %s", event.Kind, event.Message)
		}
		sections = append(sections, console.String())
	}
	switch {
	case exec.Error != "" && exec.Result != "":
		// A failed cell only has result text when the session's result
		// processor (FailureProcessor) rendered the error itself; the generic
		// line would repeat it. IsError is still set below.
	case exec.Error != "":
		sections = append(sections, fmt.Sprintf("error (%s): %s", exec.Status, exec.Error))
	case exec.Status != "" && exec.Status != "ok":
		sections = append(sections, "status: "+exec.Status)
	}
	if exec.ResultProcessorError != "" {
		sections = append(sections, "result processor error: "+exec.ResultProcessorError)
	}

	items, notes := attachContent(resp, maxImages, maxImageBytes)
	if len(notes) > 0 {
		sections = append(sections, "content:\n"+strings.Join(notes, "\n"))
	}

	body := strings.Join(sections, "\n\n")
	if len(body) > maxChars {
		omitted := len(body) - maxChars
		body = truncateUTF8(body, maxChars) + fmt.Sprintf("\n… [result truncated: %d more characters]", omitted)
	}
	result.Content = append(result.Content, &mcp.TextContent{Text: body})
	result.Content = append(result.Content, items...)
	result.IsError = exec.Error != "" || (exec.Status != "" && exec.Status != "ok")
	return result
}

// attachContent maps rendered result content to MCP content items and
// returns one note per item (attached or dropped) for the text result.
func attachContent(resp *replsession.EvaluateResponse, maxImages, maxImageBytes int) ([]mcp.Content, []string) {
	refs := resp.Cell.Execution.Content
	if len(refs) == 0 && len(resp.Content) == 0 {
		return nil, nil
	}
	items := []mcp.Content{}
	notes := []string{}
	attached := 0
	for _, ref := range refs {
		label := fmt.Sprintf("content %d", ref.Provider)
		if ref.Caption != "" {
			label += " " + strings.TrimSpace(ref.Caption)
		}
		if ref.Index < 0 || ref.Index >= len(resp.Content) {
			reason := ref.Dropped
			if reason == "" {
				reason = "not delivered"
			}
			notes = append(notes, label+": dropped ("+reason+")")
			continue
		}
		item := resp.Content[ref.Index]
		switch {
		case attached >= maxImages:
			notes = append(notes, fmt.Sprintf("%s: dropped (more than %d items)", label, maxImages))
			continue
		case len(item.Data) > maxImageBytes:
			notes = append(notes, fmt.Sprintf("%s: dropped (%d bytes > %d byte limit)", label, len(item.Data), maxImageBytes))
			continue
		}
		attached++
		items = append(items, toMCPContent(item, resp, ref))
		notes = append(notes, fmt.Sprintf("%s: attached (%s, %d bytes)", label, item.MIMEType, len(item.Data)))
	}
	return items, notes
}

func toMCPContent(item resultcontent.Content, resp *replsession.EvaluateResponse, ref resultcontent.Ref) mcp.Content {
	mimeType := strings.TrimSpace(item.MIMEType)
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return &mcp.ImageContent{Data: item.Data, MIMEType: mimeType}
	case strings.HasPrefix(mimeType, "audio/"):
		return &mcp.AudioContent{Data: item.Data, MIMEType: mimeType}
	case strings.HasPrefix(mimeType, "text/") || mimeType == "application/json":
		return &mcp.TextContent{Text: string(item.Data)}
	default:
		sessionID := ""
		if resp.Session != nil {
			sessionID = resp.Session.ID
		}
		return &mcp.EmbeddedResource{Resource: &mcp.ResourceContents{
			URI:      fmt.Sprintf("goja-repl://sessions/%s/cells/%d/content/%d", sessionID, resp.Cell.ID, ref.Provider),
			MIMEType: mimeType,
			Blob:     item.Data,
		}}
	}
}

func (h *handlers) createSession(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, CreateSessionOutput, error) {
	summary, err := h.ev.CreateSession(ctx)
	if err != nil {
		return nil, CreateSessionOutput{}, err
	}
	h.remember(summary.ID)
	return nil, CreateSessionOutput{SessionID: summary.ID}, nil
}

func (h *handlers) listSessions(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, ListSessionsOutput, error) {
	return nil, ListSessionsOutput{Sessions: h.sessionInfos(ctx)}, nil
}

// sessionInfos lists the sessions this server knows (default first) plus the
// durable ones when the evaluator has a store.
func (h *handlers) sessionInfos(ctx context.Context) []SessionInfo {
	out := []SessionInfo{}
	defaultID := h.currentDefault()
	seen := map[string]int{}
	for _, id := range h.knownSessions() {
		seen[id] = len(out)
		out = append(out, SessionInfo{SessionID: id, Default: id == defaultID})
	}
	if records, err := h.ev.ListSessions(ctx); err == nil {
		for _, record := range records {
			createdAt := record.CreatedAt
			if idx, ok := seen[record.SessionID]; ok {
				out[idx].Durable = true
				out[idx].CreatedAt = &createdAt
				continue
			}
			out = append(out, SessionInfo{SessionID: record.SessionID, CreatedAt: &createdAt, Durable: true, Default: record.SessionID == defaultID})
		}
	}
	return out
}

// unknownSessionError tells the model that sessionID does not exist, which
// sessions do, and how to proceed (agents sometimes invent ids).
func (h *handlers) unknownSessionError(ctx context.Context, sessionID string) error {
	infos := h.sessionInfos(ctx)
	var existing string
	switch {
	case len(infos) == 0:
		existing = "No sessions exist yet."
	case len(infos) == 1 && infos[0].Default:
		existing = fmt.Sprintf("There is only the default session (%s).", infos[0].SessionID)
	default:
		ids := make([]string, 0, len(infos))
		for _, info := range infos {
			if info.Default {
				ids = append(ids, info.SessionID+" (default)")
			} else {
				ids = append(ids, info.SessionID)
			}
		}
		existing = "Existing sessions: " + strings.Join(ids, ", ") + "."
	}
	return errors.Errorf("session %q does not exist. %s To use the default session (where the host's globals live), omit sessionId; to start a new one, call create_session. Only pass ids returned by create_session or list_sessions.", sessionID, existing)
}

// isSessionNotFound matches the live (replsession) and durable (repldb)
// not-found errors.
func isSessionNotFound(err error) bool {
	return errors.Is(err, replsession.ErrSessionNotFound) || errors.Is(err, repldb.ErrSessionNotFound)
}

// sessionLookupError maps "session not found" to unknownSessionError.
func (h *handlers) sessionLookupError(ctx context.Context, sessionID string, err error) error {
	if isSessionNotFound(err) {
		return h.unknownSessionError(ctx, sessionID)
	}
	return err
}

func (h *handlers) history(ctx context.Context, _ *mcp.CallToolRequest, in SessionInput) (*mcp.CallToolResult, HistoryOutput, error) {
	sessionID, err := h.resolveSession(ctx, in.SessionID)
	if err != nil {
		return nil, HistoryOutput{}, err
	}
	out := HistoryOutput{SessionID: sessionID, Cells: []HistoryCell{}}
	if records, err := h.ev.History(ctx, sessionID); err == nil && len(records) > 0 {
		for _, record := range records {
			cell := HistoryCell{CellID: record.CellID, Source: record.RawSource, Error: record.ErrorText}
			var report replsession.CellReport
			if json.Unmarshal(record.ResultJSON, &report) == nil {
				cell.Status = report.Execution.Status
				cell.Result = report.Execution.Result
			}
			out.Cells = append(out.Cells, cell)
		}
		return nil, out, nil
	}
	summary, err := h.ev.Snapshot(ctx, sessionID)
	if err != nil {
		return nil, HistoryOutput{}, h.sessionLookupError(ctx, sessionID, err)
	}
	out.Previews = true
	for _, entry := range summary.History {
		out.Cells = append(out.Cells, HistoryCell{CellID: entry.CellID, Status: entry.Status, Source: entry.SourcePreview, Result: entry.ResultPreview})
	}
	return nil, out, nil
}

func (h *handlers) bindings(ctx context.Context, _ *mcp.CallToolRequest, in SessionInput) (*mcp.CallToolResult, BindingsOutput, error) {
	sessionID, err := h.resolveSession(ctx, in.SessionID)
	if err != nil {
		return nil, BindingsOutput{}, err
	}
	views, err := h.ev.Bindings(ctx, sessionID)
	if err != nil {
		return nil, BindingsOutput{}, h.sessionLookupError(ctx, sessionID, err)
	}
	out := BindingsOutput{SessionID: sessionID, Bindings: make([]BindingInfo, 0, len(views))}
	for _, view := range views {
		out.Bindings = append(out.Bindings, BindingInfo{Name: view.Name, Kind: view.Kind, Type: view.Runtime.ValueKind, Preview: view.Runtime.Preview})
	}
	return nil, out, nil
}

func truncateUTF8(s string, n int) string {
	if n >= len(s) {
		return s
	}
	for n > 0 && (s[n]&0xC0) == 0x80 {
		n--
	}
	return s[:n]
}
