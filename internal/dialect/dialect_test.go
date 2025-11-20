package dialect

import "testing"

func TestRegistryUnknown(t *testing.T) {
	if _, err := Get("doesnotexist"); err == nil {
		t.Fatalf("expected error for unknown dialect")
	}
}
