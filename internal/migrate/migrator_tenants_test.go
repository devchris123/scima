package migrate

import (
	"context"
	"io"
	"testing"

	"github.com/scima/scima/internal/config"
)

type MockCloser struct{}

func (m MockCloser) Close() error {
	return nil
}

func TestApplyMultiTenants(t *testing.T) {
	tenants := []config.Tenant{
		{Name: "tenant1", DSN: "dsn1", Schema: "schema1"},
		{Name: "tenant2", DSN: "dsn2", Schema: "schema2"},
	}
	versions := map[int64]bool{10: true}
	pairs := []MigrationPair{
		{
			Up:   &MigrationFile{Version: 10, Name: "init", Direction: "up", SQL: "CREATE TABLE"},
			Down: &MigrationFile{Version: 10, Name: "init", Direction: "down", SQL: "DROP TABLE"},
		},
		{
			Up:   &MigrationFile{Version: 20, Name: "add_col", Direction: "up", SQL: "ALTER TABLE"},
			Down: &MigrationFile{Version: 20, Name: "add_col", Direction: "down", SQL: "ALTER TABLE DROP COLUMN"},
		},
	}
	migr := NewMigrator(mockDialect{versions: versions}, &mockConn{Versions: versions}, "")
	mtMigr := NewMultiTenantMigrator(tenants, func(_ config.Tenant) (*Migrator, io.Closer, error) {
		return migr, MockCloser{}, nil
	})
	results := mtMigr.ApplyUpAll(context.Background(), pairs)
	for _, res := range results {
		if res != nil {
			t.Errorf("expected nil error, got %v", res)
		}
	}
}

func TestApplyOnlyPendingMigrations(t *testing.T) {
	tenants := []config.Tenant{
		{Name: "tenant1", DSN: "dsn1", Schema: "schema1"},
	}
	versions1 := map[int64]bool{10: true, 20: true}
	versions2 := map[int64]bool{10: true}
	pairs := []MigrationPair{
		{
			Up:   &MigrationFile{Version: 10, Name: "init", Direction: "up", SQL: "CREATE TABLE"},
			Down: &MigrationFile{Version: 10, Name: "init", Direction: "down", SQL: "DROP TABLE"},
		},
		{
			Up:   &MigrationFile{Version: 20, Name: "add_col", Direction: "up", SQL: "ALTER TABLE"},
			Down: &MigrationFile{Version: 20, Name: "add_col", Direction: "down", SQL: "ALTER TABLE DROP COLUMN"},
		},
		{
			Up:   &MigrationFile{Version: 30, Name: "new_feature", Direction: "up", SQL: "UPDATE TABLE"},
			Down: &MigrationFile{Version: 30, Name: "new_feature", Direction: "down", SQL: "REVERT UPDATE"},
		},
	}
	migr1 := NewMigrator(mockDialect{versions: versions1}, &mockConn{Versions: versions1}, "")
	migr2 := NewMigrator(mockDialect{versions: versions2}, &mockConn{Versions: versions2}, "")
	mtMigr := NewMultiTenantMigrator(tenants, func(config config.Tenant) (*Migrator, io.Closer, error) {
		switch config.Name {
		case "tenant1":
			return migr1, MockCloser{}, nil
		case "tenant2":
			return migr2, MockCloser{}, nil
		}
		return nil, nil, nil
	})
	results := mtMigr.ApplyUpAll(context.Background(), pairs)
	for _, res := range results {
		if res != nil {
			t.Errorf("expected nil error, got %v", res)
		}
	}
}
