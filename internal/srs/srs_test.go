package srs

import (
	"testing"
	"time"
)

var testNow = time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)

func approx(a, b float64) bool {
	const eps = 1e-9
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < eps
}

func TestGradeValid(t *testing.T) {
	for _, g := range []Grade{GradeAgain, GradeHard, GradeGood, GradeEasy} {
		if !g.Valid() {
			t.Errorf("grade %q should be valid", g)
		}
	}
	if Grade("nope").Valid() {
		t.Error("unknown grade should be invalid")
	}
}

func TestFirstGood(t *testing.T) {
	s := Review(Default(), GradeGood, testNow)
	if s.Repetitions != 1 {
		t.Errorf("repetitions = %d, want 1", s.Repetitions)
	}
	if !approx(s.IntervalDays, 1) {
		t.Errorf("interval = %v, want 1", s.IntervalDays)
	}
	if s.Status != StatusReview {
		t.Errorf("status = %q, want review", s.Status)
	}
	// good keeps ease unchanged (q=4).
	if !approx(s.Ease, 2.5) {
		t.Errorf("ease = %v, want 2.5", s.Ease)
	}
	if !s.DueAt.Equal(testNow.Add(24 * time.Hour)) {
		t.Errorf("due = %v, want +24h", s.DueAt)
	}
}

func TestSecondAndThirdGood(t *testing.T) {
	s := Review(Default(), GradeGood, testNow)
	s = Review(s, GradeGood, testNow)
	if s.Repetitions != 2 {
		t.Fatalf("repetitions = %d, want 2", s.Repetitions)
	}
	if !approx(s.IntervalDays, 6) {
		t.Fatalf("interval = %v, want 6", s.IntervalDays)
	}
	// Third good: interval = round(6 * 2.5) = 15.
	s = Review(s, GradeGood, testNow)
	if s.Repetitions != 3 {
		t.Fatalf("repetitions = %d, want 3", s.Repetitions)
	}
	if !approx(s.IntervalDays, 15) {
		t.Fatalf("interval = %v, want 15", s.IntervalDays)
	}
	if !approx(s.Ease, 2.5) {
		t.Fatalf("ease = %v, want 2.5", s.Ease)
	}
}

func TestAgainResets(t *testing.T) {
	s := Review(Default(), GradeGood, testNow)
	s = Review(s, GradeGood, testNow)
	easeBefore := s.Ease

	s = Review(s, GradeAgain, testNow)
	if s.Repetitions != 0 {
		t.Errorf("repetitions = %d, want 0", s.Repetitions)
	}
	if !approx(s.IntervalDays, 0) {
		t.Errorf("interval = %v, want 0", s.IntervalDays)
	}
	if s.Status != StatusLearning {
		t.Errorf("status = %q, want learning", s.Status)
	}
	if !approx(s.Ease, easeBefore) {
		t.Errorf("ease changed on again: %v -> %v", easeBefore, s.Ease)
	}
	if !s.DueAt.Equal(testNow.Add(10 * time.Minute)) {
		t.Errorf("due = %v, want +10m", s.DueAt)
	}
}

func TestEasyIncreasesEase(t *testing.T) {
	s := Review(Default(), GradeEasy, testNow)
	// q=5 => ease += 0.1
	if !approx(s.Ease, 2.6) {
		t.Errorf("ease = %v, want 2.6", s.Ease)
	}
	if !approx(s.IntervalDays, 1) {
		t.Errorf("interval = %v, want 1", s.IntervalDays)
	}
}

func TestHardDecreasesEase(t *testing.T) {
	s := Review(Default(), GradeHard, testNow)
	// q=3 => ease += 0.1 - 2*(0.08 + 2*0.02) = -0.14
	if !approx(s.Ease, 2.36) {
		t.Errorf("ease = %v, want 2.36", s.Ease)
	}
	if !approx(s.IntervalDays, 1) {
		t.Errorf("interval = %v, want 1", s.IntervalDays)
	}
}

func TestEaseFloor(t *testing.T) {
	s := State{Ease: 1.3, IntervalDays: 10, Repetitions: 5, Status: StatusReview}
	s = Review(s, GradeHard, testNow)
	if s.Ease < MinEase {
		t.Errorf("ease = %v dropped below floor %v", s.Ease, MinEase)
	}
	if !approx(s.Ease, MinEase) {
		t.Errorf("ease = %v, want floored to %v", s.Ease, MinEase)
	}
}

func TestReviewSequenceIntervals(t *testing.T) {
	// good, good, good, good => 1, 6, 15, round(15*2.5)=38
	want := []float64{1, 6, 15, 38}
	s := Default()
	for i, wantInterval := range want {
		s = Review(s, GradeGood, testNow)
		if !approx(s.IntervalDays, wantInterval) {
			t.Errorf("step %d interval = %v, want %v", i, s.IntervalDays, wantInterval)
		}
	}
}
