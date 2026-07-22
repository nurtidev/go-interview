// Command server is the single binary for the go-interview backend. It serves
// the JSON API under /api/* and the frontend static files from disk.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nurtilek/go-interview/internal/api"
	"github.com/nurtilek/go-interview/internal/auth"
	"github.com/nurtilek/go-interview/internal/content"
	"github.com/nurtilek/go-interview/internal/runner"
	"github.com/nurtilek/go-interview/internal/store"
)

// maxConcurrentRuns bounds simultaneous livecoding executions.
const maxConcurrentRuns = 2

const defaultJWTSecret = "dev-secret-change-me"

type config struct {
	Port       string
	DBPath     string
	JWTSecret  string
	ContentDir string
	WebDist    string
}

func loadConfig(logger *slog.Logger) config {
	cfg := config{
		Port:       env("PORT", "8080"),
		DBPath:     env("DB_PATH", "./data/app.db"),
		JWTSecret:  env("JWT_SECRET", defaultJWTSecret),
		ContentDir: env("CONTENT_DIR", "./content"),
		WebDist:    env("WEB_DIST", "./web/dist"),
	}
	if cfg.JWTSecret == defaultJWTSecret {
		logger.Warn("JWT_SECRET is using the insecure default; set JWT_SECRET in production")
	}
	return cfg
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := loadConfig(logger)

	if dir := filepath.Dir(cfg.DBPath); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			logger.Error("create data directory", "dir", dir, "error", err)
			os.Exit(1)
		}
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		logger.Error("open store", "error", err)
		os.Exit(1)
	}
	defer st.Close()

	loadContent(context.Background(), st, cfg.ContentDir, logger)

	codeRunner, err := runner.New(maxConcurrentRuns)
	if err != nil {
		logger.Error("init runner", "error", err)
		os.Exit(1)
	}

	authSvc := auth.New(cfg.JWTSecret)
	srv := api.NewServer(st, authSvc, codeRunner, cfg.WebDist, logger)

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("server listening", "addr", httpServer.Addr, "web_dist", cfg.WebDist, "content_dir", cfg.ContentDir)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received, draining connections")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
	logger.Info("server stopped")
}

// loadContent parses the content directory and upserts every valid question
// and livecoding task. Failures are logged but never fatal: the content bank is
// optional at startup.
func loadContent(ctx context.Context, st *store.Store, dir string, logger *slog.Logger) {
	questions, err := content.Load(dir, logger)
	if err != nil {
		logger.Error("load content", "error", err)
		return
	}
	upserted := 0
	for _, q := range questions {
		if err := st.UpsertQuestion(ctx, toStoreQuestion(q)); err != nil {
			logger.Warn("upsert question failed", "slug", q.Slug, "error", err)
			continue
		}
		upserted++
	}
	logger.Info("content loaded", "files", len(questions), "upserted", upserted)

	loadCoding(ctx, st, filepath.Join(dir, "livecoding"), logger)
	loadLessons(ctx, st, filepath.Join(dir, "lessons"), logger)
}

// loadCoding parses the livecoding subdirectory and upserts every valid task.
// A missing directory is not an error (the content may be added later).
func loadCoding(ctx context.Context, st *store.Store, dir string, logger *slog.Logger) {
	tasks, err := content.LoadCoding(dir, logger)
	if err != nil {
		logger.Error("load livecoding", "error", err)
		return
	}
	upserted := 0
	for _, t := range tasks {
		if err := st.UpsertCodingTask(ctx, toStoreCodingTask(t)); err != nil {
			logger.Warn("upsert coding task failed", "slug", t.Slug, "error", err)
			continue
		}
		upserted++
	}
	logger.Info("livecoding loaded", "files", len(tasks), "upserted", upserted)
}

// loadLessons parses the lessons subdirectory and upserts every valid
// lesson. A missing directory is not an error (the content may be added
// later, possibly by a different process than the one running the server).
func loadLessons(ctx context.Context, st *store.Store, dir string, logger *slog.Logger) {
	lessons, err := content.LoadLessons(dir, logger)
	if err != nil {
		logger.Error("load lessons", "error", err)
		return
	}
	upserted := 0
	for _, l := range lessons {
		if err := st.UpsertLesson(ctx, toStoreLesson(l)); err != nil {
			logger.Warn("upsert lesson failed", "slug", l.Slug, "error", err)
			continue
		}
		upserted++
	}
	logger.Info("lessons loaded", "files", len(lessons), "upserted", upserted)
}

func toStoreLesson(l content.Lesson) store.Lesson {
	return store.Lesson{
		Slug:             l.Slug,
		Topic:            l.Topic,
		Title:            l.Title,
		Minutes:          l.Minutes,
		Tags:             l.Tags,
		BodyMD:           l.BodyMD,
		RelatedQuestions: l.RelatedQuestions,
		RelatedTasks:     l.RelatedTasks,
		Position:         l.Position,
	}
}

func toStoreQuestion(q content.Question) store.Question {
	levels := make([]store.AnswerLevel, len(q.AnswerLevels))
	for i, l := range q.AnswerLevels {
		levels[i] = store.AnswerLevel{Level: l.Level, TextMD: l.TextMD}
	}
	return store.Question{
		Slug:         q.Slug,
		Section:      q.Section,
		Title:        q.Title,
		Difficulty:   q.Difficulty,
		Tags:         q.Tags,
		QuestionMD:   q.QuestionMD,
		AnswerLevels: levels,
		FollowUps:    q.FollowUps,
		Position:     q.Position,
	}
}

func toStoreCodingTask(t content.CodingTask) store.CodingTask {
	return store.CodingTask{
		Slug:         t.Slug,
		Kind:         t.Kind,
		Title:        t.Title,
		Difficulty:   t.Difficulty,
		Tags:         t.Tags,
		StatementMD:  t.StatementMD,
		StarterCode:  t.StarterCode,
		Hints:        t.Hints,
		SolutionMD:   t.SolutionMD,
		TimeLimitSec: t.TimeLimitSec,
		Race:         t.Race,
		TestCode:     t.TestCode,
		SchemaSQL:    t.SchemaSQL,
		SeedSQL:      t.SeedSQL,
		Expected: store.Expected{
			Columns:      t.Expected.Columns,
			Rows:         t.Expected.Rows,
			OrderMatters: t.Expected.OrderMatters,
		},
		Position: t.Position,
	}
}
