package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/nurtilek/go-interview/internal/auth"
	"github.com/nurtilek/go-interview/internal/store"
)

// newTestServer boots a Server backed by a throwaway on-disk SQLite database
// with a nil runner: none of the endpoints exercised in this file (giveup,
// solution, solved, tasks, stats) touch the Go code runner. Registration is
// enabled, matching the default production behavior.
func newTestServer(t *testing.T) (*httptest.Server, *store.Store, *auth.Service) {
	t.Helper()
	return newTestServerWithConfig(t, true)
}

// newTestServerWithConfig is like newTestServer but lets callers pick the
// registration_enabled flag, for tests that exercise the closed-registration
// path (POST /api/auth/register and GET /api/config).
func newTestServerWithConfig(t *testing.T, registrationEnabled bool) (*httptest.Server, *store.Store, *auth.Service) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	authSvc := auth.New("test-secret")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := NewServer(st, authSvc, nil, "", registrationEnabled, logger)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, st, authSvc
}

func testUserToken(t *testing.T, st *store.Store, authSvc *auth.Service, email string) (int64, string) {
	t.Helper()
	u, err := st.CreateUser(context.Background(), email, "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	tok, err := authSvc.GenerateToken(u.ID, u.Email)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return u.ID, tok
}

// doRequest issues an HTTP request against the test server and decodes the
// JSON response body into a map for easy field assertions.
func doRequest(t *testing.T, ts *httptest.Server, method, path, token string, body any) (int, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, ts.URL+path, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func mustUpsertSQLTask(t *testing.T, st *store.Store, slug, solutionMD string) {
	t.Helper()
	if err := st.UpsertCodingTask(context.Background(), store.CodingTask{
		Slug:        slug,
		Kind:        "sql",
		Title:       "Task " + slug,
		Difficulty:  "easy",
		StatementMD: "statement",
		SolutionMD:  solutionMD,
	}); err != nil {
		t.Fatalf("upsert coding task: %v", err)
	}
}

func TestHandleCodingGiveUp_OpensSolution(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	mustUpsertSQLTask(t, st, "two-sum", "SELECT 1;")
	_, token := testUserToken(t, st, authSvc, "giveup@example.com")

	// Before giving up (or solving), the solution is forbidden.
	if status, _ := doRequest(t, ts, http.MethodGet, "/api/coding/tasks/two-sum/solution", token, nil); status != http.StatusForbidden {
		t.Fatalf("expected 403 before solving/giving up, got %d", status)
	}

	// Give up: the endpoint hands back the solution directly.
	status, body := doRequest(t, ts, http.MethodPost, "/api/coding/tasks/two-sum/giveup", token, nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200 from giveup, got %d", status)
	}
	if body["solution_md"] != "SELECT 1;" {
		t.Fatalf("expected solution_md in giveup response, got %v", body["solution_md"])
	}

	// The solution endpoint is now unlocked too.
	status, body = doRequest(t, ts, http.MethodGet, "/api/coding/tasks/two-sum/solution", token, nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200 from solution after giveup, got %d", status)
	}
	if body["solution_md"] != "SELECT 1;" {
		t.Fatalf("unexpected solution_md: %v", body["solution_md"])
	}

	// Detail and list both reflect gave_up.
	_, detail := doRequest(t, ts, http.MethodGet, "/api/coding/tasks/two-sum", token, nil)
	if detail["gave_up"] != true {
		t.Errorf("expected gave_up=true in task detail, got %v", detail["gave_up"])
	}
	_, list := doRequest(t, ts, http.MethodGet, "/api/coding/tasks", token, nil)
	tasks, _ := list["tasks"].([]any)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task in list, got %d", len(tasks))
	}
	item, _ := tasks[0].(map[string]any)
	if item["gave_up"] != true {
		t.Errorf("expected gave_up=true in task list item, got %v", item["gave_up"])
	}
}

func TestHandleCodingGiveUp_AlreadySolvedConflict(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	mustUpsertSQLTask(t, st, "reverse", "sol")
	_, token := testUserToken(t, st, authSvc, "solved@example.com")

	if status, _ := doRequest(t, ts, http.MethodPost, "/api/coding/tasks/reverse/solved", token, map[string]string{"code": "SELECT 1"}); status != http.StatusOK {
		t.Fatalf("expected 200 from solved, got %d", status)
	}

	status, body := doRequest(t, ts, http.MethodPost, "/api/coding/tasks/reverse/giveup", token, nil)
	if status != http.StatusConflict {
		t.Fatalf("expected 409 giving up an already-solved task, got %d", status)
	}
	if body["error"] != "already solved" {
		t.Fatalf(`expected error "already solved", got %v`, body["error"])
	}
}

func TestHandleCodingSolved_AdvancesDueAndClearsGaveUp(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	mustUpsertSQLTask(t, st, "joins", "sol")
	uid, token := testUserToken(t, st, authSvc, "joins@example.com")

	// Give up first, so the later solve has something to clear.
	if status, _ := doRequest(t, ts, http.MethodPost, "/api/coding/tasks/joins/giveup", token, nil); status != http.StatusOK {
		t.Fatalf("expected 200 from giveup, got %d", status)
	}

	if status, _ := doRequest(t, ts, http.MethodPost, "/api/coding/tasks/joins/solved", token, map[string]string{"code": "v1"}); status != http.StatusOK {
		t.Fatalf("expected 200 from first solved, got %d", status)
	}
	task, err := st.GetCodingTaskBySlug(context.Background(), "joins")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	st1, err := st.GetTaskState(context.Background(), uid, task.ID)
	if err != nil {
		t.Fatalf("get task state: %v", err)
	}
	if st1.GaveUp {
		t.Errorf("expected gave_up cleared after solving")
	}
	if st1.SolveCount != 1 || st1.DueAt == nil {
		t.Fatalf("expected solve_count=1 and due_at set, got %+v", st1)
	}
	due1, err := time.Parse(time.RFC3339, *st1.DueAt)
	if err != nil {
		t.Fatalf("parse due_at 1: %v", err)
	}

	if status, _ := doRequest(t, ts, http.MethodPost, "/api/coding/tasks/joins/solved", token, map[string]string{"code": "v2"}); status != http.StatusOK {
		t.Fatalf("expected 200 from re-solved, got %d", status)
	}
	st2, err := st.GetTaskState(context.Background(), uid, task.ID)
	if err != nil {
		t.Fatalf("get task state: %v", err)
	}
	if st2.SolveCount != 2 || st2.DueAt == nil {
		t.Fatalf("expected solve_count=2 and due_at set, got %+v", st2)
	}
	due2, err := time.Parse(time.RFC3339, *st2.DueAt)
	if err != nil {
		t.Fatalf("parse due_at 2: %v", err)
	}

	// 21d - 7d = 14d gap between the two due dates (tolerating the small
	// real-time elapsed between the two HTTP calls).
	gotGap := due2.Sub(due1)
	wantGap := 14 * 24 * time.Hour
	if diff := gotGap - wantGap; diff < -time.Minute || diff > time.Minute {
		t.Errorf("expected ~14 day gap between due dates (7d -> 21d), got %v", gotGap)
	}
}

func TestHandleStats_CodingDue(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	mustUpsertSQLTask(t, st, "overdue", "sol")
	mustUpsertSQLTask(t, st, "fresh", "sol")
	uid, token := testUserToken(t, st, authSvc, "stats@example.com")

	ctx := context.Background()
	overdue, err := st.GetCodingTaskBySlug(ctx, "overdue")
	if err != nil {
		t.Fatalf("get overdue task: %v", err)
	}
	fresh, err := st.GetCodingTaskBySlug(ctx, "fresh")
	if err != nil {
		t.Fatalf("get fresh task: %v", err)
	}

	// Solved 8 days ago -> due_at = -1d from now (overdue).
	past := time.Now().UTC().AddDate(0, 0, -8).Format(time.RFC3339)
	if err := st.MarkCodingSolved(ctx, uid, overdue.ID, "code", past); err != nil {
		t.Fatalf("mark overdue solved: %v", err)
	}
	// Solved just now -> due_at = +7d from now (not due yet).
	now := time.Now().UTC().Format(time.RFC3339)
	if err := st.MarkCodingSolved(ctx, uid, fresh.ID, "code", now); err != nil {
		t.Fatalf("mark fresh solved: %v", err)
	}

	_, body := doRequest(t, ts, http.MethodGet, "/api/me/stats", token, nil)
	coding, ok := body["coding"].(map[string]any)
	if !ok {
		t.Fatalf("expected coding stats object, got %v", body["coding"])
	}
	if int(coding["total"].(float64)) != 2 {
		t.Errorf("expected coding.total=2, got %v", coding["total"])
	}
	if int(coding["solved"].(float64)) != 2 {
		t.Errorf("expected coding.solved=2, got %v", coding["solved"])
	}
	if int(coding["due"].(float64)) != 1 {
		t.Errorf("expected coding.due=1, got %v", coding["due"])
	}

	// The list-level "due" flag should agree: exactly one task overdue.
	_, list := doRequest(t, ts, http.MethodGet, "/api/coding/tasks", token, nil)
	tasks, _ := list["tasks"].([]any)
	dueCount := 0
	for _, raw := range tasks {
		item, _ := raw.(map[string]any)
		if item["due"] == true {
			dueCount++
		}
	}
	if dueCount != 1 {
		t.Errorf("expected exactly 1 task flagged due in list, got %d", dueCount)
	}
}
