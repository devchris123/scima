package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanDirAndValidate(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"0010_init.up.sql":    "CREATE TABLE t (id INT);",
		"0010_init.down.sql":  "DROP TABLE t;",
		"0020_add_col.up.sql": "ALTER TABLE t ADD (name NVARCHAR(100));",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	pairs, err := ScanDir(dir)
	if err != nil {
		t.Fatalf("scan error: %v", err)
	}
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs got %d", len(pairs))
	}
	if err := Validate(pairs); err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	applied := map[int64]bool{10: true}
	pending := FilterPending(pairs, applied)
	if len(pending) != 1 || pending[0].Version != 20 {
		t.Fatalf("pending mismatch: %+v", pending)
	}
	downs := ReverseForDown(pairs, applied, 1)
	if len(downs) != 1 || downs[0].Version != 10 {
		t.Fatalf("downs mismatch: %+v", downs)
	}
	status := PrettyPrint(pairs, applied)
	if status == "" {
		t.Fatalf("status empty")
	}
}
