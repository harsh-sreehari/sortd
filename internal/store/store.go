package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

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
	OriginalSource   string
}

type Decision struct {
	File             string
	OriginalFilename string
	OriginalSource   string
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
	_, _ = db.Exec("ALTER TABLE sort_log ADD COLUMN original_source TEXT NOT NULL DEFAULT ''")
	_, _ = db.Exec("ALTER TABLE folder_index ADD COLUMN schema TEXT")

	return &Store{db: db}, nil
}

func (s *Store) LogDecision(d Decision) error {
	query := `INSERT INTO sort_log (timestamp, filename, original_filename, original_source, source, destination, tier, confidence, tags, action, reasoning) 
			  VALUES (datetime('now'), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
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

	_, err := s.db.Exec(query, d.File, orig, d.OriginalSource, d.OriginalSource, d.Destination, d.Tier, d.Confidence, tags, d.Action, d.Reasoning)
	return err
}

func (s *Store) RecentLog(n int) ([]LogEntry, error) {
	query := `SELECT id, timestamp, filename, original_filename, original_source, source, destination, tier, confidence, tags, action, reasoning, corrected 
			  FROM sort_log ORDER BY id DESC LIMIT ?`
	rows, err := s.db.Query(query, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LogEntry
	for rows.Next() {
		var e LogEntry
		var origName, reason, origSrc sql.NullString
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Filename, &origName, &origSrc, &e.Source, &e.Destination, &e.Tier, &e.Confidence, &e.Tags, &e.Action, &reason, &e.Corrected); err != nil {
			return nil, err
		}
		e.OriginalFilename = origName.String
		e.Reasoning = reason.String
		e.OriginalSource = origSrc.String
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func (s *Store) UnsortedFiles() ([]LogEntry, error) {
	query := `SELECT id, timestamp, filename, original_filename, original_source, source, destination, tier, confidence, tags, action, reasoning, corrected 
			  FROM sort_log WHERE action = 'parked' AND corrected = 0 ORDER BY id ASC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LogEntry
	for rows.Next() {
		var e LogEntry
		var origName, reason, origSrc sql.NullString
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Filename, &origName, &origSrc, &e.Source, &e.Destination, &e.Tier, &e.Confidence, &e.Tags, &e.Action, &reason, &e.Corrected); err != nil {
			return nil, err
		}
		e.OriginalFilename = origName.String
		e.Reasoning = reason.String
		e.OriginalSource = origSrc.String
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

// GetUndoableMoves returns the last N entries that were either moved or parked.
func (s *Store) GetUndoableMoves(n int) ([]LogEntry, error) {
	query := `SELECT id, timestamp, filename, original_filename, original_source, source, destination, tier, confidence, tags, action, reasoning, corrected 
			  FROM sort_log WHERE action IN ('moved', 'parked') ORDER BY timestamp DESC, id DESC LIMIT ?`
	rows, err := s.db.Query(query, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LogEntry
	for rows.Next() {
		var e LogEntry
		var origName, reason, origSrc sql.NullString
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Filename, &origName, &origSrc, &e.Source, &e.Destination, &e.Tier, &e.Confidence, &e.Tags, &e.Action, &reason, &e.Corrected); err != nil {
			return nil, err
		}
		e.OriginalFilename = origName.String
		e.Reasoning = reason.String
		e.OriginalSource = origSrc.String
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func (s *Store) DeleteLogEntry(id int) error {
	_, err := s.db.Exec("DELETE FROM sort_log WHERE id = ?", id)
	return err
}

func (s *Store) SearchLog(n int, offset int, filters map[string]string) ([]LogEntry, error) {
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
		where += " AND DATE(timestamp) = DATE('now')"
	}
	if val, ok := filters["since"]; ok && val != "" {
		if dur, err := parseHumanDuration(val); err == nil {
			cutoff := time.Now().Add(-dur).Format("2006-01-02 15:04:05")
			where += " AND timestamp >= ?"
			args = append(args, cutoff)
		}
	}
	if val, ok := filters["query"]; ok && val != "" {
		where += " AND (original_filename LIKE ? OR destination LIKE ? OR reasoning LIKE ?)"
		pattern := "%" + val + "%"
		args = append(args, pattern, pattern, pattern)
	}

	query := fmt.Sprintf(`SELECT id, timestamp, filename, original_filename, original_source, source, destination, tier, confidence, tags, action, reasoning, corrected 
			  FROM sort_log %s ORDER BY id DESC LIMIT ? OFFSET ?`, where)
	args = append(args, n, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LogEntry
	for rows.Next() {
		var e LogEntry
		var origName, reason, origSrc sql.NullString
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Filename, &origName, &origSrc, &e.Source, &e.Destination, &e.Tier, &e.Confidence, &e.Tags, &e.Action, &reason, &e.Corrected); err != nil {
			return nil, err
		}
		e.OriginalFilename = origName.String
		e.Reasoning = reason.String
		e.OriginalSource = origSrc.String
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
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

		if err := rows.Err(); err != nil {
			return nil, err
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

func (s *Store) Prune(roots []string, dryRun bool) (int, int, error) {
	// 0. Safety check: ensure at least one root is reachable to avoid wiping on empty disk
	reachable := false
	for _, r := range roots {
		if _, err := os.Stat(r); err == nil {
			reachable = true
			break
		}
	}
	if !reachable && len(roots) > 0 {
		return 0, 0, fmt.Errorf("none of the root folders are reachable. Aborting prune to protect database")
	}

	prunedIndex := 0
	prunedLog := 0

	// 1. Prune folder_index
	rows, _ := s.db.Query("SELECT path FROM folder_index")
	var toDeleteIndex []string
	for rows.Next() {
		var p string
		rows.Scan(&p)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			toDeleteIndex = append(toDeleteIndex, p)
		}
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, 0, err
	}
	rows.Close()

	for _, p := range toDeleteIndex {
		if !dryRun {
			s.db.Exec("DELETE FROM folder_index WHERE path = ?", p)
		}
		prunedIndex++
	}

	// 2. Prune sort_log (parking files)
	rows, _ = s.db.Query("SELECT id, destination FROM sort_log")
	var toDeleteLog []int
	for rows.Next() {
		var id int
		var dest string
		rows.Scan(&id, &dest)
		if _, err := os.Stat(dest); os.IsNotExist(err) {
			toDeleteLog = append(toDeleteLog, id)
		}
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return prunedIndex, prunedLog, err
	}
	rows.Close()

	for _, id := range toDeleteLog {
		if !dryRun {
			s.db.Exec("DELETE FROM sort_log WHERE id = ?", id)
		}
		prunedLog++
	}

	return prunedIndex, prunedLog, nil
}

func (s *Store) AggregatedTags(folder string) ([]TagStat, error) {
	query := "SELECT tags FROM sort_log WHERE tags IS NOT NULL AND tags != '[]'"
	var args []interface{}
	if folder != "" {
		query += " AND destination LIKE ?"
		args = append(args, folder+"%")
	}
	rows, err := s.db.Query(query, args...)
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
		if err := json.Unmarshal([]byte(tagsJSON), &tags); err != nil {
			// Legacy fallback: entries written before JSON encoding may be
			// comma-separated plain strings (e.g. "academic,pdf,report").
			// Split and trim rather than silently discarding the entry.
			for _, t := range strings.Split(tagsJSON, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
		}
		for _, t := range tags {
			if t != "" {
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

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}

func (s *Store) GetFolderCache(path string) (int64, string, string, bool) {
	var mtime int64
	var keywords, schema string
	err := s.db.QueryRow("SELECT mtime, keywords, schema FROM folder_cache WHERE path = ?", path).Scan(&mtime, &keywords, &schema)
	if err != nil {
		return 0, "", "", false
	}
	return mtime, keywords, schema, true
}

func (s *Store) UpdateFolderCache(path string, mtime int64, keywords string, schema string) error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO folder_cache (path, mtime, keywords, schema) VALUES (?, ?, ?, ?)", path, mtime, keywords, schema)
	return err
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

type GlobalStatus struct {
	TotalMoved     int
	TotalParked    int
	TotalCorrected int
	TotalFolders   int
}

func (s *Store) GetStatusMetrics() (GlobalStatus, error) {
	var gs GlobalStatus

	// Total Moved
	err := s.db.QueryRow("SELECT COUNT(*) FROM sort_log WHERE action = 'moved'").Scan(&gs.TotalMoved)
	if err != nil {
		return gs, err
	}

	// Total Parked
	err = s.db.QueryRow("SELECT COUNT(*) FROM sort_log WHERE action = 'parked'").Scan(&gs.TotalParked)
	if err != nil {
		return gs, err
	}

	// Total Corrected
	err = s.db.QueryRow("SELECT COUNT(*) FROM sort_log WHERE corrected = 1").Scan(&gs.TotalCorrected)
	if err != nil {
		return gs, err
	}

	// Total Folders Indexed
	err = s.db.QueryRow("SELECT COUNT(*) FROM folder_index").Scan(&gs.TotalFolders)
	if err != nil {
		return gs, err
	}

	return gs, nil
}

func parseHumanDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	if strings.HasSuffix(s, "w") {
		weeks, err := strconv.Atoi(strings.TrimSuffix(s, "w"))
		if err != nil {
			return 0, err
		}
		return time.Duration(weeks) * 7 * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
