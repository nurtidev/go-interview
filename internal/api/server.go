// Package api wires the HTTP router, JSON handlers and static file serving.
package api

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/nurtilek/go-interview/internal/auth"
	"github.com/nurtilek/go-interview/internal/runner"
	"github.com/nurtilek/go-interview/internal/store"
)

// Server holds the handler dependencies.
type Server struct {
	store               *store.Store
	auth                *auth.Service
	runner              *runner.Runner
	webDist             string
	registrationEnabled bool
	logger              *slog.Logger
}

// NewServer builds a Server. registrationEnabled gates POST /api/auth/register
// and is reported (never any other secret) via the public GET /api/config.
func NewServer(st *store.Store, a *auth.Service, run *runner.Runner, webDist string, registrationEnabled bool, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{store: st, auth: a, runner: run, webDist: webDist, registrationEnabled: registrationEnabled, logger: logger}
}

// Handler returns the fully configured http.Handler for the service.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Public endpoints.
	mux.HandleFunc("GET /api/healthz", s.handleHealthz)
	mux.HandleFunc("GET /api/config", s.handleConfig)
	mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)

	// Authenticated endpoints.
	mux.Handle("GET /api/me", s.protected(s.handleGetMe))
	mux.Handle("PATCH /api/me", s.protected(s.handleUpdateMe))
	mux.Handle("GET /api/sections", s.protected(s.handleSections))
	mux.Handle("GET /api/questions", s.protected(s.handleQuestions))
	mux.Handle("GET /api/questions/{slug}", s.protected(s.handleQuestionDetail))
	mux.Handle("POST /api/review/{slug}", s.protected(s.handleReview))
	mux.Handle("GET /api/review/queue", s.protected(s.handleQueue))
	mux.Handle("GET /api/me/stats", s.protected(s.handleStats))

	// Livecoding.
	mux.Handle("GET /api/coding/tasks", s.protected(s.handleCodingTasks))
	mux.Handle("GET /api/coding/tasks/{slug}", s.protected(s.handleCodingTaskDetail))
	mux.Handle("GET /api/coding/tasks/{slug}/solution", s.protected(s.handleCodingSolution))
	mux.Handle("POST /api/coding/tasks/{slug}/run", s.protected(s.handleCodingRun))
	mux.Handle("POST /api/coding/tasks/{slug}/solved", s.protected(s.handleCodingSolved))
	mux.Handle("POST /api/coding/tasks/{slug}/giveup", s.protected(s.handleCodingGiveUp))

	// Учебник (lessons).
	mux.Handle("GET /api/lessons", s.protected(s.handleLessons))
	mux.Handle("GET /api/lessons/{slug}", s.protected(s.handleLessonDetail))
	mux.Handle("POST /api/lessons/{slug}/read", s.protected(s.handleLessonRead))

	// Any other /api/* path is an API 404 (never falls back to the SPA).
	mux.HandleFunc("/api/", s.handleAPINotFound)

	// Everything else is the frontend.
	mux.Handle("/", s.staticHandler())

	return mux
}

func (s *Server) protected(h http.HandlerFunc) http.Handler {
	return s.auth.Middleware(h)
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleConfig exposes the minimal set of public, non-secret runtime flags the
// frontend needs before authentication (e.g. whether self-registration is
// open on this instance). Never add anything sensitive to this response.
func (s *Server) handleConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"registration_enabled": s.registrationEnabled})
}

func (s *Server) handleAPINotFound(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotFound, "not found")
}

// staticHandler serves the built frontend from webDist with SPA fallback to
// index.html. If webDist does not exist it responds 404 "frontend not built".
func (s *Server) staticHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.webDist == "" || !dirExists(s.webDist) {
			http.Error(w, "frontend not built", http.StatusNotFound)
			return
		}

		// Clean the request path and join it under webDist to prevent traversal.
		clean := filepath.Clean("/" + r.URL.Path)
		target := filepath.Join(s.webDist, clean)

		if info, err := os.Stat(target); err == nil && !info.IsDir() {
			http.ServeFile(w, r, target)
			return
		}

		index := filepath.Join(s.webDist, "index.html")
		if _, err := os.Stat(index); err == nil {
			http.ServeFile(w, r, index)
			return
		}

		http.Error(w, "frontend not built", http.StatusNotFound)
	})
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
