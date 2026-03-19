package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpen(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sortd-store-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "sortd.db")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer s.Close()

	// Verify tables exist
	rows, err := s.db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		t.Fatalf("failed to query sqlite_master: %v", err)
	}
	defer rows.Close()

	tables := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		tables[name] = true
	}

	expected := []string{"sort_log", "folder_index", "affinities"}
	for _, tbl := range expected {
		if !tables[tbl] {
			t.Errorf("expected table %s not found", tbl)
		}
	}
}
