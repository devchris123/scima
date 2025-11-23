package dialect

import (
	"context"
	"fmt"
	"os"
)

// HanaDialect implements Dialect for SAP HANA.
type HanaDialect struct{}

// Name returns the name of the dialect ("hana").
func (h HanaDialect) Name() string { return "hana" }

func init() { Register(HanaDialect{}) }

// EnsureMigrationTable creates the migration tracking table if it does not exist.
func (h HanaDialect) EnsureMigrationTable(ctx context.Context, c Conn, schema string) error {
	// Try create table if not exists. HANA before 2.0 lacks standard IF NOT EXISTS for some DDL; we attempt and ignore errors.
	table := qualifiedMigrationTable(schema)
	create := fmt.Sprintf("CREATE TABLE %s (version BIGINT PRIMARY KEY)", table)
	if _, err := c.ExecContext(ctx, create); err != nil {
		// Ignore 'already exists' like sqlstate 301? We do a simple substring match.
		if !containsIgnoreCase(err.Error(), "exists") {
			// Could attempt a SELECT to verify existence.
			// Fallback: check selectable.
			rows, qerr := c.QueryContext(ctx, fmt.Sprintf("SELECT version FROM %s WHERE 1=0", table))
			if qerr != nil {
				return fmt.Errorf("ensure migrations table failed: %v createErr: %v", qerr, err)
			}
			if cerr := rows.Close(); cerr != nil {
				return fmt.Errorf("error closing rows: %v", cerr)
			}
		}
	}
	return nil
}

func containsIgnoreCase(hay, needle string) bool {
	return len(hay) >= len(needle) && (stringIndexFold(hay, needle) >= 0)
}

func stringIndexFold(hay, needle string) int {
	hLower := []rune(hay)
	nLower := []rune(needle)
	for i := range hLower {
		if hLower[i] >= 'A' && hLower[i] <= 'Z' {
			hLower[i] += ('a' - 'A')
		}
	}
	for i := range nLower {
		if nLower[i] >= 'A' && nLower[i] <= 'Z' {
			nLower[i] += ('a' - 'A')
		}
	}
	for i := 0; i+len(nLower) <= len(hLower); i++ {
		match := true
		for j := range nLower {
			if hLower[i+j] != nLower[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// SelectAppliedVersions returns a map of applied migration versions from the tracking table.
func (h HanaDialect) SelectAppliedVersions(ctx context.Context, c Conn, schema string) (map[int64]bool, error) {
	table := qualifiedMigrationTable(schema)
	rows, err := c.QueryContext(ctx, fmt.Sprintf("SELECT version FROM %s", table))
	if err != nil {
		// If table not existing treat as empty; attempt create then return empty.
		if cerr := h.EnsureMigrationTable(ctx, c, schema); cerr != nil {
			return nil, err
		}
		return map[int64]bool{}, nil
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "error closing db: %v\n", err)
		}
	}()
	applied := map[int64]bool{}
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return applied, nil
}

// InsertVersion inserts a migration version into the HANA migrations table.
func (h HanaDialect) InsertVersion(ctx context.Context, c Conn, schema string, version int64) error {
	table := qualifiedMigrationTable(schema)
	_, err := c.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (version) VALUES (?)", table), version)
	return err
}

// DeleteVersion deletes a migration version from the HANA migrations table.
func (h HanaDialect) DeleteVersion(ctx context.Context, c Conn, schema string, version int64) error {
	table := qualifiedMigrationTable(schema)
	_, err := c.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE version = ?", table), version)
	return err
}
