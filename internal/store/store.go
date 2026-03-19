package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type LogEntry struct {
	ID               int
	Timestamp        string
	Filename         string
	OriginalFilename string
	Source           string
	Destination      string
	Tier             int
	Confidence       float64
	Tags             string
	Action           string
	Reasoning        string
	Corrected        int
}

// Decision representation placeholder
type Decision struct {
	File             string
	OriginalFilename string
	Destination      string
	Tags             []string
	Tier             int
	Confidence       float64
	IsNewFolder      bool
	Action           string
	Reasoning        string
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

	// Safe column additions for existing databases
	_, _ = db.Exec("ALTER TABLE sort_log ADD COLUMN original_filename TEXT NOT NULL DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE sort_log ADD COLUMN reasoning TEXT")

	return &Store{db: db}, nil
}

func (s *Store) LogDecision(d Decision) error {
	query := `INSERT INTO sort_log (timestamp, filename, original_filename, source, destination, tier, confidence, tags, action, reasoning) 
			  VALUES (datetime('now'), ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	tags := "[]"
	if len(d.Tags) > 0 {
		if b, err := json.Marshal(d.Tags); err == nil {
			tags = string(b)
		}
	}

	orig := d.OriginalFilename
	if orig == "" {
		orig = d.File
	}

	_, err := s.db.Exec(query, d.File, orig, d.File, d.Destination, d.Tier, d.Confidence, tags, d.Action, d.Reasoning)
	return err
}

func (s *Store) RecentLog(n int) ([]LogEntry, error) {
	query := `SELECT id, timestamp, filename, original_filename, source, destination, tier, confidence, tags, action, reasoning, corrected 
			  FROM sort_log ORDER BY id DESC LIMIT ?`
	rows, err := s.db.Query(query, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LogEntry
	for rows.Next() {
		var e LogEntry
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Filename, &e.OriginalFilename, &e.Source, &e.Destination, &e.Tier, &e.Confidence, &e.Tags, &e.Action, &e.Reasoning, &e.Corrected); err != nil {
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
