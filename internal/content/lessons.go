package content

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Lesson is a single "Учебник" lesson parsed from a YAML file.
type Lesson struct {
	Slug             string   `yaml:"slug"`
	Topic            string   `yaml:"topic"` // go-internals|concurrency|networks|os
	Title            string   `yaml:"title"`
	Minutes          int      `yaml:"minutes"`
	Tags             []string `yaml:"tags"`
	BodyMD           string   `yaml:"body_md"`
	RelatedQuestions []string `yaml:"related_questions"`
	RelatedTasks     []string `yaml:"related_tasks"`
	Position         int      `yaml:"-"`
}

var validLessonTopic = map[string]bool{
	"go-internals": true,
	"concurrency":  true,
	"networks":     true,
	"os":           true,
}

// LoadLessons walks dir recursively and returns every valid lesson found in
// *.yaml / *.yml files. Invalid files are logged and skipped rather than
// failing the whole load. A missing directory is not an error (the lesson
// content may be added later, possibly by a different process than the one
// loading it).
func LoadLessons(dir string, logger *slog.Logger) ([]Lesson, error) {
	if logger == nil {
		logger = slog.Default()
	}

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("lessons directory not found, skipping load", "dir", dir)
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("lessons path is not a directory: %s", dir)
	}

	var out []Lesson
	walkErr := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".yaml", ".yml":
		default:
			return nil
		}

		l, perr := parseLessonFile(path)
		if perr != nil {
			logger.Warn("skipping invalid lesson file", "file", path, "error", perr)
			return nil
		}
		out = append(out, l)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return out, nil
}

func parseLessonFile(path string) (Lesson, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Lesson{}, err
	}
	var l Lesson
	if err := yaml.Unmarshal(data, &l); err != nil {
		return Lesson{}, fmt.Errorf("parse yaml: %w", err)
	}
	l.Position = positionFromName(filepath.Base(path))
	if err := validateLesson(l); err != nil {
		return Lesson{}, err
	}
	return l, nil
}

// validateLesson checks the required fields. related_questions/related_tasks
// are intentionally not checked against the questions/coding_tasks banks
// here: unknown slugs are stored as-is and silently skipped when the API
// resolves them at read time.
func validateLesson(l Lesson) error {
	if strings.TrimSpace(l.Slug) == "" {
		return errors.New("slug is required")
	}
	if strings.TrimSpace(l.Topic) == "" {
		return errors.New("topic is required")
	}
	if !validLessonTopic[l.Topic] {
		return fmt.Errorf("invalid topic %q (want go-internals|concurrency|networks|os)", l.Topic)
	}
	if strings.TrimSpace(l.Title) == "" {
		return errors.New("title is required")
	}
	if strings.TrimSpace(l.BodyMD) == "" {
		return errors.New("body_md is required")
	}
	return nil
}
