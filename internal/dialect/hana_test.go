package dialect

import "testing"

func TestHanaRegistered(t *testing.T) {
	d, err := Get("hana")
	if err != nil {
		t.Fatalf("hana dialect not registered: %v", err)
	}
	if d.Name() != "hana" {
		t.Fatalf("unexpected name: %s", d.Name())
	}
}
