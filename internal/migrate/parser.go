package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var filePattern = regexp.MustCompile(`^(\d+)_([a-zA-Z0-9_]+)\.(up|down)\.sql$`)

// MigrationFile represents a single migration direction (up or down)
type MigrationFile struct {
	Version   int64
	Name      string
	Direction string // up or down
	FullPath  string
	SQL       string
}

// MigrationPair groups up/down
// Down may be nil for initial creation until added.
type MigrationPair struct {
	Up   *MigrationFile
	Down *MigrationFile
}

// ScanDir scans for migrations under directory.
func ScanDir(dir string) ([]MigrationPair, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	byVersion := map[int64]*MigrationPair{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := filePattern.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		versionStr := m[1]
		name := m[2]
		dirn := m[3]
		version, err := strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid version in filename %s: %w", e.Name(), err)
		}
		path := filepath.Join(dir, e.Name())
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		mf := &MigrationFile{Version: version, Name: name, Direction: dirn, FullPath: path, SQL: string(contentBytes)}
		pair := byVersion[version]
		if pair == nil {
			pair = &MigrationPair{}
			byVersion[version] = pair
		}
		if dirn == "up" {
			pair.Up = mf
		} else {
			pair.Down = mf
		}
	}
	// sort
	versions := make([]int, 0, len(byVersion))
	for v := range byVersion {
		versions = append(versions, int(v))
	}
	sort.Ints(versions)
	pairs := make([]MigrationPair, 0, len(versions))
	for _, v := range versions {
		pair := byVersion[int64(v)]
		pairs = append(pairs, *pair)
	}
	return pairs, nil
}

// FilterPending calculates pending ups given applied versions.
func FilterPending(pairs []MigrationPair, applied map[int64]bool) []MigrationFile {
	var res []MigrationFile
	for _, p := range pairs {
		if p.Up == nil {
			continue
		}
		if !applied[p.Up.Version] {
			res = append(res, *p.Up)
		}
	}
	return res
}

// ReverseForDown returns downs in reverse order restricted to already applied versions.
func ReverseForDown(pairs []MigrationPair, applied map[int64]bool, steps int) []MigrationFile {
	var downs []MigrationFile
	for _, p := range pairs {
		if p.Down == nil {
			continue
		}
		if applied[p.Down.Version] {
			downs = append(downs, *p.Down)
		}
	}
	// reverse
	for i, j := 0, len(downs)-1; i < j; i, j = i+1, j-1 {
		downs[i], downs[j] = downs[j], downs[i]
	}
	if steps > 0 && steps < len(downs) {
		return downs[:steps]
	}
	return downs
}

// Validate ensures each pair has an up file.
func Validate(pairs []MigrationPair) error {
	for _, p := range pairs {
		if p.Up == nil {
			return fmt.Errorf("missing up migration for version %d", p.Down.Version)
		}
		// Optionally ensure naming consistency.
		if p.Down != nil && p.Up.Name != p.Down.Name {
			return fmt.Errorf("name mismatch for version %d: up=%s down=%s", p.Up.Version, p.Up.Name, p.Down.Name)
		}
	}
	return nil
}

// PrettyPrint builds status output lines.
func PrettyPrint(pairs []MigrationPair, applied map[int64]bool) string {
	var sb strings.Builder
	for _, p := range pairs {
		up := p.Up
		if up == nil {
			continue
		}
		status := "pending"
		if applied[up.Version] {
			status = "applied"
		}
		fmt.Fprintf(&sb, "%04d\t%s\t%s\n", up.Version, up.Name, status)
	}
	return sb.String()
}
