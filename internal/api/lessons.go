package api

import (
	"errors"
	"net/http"

	"github.com/nurtilek/go-interview/internal/auth"
	"github.com/nurtilek/go-interview/internal/store"
)

// ---------------------------------------------------------------------------
// Lessons ("Учебник")
// ---------------------------------------------------------------------------

// handleLessons lists all lessons ordered by position, with the caller's
// read state and reinforcement progress.
func (s *Server) handleLessons(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	items, err := s.store.ListLessons(r.Context(), uid)
	if err != nil {
		s.internal(w, "list lessons", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lessons": items})
}

type lessonRelatedResp struct {
	Type   string `json:"type"` // question|task
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type lessonDetailResp struct {
	Slug    string              `json:"slug"`
	Topic   string              `json:"topic"`
	Title   string              `json:"title"`
	Minutes int                 `json:"minutes"`
	Tags    []string            `json:"tags"`
	BodyMD  string              `json:"body_md"`
	Read    bool                `json:"read"`
	Related []lessonRelatedResp `json:"related"`
}

// handleLessonDetail returns a single lesson with its related questions and
// coding tasks resolved to title/status. Related slugs that no longer exist
// in the questions/coding_tasks banks are silently skipped. Questions are
// listed before tasks.
func (s *Server) handleLessonDetail(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	slug := r.PathValue("slug")

	l, err := s.store.GetLessonBySlug(r.Context(), slug)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "lesson not found")
		return
	}
	if err != nil {
		s.internal(w, "get lesson", err)
		return
	}

	_, read, err := s.store.LessonReadAt(r.Context(), uid, l.ID)
	if err != nil {
		s.internal(w, "get lesson read state", err)
		return
	}

	related := make([]lessonRelatedResp, 0, len(l.RelatedQuestions)+len(l.RelatedTasks))
	for _, qSlug := range l.RelatedQuestions {
		q, err := s.store.GetQuestionBySlug(r.Context(), qSlug)
		if errors.Is(err, store.ErrNotFound) {
			continue
		}
		if err != nil {
			s.internal(w, "get related question", err)
			return
		}
		status, _, err := s.statusFor(r, uid, q.ID)
		if err != nil {
			s.internal(w, "get related question state", err)
			return
		}
		related = append(related, lessonRelatedResp{Type: "question", Slug: q.Slug, Title: q.Title, Status: status})
	}
	for _, tSlug := range l.RelatedTasks {
		t, err := s.store.GetCodingTaskBySlug(r.Context(), tSlug)
		if errors.Is(err, store.ErrNotFound) {
			continue
		}
		if err != nil {
			s.internal(w, "get related task", err)
			return
		}
		st, _, err := s.codingState(r, uid, t.ID)
		if err != nil {
			s.internal(w, "get related task state", err)
			return
		}
		related = append(related, lessonRelatedResp{Type: "task", Slug: t.Slug, Title: t.Title, Status: st.Status})
	}

	writeJSON(w, http.StatusOK, lessonDetailResp{
		Slug:    l.Slug,
		Topic:   l.Topic,
		Title:   l.Title,
		Minutes: l.Minutes,
		Tags:    l.Tags,
		BodyMD:  l.BodyMD,
		Read:    read,
		Related: related,
	})
}

// handleLessonRead marks a lesson as read. Idempotent: read_at is set only
// the first time.
func (s *Server) handleLessonRead(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	slug := r.PathValue("slug")

	l, err := s.store.GetLessonBySlug(r.Context(), slug)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "lesson not found")
		return
	}
	if err != nil {
		s.internal(w, "get lesson", err)
		return
	}

	if err := s.store.MarkLessonRead(r.Context(), uid, l.ID, nowString()); err != nil {
		s.internal(w, "mark lesson read", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"read": true})
}
