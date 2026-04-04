package store

const (
	SortLogSchema = `
CREATE TABLE IF NOT EXISTS sort_log (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp         TEXT NOT NULL,
    filename          TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    source            TEXT NOT NULL,
    destination       TEXT NOT NULL,
    tier              INTEGER NOT NULL,
    confidence        REAL NOT NULL,
    tags              TEXT,
    action            TEXT NOT NULL,
    reasoning         TEXT,
    corrected         INTEGER DEFAULT 0,
    original_source   TEXT NOT NULL DEFAULT ''
);`

	FolderIndexSchema = `
CREATE TABLE IF NOT EXISTS folder_index (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    path     TEXT UNIQUE NOT NULL,
    keywords TEXT NOT NULL,
    depth    INTEGER NOT NULL,
    parent   TEXT,
    schema   TEXT
);`

	AffinitiesSchema = `
CREATE TABLE IF NOT EXISTS affinities (
    tag     TEXT NOT NULL,
    folder  TEXT NOT NULL,
    weight  REAL NOT NULL DEFAULT 1.0,
    PRIMARY KEY (tag, folder)
);`

	FolderCacheSchema = `
CREATE TABLE IF NOT EXISTS folder_cache (
    path     TEXT PRIMARY KEY,
    mtime    INTEGER NOT NULL,
    keywords TEXT NOT NULL,
    schema   TEXT
);`
)

var Schemas = []string{SortLogSchema, FolderIndexSchema, AffinitiesSchema, FolderCacheSchema}
