// Package dialect provides database dialect interfaces and helpers for migration operations.
package dialect

import (
	"context"
	"database/sql"
	"errors"
)

// SQLConn adapts *sql.DB or *sql.Tx into Conn.
// Provided here so callers can wrap their own connection.

// SQLConn adapts *sql.DB or *sql.Tx into Conn for migration operations.
type SQLConn struct{ DB *sql.DB }

// ExecContext executes a query with the given arguments and returns a Result.
func (s SQLConn) ExecContext(ctx context.Context, query string, args ...any) (Result, error) {
	res, err := s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// QueryContext executes a query with the given arguments and returns Rows.
func (s SQLConn) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// Ensure SQLConn satisfies interfaces.
var _ Conn = SQLConn{}

// IsNotFound reports whether the error is a sql.ErrNoRows.
func IsNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }
