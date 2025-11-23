package migrate

import (
	"context"
	"testing"

	"github.com/scima/scima/internal/dialect"
)

type mockConn struct {
	Execs    []string
	Queries  []string
	Versions map[int64]bool
}

func (m *mockConn) ExecContext(_ context.Context, query string, _ ...any) (dialect.Result, error) {
	m.Execs = append(m.Execs, query)
	return mockResult{}, nil
}

func (m *mockConn) QueryContext(_ context.Context, query string, _ ...any) (dialect.Rows, error) {
	m.Queries = append(m.Queries, query)
	return mockRows{versions: m.Versions}, nil
}

type mockResult struct{}

func (mockResult) RowsAffected() (int64, error) { return 1, nil }

type mockRows struct {
	versions map[int64]bool
	iter     []int64
	idx      int
}

func (r mockRows) Next() bool {
	if r.iter == nil {
		for v := range r.versions {
			r.iter = append(r.iter, v)
		}
	}
	if r.idx >= len(r.iter) {
		return false
	}
	return true
}
func (r mockRows) Scan(dest ...any) error {
	if r.idx == 0 || r.idx > len(r.iter) {
		return nil
	}
	val := r.iter[r.idx-1]
	ptr := dest[0].(*int64)
	*ptr = val
	return nil
}
func (r mockRows) Close() error { return nil }
func (r mockRows) Err() error   { return nil }

// mock dialect

type mockDialect struct{ versions map[int64]bool }

func (d mockDialect) Name() string { return "mock" }
func (d mockDialect) EnsureMigrationTable(_ context.Context, _ dialect.Conn, _ string) error {
	return nil
}
func (d mockDialect) SelectAppliedVersions(_ context.Context, _ dialect.Conn, _ string) (map[int64]bool, error) {
	return d.versions, nil
}
func (d mockDialect) InsertVersion(_ context.Context, _ dialect.Conn, _ string, version int64) error {
	d.versions[version] = true
	return nil
}
func (d mockDialect) DeleteVersion(_ context.Context, _ dialect.Conn, _ string, version int64) error {
	delete(d.versions, version)
	return nil
}

func TestMigratorApplyUpDown(t *testing.T) {
	versions := map[int64]bool{10: true}
	migr := NewMigrator(mockDialect{versions: versions}, &mockConn{Versions: versions}, "")
	ups := []MigrationFile{{Version: 20, Name: "add_col", Direction: "up", SQL: "ALTER"}}
	if err := migr.ApplyUp(context.Background(), ups); err != nil {
		t.Fatalf("apply up: %v", err)
	}
	if !versions[20] {
		t.Fatalf("version 20 not inserted")
	}
	downs := []MigrationFile{{Version: 20, Name: "add_col", Direction: "down", SQL: "ALTER"}}
	if err := migr.ApplyDown(context.Background(), downs); err != nil {
		t.Fatalf("apply down: %v", err)
	}
	if versions[20] {
		t.Fatalf("version 20 still present")
	}
}
