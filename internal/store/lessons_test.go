package store

import (
	"context"
	"testing"
	"time"
)

func mustUpsertLesson(t *testing.T, st *Store, slug string, relatedQuestions, relatedTasks []string, position int) Lesson {
	t.Helper()
	ctx := context.Background()
	if err := st.UpsertLesson(ctx, Lesson{
		Slug:             slug,
		Topic:            "go-internals",
		Title:            "Lesson " + slug,
		Minutes:          5,
		Tags:             []string{"runtime"},
		BodyMD:           "body for " + slug,
		RelatedQuestions: relatedQuestions,
		RelatedTasks:     relatedTasks,
		Position:         position,
	}); err != nil {
		t.Fatalf("upsert lesson: %v", err)
	}
	l, err := st.GetLessonBySlug(ctx, slug)
	if err != nil {
		t.Fatalf("get lesson: %v", err)
	}
	return l
}

func mustUpsertQuestion(t *testing.T, st *Store, slug string) Question {
	t.Helper()
	ctx := context.Background()
	if err := st.UpsertQuestion(ctx, Question{
		Slug:       slug,
		Section:    "go-internals",
		Title:      "Question " + slug,
		Difficulty: "senior",
		QuestionMD: "question",
	}); err != nil {
		t.Fatalf("upsert question: %v", err)
	}
	q, err := st.GetQuestionBySlug(ctx, slug)
	if err != nil {
		t.Fatalf("get question: %v", err)
	}
	return q
}

func TestGetLessonBySlug_NotFound(t *testing.T) {
	st := newTestStore(t)
	if _, err := st.GetLessonBySlug(context.Background(), "does-not-exist"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpsertLesson_RoundTrip(t *testing.T) {
	st := newTestStore(t)
	l := mustUpsertLesson(t, st, "gmp-scheduler", []string{"q1", "q2"}, []string{"t1"}, 1)
	if l.Topic != "go-internals" || l.Title != "Lesson gmp-scheduler" {
		t.Errorf("unexpected lesson: %+v", l)
	}
	if len(l.RelatedQuestions) != 2 || l.RelatedQuestions[0] != "q1" {
		t.Errorf("related_questions = %v", l.RelatedQuestions)
	}
	if len(l.RelatedTasks) != 1 || l.RelatedTasks[0] != "t1" {
		t.Errorf("related_tasks = %v", l.RelatedTasks)
	}

	// Upsert again with different data (same slug): fields must be overwritten,
	// not duplicated.
	mustUpsertLesson(t, st, "gmp-scheduler", []string{"q3"}, nil, 2)
	got, err := st.GetLessonBySlug(context.Background(), "gmp-scheduler")
	if err != nil {
		t.Fatalf("get lesson: %v", err)
	}
	if len(got.RelatedQuestions) != 1 || got.RelatedQuestions[0] != "q3" {
		t.Errorf("expected related_questions to be overwritten, got %v", got.RelatedQuestions)
	}
	if got.Position != 2 {
		t.Errorf("position = %d, want 2", got.Position)
	}
}

func TestListLessons_OrderedByPosition(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u, err := st.CreateUser(ctx, "order@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	mustUpsertLesson(t, st, "second", nil, nil, 2)
	mustUpsertLesson(t, st, "first", nil, nil, 1)

	items, err := st.ListLessons(ctx, u.ID)
	if err != nil {
		t.Fatalf("list lessons: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 lessons, got %d", len(items))
	}
	if items[0].Slug != "first" || items[1].Slug != "second" {
		t.Errorf("expected order [first, second], got [%s, %s]", items[0].Slug, items[1].Slug)
	}
}

func TestListLessons_ReinforceCounts(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u, err := st.CreateUser(ctx, "reinforce@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	q := mustUpsertQuestion(t, st, "goroutine-scheduler-gmp")
	// related_questions references one real question and one that doesn't
	// exist in the questions bank: reinforce_total must only count the real one.
	mustUpsertLesson(t, st, "gmp-scheduler", []string{q.Slug, "unknown-slug"}, nil, 1)

	items, err := st.ListLessons(ctx, u.ID)
	if err != nil {
		t.Fatalf("list lessons: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 lesson, got %d", len(items))
	}
	if items[0].ReinforceTotal != 1 {
		t.Errorf("reinforce_total = %d, want 1 (unknown slug excluded)", items[0].ReinforceTotal)
	}
	if items[0].ReinforceDone != 0 {
		t.Errorf("reinforce_done = %d, want 0 before the user has any state", items[0].ReinforceDone)
	}

	// Give the user state on the question (any SRS status counts as "done").
	if err := st.UpsertState(ctx, u.ID, q.ID, State{
		Ease: 2.5, IntervalDays: 1, Repetitions: 1,
		DueAt: time.Now().UTC().Format(time.RFC3339), Status: "learning",
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("upsert state: %v", err)
	}

	items, err = st.ListLessons(ctx, u.ID)
	if err != nil {
		t.Fatalf("list lessons after state: %v", err)
	}
	if items[0].ReinforceDone != 1 {
		t.Errorf("reinforce_done = %d, want 1 after the user started the question", items[0].ReinforceDone)
	}
}

func TestListLessons_ReadFlag(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u, err := st.CreateUser(ctx, "read-flag@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	l := mustUpsertLesson(t, st, "gmp-scheduler", nil, nil, 1)

	items, err := st.ListLessons(ctx, u.ID)
	if err != nil {
		t.Fatalf("list lessons: %v", err)
	}
	if items[0].Read {
		t.Errorf("expected read=false before marking read")
	}

	if err := st.MarkLessonRead(ctx, u.ID, l.ID, time.Now().UTC().Format(time.RFC3339)); err != nil {
		t.Fatalf("mark lesson read: %v", err)
	}
	items, err = st.ListLessons(ctx, u.ID)
	if err != nil {
		t.Fatalf("list lessons after read: %v", err)
	}
	if !items[0].Read {
		t.Errorf("expected read=true after marking read")
	}
}

func TestMarkLessonRead_Idempotent(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u, err := st.CreateUser(ctx, "idempotent@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	l := mustUpsertLesson(t, st, "gmp-scheduler", nil, nil, 1)

	first := time.Now().UTC().Format(time.RFC3339)
	if err := st.MarkLessonRead(ctx, u.ID, l.ID, first); err != nil {
		t.Fatalf("mark lesson read (1st): %v", err)
	}
	second := time.Now().UTC().AddDate(0, 0, 1).Format(time.RFC3339)
	if err := st.MarkLessonRead(ctx, u.ID, l.ID, second); err != nil {
		t.Fatalf("mark lesson read (2nd): %v", err)
	}

	readAt, read, err := st.LessonReadAt(ctx, u.ID, l.ID)
	if err != nil {
		t.Fatalf("lesson read at: %v", err)
	}
	if !read {
		t.Fatal("expected lesson to be marked read")
	}
	if readAt != first {
		t.Errorf("read_at = %q, want %q (must not move on a second call)", readAt, first)
	}
}

func TestLessonReadAt_NotRead(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u, err := st.CreateUser(ctx, "not-read@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	l := mustUpsertLesson(t, st, "gmp-scheduler", nil, nil, 1)

	_, read, err := st.LessonReadAt(ctx, u.ID, l.ID)
	if err != nil {
		t.Fatalf("lesson read at: %v", err)
	}
	if read {
		t.Error("expected read=false for a never-read lesson")
	}
}

func TestCountLessonsAndCountReadLessons(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	u, err := st.CreateUser(ctx, "counts@example.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	l1 := mustUpsertLesson(t, st, "lesson-1", nil, nil, 1)
	mustUpsertLesson(t, st, "lesson-2", nil, nil, 2)

	total, err := st.CountLessons(ctx)
	if err != nil {
		t.Fatalf("count lessons: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}

	read, err := st.CountReadLessons(ctx, u.ID)
	if err != nil {
		t.Fatalf("count read lessons: %v", err)
	}
	if read != 0 {
		t.Errorf("read = %d, want 0", read)
	}

	if err := st.MarkLessonRead(ctx, u.ID, l1.ID, time.Now().UTC().Format(time.RFC3339)); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	read, err = st.CountReadLessons(ctx, u.ID)
	if err != nil {
		t.Fatalf("count read lessons after read: %v", err)
	}
	if read != 1 {
		t.Errorf("read = %d, want 1", read)
	}
}
