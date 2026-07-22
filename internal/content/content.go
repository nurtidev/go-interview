// Package content loads interview questions from YAML files on disk.
package content

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// AnswerLevel is one layered answer within a question.
type AnswerLevel struct {
	Level  string `yaml:"level"`
	TextMD string `yaml:"text_md"`
}

// Question is a single interview question parsed from a YAML file.
type Question struct {
	Slug         string        `yaml:"slug"`
	Section      string        `yaml:"section"`
	Title        string        `yaml:"title"`
	Difficulty   string        `yaml:"difficulty"`
	Tags         []string      `yaml:"tags"`
	QuestionMD   string        `yaml:"question_md"`
	AnswerLevels []AnswerLevel `yaml:"answer_levels"`
	FollowUps    []string      `yaml:"follow_ups"`
	Position     int           `yaml:"-"`
}

var (
	validSections = map[string]bool{
		"go-internals":  true,
		"concurrency":   true,
		"algorithms":    true,
		"system-design": true,
		"platform":      true,
		"networks":      true,
		"os":            true,
	}
	validDifficulty = map[string]bool{
		"middle": true,
		"senior": true,
		"staff":  true,
	}
	positionRe = regexp.MustCompile(`^(\d+)`)
)

// Load walks dir recursively and returns every valid question found in *.yaml
// / *.yml files. Invalid files are logged and skipped rather than failing the
// whole load. A missing directory is not an error (it may be populated later
// by another process).
func Load(dir string, logger *slog.Logger) ([]Question, error) {
	if logger == nil {
		logger = slog.Default()
	}

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("content directory not found, skipping load", "dir", dir)
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("content path is not a directory: %s", dir)
	}

	var out []Question
	walkErr := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Livecoding tasks and lessons live under content/livecoding and
			// content/lessons and are loaded separately by LoadCoding and
			// LoadLessons; skip them here to avoid false warnings.
			if d.Name() == "livecoding" || d.Name() == "lessons" {
				return filepath.SkipDir
			}
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".yaml", ".yml":
		default:
			return nil
		}

		q, perr := parseFile(path)
		if perr != nil {
			logger.Warn("skipping invalid content file", "file", path, "error", perr)
			return nil
		}
		out = append(out, q)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return out, nil
}

func parseFile(path string) (Question, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Question{}, err
	}
	var q Question
	if err := yaml.Unmarshal(data, &q); err != nil {
		return Question{}, fmt.Errorf("parse yaml: %w", err)
	}
	q.Position = positionFromName(filepath.Base(path))
	if err := validate(q); err != nil {
		return Question{}, err
	}
	return q, nil
}

// positionFromName extracts a leading integer prefix from a file name,
// e.g. "03-foo.yaml" -> 3. Returns 0 when there is no numeric prefix.
func positionFromName(name string) int {
	m := positionRe.FindStringSubmatch(name)
	if m == nil {
		return 0
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return n
}

func validate(q Question) error {
	if strings.TrimSpace(q.Slug) == "" {
		return errors.New("slug is required")
	}
	if strings.TrimSpace(q.Section) == "" {
		return errors.New("section is required")
	}
	if strings.TrimSpace(q.Title) == "" {
		return errors.New("title is required")
	}
	if strings.TrimSpace(q.QuestionMD) == "" {
		return errors.New("question_md is required")
	}
	if !validSections[q.Section] {
		return fmt.Errorf("invalid section %q", q.Section)
	}
	if !validDifficulty[q.Difficulty] {
		return fmt.Errorf("invalid difficulty %q", q.Difficulty)
	}
	return nil
}
