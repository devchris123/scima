package config

// Config holds runtime configuration for migrations.
type Config struct {
	Driver        string
	DSN           string
	MigrationsDir string
}
