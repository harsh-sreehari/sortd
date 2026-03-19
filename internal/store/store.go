package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	_ "modernc.org/sqlite"
)

type TagStat struct {
	Tag   string
	Count int
}

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
		var origName, reason sql.NullString
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Filename, &origName, &e.Source, &e.Destination, &e.Tier, &e.Confidence, &e.Tags, &e.Action, &reason, &e.Corrected); err != nil {
			return nil, err
		}
		e.OriginalFilename = origName.String
		e.Reasoning = reason.String
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *Store) UnsortedFiles() ([]LogEntry, error) {
	query := `SELECT id, timestamp, filename, original_filename, source, destination, tier, confidence, tags, action, reasoning, corrected 
			  FROM sort_log WHERE action = 'parked' AND corrected = 0 ORDER BY id ASC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LogEntry
	for rows.Next() {
		var e LogEntry
		var origName, reason sql.NullString
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Filename, &origName, &e.Source, &e.Destination, &e.Tier, &e.Confidence, &e.Tags, &e.Action, &reason, &e.Corrected); err != nil {
			return nil, err
		}
		e.OriginalFilename = origName.String
		e.Reasoning = reason.String
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *Store) SearchLog(n int, filters map[string]string) ([]LogEntry, error) {
	where := "WHERE 1=1"
	var args []interface{}

	if val, ok := filters["tag"]; ok && val != "" {
		where += " AND tags LIKE ?"
		args = append(args, "%"+val+"%")
	}
	if val, ok := filters["tier"]; ok && val != "" {
		where += " AND tier = ?"
		args = append(args, val)
	}
	if val, ok := filters["action"]; ok && val != "" {
		where += " AND action = ?"
		args = append(args, val)
	}
	if val, ok := filters["today"]; ok && val == "true" {
		where += " AND timestamp LIKE (date('now') || '%')"
	}
	if val, ok := filters["query"]; ok && val != "" {
		where += " AND (original_filename LIKE ? OR destination LIKE ? OR reasoning LIKE ?)"
		pattern := "%" + val + "%"
		args = append(args, pattern, pattern, pattern)
	}

	query := fmt.Sprintf(`SELECT id, timestamp, filename, original_filename, source, destination, tier, confidence, tags, action, reasoning, corrected 
			  FROM sort_log %s ORDER BY id DESC LIMIT ?`, where)
	args = append(args, n)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LogEntry
	for rows.Next() {
		var e LogEntry
		var origName, reason sql.NullString
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Filename, &origName, &e.Source, &e.Destination, &e.Tier, &e.Confidence, &e.Tags, &e.Action, &reason, &e.Corrected); err != nil {
			return nil, err
		}
		e.OriginalFilename = origName.String
		e.Reasoning = reason.String
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *Store) MarkCorrected(id int, newDest string, folderMatch string) error {
	// 1. Update the log entry
	_, err := s.db.Exec("UPDATE sort_log SET corrected = 1, destination = ? WHERE id = ?", newDest, id)
	if err != nil {
		return err
	}

	// 2. Fetch tags to update affinities
	var tagsJSON string
	err = s.db.QueryRow("SELECT tags FROM sort_log WHERE id = ?", id).Scan(&tagsJSON)
	if err != nil {
		return err
	}

	var tags []string
	if err := json.Unmarshal([]byte(tagsJSON), &tags); err != nil {
		// If tags aren't valid JSON (old entries), skip affinity update
		return nil
	}

	// 3. Upsert affinities for each tag
	for _, tag := range tags {
		query := `INSERT INTO affinities (tag, folder, weight) 
				  VALUES (?, ?, 1.0) 
				  ON CONFLICT(tag, folder) DO UPDATE SET weight = weight + 1.0`
		_, _ = s.db.Exec(query, tag, folderMatch)
	}

	return nil
}

func (s *Store) GetAffinities(tags []string) (map[string]float64, error) {
	affinities := make(map[string]float64)

	if len(tags) == 0 {
		// General learning: top preferences
		rows, err := s.db.Query("SELECT tag, folder, weight FROM affinities ORDER BY weight DESC LIMIT 20")
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var tag, folder string
			var weight float64
			if err := rows.Scan(&tag, &folder, &weight); err == nil {
				affinities[fmt.Sprintf("%s->%s", tag, folder)] = weight
			}
		}
		return affinities, nil
	}

	for _, tag := range tags {
		rows, err := s.db.Query("SELECT folder, weight FROM affinities WHERE tag = ?", tag)
		if err != nil {
			continue
		}
		for rows.Next() {
			var folder string
			var weight float64
			if err := rows.Scan(&folder, &weight); err == nil {
				affinities[folder] += weight
			}
		}
		rows.Close()
	}
	return affinities, nil
}

func (s *Store) AggregatedTags() ([]TagStat, error) {
	query := "SELECT tags FROM sort_log WHERE tags IS NOT NULL AND tags != '[]'"
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var tagsJSON string
		if err := rows.Scan(&tagsJSON); err != nil {
			continue
		}
		var tags []string
		if err := json.Unmarshal([]byte(tagsJSON), &tags); err == nil {
			for _, t := range tags {
				counts[t]++
			}
		}
	}

	var stats []TagStat
	for t, c := range counts {
		stats = append(stats, TagStat{Tag: t, Count: c})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	return stats, nil
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
