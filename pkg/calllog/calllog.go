package calllog

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dop251/goja"
	_ "github.com/mattn/go-sqlite3"
)

const (
	envCallLogDB      = "GOJA_CALLLOG_DB"
	envCallLogDisable = "GOJA_CALLLOG_DISABLE"
	defaultCallLogDB  = "goja-calllog.sqlite"
)

// Entry represents a single JS/Go call bridge log entry.
type Entry struct {
	Direction  string
	Module     string
	Function   string
	ArgsJSON   string
	ResultJSON string
	Duration   time.Duration
	Error      string
}

// Logger writes call entries to a sqlite database.
type Logger struct {
	mu      sync.Mutex
	db      *sql.DB
	insert  *sql.Stmt
	enabled bool
}

var (
	defaultOnce     sync.Once
	defaultLogger   = &Logger{}
	defaultLoggerMu sync.RWMutex
)

// ConfigureFromEnv initializes the default logger using environment variables.
//
// - GOJA_CALLLOG_DB sets the sqlite path (default: goja-calllog.sqlite)
// - GOJA_CALLLOG_DISABLE disables logging when set to a non-empty value.
func ConfigureFromEnv() error {
	var err error
	defaultOnce.Do(func() {
		if os.Getenv(envCallLogDisable) != "" {
			return
		}
		path := os.Getenv(envCallLogDB)
		if path == "" {
			path = defaultCallLogDB
		}
		err = Configure(path)
	})
	return err
}

// DefaultPath returns the default sqlite path for call logging.
func DefaultPath() string {
	return defaultCallLogDB
}

// Configure replaces the default logger with one using the provided path.
func Configure(path string) error {
	logger, err := New(path)
	if err != nil {
		return err
	}

	defaultLoggerMu.Lock()
	old := defaultLogger
	defaultLogger = logger
	defaultLoggerMu.Unlock()

	if old != nil {
		_ = old.Close()
	}

	return nil
}

// Disable stops logging and closes the default logger.
func Disable() {
	defaultLoggerMu.Lock()
	old := defaultLogger
	defaultLogger = nil
	defaultLoggerMu.Unlock()

	if old != nil {
		_ = old.Close()
	}
}

// New creates a new Logger that writes to the given sqlite path.
func New(path string) (*Logger, error) {
	if path == "" {
		return nil, errors.New("calllog: path is required")
	}

	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("calllog: create dir %s: %w", dir, err)
		}
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("calllog: open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		return nil, fmt.Errorf("calllog: enable WAL: %w", err)
	}
	if _, err := db.Exec(`PRAGMA synchronous=NORMAL;`); err != nil {
		return nil, fmt.Errorf("calllog: set synchronous: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout=5000;`); err != nil {
		return nil, fmt.Errorf("calllog: set busy_timeout: %w", err)
	}

	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS call_log (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	ts_unix_ms INTEGER NOT NULL,
	direction TEXT NOT NULL,
	module TEXT,
	function TEXT NOT NULL,
	args_json TEXT,
	result_json TEXT,
	duration_ms INTEGER,
	error TEXT
);
`); err != nil {
		return nil, fmt.Errorf("calllog: create table: %w", err)
	}

	if err := ensureColumn(db, "call_log", "result_json", "TEXT"); err != nil {
		return nil, fmt.Errorf("calllog: ensure result_json column: %w", err)
	}

	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_call_log_ts ON call_log(ts_unix_ms);`); err != nil {
		return nil, fmt.Errorf("calllog: create index ts: %w", err)
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_call_log_direction ON call_log(direction);`); err != nil {
		return nil, fmt.Errorf("calllog: create index direction: %w", err)
	}

	insert, err := db.Prepare(`
INSERT INTO call_log (ts_unix_ms, direction, module, function, args_json, result_json, duration_ms, error)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);
`)
	if err != nil {
		return nil, fmt.Errorf("calllog: prepare insert: %w", err)
	}

	return &Logger{
		db:      db,
		insert:  insert,
		enabled: true,
	}, nil
}

// Close shuts down the logger and releases sqlite resources.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = false
	if l.insert != nil {
		_ = l.insert.Close()
		l.insert = nil
	}
	if l.db != nil {
		err := l.db.Close()
		l.db = nil
		return err
	}
	return nil
}

// Log writes an entry using the default logger.
func Log(entry Entry) {
	defaultLoggerMu.RLock()
	logger := defaultLogger
	defaultLoggerMu.RUnlock()
	if logger == nil {
		return
	}
	logger.Log(entry)
}

// Log writes a single entry to sqlite.
func (l *Logger) Log(entry Entry) {
	if l == nil || !l.enabled {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.insert == nil {
		return
	}

	ts := time.Now().UTC().UnixMilli()
	direction := entry.Direction
	if direction == "" {
		direction = "unknown"
	}
	fn := entry.Function
	if fn == "" {
		fn = "unknown"
	}

	_, err := l.insert.Exec(
		ts,
		direction,
		nullable(entry.Module),
		fn,
		nullable(entry.ArgsJSON),
		nullable(entry.ResultJSON),
		entry.Duration.Milliseconds(),
		nullable(entry.Error),
	)
	if err != nil {
		log.Printf("calllog: insert failed: %v", err)
	}
}

func nullable(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func ensureColumn(db *sql.DB, table, column, columnType string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s);", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", table, column, columnType))
	return err
}

// ArgsToJSON converts goja arguments into a JSON array string for logging.
func ArgsToJSON(args []goja.Value) string {
	if len(args) == 0 {
		return "[]"
	}

	exported := make([]interface{}, len(args))
	for i, arg := range args {
		exported[i] = exportValue(arg)
	}

	if payload, err := json.Marshal(exported); err == nil {
		return string(payload)
	}

	fallback := make([]string, len(args))
	for i, arg := range args {
		fallback[i] = arg.String()
	}
	payload, err := json.Marshal(fallback)
	if err != nil {
		return "[]"
	}
	return string(payload)
}

// ValueToJSON converts a goja.Value into a JSON string.
func ValueToJSON(val goja.Value) string {
	payload, err := json.Marshal(exportValue(val))
	if err == nil {
		return string(payload)
	}
	payload, err = json.Marshal(fmt.Sprintf("%v", val))
	if err != nil {
		return "null"
	}
	return string(payload)
}

func exportValue(val goja.Value) interface{} {
	switch {
	case val == nil || goja.IsUndefined(val) || goja.IsNull(val):
		return nil
	case isFunctionValue(val):
		return "<function>"
	}

	exported := val.Export()
	switch v := exported.(type) {
	case error:
		return v.Error()
	case fmt.Stringer:
		return v.String()
	default:
		return exported
	}
}

func isFunctionValue(val goja.Value) bool {
	_, ok := goja.AssertFunction(val)
	return ok
}

// WrapGoFunction returns a goja-compatible function that logs JS->Go calls.
func WrapGoFunction(vm *goja.Runtime, moduleName, funcName string, fn interface{}) func(goja.FunctionCall) goja.Value {
	original := vm.ToValue(fn)
	callable, ok := goja.AssertFunction(original)
	if !ok {
		return func(call goja.FunctionCall) goja.Value {
			err := fmt.Errorf("calllog: %s.%s is not callable", moduleName, funcName)
			Log(Entry{
				Direction: "js->go",
				Module:    moduleName,
				Function:  funcName,
				ArgsJSON:  ArgsToJSON(call.Arguments),
				Error:     err.Error(),
			})
			panic(vm.NewGoError(err))
			return goja.Undefined()
		}
	}

	return func(call goja.FunctionCall) goja.Value {
		start := time.Now()
		result, err := callable(call.This, call.Arguments...)
		duration := time.Since(start)

		entry := Entry{
			Direction:  "js->go",
			Module:     moduleName,
			Function:   funcName,
			ArgsJSON:   ArgsToJSON(call.Arguments),
			ResultJSON: ValueToJSON(result),
			Duration:   duration,
		}
		if err != nil {
			entry.Error = err.Error()
		}
		Log(entry)

		if err != nil {
			panic(vm.NewGoError(err))
			return goja.Undefined()
		}

		return result
	}
}

// CallJSFunction logs and invokes a JS callable from Go (Go->JS direction).
func CallJSFunction(vm *goja.Runtime, moduleName, funcName string, fn goja.Callable, this goja.Value, args ...goja.Value) (goja.Value, error) {
	start := time.Now()
	result, err := fn(this, args...)
	duration := time.Since(start)

	entry := Entry{
		Direction:  "go->js",
		Module:     moduleName,
		Function:   funcName,
		ArgsJSON:   ArgsToJSON(args),
		ResultJSON: ValueToJSON(result),
		Duration:   duration,
	}
	if err != nil {
		entry.Error = err.Error()
	}
	Log(entry)

	return result, err
}
