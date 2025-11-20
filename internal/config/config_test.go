package config

import "testing"

func TestConfigStruct(t *testing.T) {
	c := Config{Driver: "hana", DSN: "hdb://user:pass@host:30015", MigrationsDir: "./migrations"}
	if c.Driver != "hana" {
		t.Fatalf("unexpected driver: %s", c.Driver)
	}
}
