package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// ---------------------------------------------------------------------------
// Livecoding tasks
// ---------------------------------------------------------------------------

// Expected is the reference result set for a SQL task. All cell values are
// stored as strings; the correctness check is performed client-side (a
// deliberate MVP trade-off).
type Expected struct {
	Columns      []string   `json:"columns"`
	Rows         [][]string `json:"rows"`
	OrderMatters bool       `json:"order_matters"`
}

// CodingTask is the full stored representation of a livecoding task.
type CodingTask struct {
	ID           int64
	Slug         string
	Kind         string // go|sql
	Title        string
	Difficulty   string
	Tags         []string
	StatementMD  string
	StarterCode  string
	Hints        []string
	SolutionMD   string
	TimeLimitSec int
	Race         bool
	TestCode     string
	SchemaSQL    string
	SeedSQL      string
	Expected     Expected
	Position     int
}

// CodingTaskListItem is a lightweight task row enriched with per-user status.
type CodingTaskListItem struct {
	Slug       string   `json:"slug"`
	Kind       string   `json:"kind"`
	Title      string   `json:"title"`
	Difficulty string   `json:"difficulty"`
	Tags       []string `json:"tags"`
	Status     string   `json:"status"` // new|attempted|solved
	GaveUp     bool     `json:"gave_up"`
	DueAt      *string  `json:"due_at"`
	Due        bool     `json:"due"` // true when due_at is set and <= now
}

// TaskState is the persisted per-user state for a single coding task.
type TaskState struct {
	Status     string // attempted|solved
	LastCode   string
	SolvedAt   *string
	UpdatedAt  string
	DueAt      *string // RFC3339, nil until first solve; "25/5" re-solve reminder
	GaveUp     bool
	SolveCount int
}

// UpsertCodingTask inserts or updates a coding task keyed by slug.
func (s *Store) UpsertCodingTask(ctx context.Context, t CodingTask) error {
	if t.Tags == nil {
		t.Tags = []string{}
	}
	if t.Hints == nil {
		t.Hints = []string{}
	}
	if t.Expected.Columns == nil {
		t.Expected.Columns = []string{}
	}
	if t.Expected.Rows == nil {
		t.Expected.Rows = [][]string{}
	}
	tags, err := json.Marshal(t.Tags)
	if err != nil {
		return err
	}
	hints, err := json.Marshal(t.Hints)
	if err != nil {
		return err
	}
	expected, err := json.Marshal(t.Expected)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO coding_tasks (slug, kind, title, difficulty, tags, statement_md, starter_code,
                          hints, solution_md, time_limit_sec, race, test_code, schema_sql,
                          seed_sql, expected, position)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(slug) DO UPDATE SET
    kind           = excluded.kind,
    title          = excluded.title,
    difficulty     = excluded.difficulty,
    tags           = excluded.tags,
    statement_md   = excluded.statement_md,
    starter_code   = excluded.starter_code,
    hints          = excluded.hints,
    solution_md    = excluded.solution_md,
    time_limit_sec = excluded.time_limit_sec,
    race           = excluded.race,
    test_code      = excluded.test_code,
    schema_sql     = excluded.schema_sql,
    seed_sql       = excluded.seed_sql,
    expected       = excluded.expected,
    position       = excluded.position`,
		t.Slug, t.Kind, t.Title, t.Difficulty, string(tags), t.StatementMD, t.StarterCode,
		string(hints), t.SolutionMD, t.TimeLimitSec, boolToInt(t.Race), t.TestCode, t.SchemaSQL,
		t.SeedSQL, string(expected), t.Position)
	return err
}

// GetCodingTaskBySlug returns a single coding task, or ErrNotFound.
func (s *Store) GetCodingTaskBySlug(ctx context.Context, slug string) (CodingTask, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, slug, kind, title, difficulty, tags, statement_md, starter_code, hints,
       solution_md, time_limit_sec, race, test_code, schema_sql, seed_sql, expected, position
FROM coding_tasks WHERE slug = ?`, slug)

	var t CodingTask
	var tags, hints, expected string
	var race int
	err := row.Scan(&t.ID, &t.Slug, &t.Kind, &t.Title, &t.Difficulty, &tags, &t.StatementMD,
		&t.StarterCode, &hints, &t.SolutionMD, &t.TimeLimitSec, &race, &t.TestCode,
		&t.SchemaSQL, &t.SeedSQL, &expected, &t.Position)
	if errors.Is(err, sql.ErrNoRows) {
		return CodingTask{}, ErrNotFound
	}
	if err != nil {
		return CodingTask{}, err
	}
	t.Race = race != 0
	_ = json.Unmarshal([]byte(tags), &t.Tags)
	_ = json.Unmarshal([]byte(hints), &t.Hints)
	_ = json.Unmarshal([]byte(expected), &t.Expected)
	if t.Tags == nil {
		t.Tags = []string{}
	}
	if t.Hints == nil {
		t.Hints = []string{}
	}
	if t.Expected.Columns == nil {
		t.Expected.Columns = []string{}
	}
	if t.Expected.Rows == nil {
		t.Expected.Rows = [][]string{}
	}
	return t, nil
}

// ListCodingTasks returns every coding task annotated with the given user's
// status. Go tasks are listed before SQL tasks, each block ordered by
// position. now (RFC3339) is used to compute the "due" ("25/5" re-solve is
// overdue) flag on each item.
func (s *Store) ListCodingTasks(ctx context.Context, userID int64, now string) ([]CodingTaskListItem, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT t.slug, t.kind, t.title, t.difficulty, t.tags, st.status, st.gave_up, st.due_at
FROM coding_tasks t
LEFT JOIN user_task_state st ON st.task_id = t.id AND st.user_id = ?
ORDER BY CASE t.kind WHEN 'go' THEN 0 ELSE 1 END, t.position, t.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []CodingTaskListItem{}
	for rows.Next() {
		var it CodingTaskListItem
		var tags string
		var status, dueAt sql.NullString
		var gaveUp sql.NullInt64
		if err := rows.Scan(&it.Slug, &it.Kind, &it.Title, &it.Difficulty, &tags, &status, &gaveUp, &dueAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(tags), &it.Tags)
		if it.Tags == nil {
			it.Tags = []string{}
		}
		if status.Valid {
			it.Status = status.String
		} else {
			it.Status = "new"
		}
		it.GaveUp = gaveUp.Valid && gaveUp.Int64 != 0
		if dueAt.Valid {
			d := dueAt.String
			it.DueAt = &d
			it.Due = d <= now
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// GetTaskState returns the per-user state for a coding task, or ErrNotFound.
func (s *Store) GetTaskState(ctx context.Context, userID, taskID int64) (TaskState, error) {
	var st TaskState
	var solvedAt, dueAt sql.NullString
	var gaveUp int
	err := s.db.QueryRowContext(ctx, `
SELECT status, last_code, solved_at, updated_at, due_at, gave_up, solve_count
FROM user_task_state WHERE user_id = ? AND task_id = ?`, userID, taskID).
		Scan(&st.Status, &st.LastCode, &solvedAt, &st.UpdatedAt, &dueAt, &gaveUp, &st.SolveCount)
	if errors.Is(err, sql.ErrNoRows) {
		return TaskState{}, ErrNotFound
	}
	if err != nil {
		return TaskState{}, err
	}
	if solvedAt.Valid {
		v := solvedAt.String
		st.SolvedAt = &v
	}
	if dueAt.Valid {
		v := dueAt.String
		st.DueAt = &v
	}
	st.GaveUp = gaveUp != 0
	return st, nil
}

// RecordCodingRun (Go tasks) atomically upserts the user's task state and
// appends a run-log entry. A "solved" status is sticky: once solved, a later
// failing run keeps the task solved and preserves the original solved_at. A
// passing run additionally advances the "25/5" re-solve cycle (see
// upsertTaskState).
func (s *Store) RecordCodingRun(ctx context.Context, userID, taskID int64, code string, passed bool, at string) error {
	status := "attempted"
	var solvedAt any
	if passed {
		status = "solved"
		solvedAt = at
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := upsertTaskState(ctx, tx, userID, taskID, status, code, solvedAt, at, passed); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO task_run_log (user_id, task_id, passed, ran_at) VALUES (?, ?, ?, ?)`,
		userID, taskID, boolToInt(passed), at); err != nil {
		return err
	}
	return tx.Commit()
}

// MarkCodingSolved (SQL tasks) marks a task solved for the user and stores the
// final code. Correctness is validated client-side for SQL tasks. Every call
// is a successful solve, so it always advances the "25/5" re-solve cycle.
func (s *Store) MarkCodingSolved(ctx context.Context, userID, taskID int64, code, at string) error {
	return upsertTaskState(ctx, s.db, userID, taskID, "solved", code, at, at, true)
}

// GiveUpCodingTask records that the user gave up on a task once its hints ran
// out: gave_up is set and the status becomes (or stays) "attempted". Callers
// must reject already-solved tasks before calling this — an already-solved
// task can't be given up, but this stays defensively sticky ('solved' status
// wins) in case of a race.
func (s *Store) GiveUpCodingTask(ctx context.Context, userID, taskID int64, at string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO user_task_state (user_id, task_id, status, last_code, solved_at, updated_at, due_at, gave_up, solve_count)
VALUES (?, ?, 'attempted', '', NULL, ?, NULL, 1, 0)
ON CONFLICT(user_id, task_id) DO UPDATE SET
    status     = CASE WHEN user_task_state.status = 'solved' THEN 'solved' ELSE 'attempted' END,
    gave_up    = 1,
    updated_at = excluded.updated_at`,
		userID, taskID, at)
	return err
}

// execer is satisfied by both *sql.DB and *sql.Tx.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// solveInterval returns how far into the future due_at should be pushed for
// the given (post-increment) solve count — the "25/5" re-solve cadence: the
// 1st solve waits a week, the 2nd three weeks, the 3rd and later two months.
func solveInterval(solveCount int) time.Duration {
	switch {
	case solveCount <= 1:
		return 7 * 24 * time.Hour
	case solveCount == 2:
		return 21 * 24 * time.Hour
	default:
		return 60 * 24 * time.Hour
	}
}

// currentSolveCount returns the existing solve_count for a (user, task) pair,
// or 0 if no state row exists yet.
func currentSolveCount(ctx context.Context, ex execer, userID, taskID int64) (int, error) {
	var n int
	err := ex.QueryRowContext(ctx,
		`SELECT solve_count FROM user_task_state WHERE user_id = ? AND task_id = ?`, userID, taskID).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return n, err
}

// upsertTaskState writes the per-user task state. When advance is true (the
// call represents a successful solve — a passing Go run or a SQL "solved"
// submission) it also drives the "25/5" re-solve cycle: solve_count is
// incremented, due_at is pushed out by solveInterval(solve_count) from
// updatedAt, and gave_up is cleared, since solving it again (even from
// memory, after giving up) closes that cycle. A non-advancing call (a
// failing run) leaves solve_count, due_at and gave_up untouched.
func upsertTaskState(ctx context.Context, ex execer, userID, taskID int64, status, code string, solvedAt any, updatedAt string, advance bool) error {
	solveCount := 0
	var dueAt any
	if advance {
		prev, err := currentSolveCount(ctx, ex, userID, taskID)
		if err != nil {
			return err
		}
		solveCount = prev + 1

		base, err := time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return err
		}
		dueAt = base.Add(solveInterval(solveCount)).Format(time.RFC3339)
	}

	_, err := ex.ExecContext(ctx, `
INSERT INTO user_task_state (user_id, task_id, status, last_code, solved_at, updated_at, due_at, gave_up, solve_count)
VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?)
ON CONFLICT(user_id, task_id) DO UPDATE SET
    status      = CASE WHEN excluded.status = 'solved' OR user_task_state.status = 'solved'
                       THEN 'solved' ELSE excluded.status END,
    last_code   = excluded.last_code,
    solved_at   = COALESCE(user_task_state.solved_at, excluded.solved_at),
    updated_at  = excluded.updated_at,
    due_at      = CASE WHEN ? THEN excluded.due_at ELSE user_task_state.due_at END,
    gave_up     = CASE WHEN ? THEN 0 ELSE user_task_state.gave_up END,
    solve_count = CASE WHEN ? THEN excluded.solve_count ELSE user_task_state.solve_count END`,
		userID, taskID, status, code, solvedAt, updatedAt, dueAt, solveCount,
		boolToInt(advance), boolToInt(advance), boolToInt(advance))
	return err
}

// CountCodingTasks returns the total number of coding tasks in the bank.
func (s *Store) CountCodingTasks(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM coding_tasks`).Scan(&n)
	return n, err
}

// CountSolvedCodingTasks returns how many coding tasks the user has solved.
func (s *Store) CountSolvedCodingTasks(ctx context.Context, userID int64) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_task_state WHERE user_id = ? AND status = 'solved'`, userID).Scan(&n)
	return n, err
}

// CountDueCodingTasks returns how many of the user's solved coding tasks are
// due for a "25/5" re-solve (due_at is set and <= now).
func (s *Store) CountDueCodingTasks(ctx context.Context, userID int64, now string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_task_state
		 WHERE user_id = ? AND status = 'solved' AND due_at IS NOT NULL AND due_at <= ?`, userID, now).Scan(&n)
	return n, err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
