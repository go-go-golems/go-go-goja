package databasemod

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	_ "github.com/mattn/go-sqlite3" // Driver for sqlite3
	"github.com/rs/zerolog/log"
)

type QueryExecer interface {
	Query(query string, args ...any) (*sql.Rows, error)
	Exec(query string, args ...any) (sql.Result, error)
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
		Name: "database",
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
	modules.SetExport(exports, m.Name(), "query", m.Query)
	modules.SetExport(exports, m.Name(), "exec", m.Exec)
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
	if m == nil || m.queryExecer == nil {
		return nil, fmt.Errorf("database not configured, call require('%s').configure(...) first", m.Name())
	}

	startTime := time.Now()
	log.Debug().Str("module", m.Name()).Str("query", query).Msg("database: executing query")

	rows, err := m.queryExecer.Query(query, flattenArgs(args)...)
	if err != nil {
		log.Error().Str("module", m.Name()).Str("query", query).Err(err).Msg("database: query error")
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Error().Str("module", m.Name()).Err(err).Msg("database: failed to close rows")
		}
	}()

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
			log.Error().Str("module", m.Name()).Err(err).Msg("database: row scan error")
			continue
		}

		rec := make(map[string]any)
		for i, col := range cols {
			rec[col] = vals[i]
		}
		result = append(result, rec)
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
	if m == nil || m.queryExecer == nil {
		return nil, fmt.Errorf("database not configured, call require('%s').configure(...) first", m.Name())
	}

	startTime := time.Now()
	log.Debug().Str("module", m.Name()).Str("query", query).Msg("database: executing exec")

	result, err := m.queryExecer.Exec(query, flattenArgs(args)...)
	if err != nil {
		log.Error().Str("module", m.Name()).Str("query", query).Err(err).Msg("database: exec error")
		return map[string]any{
			"error":   err.Error(),
			"success": false,
		}, err
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	duration := time.Since(startTime)
	log.Debug().
		Str("module", m.Name()).
		Dur("duration", duration).
		Int64("rowsAffected", rowsAffected).
		Msg("database: exec completed")

	return map[string]any{
		"success":      true,
		"rowsAffected": rowsAffected,
		"lastInsertId": lastInsertID,
	}, nil
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
}
