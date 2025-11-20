package migrate

import (
	"context"
	"fmt"

	"github.com/scima/scima/internal/dialect"
)

// Migrator executes migrations for a dialect.
type Migrator struct {
	Conn    dialect.Conn
	Dialect dialect.Dialect
}

func NewMigrator(d dialect.Dialect, c dialect.Conn) *Migrator { return &Migrator{Conn: c, Dialect: d} }

// Status returns applied version set.
func (m *Migrator) Status(ctx context.Context) (map[int64]bool, error) {
	if err := m.Dialect.EnsureMigrationTable(ctx, m.Conn); err != nil {
		return nil, err
	}
	return m.Dialect.SelectAppliedVersions(ctx, m.Conn)
}

// ApplyUp applies pending up migrations.
func (m *Migrator) ApplyUp(ctx context.Context, ups []MigrationFile) error {
	for _, up := range ups {
		if _, err := m.Conn.ExecContext(ctx, up.SQL); err != nil {
			return fmt.Errorf("apply up %d failed: %w", up.Version, err)
		}
		if err := m.Dialect.InsertVersion(ctx, m.Conn, up.Version); err != nil {
			return err
		}
	}
	return nil
}

// ApplyDown applies downs.
func (m *Migrator) ApplyDown(ctx context.Context, downs []MigrationFile) error {
	for _, down := range downs {
		if _, err := m.Conn.ExecContext(ctx, down.SQL); err != nil {
			return fmt.Errorf("apply down %d failed: %w", down.Version, err)
		}
		if err := m.Dialect.DeleteVersion(ctx, m.Conn, down.Version); err != nil {
			return err
		}
	}
	return nil
}
