package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open opens a SQLite database at the given path and initializes the schema.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	// ponyail: one-shot DDL, no migration framework needed yet
	schema := `
CREATE TABLE IF NOT EXISTS repos (
	id INTEGER PRIMARY KEY,
	owner TEXT NOT NULL,
	name TEXT NOT NULL,
	token_enc BLOB,
	ssh_key_enc BLOB,
	poll_interval_sec INTEGER DEFAULT 60,
	active INTEGER DEFAULT 1,
	UNIQUE(owner, name)
);

CREATE TABLE IF NOT EXISTS issues (
	id INTEGER PRIMARY KEY,
	repo_id INTEGER REFERENCES repos(id),
	issue_number INTEGER NOT NULL,
	title TEXT,
	type TEXT DEFAULT 'AUTO',
	status TEXT DEFAULT 'pending',
	depends_on TEXT DEFAULT '[]',
	UNIQUE(repo_id, issue_number)
);

CREATE TABLE IF NOT EXISTS runs (
	id INTEGER PRIMARY KEY,
	issue_id INTEGER REFERENCES issues(id),
	iteration INTEGER DEFAULT 1,
	outcome TEXT,
	started_at TEXT,
	finished_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_issues_repo_id ON issues(repo_id);
CREATE INDEX IF NOT EXISTS idx_runs_issue_id ON runs(issue_id);
`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return db, nil
}
