// Package audit provides small reusable audit sinks and record normalization for
// gojahttp planned-route audit events.
package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	stdlog "log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
)

// Record is a storage-friendly audit event shape. It deliberately excludes raw
// secrets such as cookies, Authorization headers, session IDs, and capability
// tokens.
type Record struct {
	Event        string         `json:"event"`
	Outcome      string         `json:"outcome"`
	Reason       string         `json:"reason,omitempty"`
	StatusCode   int            `json:"statusCode,omitempty"`
	RouteName    string         `json:"routeName,omitempty"`
	Method       string         `json:"method"`
	Pattern      string         `json:"pattern"`
	Action       string         `json:"action,omitempty"`
	ActorID      string         `json:"actorId,omitempty"`
	ActorKind    string         `json:"actorKind,omitempty"`
	TenantID     string         `json:"tenantId,omitempty"`
	ResourceType string         `json:"resourceType,omitempty"`
	ResourceID   string         `json:"resourceId,omitempty"`
	RequestID    string         `json:"requestId,omitempty"`
	IPHash       string         `json:"ipHash,omitempty"`
	UserAgent    string         `json:"userAgent,omitempty"`
	Attributes   map[string]any `json:"attributes,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
}

// Store persists normalized audit records.
type Store interface {
	InsertAuditRecord(ctx context.Context, record Record) error
}

// Normalizer maps gojahttp.AuditEvent values into Records.
type Normalizer struct {
	Now    func() time.Time
	IPHash func(ip string) string
}

func (n Normalizer) Normalize(event gojahttp.AuditEvent) Record {
	now := time.Now
	if n.Now != nil {
		now = n.Now
	}
	ipHasher := hashIP
	if n.IPHash != nil {
		ipHasher = n.IPHash
	}
	record := Record{
		Event:      event.Event,
		Outcome:    event.Outcome,
		Reason:     event.Reason,
		StatusCode: event.StatusCode,
		RouteName:  event.RouteName,
		Method:     event.Method,
		Pattern:    event.Pattern,
		Action:     event.Action,
		Attributes: RedactMap(event.Attributes),
		CreatedAt:  now(),
	}
	if event.Actor != nil {
		record.ActorID = event.Actor.ID
		record.ActorKind = event.Actor.Kind
	}
	if event.Resource != nil {
		record.ResourceType = event.Resource.Type
		record.ResourceID = event.Resource.ID
		record.TenantID = event.Resource.TenantID
	}
	if record.TenantID == "" {
		for _, resource := range event.Resources {
			if resource != nil && resource.TenantID != "" {
				record.TenantID = resource.TenantID
				break
			}
		}
	}
	if event.HTTPRequest != nil {
		record.RequestID = firstHeader(event.HTTPRequest, "X-Request-Id", "X-Correlation-Id")
		record.UserAgent = event.HTTPRequest.UserAgent()
		if ip := clientIP(event.HTTPRequest); ip != "" {
			record.IPHash = ipHasher(ip)
		}
	}
	return record
}

// Sink records audit events into a Store after normalization.
type Sink struct {
	Store      Store
	Normalizer Normalizer
}

func (s Sink) RecordAudit(ctx context.Context, event gojahttp.AuditEvent) error {
	return s.Store.InsertAuditRecord(ctx, s.Normalizer.Normalize(event))
}

// MemorySink stores normalized audit records in memory for tests and demos.
type MemorySink struct {
	mu         sync.Mutex
	Normalizer Normalizer
	Records    []Record
}

func (s *MemorySink) RecordAudit(_ context.Context, event gojahttp.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Records = append(s.Records, s.Normalizer.Normalize(event))
	return nil
}

func (s *MemorySink) Snapshot() []Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Record, len(s.Records))
	copy(out, s.Records)
	return out
}

// MemoryStore stores normalized audit records in memory for tests and demos.
type MemoryStore struct {
	mu      sync.Mutex
	Records []Record
}

func (s *MemoryStore) InsertAuditRecord(_ context.Context, record Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Records = append(s.Records, cloneRecord(record))
	return nil
}

func (s *MemoryStore) Snapshot() []Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Record, len(s.Records))
	for i, record := range s.Records {
		out[i] = cloneRecord(record)
	}
	return out
}

// LogSink logs normalized audit records as JSON for development.
type LogSink struct {
	Logger     *stdlog.Logger
	Normalizer Normalizer
}

func (s LogSink) RecordAudit(_ context.Context, event gojahttp.AuditEvent) error {
	logger := s.Logger
	if logger == nil {
		logger = stdlog.Default()
	}
	data, err := json.Marshal(s.Normalizer.Normalize(event))
	if err != nil {
		return err
	}
	logger.Print(string(data))
	return nil
}

// RedactMap returns a copy with secret-looking keys replaced by "[REDACTED]".
func RedactMap(attrs map[string]any) map[string]any {
	if len(attrs) == 0 {
		return nil
	}
	out := make(map[string]any, len(attrs))
	for key, value := range attrs {
		if secretKey(key) {
			out[key] = "[REDACTED]"
			continue
		}
		out[key] = redactValue(value)
	}
	return out
}

func redactValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return RedactMap(v)
	case map[string]string:
		out := map[string]any{}
		for key, value := range v {
			if secretKey(key) {
				out[key] = "[REDACTED]"
			} else {
				out[key] = value
			}
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = redactValue(item)
		}
		return out
	default:
		return value
	}
}

func secretKey(key string) bool {
	key = strings.ToLower(key)
	for _, fragment := range []string{"token", "secret", "password", "cookie", "session", "authorization", "credential", "code", "capability"} {
		if strings.Contains(key, fragment) {
			return true
		}
	}
	return false
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func hashIP(ip string) string {
	sum := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(sum[:])
}

func firstHeader(r *http.Request, names ...string) string {
	for _, name := range names {
		if value := r.Header.Get(name); value != "" {
			return value
		}
	}
	return ""
}

func cloneRecord(record Record) Record {
	out := record
	out.Attributes = cloneAnyMap(record.Attributes)
	return out
}

func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneAny(value)
	}
	return out
}

func cloneAny(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return cloneAnyMap(v)
	case map[string]string:
		out := make(map[string]string, len(v))
		for key, value := range v {
			out[key] = value
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = cloneAny(item)
		}
		return out
	default:
		return value
	}
}
