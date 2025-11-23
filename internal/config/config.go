// Package config holds runtime configuration for migrations.
package config

import (
	"github.com/spf13/viper"
)

// Config holds runtime configuration for migrations.
type Config struct {
	Driver        string `mapstructure:"driver"`
	DSN           string `mapstructure:"dsn"`
	MigrationsDir string `mapstructure:"migrationsdir"`
	Schema        string `mapstructure:"schema"`
}

// LoadConfig uses Viper to load config from file, env, and flags.
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType(guessType(configPath))
	v.SetDefault("driver", "hana")
	v.SetDefault("migrationsdir", "./migrations")
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	// Optionally bind env vars here if desired
	var cfg Config
	err := v.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func guessType(path string) string {
	switch {
	case len(path) > 5 && path[len(path)-5:] == ".yaml":
		return "yaml"
	case len(path) > 4 && path[len(path)-4:] == ".yml":
		return "yaml"
	case len(path) > 5 && path[len(path)-5:] == ".json":
		return "json"
	case len(path) > 5 && path[len(path)-5:] == ".toml":
		return "toml"
	default:
		return "yaml"
	}
}
