// Package dialect provides database dialect interfaces and helpers for migration operations.
package dialect

import (
	"context"
	"fmt"
)

// Conn abstracts minimal operations needed for migration execution.
// Each dialect can wrap a DB connection or tx.
// Exec should execute statements separated already (no multi-statement parsing here).
type Conn interface {
	ExecContext(ctx context.Context, query string, args ...any) (Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
}

// Result abstracts the result of a database operation.
type Result interface {
	RowsAffected() (int64, error)
}

// Rows abstracts the result set of a database query.
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
	Err() error
}

// Dialect binds SQL variants and introspection / DDL helpers.
type Dialect interface {
	Name() string
	EnsureMigrationTable(ctx context.Context, c Conn) error
	SelectAppliedVersions(ctx context.Context, c Conn) (map[int64]bool, error)
	InsertVersion(ctx context.Context, c Conn, version int64) error
	DeleteVersion(ctx context.Context, c Conn, version int64) error
}

var registry = map[string]Dialect{}

// Register adds dialect.
func Register(d Dialect) { registry[d.Name()] = d }

// Get fetches dialect by name.
func Get(name string) (Dialect, error) {
	d, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown dialect: %s", name)
	}
	return d, nil
}
