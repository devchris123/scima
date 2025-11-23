// Package main provides the CLI for scima schema migrations.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq" // postgres driver
	"github.com/scima/scima/internal/config"
	"github.com/scima/scima/internal/dialect"
	"github.com/scima/scima/internal/migrate"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{Use: "scima", Short: "Schema migrations for multiple databases (HANA first)"}

var driver string
var configPath string
var dsn string
var migrationsDir string
var schema string // optional schema qualification

func addGlobalFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&driver, "driver", "hana", "Database driver/dialect (hana, pg, mysql, sqlite, etc.)")
	cmd.PersistentFlags().StringVar(&dsn, "dsn", "", "Database DSN / connection string")
	cmd.PersistentFlags().StringVar(&migrationsDir, "migrations-dir", "./migrations", "Directory containing migration files")
	cmd.PersistentFlags().StringVar(&schema, "schema", "", "Optional database schema for migration tracking table and SQL placeholders ({{schema}}, {{schema?}})")
}

func init() {
	addGlobalFlags(rootCmd)

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	downCmd.Flags().IntVar(&steps, "steps", 1, "Number of migration steps to revert (default 1, 0=all)")
}

var initCmd = &cobra.Command{Use: "init", Short: "Initialize migration tracking table", RunE: func(_ *cobra.Command, _ []string) error {
	cfg := gatherConfig()
	migr, db, err := buildMigrator(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing db: %v\n", err)
		}
	}()
	if err := migr.EnsureMigrationTable(context.Background()); err != nil {
		return err
	}
	fmt.Println("migration table ensured")
	return nil
}}

var statusCmd = &cobra.Command{Use: "status", Short: "Show current and pending migrations", RunE: func(_ *cobra.Command, _ []string) error {
	cfg := gatherConfig()
	migr, db, err := buildMigrator(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing db: %v\n", err)
		}
	}()
	pairs, err := migrate.ScanDir(cfg.MigrationsDir)
	if err != nil {
		return err
	}
	if err := migrate.Validate(pairs); err != nil {
		return err
	}
	applied, err := migr.Status(context.Background())
	if err != nil {
		return err
	}
	fmt.Print(migrate.PrettyPrint(pairs, applied))
	return nil
}}

var upCmd = &cobra.Command{Use: "up", Short: "Apply pending up migrations", RunE: func(_ *cobra.Command, _ []string) error {
	cfg := gatherConfig()
	migr, db, err := buildMigrator(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing db: %v\n", err)
		}
	}()
	pairs, err := migrate.ScanDir(cfg.MigrationsDir)
	if err != nil {
		return err
	}
	if err := migrate.Validate(pairs); err != nil {
		return err
	}
	applied, err := migr.Status(context.Background())
	if err != nil {
		return err
	}
	pending := migrate.FilterPending(pairs, applied)
	start := time.Now()
	if err := migr.ApplyUp(context.Background(), pending); err != nil {
		return err
	}
	fmt.Printf("applied %d migrations in %s\n", len(pending), time.Since(start))
	return nil
}}

var steps int
var downCmd = &cobra.Command{Use: "down", Short: "Revert migrations (default 1 step)", RunE: func(_ *cobra.Command, _ []string) error {
	cfg := gatherConfig()
	migr, db, err := buildMigrator(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing db: %v\n", err)
		}
	}()
	pairs, err := migrate.ScanDir(cfg.MigrationsDir)
	if err != nil {
		return err
	}
	if err := migrate.Validate(pairs); err != nil {
		return err
	}
	applied, err := migr.Status(context.Background())
	if err != nil {
		return err
	}
	downs := migrate.ReverseForDown(pairs, applied, steps)
	if len(downs) == 0 {
		fmt.Println("no migrations to revert")
		return nil
	}
	start := time.Now()
	if err := migr.ApplyDown(context.Background(), downs); err != nil {
		return err
	}
	fmt.Printf("reverted %d migrations in %s\n", len(downs), time.Since(start))
	return nil
}}

func gatherConfig() config.Config {
	// Try config file first if provided or default locations
	var cfg *config.Config
	if configPath != "" {
		cfg, _ = config.LoadConfig(configPath)
	} else {
		// Try default locations
		for _, path := range []string{"./scima.yaml", "./scima.yml", "./scima.json", "./scima.toml"} {
			cfg, _ = config.LoadConfig(path)
			if cfg != nil {
				break
			}
		}
		if cfg == nil {
			cfg = &config.Config{}
		}
	}
	// CLI flags override config file
	if driver != "" {
		cfg.Driver = driver
	}
	if dsn != "" {
		cfg.DSN = dsn
	}
	if migrationsDir != "" {
		cfg.MigrationsDir = migrationsDir
	}
	if schema != "" {
		cfg.Schema = schema
	}
	return *cfg
}

func buildMigrator(cfg config.Config) (*migrate.Migrator, *sql.DB, error) {
	dial, err := dialect.Get(cfg.Driver)
	if err != nil {
		return nil, nil, err
	}
	if cfg.DSN == "" {
		return nil, nil, fmt.Errorf("dsn required")
	}
	db, err := sql.Open(driverNameFor(cfg.Driver), cfg.DSN)
	if err != nil {
		return nil, nil, err
	}
	return migrate.NewMigrator(dial, dialect.SQLConn{DB: db}, schema), db, nil
}

func driverNameFor(driver string) string {
	switch driver {
	case "hana":
		return "hdb"
	case "postgres", "pg":
		return "postgres"
	default:
		return driver // assume same
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
