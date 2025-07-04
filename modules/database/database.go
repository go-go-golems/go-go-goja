package databasemod

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
	_ "github.com/mattn/go-sqlite3" // Driver for sqlite3
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
	if err := exports.Set("configure", m.Configure); err != nil {
		log.Printf("database: failed to set configure function: %v", err)
	}
	if err := exports.Set("query", m.Query); err != nil {
		log.Printf("database: failed to set query function: %v", err)
	}
	if err := exports.Set("exec", m.Exec); err != nil {
		log.Printf("database: failed to set exec function: %v", err)
	}
	if err := exports.Set("close", m.Close); err != nil {
		log.Printf("database: failed to set close function: %v", err)
	}
}

// Configure sets up the database connection.
func (m *DBModule) Configure(driverName, dataSourceName string) error {
	if m.db != nil {
		if err := m.db.Close(); err != nil {
			log.Printf("database: failed to close existing connection: %v", err)
		}
	}

	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		log.Printf("database: failed to open connection to %s: %v", dataSourceName, err)
		return err
	}
	m.db = db
	log.Printf("database: configured for driver %s", driverName)
	return nil
}

// Close closes the database connection.
func (m *DBModule) Close() error {
	if m.db != nil {
		log.Printf("database: closing connection")
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
	log.Printf("database: executing query: %s", query)

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
		log.Printf("database: query error: %v", err)
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("database: failed to close rows: %v", err)
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
			log.Printf("database: row scan error: %v", err)
			continue
		}

		rec := make(map[string]interface{})
		for i, col := range cols {
			rec[col] = vals[i]
		}
		result = append(result, rec)
	}

	duration := time.Since(startTime)
	log.Printf("database: query completed in %v, returned %d rows", duration, len(result))

	return result, nil
}

// Exec executes a SQL statement without returning rows.
func (m *DBModule) Exec(query string, args ...interface{}) (map[string]interface{}, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not configured, call require('database').configure(...) first")
	}

	startTime := time.Now()
	log.Printf("database: executing exec: %s", query)

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
		log.Printf("database: exec error: %v", err)
		return map[string]interface{}{
			"error":   err.Error(),
			"success": false,
		}, err
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertId, _ := result.LastInsertId()

	duration := time.Since(startTime)
	log.Printf("database: exec completed in %v, affected %d rows", duration, rowsAffected)

	return map[string]interface{}{
		"success":      true,
		"rowsAffected": rowsAffected,
		"lastInsertId": lastInsertId,
	}, nil
}

func init() {
	modules.Register(&DBModule{})
}
