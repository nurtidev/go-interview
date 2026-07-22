package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/nurtilek/go-interview/internal/store"
)

func mustUpsertLesson(t *testing.T, st *store.Store, slug string, relatedQuestions, relatedTasks []string) store.Lesson {
	t.Helper()
	ctx := context.Background()
	if err := st.UpsertLesson(ctx, store.Lesson{
		Slug:             slug,
		Topic:            "go-internals",
		Title:            "Lesson " + slug,
		Minutes:          7,
		Tags:             []string{"runtime"},
		BodyMD:           "body for " + slug,
		RelatedQuestions: relatedQuestions,
		RelatedTasks:     relatedTasks,
		Position:         1,
	}); err != nil {
		t.Fatalf("upsert lesson: %v", err)
	}
	l, err := st.GetLessonBySlug(ctx, slug)
	if err != nil {
		t.Fatalf("get lesson: %v", err)
	}
	return l
}

func mustUpsertQuestion(t *testing.T, st *store.Store, slug string) store.Question {
	t.Helper()
	ctx := context.Background()
	if err := st.UpsertQuestion(ctx, store.Question{
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

func TestHandleLessons_List(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	q := mustUpsertQuestion(t, st, "goroutine-scheduler-gmp")
	mustUpsertLesson(t, st, "gmp-scheduler", []string{q.Slug, "unknown-question"}, nil)
	_, token := testUserToken(t, st, authSvc, "list@example.com")

	status, body := doRequest(t, ts, http.MethodGet, "/api/lessons", token, nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	lessons, _ := body["lessons"].([]any)
	if len(lessons) != 1 {
		t.Fatalf("expected 1 lesson, got %d", len(lessons))
	}
	item, _ := lessons[0].(map[string]any)
	if item["slug"] != "gmp-scheduler" {
		t.Errorf("slug = %v", item["slug"])
	}
	if item["read"] != false {
		t.Errorf("expected read=false, got %v", item["read"])
	}
	if int(item["reinforce_total"].(float64)) != 1 {
		t.Errorf("reinforce_total = %v, want 1 (unknown slug excluded)", item["reinforce_total"])
	}
	if int(item["reinforce_done"].(float64)) != 0 {
		t.Errorf("reinforce_done = %v, want 0", item["reinforce_done"])
	}
}

func TestHandleLessonDetail_ResolvesRelatedAndSkipsUnknown(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	q := mustUpsertQuestion(t, st, "goroutine-scheduler-gmp")
	mustUpsertSQLTask(t, st, "top-authors", "SELECT 1;")
	mustUpsertLesson(t, st, "gmp-scheduler",
		[]string{q.Slug, "unknown-question"},
		[]string{"top-authors", "unknown-task"})
	_, token := testUserToken(t, st, authSvc, "detail@example.com")

	status, body := doRequest(t, ts, http.MethodGet, "/api/lessons/gmp-scheduler", token, nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if body["body_md"] == "" {
		t.Error("expected body_md to be populated")
	}
	if body["read"] != false {
		t.Errorf("expected read=false, got %v", body["read"])
	}

	related, _ := body["related"].([]any)
	if len(related) != 2 {
		t.Fatalf("expected 2 related items (unknown slugs skipped), got %d: %v", len(related), related)
	}
	first, _ := related[0].(map[string]any)
	if first["type"] != "question" || first["slug"] != "goroutine-scheduler-gmp" || first["status"] != "new" {
		t.Errorf("unexpected first related item: %v", first)
	}
	second, _ := related[1].(map[string]any)
	if second["type"] != "task" || second["slug"] != "top-authors" || second["status"] != "new" {
		t.Errorf("unexpected second related item: %v", second)
	}
}

func TestHandleLessonDetail_NotFound(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	_, token := testUserToken(t, st, authSvc, "notfound@example.com")

	status, _ := doRequest(t, ts, http.MethodGet, "/api/lessons/does-not-exist", token, nil)
	if status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", status)
	}
}

func TestHandleLessonRead_Idempotent(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	l := mustUpsertLesson(t, st, "gmp-scheduler", nil, nil)
	uid, token := testUserToken(t, st, authSvc, "read@example.com")

	status, body := doRequest(t, ts, http.MethodPost, "/api/lessons/gmp-scheduler/read", token, nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if body["read"] != true {
		t.Errorf("expected read=true, got %v", body["read"])
	}

	firstReadAt, read, err := st.LessonReadAt(context.Background(), uid, l.ID)
	if err != nil {
		t.Fatalf("lesson read at: %v", err)
	}
	if !read {
		t.Fatal("expected lesson to be marked read")
	}

	// A second call must be idempotent: still 200, read_at unchanged.
	status, body = doRequest(t, ts, http.MethodPost, "/api/lessons/gmp-scheduler/read", token, nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200 on second read, got %d", status)
	}
	if body["read"] != true {
		t.Errorf("expected read=true on second call, got %v", body["read"])
	}
	secondReadAt, _, err := st.LessonReadAt(context.Background(), uid, l.ID)
	if err != nil {
		t.Fatalf("lesson read at (2nd): %v", err)
	}
	if secondReadAt != firstReadAt {
		t.Errorf("read_at changed on second call: %q -> %q", firstReadAt, secondReadAt)
	}

	// The detail and list endpoints must now reflect read=true.
	_, detail := doRequest(t, ts, http.MethodGet, "/api/lessons/gmp-scheduler", token, nil)
	if detail["read"] != true {
		t.Errorf("expected read=true in detail, got %v", detail["read"])
	}
	_, list := doRequest(t, ts, http.MethodGet, "/api/lessons", token, nil)
	lessons, _ := list["lessons"].([]any)
	item, _ := lessons[0].(map[string]any)
	if item["read"] != true {
		t.Errorf("expected read=true in list, got %v", item["read"])
	}
}

func TestHandleLessonRead_NotFound(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	_, token := testUserToken(t, st, authSvc, "read-404@example.com")

	status, _ := doRequest(t, ts, http.MethodPost, "/api/lessons/does-not-exist/read", token, nil)
	if status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", status)
	}
}

func TestHandleStats_Lessons(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	mustUpsertLesson(t, st, "lesson-1", nil, nil)
	mustUpsertLesson(t, st, "lesson-2", nil, nil)
	_, token := testUserToken(t, st, authSvc, "stats-lessons@example.com")

	if status, _ := doRequest(t, ts, http.MethodPost, "/api/lessons/lesson-1/read", token, nil); status != http.StatusOK {
		t.Fatalf("expected 200 marking lesson-1 read, got %d", status)
	}

	_, body := doRequest(t, ts, http.MethodGet, "/api/me/stats", token, nil)
	lessons, ok := body["lessons"].(map[string]any)
	if !ok {
		t.Fatalf("expected lessons stats object, got %v", body["lessons"])
	}
	if int(lessons["total"].(float64)) != 2 {
		t.Errorf("expected lessons.total=2, got %v", lessons["total"])
	}
	if int(lessons["read"].(float64)) != 1 {
		t.Errorf("expected lessons.read=1, got %v", lessons["read"])
	}
}
