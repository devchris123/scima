package dialect

import (
	"context"
	"database/sql"
	"errors"
)

// SQLConn adapts *sql.DB or *sql.Tx into Conn.
// Provided here so callers can wrap their own connection.

type SQLConn struct{ DB *sql.DB }

func (s SQLConn) ExecContext(ctx context.Context, query string, args ...any) (Result, error) {
	res, err := s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s SQLConn) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// Ensure SQLConn satisfies interfaces.
var _ Conn = SQLConn{}

// Simple error classification (placeholder).
func IsNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }
