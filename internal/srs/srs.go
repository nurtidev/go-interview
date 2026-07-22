// Package srs implements the classic SM-2 spaced-repetition algorithm.
// It is intentionally free of any storage or transport dependencies so it can
// be unit-tested in isolation.
package srs

import (
	"math"
	"time"
)

// Grade is the user-facing self-assessment of a review.
type Grade string

const (
	GradeAgain Grade = "again"
	GradeHard  Grade = "hard"
	GradeGood  Grade = "good"
	GradeEasy  Grade = "easy"
)

// Valid reports whether g is one of the four accepted grades.
func (g Grade) Valid() bool {
	switch g {
	case GradeAgain, GradeHard, GradeGood, GradeEasy:
		return true
	default:
		return false
	}
}

// quality maps a grade onto the SM-2 quality score q.
func (g Grade) quality() int {
	switch g {
	case GradeAgain:
		return 0
	case GradeHard:
		return 3
	case GradeGood:
		return 4
	case GradeEasy:
		return 5
	default:
		return 0
	}
}

// Status describes where a card sits in the learning lifecycle.
type Status string

const (
	StatusNew      Status = "new"
	StatusLearning Status = "learning"
	StatusReview   Status = "review"
)

// State captures the spaced-repetition state of a single card.
type State struct {
	Ease         float64
	IntervalDays float64
	Repetitions  int
	Status       Status
	DueAt        time.Time
}

// DefaultEase is the initial ease factor for a brand-new card.
const DefaultEase = 2.5

// MinEase is the floor for the ease factor.
const MinEase = 1.3

// Default returns the state of a card that has never been reviewed.
func Default() State {
	return State{
		Ease:         DefaultEase,
		IntervalDays: 0,
		Repetitions:  0,
		Status:       StatusNew,
	}
}

// Review applies grade g to state s at time now and returns the resulting
// state, including the freshly computed due date.
func Review(s State, g Grade, now time.Time) State {
	q := g.quality()

	// Failed recall: relearn in ten minutes, keep the ease factor untouched.
	if q < 3 {
		return State{
			Ease:         s.Ease,
			IntervalDays: 0,
			Repetitions:  0,
			Status:       StatusLearning,
			DueAt:        now.Add(10 * time.Minute),
		}
	}

	reps := s.Repetitions + 1

	var interval float64
	switch reps {
	case 1:
		interval = 1
	case 2:
		interval = 6
	default:
		interval = math.Round(s.IntervalDays * s.Ease)
	}

	ease := s.Ease + (0.1 - float64(5-q)*(0.08+float64(5-q)*0.02))
	if ease < MinEase {
		ease = MinEase
	}

	return State{
		Ease:         ease,
		IntervalDays: interval,
		Repetitions:  reps,
		Status:       StatusReview,
		DueAt:        now.Add(time.Duration(interval) * 24 * time.Hour),
	}
}
