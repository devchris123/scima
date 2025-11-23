//go:build integration
// +build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	_ "github.com/lib/pq"
	"github.com/scima/scima/internal/dialect"
	"github.com/scima/scima/internal/migrate"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Integration test configuration via environment variables:
//   SCIMA_TEST_PG_IMAGE     (default: postgres:15-alpine)
//   SCIMA_TEST_PG_PORT      (default: 5432)
//   SCIMA_TEST_PG_USER      (default: test)
//   SCIMA_TEST_PG_PASSWORD  (default: test)
//   SCIMA_TEST_PG_DB        (default: testdb)
//   SCIMA_TEST_PG_REGISTRY  (optional, prepends to image if set)
//
// These are for integration testing only and will not clash with production env vars.

func TestPostgresMigrationsIntegration(t *testing.T) {
	// ---------------------------------------------------------------------
	// SETUP: Start ephemeral Postgres container and establish connection
	// ---------------------------------------------------------------------
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
	// Setup test schema
	_, err = db.ExecContext(ctx, `CREATE SCHEMA IF NOT EXISTS test_schema`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	// ---------------------------------------------------------------------
	// MIGRATOR: Acquire dialect and create migrator wrapper
	// ---------------------------------------------------------------------
	d, err := dialect.Get("postgres")
	if err != nil {
		t.Fatalf("get dialect: %v", err)
	}
	migr := migrate.NewMigrator(d, dialect.SQLConn{DB: db}, "test_schema")

	// ---------------------------------------------------------------------
	// DISCOVERY: Locate migrations directory and parse & validate files
	// ---------------------------------------------------------------------
	pairs, err := migrate.ScanDir(migDir)
	if err != nil {
		t.Fatalf("scan migrations: %v", err)
	}
	if err := migrate.Validate(pairs); err != nil {
		t.Fatalf("validate: %v", err)
	}

	// ---------------------------------------------------------------------
	// STATUS (PRE): Expect no applied migrations on fresh container
	// ---------------------------------------------------------------------
	applied, err := migr.Status(ctx)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if len(applied) != 0 {
		t.Fatalf("expected 0 applied initially, got %d", len(applied))
	}

	// ---------------------------------------------------------------------
	// EXECUTION: Apply all pending "up" migrations
	// ---------------------------------------------------------------------
	pending := migrate.FilterPending(pairs, applied)
	if len(pending) == 0 {
		t.Fatalf("expected pending migrations")
	}
	if err := migr.ApplyUp(ctx, pending); err != nil {
		t.Fatalf("apply up: %v", err)
	}

	// ---------------------------------------------------------------------
	// STATUS (POST): Verify all migrations applied
	// ---------------------------------------------------------------------
	applied2, err := migr.Status(ctx)
	if err != nil {
		t.Fatalf("status2: %v", err)
	}
	if len(applied2) != len(pairs) {
		t.Fatalf("expected %d applied got %d", len(pairs), len(applied2))
	}
}

func getenvDefault(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}
