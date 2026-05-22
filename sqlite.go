package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func OpenProfileDB(profileDir string) (*sql.DB, error) {
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		return nil, fmt.Errorf("Create profile dir: %w", err)
	}

	dbPath := filepath.Join(profileDir, "data.sqlite3")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Good defaults for desktop apps
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("Pragma foreign_keys: %w", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("Pragma journal_mode: %w", err)
	}

	if err := initSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func initSchema(db *sql.DB) error {
	// Use PRAGMA user_version for schema migrations.
	var v int
	if err := db.QueryRow(`PRAGMA user_version;`).Scan(&v); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}

	if v == 0 {
		// Initial schema
		_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS entries (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	body TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
	updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

PRAGMA user_version = 1;
`)
		if err != nil {
			return fmt.Errorf("Create schema v1: %w", err)
		}
		v = 1
	}

	// Future migrations would go here:
	// if v == 1 { ... set user_version = 2 }
	if v == 1 {
		_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS elabftw_instances (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	site_url TEXT NOT NULL UNIQUE,
	api_key TEXT NOT NULL,
	verify_tls INTEGER NOT NULL DEFAULT 1,
	created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
	updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE IF NOT EXISTS local2remote (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	instance INTEGER NOT NULL,
	remote_id INTEGER NOT NULL,
	local_id INTEGER NOT NULL,
	type TEXT NOT NULL CHECK (type IN ('experiment', 'resource', 'template')),
	created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
	updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),

	FOREIGN KEY (instance) REFERENCES elabftw_instances(id) ON DELETE CASCADE,
	UNIQUE(instance, local_id, type),
	UNIQUE(instance, remote_id, type)
);

CREATE INDEX IF NOT EXISTS idx_local2remote_local
ON local2remote(local_id, type);

CREATE INDEX IF NOT EXISTS idx_local2remote_remote
ON local2remote(instance, remote_id, type);

PRAGMA user_version = 2;
`)
		if err != nil {
			return fmt.Errorf("Create schema v2: %w", err)
		}
		v = 2
	}

	return nil
}
