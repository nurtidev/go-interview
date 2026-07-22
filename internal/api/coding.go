package api

import (
	"errors"
	"net/http"

	"github.com/nurtilek/go-interview/internal/auth"
	"github.com/nurtilek/go-interview/internal/runner"
	"github.com/nurtilek/go-interview/internal/store"
)

// ---------------------------------------------------------------------------
// Livecoding
// ---------------------------------------------------------------------------

// handleCodingTasks lists all coding tasks (Go first, then SQL) with the
// caller's per-task status.
func (s *Server) handleCodingTasks(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	items, err := s.store.ListCodingTasks(r.Context(), uid, nowString())
	if err != nil {
		s.internal(w, "list coding tasks", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": items})
}

type codingExpectedResp struct {
	Columns      []string   `json:"columns"`
	Rows         [][]string `json:"rows"`
	OrderMatters bool       `json:"order_matters"`
}

type codingTaskResp struct {
	Slug        string   `json:"slug"`
	Kind        string   `json:"kind"`
	Title       string   `json:"title"`
	Difficulty  string   `json:"difficulty"`
	Tags        []string `json:"tags"`
	StatementMD string   `json:"statement_md"`
	StarterCode string   `json:"starter_code"`
	Hints       []string `json:"hints"`
	Status      string   `json:"status"`
	LastCode    *string  `json:"last_code"`
	GaveUp      bool     `json:"gave_up"`
	DueAt       *string  `json:"due_at"`
	// test_code is never exposed; solution_md is served by a separate endpoint.
	SolutionMDAvailable bool `json:"solution_md_available"`

	// SQL-only fields (omitted for Go tasks).
	SchemaSQL string              `json:"schema_sql,omitempty"`
	SeedSQL   string              `json:"seed_sql,omitempty"`
	Expected  *codingExpectedResp `json:"expected,omitempty"`
}

// handleCodingTaskDetail returns a single task. It never leaks test_code or the
// solution; SQL tasks additionally carry the schema/seed/expected result set.
func (s *Server) handleCodingTaskDetail(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	slug := r.PathValue("slug")

	t, err := s.store.GetCodingTaskBySlug(r.Context(), slug)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		s.internal(w, "get coding task", err)
		return
	}

	st, found, err := s.codingState(r, uid, t.ID)
	if err != nil {
		s.internal(w, "get task state", err)
		return
	}
	var lastCode *string
	if found {
		code := st.LastCode
		lastCode = &code
	}

	resp := codingTaskResp{
		Slug:                t.Slug,
		Kind:                t.Kind,
		Title:               t.Title,
		Difficulty:          t.Difficulty,
		Tags:                t.Tags,
		StatementMD:         t.StatementMD,
		StarterCode:         t.StarterCode,
		Hints:               t.Hints,
		Status:              st.Status,
		LastCode:            lastCode,
		GaveUp:              st.GaveUp,
		DueAt:               st.DueAt,
		SolutionMDAvailable: t.SolutionMD != "",
	}
	if t.Kind == "sql" {
		resp.SchemaSQL = t.SchemaSQL
		resp.SeedSQL = t.SeedSQL
		resp.Expected = &codingExpectedResp{
			Columns:      t.Expected.Columns,
			Rows:         t.Expected.Rows,
			OrderMatters: t.Expected.OrderMatters,
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleCodingSolution returns the reference solution, but only once the caller
// has solved the task.
func (s *Server) handleCodingSolution(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	slug := r.PathValue("slug")

	t, err := s.store.GetCodingTaskBySlug(r.Context(), slug)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		s.internal(w, "get coding task", err)
		return
	}

	st, _, err := s.codingState(r, uid, t.ID)
	if err != nil {
		s.internal(w, "get task state", err)
		return
	}
	// The solution unlocks once the task is solved, or once the caller has
	// given up on it (hints exhausted, "25/5" give-up path).
	if st.Status != "solved" && !st.GaveUp {
		writeError(w, http.StatusForbidden, "solve the task first")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"solution_md": t.SolutionMD})
}

// handleCodingGiveUp marks a task as given up: hints are exhausted, so the
// caller opens the reference solution and is expected to re-attempt the task
// from memory later (the "25/5" framework's give-up path). A task that's
// already solved can't be given up.
func (s *Server) handleCodingGiveUp(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	slug := r.PathValue("slug")

	t, err := s.store.GetCodingTaskBySlug(r.Context(), slug)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		s.internal(w, "get coding task", err)
		return
	}

	st, _, err := s.codingState(r, uid, t.ID)
	if err != nil {
		s.internal(w, "get task state", err)
		return
	}
	if st.Status == "solved" {
		writeError(w, http.StatusConflict, "already solved")
		return
	}

	if err := s.store.GiveUpCodingTask(r.Context(), uid, t.ID, nowString()); err != nil {
		s.internal(w, "give up coding task", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"solution_md": t.SolutionMD})
}

// handleCodingRun compiles and tests a Go submission, records the attempt and
// returns the outcome.
func (s *Server) handleCodingRun(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	slug := r.PathValue("slug")

	var req struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	t, err := s.store.GetCodingTaskBySlug(r.Context(), slug)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		s.internal(w, "get coding task", err)
		return
	}
	if t.Kind != "go" {
		writeError(w, http.StatusBadRequest, "run is only available for go tasks")
		return
	}

	res, err := s.runner.Run(r.Context(), runner.Request{
		Code:         req.Code,
		TestCode:     t.TestCode,
		TimeLimitSec: t.TimeLimitSec,
		Race:         t.Race,
	})
	switch {
	case errors.Is(err, runner.ErrCodeTooLarge):
		writeError(w, http.StatusBadRequest, "code too large")
		return
	case errors.Is(err, runner.ErrBusy):
		writeError(w, http.StatusServiceUnavailable, "runner busy, please try again")
		return
	case err != nil:
		s.internal(w, "run code", err)
		return
	}

	passed := res.Result == runner.ResultPassed
	if err := s.store.RecordCodingRun(r.Context(), uid, t.ID, req.Code, passed, nowString()); err != nil {
		s.internal(w, "record coding run", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result": string(res.Result),
		"output": res.Output,
	})
}

// handleCodingSolved marks a SQL task solved. Result correctness is validated
// client-side (a deliberate MVP trade-off); the server only stores the code.
func (s *Server) handleCodingSolved(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	slug := r.PathValue("slug")

	var req struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	t, err := s.store.GetCodingTaskBySlug(r.Context(), slug)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		s.internal(w, "get coding task", err)
		return
	}
	if t.Kind != "sql" {
		writeError(w, http.StatusBadRequest, "solved is only available for sql tasks")
		return
	}

	if err := s.store.MarkCodingSolved(r.Context(), uid, t.ID, req.Code, nowString()); err != nil {
		s.internal(w, "mark coding solved", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "solved"})
}

// codingState returns the caller's per-task state. A missing row (task never
// attempted) reports a fresh "new" state with found=false, so callers can
// tell a real empty last_code/solution from "no state yet".
func (s *Server) codingState(r *http.Request, uid, taskID int64) (store.TaskState, bool, error) {
	st, err := s.store.GetTaskState(r.Context(), uid, taskID)
	if errors.Is(err, store.ErrNotFound) {
		return store.TaskState{Status: "new"}, false, nil
	}
	if err != nil {
		return store.TaskState{}, false, err
	}
	return st, true, nil
}
