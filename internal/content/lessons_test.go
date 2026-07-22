package content

import (
	"testing"
)

func TestParseLessonValid(t *testing.T) {
	l, err := parseLessonFile("testdata/lessons/01-gmp-scheduler.yaml")
	if err != nil {
		t.Fatalf("parseLessonFile: %v", err)
	}
	if l.Slug != "gmp-scheduler" {
		t.Errorf("slug = %q", l.Slug)
	}
	if l.Topic != "go-internals" {
		t.Errorf("topic = %q", l.Topic)
	}
	if l.Title == "" {
		t.Error("title must be populated")
	}
	if l.Minutes != 8 {
		t.Errorf("minutes = %d, want 8", l.Minutes)
	}
	if l.Position != 1 {
		t.Errorf("position = %d, want 1", l.Position)
	}
	if len(l.Tags) != 2 {
		t.Errorf("tags = %v", l.Tags)
	}
	if l.BodyMD == "" {
		t.Error("body_md must be populated")
	}
	if len(l.RelatedQuestions) != 2 || l.RelatedQuestions[0] != "goroutine-scheduler-gmp" {
		t.Errorf("related_questions = %v", l.RelatedQuestions)
	}
	if len(l.RelatedTasks) != 2 || l.RelatedTasks[0] != "sum-slice" {
		t.Errorf("related_tasks = %v", l.RelatedTasks)
	}
}

func TestParseLessonValid_EmptyRelated(t *testing.T) {
	l, err := parseLessonFile("testdata/lessons/02-channel-internals.yaml")
	if err != nil {
		t.Fatalf("parseLessonFile: %v", err)
	}
	if l.Topic != "concurrency" {
		t.Errorf("topic = %q", l.Topic)
	}
	if len(l.RelatedQuestions) != 0 {
		t.Errorf("related_questions = %v, want empty", l.RelatedQuestions)
	}
	if len(l.RelatedTasks) != 0 {
		t.Errorf("related_tasks = %v, want empty", l.RelatedTasks)
	}
}

func TestParseLessonInvalid(t *testing.T) {
	if _, err := parseLessonFile("testdata/lessons/99-invalid.yaml"); err == nil {
		t.Fatal("expected error for lesson missing body_md")
	}
}

func TestValidateLesson_UnknownTopic(t *testing.T) {
	l := Lesson{Slug: "x", Topic: "algorithms", Title: "x", BodyMD: "x"}
	if err := validateLesson(l); err == nil {
		t.Fatal("expected error for topic outside go-internals|concurrency")
	}
}

func TestLoadLessonsSkipsInvalid(t *testing.T) {
	lessons, err := LoadLessons("testdata/lessons", nil)
	if err != nil {
		t.Fatalf("LoadLessons: %v", err)
	}
	// Two valid lessons; the invalid one (missing body_md) must be skipped.
	if len(lessons) != 2 {
		t.Fatalf("loaded %d lessons, want 2", len(lessons))
	}
	for _, l := range lessons {
		if l.Slug == "" {
			t.Errorf("lesson with empty slug leaked through: %+v", l)
		}
	}
}

func TestLoadLessonsMissingDirIsNotError(t *testing.T) {
	lessons, err := LoadLessons("testdata/lessons/does-not-exist", nil)
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
	if lessons != nil {
		t.Errorf("expected nil lessons, got %v", lessons)
	}
}
