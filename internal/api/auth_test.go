package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/nurtilek/go-interview/internal/auth"
)

// ---------------------------------------------------------------------------
// GET /api/config
// ---------------------------------------------------------------------------

func TestHandleConfig_RegistrationEnabled(t *testing.T) {
	ts, _, _ := newTestServerWithConfig(t, true)

	status, body := doRequest(t, ts, http.MethodGet, "/api/config", "", nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if body["registration_enabled"] != true {
		t.Errorf("registration_enabled = %v, want true", body["registration_enabled"])
	}
	// Nothing else — the endpoint must never leak anything beyond this flag.
	if len(body) != 1 {
		t.Errorf("expected exactly one field in /api/config response, got %v", body)
	}
}

func TestHandleConfig_RegistrationDisabled(t *testing.T) {
	ts, _, _ := newTestServerWithConfig(t, false)

	status, body := doRequest(t, ts, http.MethodGet, "/api/config", "", nil)
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if body["registration_enabled"] != false {
		t.Errorf("registration_enabled = %v, want false", body["registration_enabled"])
	}
}

// ---------------------------------------------------------------------------
// POST /api/auth/register
// ---------------------------------------------------------------------------

func TestHandleRegister_ClosedReturns403BeforeValidation(t *testing.T) {
	ts, _, _ := newTestServerWithConfig(t, false)

	// An obviously invalid body (bad email, short password) must still yield
	// 403 "registration is closed" — the gate runs before any validation.
	status, body := doRequest(t, ts, http.MethodPost, "/api/auth/register", "", map[string]any{
		"email":    "not-an-email",
		"password": "x",
	})
	if status != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %v", status, body)
	}
	if body["error"] != "registration is closed" {
		t.Errorf("error = %v, want %q", body["error"], "registration is closed")
	}
}

func TestHandleRegister_OpenSucceeds(t *testing.T) {
	ts, _, _ := newTestServerWithConfig(t, true)

	status, body := doRequest(t, ts, http.MethodPost, "/api/auth/register", "", map[string]any{
		"email":    "new@example.com",
		"password": "password123",
	})
	if status != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %v", status, body)
	}
	if body["token"] == nil || body["token"] == "" {
		t.Errorf("expected a token in the response, got %v", body["token"])
	}
}

func TestHandleLogin_WorksEvenWhenRegistrationClosed(t *testing.T) {
	ts, st, _ := newTestServerWithConfig(t, false)

	hash, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := st.CreateUser(context.Background(), "loginworks@example.com", hash); err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Registration is closed, but login must be unaffected.
	status, body := doRequest(t, ts, http.MethodPost, "/api/auth/login", "", map[string]any{
		"email":    "loginworks@example.com",
		"password": "password123",
	})
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d: %v", status, body)
	}
	if body["token"] == nil || body["token"] == "" {
		t.Errorf("expected a token in the response, got %v", body["token"])
	}
}
