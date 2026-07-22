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

// defaultTimeLimitSec is used for Go tasks that omit an explicit time limit.
const defaultTimeLimitSec = 10

// Expected is the reference result set for a SQL task. Cell values are
// normalised to strings so the contract is stable regardless of how they are
// written in YAML (e.g. 3 and "3" are equivalent).
type Expected struct {
	Columns      []string
	Rows         [][]string
	OrderMatters bool
}

// CodingTask is a single livecoding task parsed from a YAML file.
type CodingTask struct {
	Slug         string
	Kind         string // go|sql
	Title        string
	Difficulty   string
	Tags         []string
	StatementMD  string
	StarterCode  string
	Hints        []string
	SolutionMD   string
	TimeLimitSec int
	Race         bool
	TestCode     string
	SchemaSQL    string
	SeedSQL      string
	Expected     Expected
	Position     int
}

// codingTaskYAML mirrors the on-disk YAML schema. It is decoded first and then
// converted to CodingTask so that Expected.Rows can be normalised to strings.
type codingTaskYAML struct {
	Slug         string       `yaml:"slug"`
	Kind         string       `yaml:"kind"`
	Title        string       `yaml:"title"`
	Difficulty   string       `yaml:"difficulty"`
	Tags         []string     `yaml:"tags"`
	StatementMD  string       `yaml:"statement_md"`
	StarterCode  string       `yaml:"starter_code"`
	Hints        []string     `yaml:"hints"`
	SolutionMD   string       `yaml:"solution_md"`
	TimeLimitSec int          `yaml:"time_limit_sec"`
	Race         bool         `yaml:"race"`
	TestCode     string       `yaml:"test_code"`
	SchemaSQL    string       `yaml:"schema_sql"`
	SeedSQL      string       `yaml:"seed_sql"`
	Expected     expectedYAML `yaml:"expected"`
}

type expectedYAML struct {
	Columns      []string `yaml:"columns"`
	Rows         [][]any  `yaml:"rows"`
	OrderMatters bool     `yaml:"order_matters"`
}

var validCodingKind = map[string]bool{"go": true, "sql": true}

// LoadCoding walks dir recursively and returns every valid livecoding task
// found in *.yaml / *.yml files. Invalid files are logged and skipped rather
// than failing the whole load. A missing directory is not an error (the
// livecoding content may be added later or by another process).
func LoadCoding(dir string, logger *slog.Logger) ([]CodingTask, error) {
	if logger == nil {
		logger = slog.Default()
	}

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("livecoding directory not found, skipping load", "dir", dir)
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("livecoding path is not a directory: %s", dir)
	}

	var out []CodingTask
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

		t, perr := parseCodingFile(path)
		if perr != nil {
			logger.Warn("skipping invalid livecoding file", "file", path, "error", perr)
			return nil
		}
		out = append(out, t)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return out, nil
}

func parseCodingFile(path string) (CodingTask, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CodingTask{}, err
	}
	var raw codingTaskYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return CodingTask{}, fmt.Errorf("parse yaml: %w", err)
	}

	t := CodingTask{
		Slug:         raw.Slug,
		Kind:         strings.ToLower(strings.TrimSpace(raw.Kind)),
		Title:        raw.Title,
		Difficulty:   raw.Difficulty,
		Tags:         raw.Tags,
		StatementMD:  raw.StatementMD,
		StarterCode:  raw.StarterCode,
		Hints:        raw.Hints,
		SolutionMD:   raw.SolutionMD,
		TimeLimitSec: raw.TimeLimitSec,
		Race:         raw.Race,
		TestCode:     raw.TestCode,
		SchemaSQL:    raw.SchemaSQL,
		SeedSQL:      raw.SeedSQL,
		Expected:     convertExpected(raw.Expected),
		Position:     positionFromName(filepath.Base(path)),
	}
	if t.Kind == "go" && t.TimeLimitSec <= 0 {
		t.TimeLimitSec = defaultTimeLimitSec
	}
	if err := validateCoding(t); err != nil {
		return CodingTask{}, err
	}
	return t, nil
}

func convertExpected(e expectedYAML) Expected {
	out := Expected{Columns: e.Columns, OrderMatters: e.OrderMatters}
	for _, row := range e.Rows {
		cells := make([]string, len(row))
		for i, c := range row {
			cells[i] = cellToString(c)
		}
		out.Rows = append(out.Rows, cells)
	}
	return out
}

// cellToString renders a YAML scalar as the string the contract stores.
func cellToString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func validateCoding(t CodingTask) error {
	if strings.TrimSpace(t.Slug) == "" {
		return errors.New("slug is required")
	}
	if !validCodingKind[t.Kind] {
		return fmt.Errorf("invalid kind %q (want go|sql)", t.Kind)
	}
	if strings.TrimSpace(t.Title) == "" {
		return errors.New("title is required")
	}
	if !validDifficulty[t.Difficulty] {
		return fmt.Errorf("invalid difficulty %q", t.Difficulty)
	}
	if strings.TrimSpace(t.StatementMD) == "" {
		return errors.New("statement_md is required")
	}
	switch t.Kind {
	case "go":
		if strings.TrimSpace(t.TestCode) == "" {
			return errors.New("test_code is required for go tasks")
		}
	case "sql":
		if strings.TrimSpace(t.SchemaSQL) == "" {
			return errors.New("schema_sql is required for sql tasks")
		}
		if len(t.Expected.Columns) == 0 {
			return errors.New("expected.columns is required for sql tasks")
		}
	}
	return nil
}
