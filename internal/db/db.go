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
	body TEXT,
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

	// ponyail: migration — add columns that may be missing in existing databases
	// Ignore errors if column already exists (SQLite doesn't support IF NOT EXISTS for ALTER TABLE)
	migrations := []string{
		"ALTER TABLE issues ADD COLUMN body TEXT",
	}
	for _, m := range migrations {
		db.Exec(m) // best-effort, fails silently if column exists
	}

	return db, nil
}

// Repo represents a watched GitHub repository.
type Repo struct {
	ID              int64
	Owner           string
	Name            string
	TokenEnc        []byte
	PollIntervalSec int
	Active          bool
}

// Issue represents a mirrored GitHub issue.
type Issue struct {
	ID          int64
	RepoID      int64
	IssueNumber int
	Title       string
	Body        string
	Type        string
	Status      string
	DependsOn   string
}

// ListActiveRepos returns all active repos from the database.
func ListActiveRepos(db *sql.DB) ([]Repo, error) {
	rows, err := db.Query("SELECT id, owner, name, token_enc, poll_interval_sec, active FROM repos WHERE active = 1")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []Repo
	for rows.Next() {
		var r Repo
		if err := rows.Scan(&r.ID, &r.Owner, &r.Name, &r.TokenEnc, &r.PollIntervalSec, &r.Active); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

// UpsertIssue inserts or updates an issue in the database.
func UpsertIssue(db *sql.DB, issue Issue) error {
	_, err := db.Exec(`
		INSERT INTO issues (repo_id, issue_number, title, body, type, status, depends_on)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repo_id, issue_number) DO UPDATE SET
			title = excluded.title,
			body = excluded.body,
			type = excluded.type,
			status = excluded.status,
			depends_on = excluded.depends_on
	`, issue.RepoID, issue.IssueNumber, issue.Title, issue.Body, issue.Type, issue.Status, issue.DependsOn)
	return err
}

// GetPendingIssue returns the first pending AUTO issue for a repo, oldest first.
func GetPendingIssue(db *sql.DB, repoID int64) (*Issue, error) {
	var i Issue
	err := db.QueryRow(`
		SELECT id, repo_id, issue_number, title, body, type, status, depends_on
		FROM issues
		WHERE repo_id = ? AND status = 'pending' AND type = 'AUTO'
		ORDER BY issue_number ASC
		LIMIT 1
	`, repoID).Scan(&i.ID, &i.RepoID, &i.IssueNumber, &i.Title, &i.Body, &i.Type, &i.Status, &i.DependsOn)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &i, nil
}

// AreDepsDone checks whether all dependency issue numbers have status 'done'.
func AreDepsDone(db *sql.DB, repoID int64, depNumbers []int) (bool, error) {
	if len(depNumbers) == 0 {
		return true, nil
	}
	for _, n := range depNumbers {
		var status string
		err := db.QueryRow("SELECT status FROM issues WHERE repo_id = ? AND issue_number = ?", repoID, n).Scan(&status)
		if err == sql.ErrNoRows {
			return false, nil // dep not yet mirrored
		}
		if err != nil {
			return false, err
		}
		if status != "done" {
			return false, nil
		}
	}
	return true, nil
}

// UpdateIssueStatus sets the status of an issue.
func UpdateIssueStatus(db *sql.DB, issueID int64, status string) error {
	_, err := db.Exec("UPDATE issues SET status = ? WHERE id = ?", status, issueID)
	return err
}

// InsertRun records an execution run for an issue.
// startedAt and finishedAt are ISO 8601 strings.
func InsertRun(db *sql.DB, issueID int64, iteration int, outcome, startedAt, finishedAt string) error {
	_, err := db.Exec(
		"INSERT INTO runs (issue_id, iteration, outcome, started_at, finished_at) VALUES (?, ?, ?, ?, ?)",
		issueID, iteration, outcome, startedAt, finishedAt,
	)
	return err
}
