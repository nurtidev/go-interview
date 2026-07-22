package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func mustUpsertTask(t *testing.T, st *Store, slug, kind string) CodingTask {
	t.Helper()
	ctx := context.Background()
	if err := st.UpsertCodingTask(ctx, CodingTask{
		Slug:        slug,
		Kind:        kind,
		Title:       "Task " + slug,
		Difficulty:  "easy",
		StatementMD: "statement",
		StarterCode: "starter",
		SolutionMD:  "solution",
		TestCode:    "test",
	}); err != nil {
		t.Fatalf("upsert coding task: %v", err)
	}
	task, err := st.GetCodingTaskBySlug(ctx, slug)
	if err != nil {
		t.Fatalf("get coding task: %v", err)
	}
	return task
}

func TestGiveUpCodingTask(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u, err := st.CreateUser(ctx, "give-up@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	task := mustUpsertTask(t, st, "give-up-task", "sql")

	now := time.Now().UTC().Format(time.RFC3339)
	if err := st.GiveUpCodingTask(ctx, u.ID, task.ID, now); err != nil {
		t.Fatalf("give up: %v", err)
	}

	got, err := st.GetTaskState(ctx, u.ID, task.ID)
	if err != nil {
		t.Fatalf("get task state: %v", err)
	}
	if !got.GaveUp {
		t.Errorf("expected gave_up=true after giving up")
	}
	if got.Status != "attempted" {
		t.Errorf("expected status=attempted after giving up, got %q", got.Status)
	}
	if got.DueAt != nil {
		t.Errorf("expected due_at to stay nil for a never-solved task, got %v", *got.DueAt)
	}
}

func TestGiveUpCodingTask_KeepsSolvedStatusSticky(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u, err := st.CreateUser(ctx, "sticky@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	task := mustUpsertTask(t, st, "sticky-task", "sql")

	now := time.Now().UTC().Format(time.RFC3339)
	if err := st.MarkCodingSolved(ctx, u.ID, task.ID, "code", now); err != nil {
		t.Fatalf("mark solved: %v", err)
	}
	// The HTTP handler is expected to reject giving up an already-solved
	// task; this only checks the store stays defensively sticky if called
	// directly anyway.
	if err := st.GiveUpCodingTask(ctx, u.ID, task.ID, now); err != nil {
		t.Fatalf("give up: %v", err)
	}

	got, err := st.GetTaskState(ctx, u.ID, task.ID)
	if err != nil {
		t.Fatalf("get task state: %v", err)
	}
	if got.Status != "solved" {
		t.Errorf("expected status to stay solved, got %q", got.Status)
	}
}

func TestRecordCodingRun_AdvancesDueAtAndResetsGaveUp(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u, err := st.CreateUser(ctx, "resolve@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	task := mustUpsertTask(t, st, "resolve-task", "go")

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at1 := base.Format(time.RFC3339)

	if err := st.GiveUpCodingTask(ctx, u.ID, task.ID, at1); err != nil {
		t.Fatalf("give up: %v", err)
	}

	// First passing run: solve_count -> 1, due_at -> at1 + 7d, gave_up cleared.
	if err := st.RecordCodingRun(ctx, u.ID, task.ID, "code-1", true, at1); err != nil {
		t.Fatalf("record run 1: %v", err)
	}
	got, err := st.GetTaskState(ctx, u.ID, task.ID)
	if err != nil {
		t.Fatalf("get task state: %v", err)
	}
	if got.GaveUp {
		t.Errorf("expected gave_up cleared after a passing run")
	}
	if got.SolveCount != 1 {
		t.Errorf("expected solve_count=1, got %d", got.SolveCount)
	}
	wantDue1 := base.AddDate(0, 0, 7).Format(time.RFC3339)
	if got.DueAt == nil || *got.DueAt != wantDue1 {
		t.Fatalf("expected due_at=%s, got %v", wantDue1, got.DueAt)
	}

	// Second passing run (re-solve, before or after due doesn't matter for
	// this MVP): solve_count -> 2, due_at -> at2 + 21d.
	at2 := base.AddDate(0, 0, 3).Format(time.RFC3339)
	if err := st.RecordCodingRun(ctx, u.ID, task.ID, "code-2", true, at2); err != nil {
		t.Fatalf("record run 2: %v", err)
	}
	got, err = st.GetTaskState(ctx, u.ID, task.ID)
	if err != nil {
		t.Fatalf("get task state: %v", err)
	}
	if got.SolveCount != 2 {
		t.Errorf("expected solve_count=2, got %d", got.SolveCount)
	}
	wantDue2 := base.AddDate(0, 0, 3).AddDate(0, 0, 21).Format(time.RFC3339)
	if got.DueAt == nil || *got.DueAt != wantDue2 {
		t.Fatalf("expected due_at=%s, got %v", wantDue2, got.DueAt)
	}

	// Third passing run: interval becomes 60 days.
	at3 := base.AddDate(0, 0, 10).Format(time.RFC3339)
	if err := st.RecordCodingRun(ctx, u.ID, task.ID, "code-3", true, at3); err != nil {
		t.Fatalf("record run 3: %v", err)
	}
	got, err = st.GetTaskState(ctx, u.ID, task.ID)
	if err != nil {
		t.Fatalf("get task state: %v", err)
	}
	if got.SolveCount != 3 {
		t.Errorf("expected solve_count=3, got %d", got.SolveCount)
	}
	wantDue3 := base.AddDate(0, 0, 10).AddDate(0, 0, 60).Format(time.RFC3339)
	if got.DueAt == nil || *got.DueAt != wantDue3 {
		t.Fatalf("expected due_at=%s, got %v", wantDue3, got.DueAt)
	}

	// A failing run afterwards must not touch solve_count/due_at/gave_up, and
	// must not downgrade the sticky "solved" status.
	at4 := base.AddDate(0, 0, 11).Format(time.RFC3339)
	if err := st.RecordCodingRun(ctx, u.ID, task.ID, "bad code", false, at4); err != nil {
		t.Fatalf("record run 4: %v", err)
	}
	got, err = st.GetTaskState(ctx, u.ID, task.ID)
	if err != nil {
		t.Fatalf("get task state: %v", err)
	}
	if got.Status != "solved" {
		t.Errorf("expected status to stay solved after a failing rerun, got %q", got.Status)
	}
	if got.SolveCount != 3 {
		t.Errorf("expected solve_count to stay 3 after a failing rerun, got %d", got.SolveCount)
	}
	if got.DueAt == nil || *got.DueAt != wantDue3 {
		t.Errorf("expected due_at to stay %s after a failing rerun, got %v", wantDue3, got.DueAt)
	}
	if got.GaveUp {
		t.Errorf("expected gave_up to stay false after a failing rerun")
	}
}

func TestCountDueCodingTasks(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u, err := st.CreateUser(ctx, "due@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	overdue := mustUpsertTask(t, st, "overdue-task", "sql")
	fresh := mustUpsertTask(t, st, "fresh-task", "sql")

	// Solved 8 days ago -> due_at = -1d from now (overdue).
	past := time.Now().UTC().AddDate(0, 0, -8).Format(time.RFC3339)
	if err := st.MarkCodingSolved(ctx, u.ID, overdue.ID, "code", past); err != nil {
		t.Fatalf("mark overdue solved: %v", err)
	}
	// Solved just now -> due_at = +7d from now (not due yet).
	now := time.Now().UTC().Format(time.RFC3339)
	if err := st.MarkCodingSolved(ctx, u.ID, fresh.ID, "code", now); err != nil {
		t.Fatalf("mark fresh solved: %v", err)
	}

	n, err := st.CountDueCodingTasks(ctx, u.ID, now)
	if err != nil {
		t.Fatalf("count due coding tasks: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 due coding task, got %d", n)
	}
}

func TestListCodingTasks_DueFlag(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u, err := st.CreateUser(ctx, "list-due@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	task := mustUpsertTask(t, st, "list-due-task", "sql")

	past := time.Now().UTC().AddDate(0, 0, -8).Format(time.RFC3339)
	if err := st.MarkCodingSolved(ctx, u.ID, task.ID, "code", past); err != nil {
		t.Fatalf("mark solved: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	items, err := st.ListCodingTasks(ctx, u.ID, now)
	if err != nil {
		t.Fatalf("list coding tasks: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 task, got %d", len(items))
	}
	if !items[0].Due {
		t.Errorf("expected due=true for an overdue solved task")
	}
	if items[0].DueAt == nil {
		t.Errorf("expected due_at to be set")
	}
}
