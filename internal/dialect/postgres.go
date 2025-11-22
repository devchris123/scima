package dialect

import (
	"context"
	"fmt"
	"os"
)

// PostgresDialect implements Dialect for PostgreSQL.
type PostgresDialect struct{}

// Name returns the name of the dialect ("postgres").
func (p PostgresDialect) Name() string { return "postgres" }

const pgMigrationsTable = "schema_migrations"

// EnsureMigrationTable creates the migration tracking table if it does not exist.
func (p PostgresDialect) EnsureMigrationTable(ctx context.Context, c Conn) error {
	stmt := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (version BIGINT PRIMARY KEY)", pgMigrationsTable)
	_, err := c.ExecContext(ctx, stmt)
	return err
}

// SelectAppliedVersions returns a map of applied migration versions from the tracking table.
func (p PostgresDialect) SelectAppliedVersions(ctx context.Context, c Conn) (map[int64]bool, error) {
	rows, err := c.QueryContext(ctx, fmt.Sprintf("SELECT version FROM %s", pgMigrationsTable))
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "warning: error closing rows: %v\n", cerr)
		}
	}()
	res := map[int64]bool{}
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		res[v] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

// InsertVersion inserts a migration version into the Postgres migrations table.
func (p PostgresDialect) InsertVersion(ctx context.Context, c Conn, version int64) error {
	_, err := c.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (version) VALUES ($1)", pgMigrationsTable), version)
	return err
}

// DeleteVersion deletes a migration version from the Postgres migrations table.
func (p PostgresDialect) DeleteVersion(ctx context.Context, c Conn, version int64) error {
	_, err := c.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE version = $1", pgMigrationsTable), version)
	return err
}

func init() { Register(PostgresDialect{}) }
