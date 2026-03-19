package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"os"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type LogEntry struct {
	ID          int
	Timestamp   string
	Filename    string
	Source      string
	Destination string
	Tier        int
	Confidence  float64
	Tags        string
	Action      string
	Corrected   int
}

// Decision representation placeholder
type Decision struct {
	File        string
	Destination string
	Tags        []string
	Tier        int
	Confidence  float64
	IsNewFolder bool
	Action      string
}

func Open(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	for _, schema := range Schemas {
		if _, err := db.Exec(schema); err != nil {
			return nil, fmt.Errorf("failed to migrate schema: %w", err)
		}
	}

	return &Store{db: db}, nil
}

func (s *Store) LogDecision(d Decision) error {
	// Simplified mock implementation for now
	return nil
}

func (s *Store) RecentLog(n int) ([]LogEntry, error) {
	return nil, nil
}

func (s *Store) UnsortedFiles() ([]string, error) {
	return nil, nil
}

func (s *Store) MarkCorrected(id int, newDest string) error {
	return nil
}

func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
