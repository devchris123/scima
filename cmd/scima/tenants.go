package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/scima/scima/internal/config"
	"github.com/scima/scima/internal/migrate"
	"github.com/spf13/cobra"
)

var tenantsCmd = &cobra.Command{
	Use:   "tenants",
	Short: "Multi-tenant schema migrations",
}

var tenantsUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply pending up migrations for all tenants",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg := gatherConfig()
		if len(cfg.Tenants) == 0 {
			return fmt.Errorf("no tenants configured")
		}
		pairs, err := migrate.ScanDir(cfg.MigrationsDir)
		if err != nil {
			return err
		}
		if err := migrate.Validate(pairs); err != nil {
			return err
		}
		mtMigr := migrate.NewMultiTenantMigrator(cfg.Tenants, func(t config.Tenant) (*migrate.Migrator, io.Closer, error) {
			return migrate.BuildTenantMigrator(cfg.Driver, t)
		})
		start := time.Now()
		results := mtMigr.ApplyUpAll(context.Background(), pairs)
		for tenant, err := range results {
			if err != nil {
				fmt.Fprintf(os.Stderr, "tenant %s: error: %v\n", tenant, err)
			} else {
				fmt.Printf("tenant %s: migrations applied successfully\n", tenant)
			}
		}
		fmt.Printf("completed in %s\n", time.Since(start))
		return nil
	},
}

// Register the tenants command and its subcommands in main.go's init()
func init() {
	rootCmd.AddCommand(tenantsCmd)
	tenantsCmd.AddCommand(tenantsUpCmd)
}
