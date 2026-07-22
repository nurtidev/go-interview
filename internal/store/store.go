// Package store is the SQLite persistence layer. It uses the pure-Go
// modernc.org/sqlite driver (no cgo) with the standard database/sql package.
package store

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// Sentinel errors returned by the store.
var (
	ErrNotFound   = errors.New("not found")
	ErrEmailTaken = errors.New("email already registered")
)

// Store wraps a *sql.DB and exposes the query methods used by the API.
type Store struct {
	db *sql.DB
}

// Open opens (creating if necessary) the SQLite database at path and applies
// the embedded schema.
func Open(path string) (*Store, error) {
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	if err := migrateUserTaskStateColumns(db); err != nil {
		return nil, fmt.Errorf("migrate user_task_state columns: %w", err)
	}
	if err := migrateUserProfileColumns(db); err != nil {
		return nil, fmt.Errorf("migrate users columns: %w", err)
	}
	return &Store{db: db}, nil
}

// migrateUserTaskStateColumns adds the "25/5" give-up/re-solve columns to
// user_task_state for databases created before they existed. schema.sql uses
// CREATE TABLE IF NOT EXISTS, which never alters an already-existing table,
// so pre-existing local databases need these idempotent ALTER TABLE steps.
// Each statement is a no-op once its column already exists.
func migrateUserTaskStateColumns(db *sql.DB) error {
	stmts := []string{
		`ALTER TABLE user_task_state ADD COLUMN due_at TEXT`,
		`ALTER TABLE user_task_state ADD COLUMN gave_up INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE user_task_state ADD COLUMN solve_count INTEGER NOT NULL DEFAULT 0`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil && !isDuplicateColumn(err) {
			return err
		}
	}
	return nil
}

// migrateUserProfileColumns adds the profile columns (name, interview_date)
// to users for databases created before they existed. schema.sql uses
// CREATE TABLE IF NOT EXISTS, which never alters an already-existing table,
// so pre-existing local databases need these idempotent ALTER TABLE steps.
// Each statement is a no-op once its column already exists.
func migrateUserProfileColumns(db *sql.DB) error {
	stmts := []string{
		`ALTER TABLE users ADD COLUMN name TEXT`,
		`ALTER TABLE users ADD COLUMN interview_date TEXT`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil && !isDuplicateColumn(err) {
			return err
		}
	}
	return nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// isDuplicateColumn reports whether err is modernc.org/sqlite's error for an
// ALTER TABLE ADD COLUMN that names a column which already exists — the
// signal that a migration step has already been applied.
func isDuplicateColumn(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate column name")
}

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

// User is a stored account. Name and InterviewDate are nullable profile
// fields set via PATCH /api/me; both are nil until the user fills them in.
type User struct {
	ID            int64
	Email         string
	PasswordHash  string
	CreatedAt     string
	Name          *string
	InterviewDate *string // YYYY-MM-DD
}

// CreateUser inserts a new user. It returns ErrEmailTaken if the email exists.
func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	now := nowRFC3339()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, created_at) VALUES (?, ?, ?)`,
		email, passwordHash, now)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return User{}, err
	}
	return User{ID: id, Email: email, PasswordHash: passwordHash, CreatedAt: now}, nil
}

// GetUserByEmail returns the user with the given email, or ErrNotFound.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE email = ?`, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return u, nil
}

// GetUserByID returns the user with the given id, or ErrNotFound.
func (s *Store) GetUserByID(ctx context.Context, id int64) (User, error) {
	var u User
	var name, interviewDate sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at, name, interview_date FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &name, &interviewDate)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	if name.Valid {
		v := name.String
		u.Name = &v
	}
	if interviewDate.Valid {
		v := interviewDate.String
		u.InterviewDate = &v
	}
	return u, nil
}

// UpdateUserProfile sets the user's name and interview_date (either may be
// nil to clear the column) and returns the updated user. Callers pass the
// full desired value for each field — partial-update semantics (leaving a
// field untouched when absent from the request) are the API layer's job.
func (s *Store) UpdateUserProfile(ctx context.Context, userID int64, name, interviewDate *string) (User, error) {
	var nameArg, dateArg any
	if name != nil {
		nameArg = *name
	}
	if interviewDate != nil {
		dateArg = *interviewDate
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE users SET name = ?, interview_date = ? WHERE id = ?`, nameArg, dateArg, userID); err != nil {
		return User{}, err
	}
	return s.GetUserByID(ctx, userID)
}

// ---------------------------------------------------------------------------
// Questions
// ---------------------------------------------------------------------------

// AnswerLevel is one layered answer for a question.
type AnswerLevel struct {
	Level  string `json:"level"`
	TextMD string `json:"text_md"`
}

// Question is the full stored representation of an interview question.
type Question struct {
	ID           int64
	Slug         string
	Section      string
	Title        string
	Difficulty   string
	Tags         []string
	QuestionMD   string
	AnswerLevels []AnswerLevel
	FollowUps    []string
	Position     int
}

// QuestionListItem is a lightweight question row enriched with per-user state.
type QuestionListItem struct {
	Slug       string   `json:"slug"`
	Title      string   `json:"title"`
	Difficulty string   `json:"difficulty"`
	Tags       []string `json:"tags"`
	Status     string   `json:"status"`
	DueAt      *string  `json:"due_at"`
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanQuestion(sc rowScanner) (Question, error) {
	var q Question
	var tags, levels, fus string
	err := sc.Scan(&q.ID, &q.Slug, &q.Section, &q.Title, &q.Difficulty,
		&tags, &q.QuestionMD, &levels, &fus, &q.Position)
	if errors.Is(err, sql.ErrNoRows) {
		return Question{}, ErrNotFound
	}
	if err != nil {
		return Question{}, err
	}
	_ = json.Unmarshal([]byte(tags), &q.Tags)
	_ = json.Unmarshal([]byte(levels), &q.AnswerLevels)
	_ = json.Unmarshal([]byte(fus), &q.FollowUps)
	if q.Tags == nil {
		q.Tags = []string{}
	}
	if q.AnswerLevels == nil {
		q.AnswerLevels = []AnswerLevel{}
	}
	if q.FollowUps == nil {
		q.FollowUps = []string{}
	}
	return q, nil
}

// UpsertQuestion inserts or updates a question keyed by slug.
func (s *Store) UpsertQuestion(ctx context.Context, q Question) error {
	if q.Tags == nil {
		q.Tags = []string{}
	}
	if q.AnswerLevels == nil {
		q.AnswerLevels = []AnswerLevel{}
	}
	if q.FollowUps == nil {
		q.FollowUps = []string{}
	}
	tags, err := json.Marshal(q.Tags)
	if err != nil {
		return err
	}
	levels, err := json.Marshal(q.AnswerLevels)
	if err != nil {
		return err
	}
	fus, err := json.Marshal(q.FollowUps)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO questions (slug, section, title, difficulty, tags, question_md, answer_levels, follow_ups, position)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(slug) DO UPDATE SET
    section       = excluded.section,
    title         = excluded.title,
    difficulty    = excluded.difficulty,
    tags          = excluded.tags,
    question_md   = excluded.question_md,
    answer_levels = excluded.answer_levels,
    follow_ups    = excluded.follow_ups,
    position      = excluded.position`,
		q.Slug, q.Section, q.Title, q.Difficulty, string(tags), q.QuestionMD, string(levels), string(fus), q.Position)
	return err
}

// GetQuestionBySlug returns a single question, or ErrNotFound.
func (s *Store) GetQuestionBySlug(ctx context.Context, slug string) (Question, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, slug, section, title, difficulty, tags, question_md, answer_levels, follow_ups, position
FROM questions WHERE slug = ?`, slug)
	return scanQuestion(row)
}

// ListQuestionsBySection returns questions in a section ordered by position,
// each annotated with the given user's learning state.
func (s *Store) ListQuestionsBySection(ctx context.Context, userID int64, section string) ([]QuestionListItem, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT q.slug, q.title, q.difficulty, q.tags, s.status, s.due_at
FROM questions q
LEFT JOIN user_question_state s ON s.question_id = q.id AND s.user_id = ?
WHERE q.section = ?
ORDER BY q.position, q.id`, userID, section)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []QuestionListItem{}
	for rows.Next() {
		var it QuestionListItem
		var tags string
		var status, due sql.NullString
		if err := rows.Scan(&it.Slug, &it.Title, &it.Difficulty, &tags, &status, &due); err != nil {
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
		if due.Valid {
			d := due.String
			it.DueAt = &d
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// ---------------------------------------------------------------------------
// User question state
// ---------------------------------------------------------------------------

// State is the persisted spaced-repetition state for a (user, question) pair.
type State struct {
	Ease         float64
	IntervalDays float64
	Repetitions  int
	DueAt        string
	Status       string
	UpdatedAt    string
}

// GetState returns the state for a (user, question), or ErrNotFound.
func (s *Store) GetState(ctx context.Context, userID, questionID int64) (State, error) {
	var st State
	err := s.db.QueryRowContext(ctx, `
SELECT ease, interval_days, repetitions, due_at, status, updated_at
FROM user_question_state WHERE user_id = ? AND question_id = ?`, userID, questionID).
		Scan(&st.Ease, &st.IntervalDays, &st.Repetitions, &st.DueAt, &st.Status, &st.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return State{}, ErrNotFound
	}
	if err != nil {
		return State{}, err
	}
	return st, nil
}

// UpsertState inserts or updates the state for a (user, question) pair.
func (s *Store) UpsertState(ctx context.Context, userID, questionID int64, st State) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO user_question_state (user_id, question_id, ease, interval_days, repetitions, due_at, status, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id, question_id) DO UPDATE SET
    ease          = excluded.ease,
    interval_days = excluded.interval_days,
    repetitions   = excluded.repetitions,
    due_at        = excluded.due_at,
    status        = excluded.status,
    updated_at    = excluded.updated_at`,
		userID, questionID, st.Ease, st.IntervalDays, st.Repetitions, st.DueAt, st.Status, st.UpdatedAt)
	return err
}

// InsertReviewLog appends a review event.
func (s *Store) InsertReviewLog(ctx context.Context, userID, questionID int64, grade, reviewedAt string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO review_log (user_id, question_id, grade, reviewed_at) VALUES (?, ?, ?, ?)`,
		userID, questionID, grade, reviewedAt)
	return err
}

// RecordReview atomically upserts the user's SRS state and appends the
// review log entry for a single review submission.
func (s *Store) RecordReview(ctx context.Context, userID, questionID int64, st State, grade, reviewedAt string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
INSERT INTO user_question_state (user_id, question_id, ease, interval_days, repetitions, due_at, status, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id, question_id) DO UPDATE SET
    ease          = excluded.ease,
    interval_days = excluded.interval_days,
    repetitions   = excluded.repetitions,
    due_at        = excluded.due_at,
    status        = excluded.status,
    updated_at    = excluded.updated_at`,
		userID, questionID, st.Ease, st.IntervalDays, st.Repetitions, st.DueAt, st.Status, st.UpdatedAt); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO review_log (user_id, question_id, grade, reviewed_at) VALUES (?, ?, ?, ?)`,
		userID, questionID, grade, reviewedAt); err != nil {
		return err
	}

	return tx.Commit()
}

// ---------------------------------------------------------------------------
// Queue, sections & stats
// ---------------------------------------------------------------------------

// QueueItem is a due card in the review queue.
type QueueItem struct {
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Section string `json:"section"`
	DueAt   string `json:"due_at"`
}

// ReviewQueue returns cards due at or before now, ordered by due date.
func (s *Store) ReviewQueue(ctx context.Context, userID int64, now string, limit int) ([]QueueItem, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT q.slug, q.title, q.section, s.due_at
FROM user_question_state s
JOIN questions q ON q.id = s.question_id
WHERE s.user_id = ? AND s.due_at <= ?
ORDER BY s.due_at ASC
LIMIT ?`, userID, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []QueueItem{}
	for rows.Next() {
		var it QueueItem
		if err := rows.Scan(&it.Slug, &it.Title, &it.Section, &it.DueAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// SectionProgress holds per-section counters for a single user.
type SectionProgress struct {
	Done int
	Due  int
}

// CountQuestionsBySection returns the total number of questions per section.
func (s *Store) CountQuestionsBySection(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT section, COUNT(*) FROM questions GROUP BY section`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]int{}
	for rows.Next() {
		var section string
		var n int
		if err := rows.Scan(&section, &n); err != nil {
			return nil, err
		}
		out[section] = n
	}
	return out, rows.Err()
}

// StateStatsBySection returns per-section done/due counters for a user.
func (s *Store) StateStatsBySection(ctx context.Context, userID int64, now string) (map[string]SectionProgress, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT q.section,
       COUNT(*) AS done,
       COALESCE(SUM(CASE WHEN s.due_at <= ? THEN 1 ELSE 0 END), 0) AS due
FROM user_question_state s
JOIN questions q ON q.id = s.question_id
WHERE s.user_id = ?
GROUP BY q.section`, now, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]SectionProgress{}
	for rows.Next() {
		var section string
		var p SectionProgress
		if err := rows.Scan(&section, &p.Done, &p.Due); err != nil {
			return nil, err
		}
		out[section] = p
	}
	return out, rows.Err()
}

// TotalQuestions returns the total number of questions in the bank.
func (s *Store) TotalQuestions(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM questions`).Scan(&n)
	return n, err
}

// CountUserStates returns how many questions the user has started (has state for).
func (s *Store) CountUserStates(ctx context.Context, userID int64) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_question_state WHERE user_id = ?`, userID).Scan(&n)
	return n, err
}

// CountDue returns how many of the user's cards are due at or before now.
func (s *Store) CountDue(ctx context.Context, userID int64, now string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_question_state WHERE user_id = ? AND due_at <= ?`, userID, now).Scan(&n)
	return n, err
}

// ReviewDates returns the distinct UTC review dates (YYYY-MM-DD) for a user,
// newest first. Used to compute the review streak.
func (s *Store) ReviewDates(ctx context.Context, userID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT substr(reviewed_at, 1, 10) FROM review_log WHERE user_id = ? ORDER BY 1 DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dates := []string{}
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		dates = append(dates, d)
	}
	return dates, rows.Err()
}

// ActivityDay is one calendar day's event count, used both to render the
// recent-activity heatmap and to compute the all-time streak record.
type ActivityDay struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// ActivityByDay returns every UTC calendar date (YYYY-MM-DD) on which the
// user had any recorded activity, each annotated with the number of events
// that day: review_log rows, task_run_log rows, and user_lesson_state rows
// (one per lesson read) all count as separate events. Ordered by date
// ascending, oldest first. There is no date range filter here — callers
// trim to the recent window they need (e.g. the last 84 days for the
// heatmap) or use the full history (to compute the streak record).
func (s *Store) ActivityByDay(ctx context.Context, userID int64) ([]ActivityDay, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT date, SUM(cnt) AS total FROM (
    SELECT substr(reviewed_at, 1, 10) AS date, COUNT(*) AS cnt FROM review_log WHERE user_id = ? GROUP BY date
    UNION ALL
    SELECT substr(ran_at, 1, 10) AS date, COUNT(*) AS cnt FROM task_run_log WHERE user_id = ? GROUP BY date
    UNION ALL
    SELECT substr(read_at, 1, 10) AS date, COUNT(*) AS cnt FROM user_lesson_state WHERE user_id = ? GROUP BY date
) GROUP BY date ORDER BY date ASC`, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	days := []ActivityDay{}
	for rows.Next() {
		var d ActivityDay
		if err := rows.Scan(&d.Date, &d.Count); err != nil {
			return nil, err
		}
		days = append(days, d)
	}
	return days, rows.Err()
}
