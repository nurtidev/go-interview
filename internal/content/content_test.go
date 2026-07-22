package content

import (
	"testing"
)

func TestParseFileValid(t *testing.T) {
	q, err := parseFile("testdata/01-goroutine-scheduler.yaml")
	if err != nil {
		t.Fatalf("parseFile: %v", err)
	}
	if q.Slug != "goroutine-scheduler-gmp" {
		t.Errorf("slug = %q", q.Slug)
	}
	if q.Section != "go-internals" {
		t.Errorf("section = %q", q.Section)
	}
	if q.Difficulty != "senior" {
		t.Errorf("difficulty = %q", q.Difficulty)
	}
	if q.Position != 1 {
		t.Errorf("position = %d, want 1", q.Position)
	}
	if len(q.Tags) != 2 {
		t.Errorf("tags = %v", q.Tags)
	}
	if len(q.AnswerLevels) != 3 {
		t.Errorf("answer levels = %d, want 3", len(q.AnswerLevels))
	}
	if q.AnswerLevels[0].Level != "middle" || q.AnswerLevels[0].TextMD == "" {
		t.Errorf("unexpected first answer level: %+v", q.AnswerLevels[0])
	}
	if len(q.FollowUps) != 2 {
		t.Errorf("follow ups = %d, want 2", len(q.FollowUps))
	}
}

func TestParseFileInvalid(t *testing.T) {
	if _, err := parseFile("testdata/99-invalid-missing-fields.yaml"); err == nil {
		t.Fatal("expected error for file missing slug")
	}
}

func TestPositionFromName(t *testing.T) {
	cases := map[string]int{
		"03-foo.yaml":    3,
		"12-bar-baz.yml": 12,
		"no-prefix.yaml": 0,
		"007-agent.yaml": 7,
		"1question.yaml": 1,
	}
	for name, want := range cases {
		if got := positionFromName(name); got != want {
			t.Errorf("positionFromName(%q) = %d, want %d", name, got, want)
		}
	}
}

func TestLoadSkipsInvalidAndReturnsValid(t *testing.T) {
	qs, err := Load("testdata", nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(qs) != 2 {
		t.Fatalf("loaded %d questions, want 2 (invalid one must be skipped)", len(qs))
	}
	for _, q := range qs {
		if q.Slug == "" {
			t.Errorf("question with empty slug leaked through: %+v", q)
		}
	}
}

func TestLoadMissingDirIsNotError(t *testing.T) {
	qs, err := Load("testdata/does-not-exist", nil)
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
	if qs != nil {
		t.Errorf("expected nil questions, got %v", qs)
	}
}
