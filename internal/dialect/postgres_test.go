package dialect

import "testing"

func TestPostgresDialectRegistered(t *testing.T) {
	d, err := Get("postgres")
	if err != nil {
		t.Fatalf("postgres dialect not registered: %v", err)
	}
	if d.Name() != "postgres" {
		t.Fatalf("unexpected name: %s", d.Name())
	}
}
