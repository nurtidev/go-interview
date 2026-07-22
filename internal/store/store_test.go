package store

import (
	"context"
	"testing"
)

// ---------------------------------------------------------------------------
// User profile (GetUserByID / UpdateUserProfile)
// ---------------------------------------------------------------------------

func TestGetUserByID_RoundTrip(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	created, err := st.CreateUser(ctx, "profile@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	got, err := st.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get user by id: %v", err)
	}
	if got.Email != "profile@example.com" {
		t.Errorf("email = %q", got.Email)
	}
	if got.Name != nil {
		t.Errorf("expected name nil for a new user, got %v", *got.Name)
	}
	if got.InterviewDate != nil {
		t.Errorf("expected interview_date nil for a new user, got %v", *got.InterviewDate)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	st := newTestStore(t)
	if _, err := st.GetUserByID(context.Background(), 999); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateUserProfile_SetsAndClears(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u, err := st.CreateUser(ctx, "update@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	name := "Aigerim"
	date := "2026-09-01"
	updated, err := st.UpdateUserProfile(ctx, u.ID, &name, &date)
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if updated.Name == nil || *updated.Name != name {
		t.Errorf("name = %v, want %q", updated.Name, name)
	}
	if updated.InterviewDate == nil || *updated.InterviewDate != date {
		t.Errorf("interview_date = %v, want %q", updated.InterviewDate, date)
	}

	// Passing nil for interview_date clears it while leaving name as given
	// (UpdateUserProfile always writes the full value it's handed for each
	// field -- partial-update semantics live in the API layer, not here).
	updated2, err := st.UpdateUserProfile(ctx, u.ID, &name, nil)
	if err != nil {
		t.Fatalf("update profile (clear date): %v", err)
	}
	if updated2.Name == nil || *updated2.Name != name {
		t.Errorf("name = %v, want %q to survive the second update", updated2.Name, name)
	}
	if updated2.InterviewDate != nil {
		t.Errorf("expected interview_date cleared, got %v", *updated2.InterviewDate)
	}

	// Round-trip through GetUserByID too.
	reloaded, err := st.GetUserByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("get user by id: %v", err)
	}
	if reloaded.Name == nil || *reloaded.Name != name {
		t.Errorf("reloaded name = %v, want %q", reloaded.Name, name)
	}
	if reloaded.InterviewDate != nil {
		t.Errorf("reloaded interview_date = %v, want nil", *reloaded.InterviewDate)
	}
}

// ---------------------------------------------------------------------------
// ActivityByDay
// ---------------------------------------------------------------------------

func TestActivityByDay_AggregatesAcrossSources(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u, err := st.CreateUser(ctx, "activity@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	q1 := mustUpsertQuestion(t, st, "activity-question-1")
	q2 := mustUpsertQuestion(t, st, "activity-question-2")
	task := mustUpsertTask(t, st, "activity-task", "sql")
	lesson := mustUpsertLesson(t, st, "activity-lesson", nil, nil, 1)

	const day1 = "2026-01-01"
	const day2 = "2026-01-02"

	// Day 1: a review + a coding run + a lesson read = 3 events.
	if err := st.InsertReviewLog(ctx, u.ID, q1.ID, "good", day1+"T09:00:00Z"); err != nil {
		t.Fatalf("insert review log: %v", err)
	}
	if err := st.RecordCodingRun(ctx, u.ID, task.ID, "code", true, day1+"T10:00:00Z"); err != nil {
		t.Fatalf("record coding run: %v", err)
	}
	if err := st.MarkLessonRead(ctx, u.ID, lesson.ID, day1+"T11:00:00Z"); err != nil {
		t.Fatalf("mark lesson read: %v", err)
	}

	// Day 2: two separate reviews = 2 events.
	if err := st.InsertReviewLog(ctx, u.ID, q1.ID, "again", day2+"T09:00:00Z"); err != nil {
		t.Fatalf("insert review log (2a): %v", err)
	}
	if err := st.InsertReviewLog(ctx, u.ID, q2.ID, "easy", day2+"T09:30:00Z"); err != nil {
		t.Fatalf("insert review log (2b): %v", err)
	}

	days, err := st.ActivityByDay(ctx, u.ID)
	if err != nil {
		t.Fatalf("activity by day: %v", err)
	}
	if len(days) != 2 {
		t.Fatalf("expected 2 active days, got %d: %+v", len(days), days)
	}
	if days[0].Date != day1 || days[0].Count != 3 {
		t.Errorf("day1 = %+v, want {%s 3}", days[0], day1)
	}
	if days[1].Date != day2 || days[1].Count != 2 {
		t.Errorf("day2 = %+v, want {%s 2}", days[1], day2)
	}
}

func TestActivityByDay_NoActivity(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u, err := st.CreateUser(ctx, "noactivity@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	days, err := st.ActivityByDay(ctx, u.ID)
	if err != nil {
		t.Fatalf("activity by day: %v", err)
	}
	if len(days) != 0 {
		t.Errorf("expected no active days, got %+v", days)
	}
}

func TestActivityByDay_ScopedToUser(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u1, err := st.CreateUser(ctx, "scoped-1@example.com", "hash")
	if err != nil {
		t.Fatalf("create user 1: %v", err)
	}
	u2, err := st.CreateUser(ctx, "scoped-2@example.com", "hash")
	if err != nil {
		t.Fatalf("create user 2: %v", err)
	}
	q := mustUpsertQuestion(t, st, "scoped-question")

	if err := st.InsertReviewLog(ctx, u1.ID, q.ID, "good", "2026-03-01T00:00:00Z"); err != nil {
		t.Fatalf("insert review log for u1: %v", err)
	}

	days, err := st.ActivityByDay(ctx, u2.ID)
	if err != nil {
		t.Fatalf("activity by day for u2: %v", err)
	}
	if len(days) != 0 {
		t.Errorf("expected user 2 to have no activity, got %+v", days)
	}
}
