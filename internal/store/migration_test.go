package store

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

// TestOpen_MigratesPreExistingDatabase verifies that Open() applies the
// idempotent ALTER TABLE steps against a database created before the
// due_at/gave_up/solve_count columns existed, without erroring, and that
// running it twice (columns already present) is still a no-op.
func TestOpen_MigratesPreExistingDatabase(t *testing.T) {
	path := t.TempDir() + "/old.db"
	const oldSchema = `
CREATE TABLE IF NOT EXISTS user_task_state (
    user_id    INTEGER NOT NULL,
    task_id    INTEGER NOT NULL,
    status     TEXT NOT NULL,
    last_code  TEXT NOT NULL,
    solved_at  TEXT,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (user_id, task_id)
);
INSERT INTO user_task_state (user_id, task_id, status, last_code, solved_at, updated_at)
VALUES (1, 1, 'solved', 'old code', '2025-01-01T00:00:00Z', '2025-01-01T00:00:00Z');
`
	seedDB, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("open seed db: %v", err)
	}
	if _, err := seedDB.Exec(oldSchema); err != nil {
		t.Fatalf("seed old schema: %v", err)
	}
	if err := seedDB.Close(); err != nil {
		t.Fatalf("close seed db: %v", err)
	}

	// First Open: must ALTER TABLE ADD COLUMN successfully.
	st1, err := Open(path)
	if err != nil {
		t.Fatalf("first Open() on pre-existing db: %v", err)
	}
	got, err := st1.GetTaskState(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("get task state after migration: %v", err)
	}
	if got.Status != "solved" || got.LastCode != "old code" {
		t.Errorf("expected pre-existing row preserved, got %+v", got)
	}
	if got.GaveUp {
		t.Errorf("expected gave_up to default to false, got true")
	}
	if got.SolveCount != 0 {
		t.Errorf("expected solve_count to default to 0, got %d", got.SolveCount)
	}
	if got.DueAt != nil {
		t.Errorf("expected due_at to default to nil, got %v", *got.DueAt)
	}
	st1.Close()

	// Second Open: columns already exist, must be a clean no-op (this is the
	// "duplicate column name" error-swallowing path).
	st2, err := Open(path)
	if err != nil {
		t.Fatalf("second Open() on already-migrated db: %v", err)
	}
	st2.Close()

	os.Remove(path)
}

// TestOpen_MigratesPreExistingUsersTable verifies that Open() applies the
// idempotent ALTER TABLE steps that add name/interview_date to a users
// table created before those columns existed, without erroring, and that
// running it twice (columns already present) is still a no-op.
func TestOpen_MigratesPreExistingUsersTable(t *testing.T) {
	path := t.TempDir() + "/old_users.db"
	const oldSchema = `
CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    email         TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TEXT NOT NULL
);
INSERT INTO users (email, password_hash, created_at)
VALUES ('old@example.com', 'hash', '2025-01-01T00:00:00Z');
`
	seedDB, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("open seed db: %v", err)
	}
	if _, err := seedDB.Exec(oldSchema); err != nil {
		t.Fatalf("seed old schema: %v", err)
	}
	if err := seedDB.Close(); err != nil {
		t.Fatalf("close seed db: %v", err)
	}

	// First Open: must ALTER TABLE ADD COLUMN successfully.
	st1, err := Open(path)
	if err != nil {
		t.Fatalf("first Open() on pre-existing users table: %v", err)
	}
	u, err := st1.GetUserByEmail(context.Background(), "old@example.com")
	if err != nil {
		t.Fatalf("get user after migration: %v", err)
	}
	got, err := st1.GetUserByID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("get user by id after migration: %v", err)
	}
	if got.Name != nil {
		t.Errorf("expected name to default to nil, got %v", *got.Name)
	}
	if got.InterviewDate != nil {
		t.Errorf("expected interview_date to default to nil, got %v", *got.InterviewDate)
	}
	st1.Close()

	// Second Open: columns already exist, must be a clean no-op.
	st2, err := Open(path)
	if err != nil {
		t.Fatalf("second Open() on already-migrated users table: %v", err)
	}
	st2.Close()

	os.Remove(path)
}
