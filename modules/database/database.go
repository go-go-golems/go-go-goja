package databasemod

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
	_ "github.com/mattn/go-sqlite3" // Driver for sqlite3
	"github.com/rs/zerolog/log"
)

// DBModule provides a database connection for a goja runtime.
type DBModule struct {
	db *sql.DB
}

var _ modules.NativeModule = (*DBModule)(nil)

// Name returns the module name.
func (m *DBModule) Name() string { return "database" }

// Doc returns the documentation for the module.
func (m *DBModule) Doc() string {
	return `
Database module provides a simple SQL interface.

Functions:
  configure(driverName, dataSourceName): Configures the database connection.
    Example: require('database').configure('sqlite3', ':memory:');
  query(sql, ...args): Executes a query and returns rows.
    Example: require('database').query('SELECT * FROM users WHERE id = ?', 1);
  exec(sql, ...args): Executes a statement and returns result summary.
    Example: require('database').exec('INSERT INTO users (name) VALUES (?)', 'John');
  close(): Closes the database connection.
`
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
	if m.db != nil {
		if err := m.db.Close(); err != nil {
			log.Error().Err(err).Msg("database: failed to close existing connection")
		}
	}

	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		log.Error().Str("dsn", dataSourceName).Err(err).Msg("database: failed to open connection")
		return err
	}
	m.db = db
	log.Debug().Str("driver", driverName).Msg("database: configured")
	return nil
}

// Close closes the database connection.
func (m *DBModule) Close() error {
	if m.db != nil {
		log.Debug().Msg("database: closing connection")
		return m.db.Close()
	}
	return nil
}

// Query executes a SQL query and returns results as JavaScript objects.
func (m *DBModule) Query(query string, args ...interface{}) ([]map[string]interface{}, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not configured, call require('database').configure(...) first")
	}

	startTime := time.Now()
	log.Debug().Str("query", query).Msg("database: executing query")

	var flatArgs []interface{}
	for _, arg := range args {
		if slice, ok := arg.([]interface{}); ok {
			flatArgs = append(flatArgs, slice...)
		} else {
			flatArgs = append(flatArgs, arg)
		}
	}

	rows, err := m.db.Query(query, flatArgs...)
	if err != nil {
		log.Error().Str("query", query).Err(err).Msg("database: query error")
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Error().Err(err).Msg("database: failed to close rows")
		}
	}()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		scan := make([]interface{}, len(cols))
		for i := range vals {
			scan[i] = &vals[i]
		}

		if err := rows.Scan(scan...); err != nil {
			log.Error().Err(err).Msg("database: row scan error")
			continue
		}

		rec := make(map[string]interface{})
		for i, col := range cols {
			rec[col] = vals[i]
		}
		result = append(result, rec)
	}

	duration := time.Since(startTime)
	log.Debug().
		Dur("duration", duration).
		Int("rows", len(result)).
		Msg("database: query completed")

	return result, nil
}

// Exec executes a SQL statement without returning rows.
func (m *DBModule) Exec(query string, args ...interface{}) (map[string]interface{}, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not configured, call require('database').configure(...) first")
	}

	startTime := time.Now()
	log.Debug().Str("query", query).Msg("database: executing exec")

	var flatArgs []interface{}
	for _, arg := range args {
		if slice, ok := arg.([]interface{}); ok {
			flatArgs = append(flatArgs, slice...)
		} else {
			flatArgs = append(flatArgs, arg)
		}
	}

	result, err := m.db.Exec(query, flatArgs...)
	if err != nil {
		log.Error().Str("query", query).Err(err).Msg("database: exec error")
		return map[string]interface{}{
			"error":   err.Error(),
			"success": false,
		}, err
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertId, _ := result.LastInsertId()

	duration := time.Since(startTime)
	log.Debug().
		Dur("duration", duration).
		Int64("rowsAffected", rowsAffected).
		Msg("database: exec completed")

	return map[string]interface{}{
		"success":      true,
		"rowsAffected": rowsAffected,
		"lastInsertId": lastInsertId,
	}, nil
}

func init() {
	modules.Register(&DBModule{})
}
