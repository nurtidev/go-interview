package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

// ---------------------------------------------------------------------------
// Lessons ("Учебник")
// ---------------------------------------------------------------------------

// Lesson is the full stored representation of a lesson.
type Lesson struct {
	ID               int64
	Slug             string
	Topic            string // go-internals|concurrency|networks|os
	Title            string
	Minutes          int
	Tags             []string
	BodyMD           string
	RelatedQuestions []string
	RelatedTasks     []string
	Position         int
}

// LessonListItem is a lightweight lesson row enriched with per-user progress.
type LessonListItem struct {
	Slug           string   `json:"slug"`
	Topic          string   `json:"topic"`
	Title          string   `json:"title"`
	Minutes        int      `json:"minutes"`
	Tags           []string `json:"tags"`
	Read           bool     `json:"read"`
	ReinforceDone  int      `json:"reinforce_done"`
	ReinforceTotal int      `json:"reinforce_total"`
}

// UpsertLesson inserts or updates a lesson keyed by slug.
func (s *Store) UpsertLesson(ctx context.Context, l Lesson) error {
	if l.Tags == nil {
		l.Tags = []string{}
	}
	if l.RelatedQuestions == nil {
		l.RelatedQuestions = []string{}
	}
	if l.RelatedTasks == nil {
		l.RelatedTasks = []string{}
	}
	tags, err := json.Marshal(l.Tags)
	if err != nil {
		return err
	}
	relQuestions, err := json.Marshal(l.RelatedQuestions)
	if err != nil {
		return err
	}
	relTasks, err := json.Marshal(l.RelatedTasks)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO lessons (slug, topic, title, minutes, tags, body_md, related_questions, related_tasks, position)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(slug) DO UPDATE SET
    topic             = excluded.topic,
    title             = excluded.title,
    minutes           = excluded.minutes,
    tags              = excluded.tags,
    body_md           = excluded.body_md,
    related_questions = excluded.related_questions,
    related_tasks     = excluded.related_tasks,
    position          = excluded.position`,
		l.Slug, l.Topic, l.Title, l.Minutes, string(tags), l.BodyMD, string(relQuestions), string(relTasks), l.Position)
	return err
}

// GetLessonBySlug returns a single lesson, or ErrNotFound.
func (s *Store) GetLessonBySlug(ctx context.Context, slug string) (Lesson, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, slug, topic, title, minutes, tags, body_md, related_questions, related_tasks, position
FROM lessons WHERE slug = ?`, slug)

	var l Lesson
	var tags, relQuestions, relTasks string
	err := row.Scan(&l.ID, &l.Slug, &l.Topic, &l.Title, &l.Minutes, &tags, &l.BodyMD, &relQuestions, &relTasks, &l.Position)
	if errors.Is(err, sql.ErrNoRows) {
		return Lesson{}, ErrNotFound
	}
	if err != nil {
		return Lesson{}, err
	}
	_ = json.Unmarshal([]byte(tags), &l.Tags)
	_ = json.Unmarshal([]byte(relQuestions), &l.RelatedQuestions)
	_ = json.Unmarshal([]byte(relTasks), &l.RelatedTasks)
	if l.Tags == nil {
		l.Tags = []string{}
	}
	if l.RelatedQuestions == nil {
		l.RelatedQuestions = []string{}
	}
	if l.RelatedTasks == nil {
		l.RelatedTasks = []string{}
	}
	return l, nil
}

// ListLessons returns every lesson ordered by position, each annotated with
// the given user's read state and reinforce_done/reinforce_total counters.
// reinforce_total counts the related_questions slugs that exist in the
// questions bank (unknown slugs are silently excluded); reinforce_done
// counts how many of those the user has started (has a user_question_state
// row for, regardless of SRS status).
func (s *Store) ListLessons(ctx context.Context, userID int64) ([]LessonListItem, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT l.id, l.slug, l.topic, l.title, l.minutes, l.tags, l.related_questions, ur.read_at
FROM lessons l
LEFT JOIN user_lesson_state ur ON ur.lesson_id = l.id AND ur.user_id = ?
ORDER BY l.position, l.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type pending struct {
		item             LessonListItem
		relatedQuestions []string
	}
	var raws []pending
	for rows.Next() {
		var id int64
		var it LessonListItem
		var tags, relQuestions string
		var readAt sql.NullString
		if err := rows.Scan(&id, &it.Slug, &it.Topic, &it.Title, &it.Minutes, &tags, &relQuestions, &readAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(tags), &it.Tags)
		if it.Tags == nil {
			it.Tags = []string{}
		}
		it.Read = readAt.Valid

		var relatedQuestions []string
		_ = json.Unmarshal([]byte(relQuestions), &relatedQuestions)
		raws = append(raws, pending{item: it, relatedQuestions: relatedQuestions})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	existing, done, err := s.questionReinforceSets(ctx, userID)
	if err != nil {
		return nil, err
	}

	items := make([]LessonListItem, 0, len(raws))
	for _, r := range raws {
		it := r.item
		for _, slug := range r.relatedQuestions {
			if !existing[slug] {
				continue
			}
			it.ReinforceTotal++
			if done[slug] {
				it.ReinforceDone++
			}
		}
		items = append(items, it)
	}
	return items, nil
}

// questionReinforceSets returns the set of all existing question slugs and
// the subset of those the given user has state for. Used to compute lesson
// reinforce_total/reinforce_done without an N+1 query per lesson.
func (s *Store) questionReinforceSets(ctx context.Context, userID int64) (existing, done map[string]bool, err error) {
	existing = map[string]bool{}
	existingRows, err := s.db.QueryContext(ctx, `SELECT slug FROM questions`)
	if err != nil {
		return nil, nil, err
	}
	defer existingRows.Close()
	for existingRows.Next() {
		var slug string
		if err := existingRows.Scan(&slug); err != nil {
			return nil, nil, err
		}
		existing[slug] = true
	}
	if err := existingRows.Err(); err != nil {
		return nil, nil, err
	}

	done = map[string]bool{}
	doneRows, err := s.db.QueryContext(ctx, `
SELECT q.slug FROM questions q
JOIN user_question_state st ON st.question_id = q.id
WHERE st.user_id = ?`, userID)
	if err != nil {
		return nil, nil, err
	}
	defer doneRows.Close()
	for doneRows.Next() {
		var slug string
		if err := doneRows.Scan(&slug); err != nil {
			return nil, nil, err
		}
		done[slug] = true
	}
	return existing, done, doneRows.Err()
}

// LessonReadAt returns the read_at timestamp for a (user, lesson) pair, and
// whether the lesson has been read at all.
func (s *Store) LessonReadAt(ctx context.Context, userID, lessonID int64) (string, bool, error) {
	var readAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT read_at FROM user_lesson_state WHERE user_id = ? AND lesson_id = ?`, userID, lessonID).Scan(&readAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return readAt, true, nil
}

// MarkLessonRead records that the user has read a lesson. Idempotent: once a
// row exists, read_at is never overwritten by a later call.
func (s *Store) MarkLessonRead(ctx context.Context, userID, lessonID int64, at string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO user_lesson_state (user_id, lesson_id, read_at)
VALUES (?, ?, ?)
ON CONFLICT(user_id, lesson_id) DO NOTHING`, userID, lessonID, at)
	return err
}

// CountLessons returns the total number of lessons in the bank.
func (s *Store) CountLessons(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM lessons`).Scan(&n)
	return n, err
}

// CountReadLessons returns how many lessons the user has read.
func (s *Store) CountReadLessons(ctx context.Context, userID int64) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_lesson_state WHERE user_id = ?`, userID).Scan(&n)
	return n, err
}
