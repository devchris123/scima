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
var dsn string
var migrationsDir string

func addGlobalFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&driver, "driver", "hana", "Database driver/dialect (hana, pg, mysql, sqlite, etc.)")
	cmd.PersistentFlags().StringVar(&dsn, "dsn", "", "Database DSN / connection string")
	cmd.PersistentFlags().StringVar(&migrationsDir, "migrations-dir", "./migrations", "Directory containing migration files")
}

func init() {
	addGlobalFlags(rootCmd)

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	downCmd.Flags().IntVar(&steps, "steps", 1, "Number of migration steps to revert (default 1, 0=all)")
}

var initCmd = &cobra.Command{Use: "init", Short: "Initialize migration tracking table", RunE: func(cmd *cobra.Command, args []string) error {
	cfg := gatherConfig()
	migr, db, err := buildMigrator(cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := migr.Dialect.EnsureMigrationTable(context.Background(), migr.Conn); err != nil {
		return err
	}
	fmt.Println("migration table ensured")
	return nil
}}

var statusCmd = &cobra.Command{Use: "status", Short: "Show current and pending migrations", RunE: func(cmd *cobra.Command, args []string) error {
	cfg := gatherConfig()
	migr, db, err := buildMigrator(cfg)
	if err != nil {
		return err
	}
	defer db.Close()
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

var upCmd = &cobra.Command{Use: "up", Short: "Apply pending up migrations", RunE: func(cmd *cobra.Command, args []string) error {
	cfg := gatherConfig()
	migr, db, err := buildMigrator(cfg)
	if err != nil {
		return err
	}
	defer db.Close()
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
var downCmd = &cobra.Command{Use: "down", Short: "Revert migrations (default 1 step)", RunE: func(cmd *cobra.Command, args []string) error {
	cfg := gatherConfig()
	migr, db, err := buildMigrator(cfg)
	if err != nil {
		return err
	}
	defer db.Close()
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
	return config.Config{Driver: driver, DSN: dsn, MigrationsDir: migrationsDir}
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
	return migrate.NewMigrator(dial, dialect.SQLConn{DB: db}), db, nil
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
