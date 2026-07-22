package api

import (
	"net/http"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// GET /api/me
// ---------------------------------------------------------------------------

func TestHandleGetMe_Defaults(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	_, token := testUserToken(t, st, authSvc, "me@example.com")

	status, body := doRequest(t, ts, http.MethodGet, "/api/me", token, nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if body["email"] != "me@example.com" {
		t.Errorf("email = %v", body["email"])
	}
	if body["name"] != nil {
		t.Errorf("expected name nil for a fresh user, got %v", body["name"])
	}
	if body["interview_date"] != nil {
		t.Errorf("expected interview_date nil for a fresh user, got %v", body["interview_date"])
	}
}

func TestHandleGetMe_Unauthorized(t *testing.T) {
	ts, _, _ := newTestServer(t)
	status, _ := doRequest(t, ts, http.MethodGet, "/api/me", "", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", status)
	}
}

// ---------------------------------------------------------------------------
// PATCH /api/me
// ---------------------------------------------------------------------------

func TestHandleUpdateMe_PartialUpdate(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	_, token := testUserToken(t, st, authSvc, "patch@example.com")

	// Set both fields.
	status, body := doRequest(t, ts, http.MethodPatch, "/api/me", token, map[string]any{
		"name":           "Aigerim",
		"interview_date": "2026-09-01",
	})
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d: %v", status, body)
	}
	if body["name"] != "Aigerim" {
		t.Errorf("name = %v", body["name"])
	}
	if body["interview_date"] != "2026-09-01" {
		t.Errorf("interview_date = %v", body["interview_date"])
	}

	// Partial update: only name given, interview_date must survive untouched.
	status, body = doRequest(t, ts, http.MethodPatch, "/api/me", token, map[string]any{
		"name": "Askar",
	})
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d: %v", status, body)
	}
	if body["name"] != "Askar" {
		t.Errorf("name = %v, want Askar", body["name"])
	}
	if body["interview_date"] != "2026-09-01" {
		t.Errorf("expected interview_date to survive an update that omits it, got %v", body["interview_date"])
	}

	// Explicit null clears interview_date; name must still survive untouched.
	status, body = doRequest(t, ts, http.MethodPatch, "/api/me", token, map[string]any{
		"interview_date": nil,
	})
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d: %v", status, body)
	}
	if body["interview_date"] != nil {
		t.Errorf("expected interview_date cleared by explicit null, got %v", body["interview_date"])
	}
	if body["name"] != "Askar" {
		t.Errorf("expected name to survive untouched, got %v", body["name"])
	}

	// Confirm it stuck via GET too.
	_, get := doRequest(t, ts, http.MethodGet, "/api/me", token, nil)
	if get["name"] != "Askar" || get["interview_date"] != nil {
		t.Errorf("unexpected profile after updates: %v", get)
	}
}

func TestHandleUpdateMe_ValidationErrors(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	_, token := testUserToken(t, st, authSvc, "validate@example.com")

	tests := []struct {
		name string
		body map[string]any
	}{
		{"name too long", map[string]any{"name": strings.Repeat("a", 101)}},
		{"name explicit null", map[string]any{"name": nil}},
		{"name wrong type", map[string]any{"name": 123}},
		{"interview_date bad format", map[string]any{"interview_date": "09-01-2026"}},
		{"interview_date garbage", map[string]any{"interview_date": "not-a-date"}},
		{"interview_date wrong type", map[string]any{"interview_date": 20260901}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, body := doRequest(t, ts, http.MethodPatch, "/api/me", token, tc.body)
			if status != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %v", status, body)
			}
		})
	}
}

func TestHandleUpdateMe_NameExactly100CharsOK(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	_, token := testUserToken(t, st, authSvc, "boundary@example.com")

	name := strings.Repeat("a", 100)
	status, body := doRequest(t, ts, http.MethodPatch, "/api/me", token, map[string]any{"name": name})
	if status != http.StatusOK {
		t.Fatalf("expected 200 for a 100-char name, got %d: %v", status, body)
	}
	if body["name"] != name {
		t.Errorf("name = %v", body["name"])
	}
}

func TestHandleUpdateMe_ClearingInterviewDateOnCreateAndClear(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	_, token := testUserToken(t, st, authSvc, "clear@example.com")

	// Clearing a field that was never set is a harmless no-op.
	status, body := doRequest(t, ts, http.MethodPatch, "/api/me", token, map[string]any{"interview_date": nil})
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d: %v", status, body)
	}
	if body["interview_date"] != nil {
		t.Errorf("expected interview_date nil, got %v", body["interview_date"])
	}
}

func TestHandleUpdateMe_Unauthorized(t *testing.T) {
	ts, _, _ := newTestServer(t)
	status, _ := doRequest(t, ts, http.MethodPatch, "/api/me", "", map[string]any{"name": "x"})
	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", status)
	}
}

func TestHandleUpdateMe_InvalidJSON(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	_, token := testUserToken(t, st, authSvc, "badjson@example.com")

	req, err := http.NewRequest(http.MethodPatch, ts.URL+"/api/me", strings.NewReader("{not valid json"))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
