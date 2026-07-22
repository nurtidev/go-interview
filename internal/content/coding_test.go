package content

import (
	"testing"
)

func TestParseCodingGo(t *testing.T) {
	task, err := parseCodingFile("testdata/livecoding/go/01-sum-slice.yaml")
	if err != nil {
		t.Fatalf("parseCodingFile: %v", err)
	}
	if task.Slug != "sum-slice" {
		t.Errorf("slug = %q", task.Slug)
	}
	if task.Kind != "go" {
		t.Errorf("kind = %q", task.Kind)
	}
	if task.Difficulty != "middle" {
		t.Errorf("difficulty = %q", task.Difficulty)
	}
	if task.Position != 1 {
		t.Errorf("position = %d, want 1", task.Position)
	}
	if task.TimeLimitSec != 5 {
		t.Errorf("time_limit_sec = %d, want 5", task.TimeLimitSec)
	}
	if task.TestCode == "" {
		t.Error("test_code must be populated")
	}
	if len(task.Hints) != 2 {
		t.Errorf("hints = %d, want 2", len(task.Hints))
	}
}

func TestParseCodingSQL(t *testing.T) {
	task, err := parseCodingFile("testdata/livecoding/sql/01-top-authors.yaml")
	if err != nil {
		t.Fatalf("parseCodingFile: %v", err)
	}
	if task.Kind != "sql" {
		t.Fatalf("kind = %q", task.Kind)
	}
	if task.SchemaSQL == "" || task.SeedSQL == "" {
		t.Error("schema_sql and seed_sql must be populated")
	}
	want := [][]string{{"Alice", "3"}, {"Bob", "1"}}
	if len(task.Expected.Rows) != len(want) {
		t.Fatalf("expected rows = %v, want %v", task.Expected.Rows, want)
	}
	for i := range want {
		for j := range want[i] {
			if task.Expected.Rows[i][j] != want[i][j] {
				t.Errorf("expected[%d][%d] = %q, want %q (numbers must be normalised to strings)",
					i, j, task.Expected.Rows[i][j], want[i][j])
			}
		}
	}
	if !task.Expected.OrderMatters {
		t.Error("order_matters should be true")
	}
}

func TestParseCodingInvalid(t *testing.T) {
	if _, err := parseCodingFile("testdata/livecoding/go/99-invalid.yaml"); err == nil {
		t.Fatal("expected error for go task missing test_code")
	}
}

func TestLoadCodingSkipsInvalid(t *testing.T) {
	tasks, err := LoadCoding("testdata/livecoding", nil)
	if err != nil {
		t.Fatalf("LoadCoding: %v", err)
	}
	// One valid go + one valid sql; the invalid file must be skipped.
	if len(tasks) != 2 {
		t.Fatalf("loaded %d tasks, want 2", len(tasks))
	}
	kinds := map[string]int{}
	for _, task := range tasks {
		kinds[task.Kind]++
	}
	if kinds["go"] != 1 || kinds["sql"] != 1 {
		t.Errorf("kinds = %v, want one go and one sql", kinds)
	}
}

func TestLoadCodingMissingDirIsNotError(t *testing.T) {
	tasks, err := LoadCoding("testdata/livecoding/does-not-exist", nil)
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
	if tasks != nil {
		t.Errorf("expected nil tasks, got %v", tasks)
	}
}
