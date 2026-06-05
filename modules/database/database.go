package databasemod

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	_ "github.com/mattn/go-sqlite3" // Driver for sqlite3
)

type QueryExecer interface {
	Query(query string, args ...any) (*sql.Rows, error)
	Exec(query string, args ...any) (sql.Result, error)
}

type QueryExecerContext interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Transaction is the minimal SQL transaction surface exposed by the database
// module. It matches *sql.Tx while still allowing guarded wrappers to enforce
// host policy inside transaction exec/query calls.
type Transaction interface {
	Query(query string, args ...any) (*sql.Rows, error)
	Exec(query string, args ...any) (sql.Result, error)
	Commit() error
	Rollback() error
}

// TransactionContext is the context-aware transaction surface. *sql.Tx
// implements this interface, as can wrapper transactions.
type TransactionContext interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Commit() error
	Rollback() error
}

// TransactionBeginner is implemented by database wrappers that can return an
// abstract transaction handle. Use this when the wrapper must preserve policy,
// for example read-only guards, inside the transaction.
type TransactionBeginner interface {
	BeginTransaction() (Transaction, error)
}

// TransactionBeginnerContext is the context-aware variant of
// TransactionBeginner. It is preferred when available.
type TransactionBeginnerContext interface {
	BeginTransactionContext(ctx context.Context, opts *sql.TxOptions) (Transaction, error)
}

type sqlTransactionBeginner interface {
	Begin() (*sql.Tx, error)
}

type sqlTransactionBeginnerContext interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type Option func(*DBModule)

func WithName(name string) Option {
	return func(m *DBModule) {
		if m == nil || name == "" {
			return
		}
		m.name = name
	}
}

func WithPreconfiguredDB(db QueryExecer) Option {
	return func(m *DBModule) {
		if m == nil || db == nil {
			return
		}
		m.queryExecer = db
		m.allowConfigure = false
	}
}

func WithCloseFn(closeFn func() error) Option {
	return func(m *DBModule) {
		if m == nil {
			return
		}
		m.closeFn = closeFn
	}
}

func WithConfigureEnabled(enabled bool) Option {
	return func(m *DBModule) {
		if m == nil {
			return
		}
		m.allowConfigure = enabled
	}
}

// DBModule provides a database connection for a goja runtime.
type DBModule struct {
	name           string
	queryExecer    QueryExecer
	closeFn        func() error
	allowConfigure bool
}

var _ modules.NativeModule = (*DBModule)(nil)
var _ modules.TypeScriptDeclarer = (*DBModule)(nil)

func New(options ...Option) *DBModule {
	ret := &DBModule{
		name:           "database",
		allowConfigure: true,
	}
	for _, option := range options {
		if option != nil {
			option(ret)
		}
	}
	return ret
}

// Name returns the module name.
func (m *DBModule) Name() string {
	if m == nil || m.name == "" {
		return "database"
	}
	return m.name
}

func (m *DBModule) TypeScriptModule() *spec.Module {
	return &spec.Module{
		Name: m.Name(),
		RawDTS: []string{
			"interface DatabaseExecResult {",
			"  success: boolean;",
			"  rowsAffected?: number;",
			"  lastInsertId?: number;",
			"  error?: string;",
			"}",
			"interface DatabaseTransaction {",
			"  query(query: string, ...args: unknown[]): Array<Record<string, unknown>>;",
			"  exec(query: string, ...args: unknown[]): DatabaseExecResult;",
			"  commit(): { success: boolean; error?: string };",
			"  rollback(): { success: boolean; error?: string };",
			"}",
		},
		Functions: []spec.Function{
			{
				Name: "configure",
				Params: []spec.Param{
					{Name: "driverName", Type: spec.String()},
					{Name: "dataSourceName", Type: spec.String()},
				},
				Returns: spec.Void(),
			},
			{
				Name: "query",
				Params: []spec.Param{
					{Name: "query", Type: spec.String()},
					{Name: "args", Type: spec.Unknown(), Variadic: true},
				},
				Returns: spec.Unknown(),
			},
			{
				Name: "exec",
				Params: []spec.Param{
					{Name: "query", Type: spec.String()},
					{Name: "args", Type: spec.Unknown(), Variadic: true},
				},
				Returns: spec.Unknown(),
			},
			{
				Name:    "begin",
				Returns: spec.Named("DatabaseTransaction"),
			},
			{
				Name:    "close",
				Returns: spec.Void(),
			},
		},
	}
}

// Doc returns the documentation for the module.
func (m *DBModule) Doc() string {
	doc := `
Database module provides a simple SQL interface.

Functions:
  query(sql, ...args): Executes a query and returns rows.
    Example: require('database').query('SELECT * FROM users WHERE id = ?', 1);
  exec(sql, ...args): Executes a statement and returns result summary.
    Example: require('database').exec('INSERT INTO users (name) VALUES (?)', 'John');
  begin(): Starts a transaction. The returned object has query, exec, commit, and rollback.
    Example: const tx = require('database').begin(); tx.exec('INSERT INTO users(name) VALUES (?)', 'Ada'); tx.commit();
  close(): Closes the database connection if the module owns it.
`
	if m.allowConfigure {
		doc += `
  configure(driverName, dataSourceName): Configures the database connection.
    Example: require('database').configure('sqlite3', ':memory:');
`
	} else {
		doc += `
This module is preconfigured by Go and does not allow configure().
`
	}

	return doc
}

// Loader exposes the database functions to the JavaScript module.
func (m *DBModule) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	exports := moduleObj.Get("exports").(*goja.Object)
	modules.SetExport(exports, m.Name(), "configure", m.Configure)
	modules.SetExport(exports, m.Name(), "query", func(query string, args ...any) ([]map[string]any, error) {
		return m.QueryContext(runtimebridge.CurrentOwnerContext(vm), query, args...)
	})
	modules.SetExport(exports, m.Name(), "exec", func(query string, args ...any) (map[string]any, error) {
		return m.ExecContext(runtimebridge.CurrentOwnerContext(vm), query, args...)
	})
	modules.SetExport(exports, m.Name(), "begin", func() (*goja.Object, error) {
		tx, err := m.BeginContext(runtimebridge.CurrentOwnerContext(vm))
		if err != nil {
			return nil, err
		}
		return tx.ToObject(vm), nil
	})
	modules.SetExport(exports, m.Name(), "close", m.Close)
}

// Configure sets up the database connection.
func (m *DBModule) Configure(driverName, dataSourceName string) error {
	if !m.allowConfigure {
		return fmt.Errorf("database module %q is preconfigured and does not allow configure()", m.Name())
	}
	if err := m.closeOwnedConnection(); err != nil {
		log.Error().Err(err).Msg("database: failed to close existing connection")
	}

	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		log.Error().Str("dsn", dataSourceName).Err(err).Msg("database: failed to open connection")
		return err
	}
	m.queryExecer = db
	m.closeFn = db.Close
	log.Debug().Str("driver", driverName).Str("module", m.Name()).Msg("database: configured")
	return nil
}

// Close closes the database connection if this module owns it.
func (m *DBModule) Close() error {
	if m == nil || m.closeFn == nil {
		return nil
	}
	if err := m.closeFn(); err != nil {
		return err
	}
	m.closeFn = nil
	m.queryExecer = nil
	return nil
}

// Query executes a SQL query and returns results as JavaScript objects.
func (m *DBModule) Query(query string, args ...any) ([]map[string]any, error) {
	return m.QueryContext(context.Background(), query, args...)
}

// QueryContext executes a SQL query with ctx and returns results as JavaScript objects.
func (m *DBModule) QueryContext(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	if m == nil || m.queryExecer == nil {
		return nil, fmt.Errorf("database not configured, call require('%s').configure(...) first", m.Name())
	}
	if ctx == nil {
		ctx = context.Background()
	}

	startTime := time.Now()
	log.Debug().Str("module", m.Name()).Str("query", query).Msg("database: executing query")

	rows, err := queryRows(ctx, m.queryExecer, query, flattenArgs(args)...)
	if err != nil {
		log.Error().Str("module", m.Name()).Str("query", query).Err(err).Msg("database: query error")
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Error().Str("module", m.Name()).Err(err).Msg("database: failed to close rows")
		}
	}()

	result, err := rowsToRecords(m.Name(), rows)
	if err != nil {
		return nil, err
	}

	duration := time.Since(startTime)
	log.Debug().
		Str("module", m.Name()).
		Dur("duration", duration).
		Int("rows", len(result)).
		Msg("database: query completed")

	return result, nil
}

// Exec executes a SQL statement without returning rows.
func (m *DBModule) Exec(query string, args ...any) (map[string]any, error) {
	return m.ExecContext(context.Background(), query, args...)
}

// ExecContext executes a SQL statement with ctx without returning rows.
func (m *DBModule) ExecContext(ctx context.Context, query string, args ...any) (map[string]any, error) {
	if m == nil || m.queryExecer == nil {
		return nil, fmt.Errorf("database not configured, call require('%s').configure(...) first", m.Name())
	}
	if ctx == nil {
		ctx = context.Background()
	}

	startTime := time.Now()
	log.Debug().Str("module", m.Name()).Str("query", query).Msg("database: executing exec")

	result, err := execResult(ctx, m.queryExecer, query, flattenArgs(args)...)
	if err != nil {
		log.Error().Str("module", m.Name()).Str("query", query).Err(err).Msg("database: exec error")
		return map[string]any{
			"error":   err.Error(),
			"success": false,
		}, err
	}

	resultMap := resultToMap(result)
	rowsAffected, _ := result.RowsAffected()

	duration := time.Since(startTime)
	log.Debug().
		Str("module", m.Name()).
		Dur("duration", duration).
		Int64("rowsAffected", rowsAffected).
		Msg("database: exec completed")

	return resultMap, nil
}

// Begin starts a transaction using context.Background().
func (m *DBModule) Begin() (*TransactionHandle, error) {
	return m.BeginContext(context.Background())
}

// BeginContext starts a transaction on the configured database.
func (m *DBModule) BeginContext(ctx context.Context) (*TransactionHandle, error) {
	if m == nil || m.queryExecer == nil {
		return nil, fmt.Errorf("database not configured, call require('%s').configure(...) first", m.Name())
	}
	if ctx == nil {
		ctx = context.Background()
	}

	tx, err := beginTransaction(ctx, m.queryExecer)
	if err != nil {
		return nil, err
	}
	return &TransactionHandle{moduleName: m.Name(), tx: tx}, nil
}

// TransactionHandle is the JavaScript-facing database transaction handle.
type TransactionHandle struct {
	moduleName string
	tx         Transaction
	closed     bool
	mu         sync.Mutex
}

// ToObject creates the JavaScript transaction object exported by begin().
func (h *TransactionHandle) ToObject(vm *goja.Runtime) *goja.Object {
	obj := vm.NewObject()
	modules.SetExport(obj, h.moduleName, "query", func(query string, args ...any) ([]map[string]any, error) {
		return h.QueryContext(runtimebridge.CurrentOwnerContext(vm), query, args...)
	})
	modules.SetExport(obj, h.moduleName, "exec", func(query string, args ...any) (map[string]any, error) {
		return h.ExecContext(runtimebridge.CurrentOwnerContext(vm), query, args...)
	})
	modules.SetExport(obj, h.moduleName, "commit", h.Commit)
	modules.SetExport(obj, h.moduleName, "rollback", h.Rollback)
	return obj
}

// QueryContext executes a query inside the transaction.
func (h *TransactionHandle) QueryContext(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed || h.tx == nil {
		return nil, fmt.Errorf("database transaction is closed")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	rows, err := queryRows(ctx, h.tx, query, flattenArgs(args)...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Error().Str("module", h.moduleName).Err(err).Msg("database: failed to close transaction rows")
		}
	}()
	return rowsToRecords(h.moduleName, rows)
}

// ExecContext executes a statement inside the transaction.
func (h *TransactionHandle) ExecContext(ctx context.Context, query string, args ...any) (map[string]any, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed || h.tx == nil {
		return nil, fmt.Errorf("database transaction is closed")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := execResult(ctx, h.tx, query, flattenArgs(args)...)
	if err != nil {
		return map[string]any{"error": err.Error(), "success": false}, err
	}
	return resultToMap(result), nil
}

// Commit commits the transaction and closes the handle.
func (h *TransactionHandle) Commit() (map[string]any, error) {
	return h.closeWith("commit", func(tx Transaction) error { return tx.Commit() })
}

// Rollback rolls the transaction back and closes the handle.
func (h *TransactionHandle) Rollback() (map[string]any, error) {
	return h.closeWith("rollback", func(tx Transaction) error { return tx.Rollback() })
}

func (h *TransactionHandle) closeWith(action string, fn func(Transaction) error) (map[string]any, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed || h.tx == nil {
		err := fmt.Errorf("database transaction is closed")
		return map[string]any{"success": false, "error": err.Error()}, err
	}
	tx := h.tx
	err := fn(tx)
	h.closed = true
	h.tx = nil
	if err != nil {
		return map[string]any{"success": false, "error": err.Error()}, fmt.Errorf("transaction %s: %w", action, err)
	}
	return map[string]any{"success": true}, nil
}

func (m *DBModule) closeOwnedConnection() error {
	if m == nil || m.closeFn == nil {
		return nil
	}
	if err := m.closeFn(); err != nil {
		return err
	}
	m.closeFn = nil
	m.queryExecer = nil
	return nil
}

func beginTransaction(ctx context.Context, qe QueryExecer) (Transaction, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if beginner, ok := qe.(TransactionBeginnerContext); ok {
		return beginner.BeginTransactionContext(ctx, nil)
	}
	if beginner, ok := qe.(sqlTransactionBeginnerContext); ok {
		return beginner.BeginTx(ctx, nil)
	}
	if beginner, ok := qe.(TransactionBeginner); ok {
		return beginner.BeginTransaction()
	}
	if beginner, ok := qe.(sqlTransactionBeginner); ok {
		return beginner.Begin()
	}
	return nil, fmt.Errorf("database %T does not support transactions", qe)
}

func rowsToRecords(moduleName string, rows *sql.Rows) ([]map[string]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		scan := make([]any, len(cols))
		for i := range vals {
			scan[i] = &vals[i]
		}

		if err := rows.Scan(scan...); err != nil {
			log.Error().Str("module", moduleName).Err(err).Msg("database: row scan error")
			continue
		}

		rec := make(map[string]any)
		for i, col := range cols {
			rec[col] = vals[i]
		}
		result = append(result, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func resultToMap(result sql.Result) map[string]any {
	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()
	return map[string]any{
		"success":      true,
		"rowsAffected": rowsAffected,
		"lastInsertId": lastInsertID,
	}
}

func queryRows(ctx context.Context, qe QueryExecer, query string, args ...any) (*sql.Rows, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if qec, ok := qe.(QueryExecerContext); ok {
		return qec.QueryContext(ctx, query, args...)
	}
	return qe.Query(query, args...)
}

func execResult(ctx context.Context, qe QueryExecer, query string, args ...any) (sql.Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if qec, ok := qe.(QueryExecerContext); ok {
		return qec.ExecContext(ctx, query, args...)
	}
	return qe.Exec(query, args...)
}

func flattenArgs(args []any) []any {
	var flatArgs []any
	for _, arg := range args {
		if slice, ok := arg.([]any); ok {
			flatArgs = append(flatArgs, slice...)
		} else {
			flatArgs = append(flatArgs, arg)
		}
	}
	return flatArgs
}

func init() {
	modules.Register(New())
	modules.Register(New(WithName("db")))
}
