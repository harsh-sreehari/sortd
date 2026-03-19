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
	query := `INSERT INTO sort_log (timestamp, filename, source, destination, tier, confidence, tags, action) 
			  VALUES (datetime('now'), ?, ?, ?, ?, ?, ?, ?)`
	tags := ""
	if len(d.Tags) > 0 {
		tags = d.Tags[0] // Simplify tags
	}
	_, err := s.db.Exec(query, d.File, d.File, d.Destination, d.Tier, d.Confidence, tags, d.Action)
	return err
}

func (s *Store) RecentLog(n int) ([]LogEntry, error) {
	query := `SELECT id, timestamp, filename, source, destination, tier, confidence, tags, action, corrected 
			  FROM sort_log ORDER BY id DESC LIMIT ?`
	rows, err := s.db.Query(query, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LogEntry
	for rows.Next() {
		var e LogEntry
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Filename, &e.Source, &e.Destination, &e.Tier, &e.Confidence, &e.Tags, &e.Action, &e.Corrected); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *Store) UnsortedFiles() ([]string, error) {
	return nil, nil
}

func (s *Store) MarkCorrected(id int, newDest string) error {
	return nil
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
