// Package migrate provides migration execution logic for supported database dialects.
package migrate

import (
	"context"
	"fmt"
	"strings"

	"github.com/scima/scima/internal/dialect"
)

// Migrator executes migrations for a dialect.
type Migrator struct {
	Conn    dialect.Conn
	Dialect dialect.Dialect
	Schema  string // optional schema qualifier
}

// NewMigrator creates a new Migrator for the given dialect and connection.
func NewMigrator(d dialect.Dialect, c dialect.Conn, schema string) *Migrator {
	return &Migrator{Conn: c, Dialect: d, Schema: schema}
}

// EnsureMigrationTable ensures the migration tracking table exists.
func (m *Migrator) EnsureMigrationTable(ctx context.Context) error {
	return m.Dialect.EnsureMigrationTable(ctx, m.Conn, m.Schema)
}

// Status returns applied version set.
func (m *Migrator) Status(ctx context.Context) (map[int64]bool, error) {
	if err := m.Dialect.EnsureMigrationTable(ctx, m.Conn, m.Schema); err != nil {
		return nil, err
	}
	return m.Dialect.SelectAppliedVersions(ctx, m.Conn, m.Schema)
}

// ApplyUp applies pending up migrations.
func (m *Migrator) ApplyUp(ctx context.Context, ups []MigrationFile) error {
	for _, up := range ups {
		expanded, err := expandPlaceholders(up.SQL, m.Schema)
		if err != nil {
			return fmt.Errorf("placeholder expansion up %d: %w", up.Version, err)
		}
		if _, err := m.Conn.ExecContext(ctx, expanded); err != nil {
			return fmt.Errorf("apply up %d failed: %w", up.Version, err)
		}
		if err := m.Dialect.InsertVersion(ctx, m.Conn, m.Schema, up.Version); err != nil {
			return err
		}
	}
	return nil
}

// ApplyDown applies downs.
func (m *Migrator) ApplyDown(ctx context.Context, downs []MigrationFile) error {
	for _, down := range downs {
		expanded, err := expandPlaceholders(down.SQL, m.Schema)
		if err != nil {
			return fmt.Errorf("placeholder expansion down %d: %w", down.Version, err)
		}
		if _, err := m.Conn.ExecContext(ctx, expanded); err != nil {
			return fmt.Errorf("apply down %d failed: %w", down.Version, err)
		}
		if err := m.Dialect.DeleteVersion(ctx, m.Conn, m.Schema, down.Version); err != nil {
			return err
		}
	}
	return nil
}

const (
	requiredSchemaToken = "{{schema}}"
	optionalSchemaToken = "{{schema?}}"
)

// expandPlaceholders substitutes schema tokens in SQL.
// {{schema}} requires a non-empty schema; {{schema?}} inserts schema plus dot or nothing.
func expandPlaceholders(sql string, schema string) (string, error) {
	// We process with a manual scan to support escaping via backslash: \{{schema}} or \{{schema?}} remain literal.
	// Strategy: iterate runes, detect backslash before placeholder start; build output.
	var b strings.Builder
	// Precompute for efficiency.
	req := requiredSchemaToken
	opt := optionalSchemaToken
	for i := 0; i < len(sql); {
		// Handle escaped required placeholder
		if i+1+len(req) <= len(sql) && sql[i] == '\\' && sql[i+1:i+1+len(req)] == req {
			b.WriteString(req) // drop escape, keep literal token
			i += 1 + len(req)
			continue
		}
		// Handle escaped optional placeholder
		if i+1+len(opt) <= len(sql) && sql[i] == '\\' && sql[i+1:i+1+len(opt)] == opt {
			b.WriteString(opt)
			i += 1 + len(opt)
			continue
		}
		// Unescaped optional
		if i+len(opt) <= len(sql) && sql[i:i+len(opt)] == opt {
			if schema != "" {
				b.WriteString(schema)
				b.WriteByte('.')
			}
			i += len(opt)
			continue
		}
		// Unescaped required
		if i+len(req) <= len(sql) && sql[i:i+len(req)] == req {
			if schema == "" {
				return "", fmt.Errorf("%s used but schema not set", requiredSchemaToken)
			}
			b.WriteString(schema)
			i += len(req)
			continue
		}
		// Default: copy one byte
		b.WriteByte(sql[i])
		i++
	}
	return b.String(), nil
}
