package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/scima/scima/internal/config"
	"github.com/scima/scima/internal/dialect"
)

// MultiTenantMigrator manages migrations across multiple tenants.
type MultiTenantMigrator struct {
	tenants       []config.Tenant
	buildMigrator func(cfg config.Tenant) (*Migrator, io.Closer, error)
}

// BuildDefaultMigrator constructs a Migrator and underlying io.Closer from config.
func BuildDefaultMigrator(driver string, cfg config.Tenant) (*Migrator, io.Closer, error) {
	dial, err := dialect.Get(driver)
	if err != nil {
		return nil, nil, err
	}
	if cfg.DSN == "" {
		return nil, nil, fmt.Errorf("dsn required")
	}
	db, err := sql.Open(driverNameFor(driver), cfg.DSN)
	if err != nil {
		return nil, nil, err
	}
	return NewMigrator(dial, dialect.SQLConn{DB: db}, cfg.Schema), db, nil
}

// NewMultiTenantMigrator creates a MultiTenantMigrator.
func NewMultiTenantMigrator(
	tenants []config.Tenant,
	buildMigrator func(cfg config.Tenant) (*Migrator, io.Closer, error),
) *MultiTenantMigrator {
	return &MultiTenantMigrator{
		tenants,
		buildMigrator,
	}
}

// ApplyUpAll applies all pending up migrations for all tenants.
func (m *MultiTenantMigrator) ApplyUpAll(ctx context.Context, pairs []MigrationPair) map[string]error {
	results := make(map[string]error)
	for _, t := range m.tenants {
		migr, db, err := m.buildMigrator(t)
		if err != nil {
			results[t.Name] = err
			continue
		}
		applied, err := migr.Status(ctx)
		if err != nil {
			results[t.Name] = err
			closeDb(db)
			continue
		}
		ups := FilterPending(pairs, applied) // You may need to implement this helper
		err = migr.ApplyUp(ctx, ups)
		results[t.Name] = err
		closeDb(db)
	}
	return results
}

// StatusAll retrieves the migration status for all tenants.
func (m *MultiTenantMigrator) StatusAll(ctx context.Context) map[string]map[int64]bool {
	results := make(map[string]map[int64]bool)
	for _, t := range m.tenants {
		migr, db, err := m.buildMigrator(t)
		if err != nil {
			results[t.Name] = nil
			continue
		}
		status, err := migr.Status(ctx)
		if err != nil {
			results[t.Name] = nil
		} else {
			results[t.Name] = status
		}
		closeDb(db)
	}
	return results
}

// BuildTenantMigrator constructs a Migrator and underlying *sql.DB for a tenant.
func BuildTenantMigrator(driver string, cfg config.Tenant) (*Migrator, *sql.DB, error) {
	dial, err := dialect.Get(driver)
	if err != nil {
		return nil, nil, err
	}
	if cfg.DSN == "" {
		return nil, nil, fmt.Errorf("dsn required")
	}
	db, err := sql.Open(driverNameFor(driver), cfg.DSN)
	if err != nil {
		return nil, nil, err
	}
	return NewMigrator(dial, dialect.SQLConn{DB: db}, cfg.Schema), db, nil
}
