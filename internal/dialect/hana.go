package dialect

import (
	"context"
	"fmt"
)

// HanaDialect implements Dialect for SAP HANA.
type HanaDialect struct{}

func (h HanaDialect) Name() string { return "hana" }

func init() { Register(HanaDialect{}) }

const hanaMigrationsTable = "SCIMA_SCHEMA_MIGRATIONS" // uppercase by convention in HANA

func (h HanaDialect) EnsureMigrationTable(ctx context.Context, c Conn) error {
	// Try create table if not exists. HANA before 2.0 lacks standard IF NOT EXISTS for some DDL; we attempt and ignore errors.
	create := fmt.Sprintf("CREATE TABLE %s (version BIGINT PRIMARY KEY)", hanaMigrationsTable)
	if _, err := c.ExecContext(ctx, create); err != nil {
		// Ignore 'already exists' like sqlstate 301? We do a simple substring match.
		if !containsIgnoreCase(err.Error(), "exists") {
			// Could attempt a SELECT to verify existence.
			// Fallback: check selectable.
			rows, qerr := c.QueryContext(ctx, fmt.Sprintf("SELECT version FROM %s WHERE 1=0", hanaMigrationsTable))
			if qerr != nil {
				return fmt.Errorf("ensure migrations table failed: %v createErr: %v", qerr, err)
			}
			rows.Close()
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
			hLower[i] = hLower[i] + ('a' - 'A')
		}
	}
	for i := range nLower {
		if nLower[i] >= 'A' && nLower[i] <= 'Z' {
			nLower[i] = nLower[i] + ('a' - 'A')
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

func (h HanaDialect) SelectAppliedVersions(ctx context.Context, c Conn) (map[int64]bool, error) {
	rows, err := c.QueryContext(ctx, fmt.Sprintf("SELECT version FROM %s", hanaMigrationsTable))
	if err != nil {
		// If table not existing treat as empty; attempt create then return empty.
		if cerr := h.EnsureMigrationTable(ctx, c); cerr != nil {
			return nil, err
		}
		return map[int64]bool{}, nil
	}
	defer rows.Close()
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

func (h HanaDialect) InsertVersion(ctx context.Context, c Conn, version int64) error {
	_, err := c.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (version) VALUES (?)", hanaMigrationsTable), version)
	return err
}

func (h HanaDialect) DeleteVersion(ctx context.Context, c Conn, version int64) error {
	_, err := c.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE version = ?", hanaMigrationsTable), version)
	return err
}
