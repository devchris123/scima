//go:build integration
// +build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	_ "github.com/lib/pq"
	"github.com/scima/scima/internal/config"
	"github.com/scima/scima/internal/dialect"
	"github.com/scima/scima/internal/migrate"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestPostgresMultiTenantMigrationsIntegration(t *testing.T) {
	ctx := context.Background()

	rootDir := getenvDefault("SCIMA_TEST_ROOT_DIR", "/Users/I758791/github.com/scima")
	migDir := filepath.Join(rootDir, "tests", "integration", "postgres", "migrations")
	if _, err := os.Stat(migDir); os.IsNotExist(err) {
		t.Fatalf("migrations folder not found: %s", migDir)
	}
	image := getenvDefault("SCIMA_TEST_PG_IMAGE", "postgres:15-alpine")
	port := getenvDefault("SCIMA_TEST_PG_PORT", "5432")
	user := getenvDefault("SCIMA_TEST_PG_USER", "test")
	password := getenvDefault("SCIMA_TEST_PG_PASSWORD", "test")
	dbname := getenvDefault("SCIMA_TEST_PG_DB", "testdb")
	registry := os.Getenv("SCIMA_TEST_PG_REGISTRY")
	if registry != "" {
		image = registry + "/" + image
	}

	pgPort := nat.Port(port + "/tcp")

	req := tc.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{string(pgPort)},
		Env: map[string]string{
			"POSTGRES_USER":     user,
			"POSTGRES_PASSWORD": password,
			"POSTGRES_DB":       dbname,
		},
		WaitingFor: wait.ForListeningPort(pgPort).WithStartupTimeout(60 * time.Second),
	}
	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		t.Fatalf("container start: %v", err)
	}
	defer container.Terminate(ctx)

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("host: %v", err)
	}
	mappedPort, err := container.MappedPort(ctx, pgPort)
	if err != nil {
		t.Fatalf("mapped port: %v", err)
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, mappedPort.Port(), dbname)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	// Create schemas for tenants
	tenantSchemas := []string{"tenant1", "tenant2"}
	for _, schema := range tenantSchemas {
		_, err = db.ExecContext(ctx, fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, schema))
		if err != nil {
			t.Fatalf("create schema %s: %v", schema, err)
		}
	}

	// Prepare tenants config
	var tenants []config.Tenant
	for _, schema := range tenantSchemas {
		tenants = append(tenants, config.Tenant{
			Name:   schema,
			DSN:    dsn,
			Schema: schema,
		})
	}

	// Parse and validate migrations
	pairs, err := migrate.ScanDir(migDir)
	if err != nil {
		t.Fatalf("scan migrations: %v", err)
	}
	if err := migrate.Validate(pairs); err != nil {
		t.Fatalf("validate: %v", err)
	}

	// Multi-tenant migrator
	mtMigr := migrate.NewMultiTenantMigrator(tenants, func(t config.Tenant) (*migrate.Migrator, io.Closer, error) {
		d, err := dialect.Get("postgres")
		if err != nil {
			return nil, nil, err
		}
		db, err := sql.Open("postgres", t.DSN)
		if err != nil {
			return nil, nil, err
		}
		return migrate.NewMigrator(d, dialect.SQLConn{DB: db}, t.Schema), db, nil
	})

	// Apply migrations for all tenants
	results := mtMigr.ApplyUpAll(ctx, pairs)
	for tenant, err := range results {
		if err != nil {
			t.Errorf("tenant %s: migration error: %v", tenant, err)
		}
	}

	// Verify all migrations applied for each tenant
	statusAll := mtMigr.StatusAll(ctx)
	for _, status := range statusAll {
		if len(status) != len(pairs) {
			t.Errorf("expected %d applied migrations, got %d", len(pairs), len(status))
		}
	}
}
