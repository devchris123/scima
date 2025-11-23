package config

import (
	"os"
	"testing"
)

func TestConfigStruct(t *testing.T) {
	c := Config{Driver: "hana", DSN: "hdb://user:pass@host:30015", MigrationsDir: "./migrations"}
	if c.Driver != "hana" {
		t.Fatalf("unexpected driver: %s", c.Driver)
	}
}

func TestLoadConfigYAML(t *testing.T) {
	f, err := os.CreateTemp("", "scima_test_config_*.yaml")
	if err != nil {
		t.Fatalf("tempfile: %v", err)
	}
	defer func() {
		if err := os.Remove(f.Name()); err != nil {
			t.Fatalf("remove: %v", err)
		}
	}()
	data := []byte(`driver: pg
dsn: "postgres://user:pass@localhost:5432/db"
migrationsdir: "./migrations"
schema: "tenant42"
`)
	if _, err := f.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	cfg, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Driver != "pg" || cfg.DSN != "postgres://user:pass@localhost:5432/db" || cfg.MigrationsDir != "./migrations" || cfg.Schema != "tenant42" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestLoadConfigJSON(t *testing.T) {
	f, err := os.CreateTemp("", "scima_test_config_*.json")
	if err != nil {
		t.Fatalf("tempfile: %v", err)
	}
	defer func() {
		if err := os.Remove(f.Name()); err != nil {
			t.Fatalf("remove: %v", err)
		}
	}()
	data := []byte(`{
		"driver": "pg",
		"dsn": "postgres://user:pass@localhost:5432/db",
		"migrationsdir": "./migrations",
		"schema": "tenant42"
	}`)
	if _, err := f.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	cfg, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Driver != "pg" || cfg.DSN != "postgres://user:pass@localhost:5432/db" || cfg.MigrationsDir != "./migrations" || cfg.Schema != "tenant42" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestLoadConfigTOML(t *testing.T) {
	f, err := os.CreateTemp("", "scima_test_config_*.toml")
	if err != nil {
		t.Fatalf("tempfile: %v", err)
	}
	defer func() {
		if err := os.Remove(f.Name()); err != nil {
			t.Fatalf("remove: %v", err)
		}
	}()
	data := []byte(`driver = "pg"
dsn = "postgres://user:pass@localhost:5432/db"
migrationsdir = "./migrations"
schema = "tenant42"
`)
	if _, err := f.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	cfg, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Driver != "pg" || cfg.DSN != "postgres://user:pass@localhost:5432/db" || cfg.MigrationsDir != "./migrations" || cfg.Schema != "tenant42" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}
