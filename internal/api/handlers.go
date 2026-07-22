package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/nurtilek/go-interview/internal/auth"
	"github.com/nurtilek/go-interview/internal/srs"
	"github.com/nurtilek/go-interview/internal/store"
)

type userDTO struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

func nowUTC() time.Time          { return time.Now().UTC() }
func nowString() string          { return nowUTC().Format(time.RFC3339) }
func fmtTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if !validEmail(req.Email) {
		writeError(w, http.StatusBadRequest, "invalid email")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.logger.Error("hash password", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	u, err := s.store.CreateUser(r.Context(), req.Email, hash)
	if errors.Is(err, store.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "email already registered")
		return
	}
	if err != nil {
		s.logger.Error("create user", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	s.issueToken(w, http.StatusCreated, u)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	u, err := s.store.GetUserByEmail(r.Context(), req.Email)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		s.logger.Error("get user", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !auth.CheckPassword(u.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	s.issueToken(w, http.StatusOK, u)
}

func (s *Server) issueToken(w http.ResponseWriter, status int, u store.User) {
	token, err := s.auth.GenerateToken(u.ID, u.Email)
	if err != nil {
		s.logger.Error("generate token", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, status, map[string]any{
		"token": token,
		"user":  userDTO{ID: u.ID, Email: u.Email},
	})
}

// ---------------------------------------------------------------------------
// Profile
// ---------------------------------------------------------------------------

type meDTO struct {
	ID            int64   `json:"id"`
	Email         string  `json:"email"`
	Name          *string `json:"name"`
	InterviewDate *string `json:"interview_date"`
}

func meResponse(u store.User) meDTO {
	return meDTO{ID: u.ID, Email: u.Email, Name: u.Name, InterviewDate: u.InterviewDate}
}

func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	u, err := s.store.GetUserByID(r.Context(), uid)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err != nil {
		s.internal(w, "get user", err)
		return
	}
	writeJSON(w, http.StatusOK, meResponse(u))
}

// handleUpdateMe partially updates the caller's profile: only fields present
// in the JSON body are changed. name must be a non-null string of at most
// 100 characters; interview_date must be a YYYY-MM-DD string, or explicit
// null to clear it. The body is decoded into a raw map[string]json.RawMessage
// first (rather than a plain struct) because a struct with pointer fields
// can't distinguish "field absent" (leave untouched) from "field explicitly
// null" (clear it) — both decode to a nil pointer.
func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var raw map[string]json.RawMessage
	if err := decodeJSON(r, &raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	u, err := s.store.GetUserByID(r.Context(), uid)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err != nil {
		s.internal(w, "get user", err)
		return
	}

	name := u.Name
	if v, present := raw["name"]; present {
		var n *string
		if err := json.Unmarshal(v, &n); err != nil || n == nil {
			writeError(w, http.StatusBadRequest, "name must be a non-null string")
			return
		}
		if len(*n) > 100 {
			writeError(w, http.StatusBadRequest, "name must be at most 100 characters")
			return
		}
		name = n
	}

	interviewDate := u.InterviewDate
	if v, present := raw["interview_date"]; present {
		var d *string
		if err := json.Unmarshal(v, &d); err != nil {
			writeError(w, http.StatusBadRequest, "invalid interview_date")
			return
		}
		if d != nil && !validDateYYYYMMDD(*d) {
			writeError(w, http.StatusBadRequest, "interview_date must be YYYY-MM-DD or null")
			return
		}
		interviewDate = d
	}

	updated, err := s.store.UpdateUserProfile(r.Context(), uid, name, interviewDate)
	if err != nil {
		s.internal(w, "update user profile", err)
		return
	}
	writeJSON(w, http.StatusOK, meResponse(updated))
}

// ---------------------------------------------------------------------------
// Sections
// ---------------------------------------------------------------------------

func (s *Server) handleSections(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	now := nowString()

	totals, err := s.store.CountQuestionsBySection(ctx)
	if err != nil {
		s.internal(w, "count questions", err)
		return
	}
	progress, err := s.store.StateStatsBySection(ctx, uid, now)
	if err != nil {
		s.internal(w, "section stats", err)
		return
	}

	type sectionResp struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Total       int    `json:"total"`
		Done        int    `json:"done"`
		Due         int    `json:"due"`
	}

	out := make([]sectionResp, 0, len(Sections))
	for _, sec := range Sections {
		p := progress[sec.ID]
		out = append(out, sectionResp{
			ID:          sec.ID,
			Title:       sec.Title,
			Description: sec.Description,
			Total:       totals[sec.ID],
			Done:        p.Done,
			Due:         p.Due,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"sections": out})
}

// ---------------------------------------------------------------------------
// Questions
// ---------------------------------------------------------------------------

func (s *Server) handleQuestions(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	section := strings.TrimSpace(r.URL.Query().Get("section"))
	if section == "" {
		writeError(w, http.StatusBadRequest, "section query parameter is required")
		return
	}

	items, err := s.store.ListQuestionsBySection(r.Context(), uid, section)
	if err != nil {
		s.internal(w, "list questions", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"questions": items})
}

type questionDetailResp struct {
	Slug         string              `json:"slug"`
	Section      string              `json:"section"`
	Title        string              `json:"title"`
	Difficulty   string              `json:"difficulty"`
	Tags         []string            `json:"tags"`
	QuestionMD   string              `json:"question_md"`
	AnswerLevels []store.AnswerLevel `json:"answer_levels"`
	FollowUps    []string            `json:"follow_ups"`
	Status       string              `json:"status"`
	DueAt        *string             `json:"due_at"`
}

func (s *Server) handleQuestionDetail(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	slug := r.PathValue("slug")

	q, err := s.store.GetQuestionBySlug(r.Context(), slug)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "question not found")
		return
	}
	if err != nil {
		s.internal(w, "get question", err)
		return
	}

	status, due, err := s.statusFor(r, uid, q.ID)
	if err != nil {
		s.internal(w, "get state", err)
		return
	}

	writeJSON(w, http.StatusOK, questionDetailResp{
		Slug:         q.Slug,
		Section:      q.Section,
		Title:        q.Title,
		Difficulty:   q.Difficulty,
		Tags:         q.Tags,
		QuestionMD:   q.QuestionMD,
		AnswerLevels: q.AnswerLevels,
		FollowUps:    q.FollowUps,
		Status:       status,
		DueAt:        due,
	})
}

// statusFor returns the display status ("new"/"learning"/"review") and due_at
// for a (user, question) pair.
func (s *Server) statusFor(r *http.Request, uid, questionID int64) (string, *string, error) {
	st, err := s.store.GetState(r.Context(), uid, questionID)
	if errors.Is(err, store.ErrNotFound) {
		return "new", nil, nil
	}
	if err != nil {
		return "", nil, err
	}
	due := st.DueAt
	return st.Status, &due, nil
}

// ---------------------------------------------------------------------------
// Review
// ---------------------------------------------------------------------------

func (s *Server) handleReview(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	slug := r.PathValue("slug")

	var req struct {
		Grade string `json:"grade"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	grade := srs.Grade(req.Grade)
	if !grade.Valid() {
		writeError(w, http.StatusBadRequest, "grade must be one of again|hard|good|easy")
		return
	}

	ctx := r.Context()
	q, err := s.store.GetQuestionBySlug(ctx, slug)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "question not found")
		return
	}
	if err != nil {
		s.internal(w, "get question", err)
		return
	}

	prev, err := s.store.GetState(ctx, uid, q.ID)
	var current srs.State
	switch {
	case errors.Is(err, store.ErrNotFound):
		current = srs.Default()
	case err != nil:
		s.internal(w, "get state", err)
		return
	default:
		current = srs.State{
			Ease:         prev.Ease,
			IntervalDays: prev.IntervalDays,
			Repetitions:  prev.Repetitions,
			Status:       srs.Status(prev.Status),
		}
	}

	now := nowUTC()
	next := srs.Review(current, grade, now)

	newState := store.State{
		Ease:         next.Ease,
		IntervalDays: next.IntervalDays,
		Repetitions:  next.Repetitions,
		DueAt:        fmtTime(next.DueAt),
		Status:       string(next.Status),
		UpdatedAt:    fmtTime(now),
	}
	if err := s.store.RecordReview(ctx, uid, q.ID, newState, string(grade), fmtTime(now)); err != nil {
		s.internal(w, "record review", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":        string(next.Status),
		"due_at":        newState.DueAt,
		"interval_days": next.IntervalDays,
	})
}

func (s *Server) handleQueue(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	items, err := s.store.ReviewQueue(r.Context(), uid, nowString(), 50)
	if err != nil {
		s.internal(w, "review queue", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"questions": items})
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	now := nowUTC()
	nowStr := fmtTime(now)

	total, err := s.store.TotalQuestions(ctx)
	if err != nil {
		s.internal(w, "total questions", err)
		return
	}
	reviewed, err := s.store.CountUserStates(ctx, uid)
	if err != nil {
		s.internal(w, "count states", err)
		return
	}
	dueToday, err := s.store.CountDue(ctx, uid, nowStr)
	if err != nil {
		s.internal(w, "count due", err)
		return
	}
	dates, err := s.store.ReviewDates(ctx, uid)
	if err != nil {
		s.internal(w, "review dates", err)
		return
	}
	totals, err := s.store.CountQuestionsBySection(ctx)
	if err != nil {
		s.internal(w, "count by section", err)
		return
	}
	progress, err := s.store.StateStatsBySection(ctx, uid, nowStr)
	if err != nil {
		s.internal(w, "section stats", err)
		return
	}
	codingTotal, err := s.store.CountCodingTasks(ctx)
	if err != nil {
		s.internal(w, "count coding tasks", err)
		return
	}
	codingSolved, err := s.store.CountSolvedCodingTasks(ctx, uid)
	if err != nil {
		s.internal(w, "count solved coding tasks", err)
		return
	}
	codingDue, err := s.store.CountDueCodingTasks(ctx, uid, nowStr)
	if err != nil {
		s.internal(w, "count due coding tasks", err)
		return
	}
	lessonsTotal, err := s.store.CountLessons(ctx)
	if err != nil {
		s.internal(w, "count lessons", err)
		return
	}
	lessonsRead, err := s.store.CountReadLessons(ctx, uid)
	if err != nil {
		s.internal(w, "count read lessons", err)
		return
	}
	activityAll, err := s.store.ActivityByDay(ctx, uid)
	if err != nil {
		s.internal(w, "activity by day", err)
		return
	}

	// activity (the heatmap) is trimmed to the last 84 days, UTC, including
	// today; days without activity are simply absent from the slice.
	// streak_record (the all-time longest run of consecutive active days) is
	// computed from the full, untrimmed history, per the same three sources.
	const heatmapDays = 84
	cutoff := now.AddDate(0, 0, -(heatmapDays - 1)).Format("2006-01-02")
	activity := make([]store.ActivityDay, 0, len(activityAll))
	allDates := make([]string, 0, len(activityAll))
	for _, d := range activityAll {
		allDates = append(allDates, d.Date)
		if d.Date >= cutoff {
			activity = append(activity, d)
		}
	}
	streakRecord := longestConsecutiveRun(allDates)

	type sectionStat struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Total int    `json:"total"`
		Done  int    `json:"done"`
	}
	bySection := make([]sectionStat, 0, len(Sections))
	for _, sec := range Sections {
		bySection = append(bySection, sectionStat{
			ID:    sec.ID,
			Title: sec.Title,
			Total: totals[sec.ID],
			Done:  progress[sec.ID].Done,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total":         total,
		"reviewed":      reviewed,
		"due_today":     dueToday,
		"streak_days":   computeStreak(dates, now),
		"streak_record": streakRecord,
		"activity":      activity,
		"by_section":    bySection,
		"coding": map[string]any{
			"total":  codingTotal,
			"solved": codingSolved,
			"due":    codingDue,
		},
		"lessons": map[string]any{
			"total": lessonsTotal,
			"read":  lessonsRead,
		},
	})
}

// computeStreak counts consecutive UTC days (ending today, or yesterday if
// today has no reviews yet) that have at least one review.
func computeStreak(dates []string, now time.Time) int {
	set := make(map[string]bool, len(dates))
	for _, d := range dates {
		set[d] = true
	}

	const layout = "2006-01-02"
	cursor := now
	if !set[now.Format(layout)] {
		// No reviews today: start counting from yesterday.
		cursor = now.AddDate(0, 0, -1)
		if !set[cursor.Format(layout)] {
			return 0
		}
	}

	streak := 0
	for set[cursor.Format(layout)] {
		streak++
		cursor = cursor.AddDate(0, 0, -1)
	}
	return streak
}

// longestConsecutiveRun returns the length of the longest run of
// consecutive UTC calendar days (each formatted YYYY-MM-DD, in any order,
// duplicates allowed) present in dates. Used for streak_record: unlike
// computeStreak, it is not anchored to "today" — it finds the longest run
// anywhere in the history, including ones broken by later gaps.
func longestConsecutiveRun(dates []string) int {
	set := make(map[string]bool, len(dates))
	for _, d := range dates {
		set[d] = true
	}

	const layout = "2006-01-02"
	best := 0
	for d := range set {
		t, err := time.Parse(layout, d)
		if err != nil {
			continue
		}
		// Only start counting from the first day of each run, so every run
		// is measured exactly once regardless of iteration order.
		prevDay := t.AddDate(0, 0, -1).Format(layout)
		if set[prevDay] {
			continue
		}
		run := 0
		cursor := t
		for set[cursor.Format(layout)] {
			run++
			cursor = cursor.AddDate(0, 0, 1)
		}
		if run > best {
			best = run
		}
	}
	return best
}

func (s *Server) internal(w http.ResponseWriter, msg string, err error) {
	s.logger.Error(msg, "error", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}
