package api

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// longestConsecutiveRun (pure function)
// ---------------------------------------------------------------------------

func TestLongestConsecutiveRun(t *testing.T) {
	tests := []struct {
		name  string
		dates []string
		want  int
	}{
		{"empty", nil, 0},
		{"single day", []string{"2026-01-01"}, 1},
		{"simple run", []string{"2026-01-01", "2026-01-02", "2026-01-03"}, 3},
		{
			"broken sequence: the longest run wins even when it's not the most recent",
			[]string{
				"2026-01-01", "2026-01-02", "2026-01-03", "2026-01-04", // run of 4
				"2026-01-10", "2026-01-11", // gap, then a shorter run of 2
			},
			4,
		},
		{
			"unordered input still finds the run",
			[]string{"2026-01-03", "2026-01-01", "2026-01-02", "2026-02-15"},
			3,
		},
		{"duplicate dates do not inflate the run", []string{"2026-01-01", "2026-01-01", "2026-01-02"}, 2},
		{
			"two equal-length runs: either length is the answer",
			[]string{"2026-01-01", "2026-01-02", "2026-01-10", "2026-01-11"},
			2,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := longestConsecutiveRun(tc.dates); got != tc.want {
				t.Errorf("longestConsecutiveRun(%v) = %d, want %d", tc.dates, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GET /api/me/stats: activity + streak_record
// ---------------------------------------------------------------------------

func TestHandleStats_Activity(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	uid, token := testUserToken(t, st, authSvc, "activity@example.com")

	ctx := context.Background()
	today := time.Now().UTC()
	todayStr := today.Format("2006-01-02")

	q1 := mustUpsertQuestion(t, st, "activity-question-1")
	q2 := mustUpsertQuestion(t, st, "activity-question-2")
	q3 := mustUpsertQuestion(t, st, "activity-question-old")

	// Two reviews today -> count 2 for today.
	if err := st.InsertReviewLog(ctx, uid, q1.ID, "good", today.Format(time.RFC3339)); err != nil {
		t.Fatalf("insert review log: %v", err)
	}
	if err := st.InsertReviewLog(ctx, uid, q2.ID, "easy", today.Add(time.Hour).Format(time.RFC3339)); err != nil {
		t.Fatalf("insert review log 2: %v", err)
	}

	// An event 90 days ago must be excluded from the 84-day heatmap window.
	old := today.AddDate(0, 0, -90)
	if err := st.InsertReviewLog(ctx, uid, q3.ID, "good", old.Format(time.RFC3339)); err != nil {
		t.Fatalf("insert old review log: %v", err)
	}

	_, body := doRequest(t, ts, http.MethodGet, "/api/me/stats", token, nil)
	activity, ok := body["activity"].([]any)
	if !ok {
		t.Fatalf("expected activity array, got %v", body["activity"])
	}
	if len(activity) != 1 {
		t.Fatalf("expected 1 active day within the 84-day window, got %d: %v", len(activity), activity)
	}
	item, _ := activity[0].(map[string]any)
	if item["date"] != todayStr {
		t.Errorf("date = %v, want %v", item["date"], todayStr)
	}
	if int(item["count"].(float64)) != 2 {
		t.Errorf("count = %v, want 2", item["count"])
	}
}

func TestHandleStats_ActivityEmptyWhenNoEvents(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	_, token := testUserToken(t, st, authSvc, "no-activity@example.com")

	_, body := doRequest(t, ts, http.MethodGet, "/api/me/stats", token, nil)
	activity, ok := body["activity"].([]any)
	if !ok {
		t.Fatalf("expected activity array, got %v", body["activity"])
	}
	if len(activity) != 0 {
		t.Errorf("expected empty activity, got %v", activity)
	}
	if int(body["streak_record"].(float64)) != 0 {
		t.Errorf("expected streak_record=0, got %v", body["streak_record"])
	}
}

func TestHandleStats_StreakRecordOnBrokenSequence(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	uid, token := testUserToken(t, st, authSvc, "streak@example.com")
	q := mustUpsertQuestion(t, st, "streak-question")

	ctx := context.Background()
	today := time.Now().UTC()

	// A run of 5 consecutive days well in the past...
	longRunStart := today.AddDate(0, 0, -20)
	for i := 0; i < 5; i++ {
		day := longRunStart.AddDate(0, 0, i)
		if err := st.InsertReviewLog(ctx, uid, q.ID, "good", day.Format(time.RFC3339)); err != nil {
			t.Fatalf("insert review log (long run, day %d): %v", i, err)
		}
	}
	// ...a gap...
	// ...then a shorter run of 2, ending today (the "current" streak).
	for i := 0; i < 2; i++ {
		day := today.AddDate(0, 0, -1+i)
		if err := st.InsertReviewLog(ctx, uid, q.ID, "good", day.Format(time.RFC3339)); err != nil {
			t.Fatalf("insert review log (recent run, day %d): %v", i, err)
		}
	}

	_, body := doRequest(t, ts, http.MethodGet, "/api/me/stats", token, nil)
	if int(body["streak_record"].(float64)) != 5 {
		t.Errorf("streak_record = %v, want 5 (the older, longer run)", body["streak_record"])
	}
	// streak_days is the *current* streak, anchored to today: it must reflect
	// only the recent 2-day run, unaffected by the older, longer one.
	if int(body["streak_days"].(float64)) != 2 {
		t.Errorf("streak_days = %v, want 2", body["streak_days"])
	}
}

func TestHandleStats_StreakRecordAcrossAllThreeSources(t *testing.T) {
	ts, st, authSvc := newTestServer(t)
	uid, token := testUserToken(t, st, authSvc, "mixed-sources@example.com")

	ctx := context.Background()
	today := time.Now().UTC()
	q := mustUpsertQuestion(t, st, "mixed-question")
	mustUpsertSQLTask(t, st, "mixed-task", "solution")
	task, err := st.GetCodingTaskBySlug(ctx, "mixed-task")
	if err != nil {
		t.Fatalf("get coding task: %v", err)
	}
	lesson := mustUpsertLesson(t, st, "mixed-lesson", nil, nil)

	// Three consecutive days, each active via a different source:
	// day-2 review, day-1 coding run, day-0 lesson read.
	day0 := today
	day1 := today.AddDate(0, 0, -1)
	day2 := today.AddDate(0, 0, -2)

	if err := st.InsertReviewLog(ctx, uid, q.ID, "good", day2.Format(time.RFC3339)); err != nil {
		t.Fatalf("insert review log: %v", err)
	}
	if err := st.RecordCodingRun(ctx, uid, task.ID, "code", true, day1.Format(time.RFC3339)); err != nil {
		t.Fatalf("record coding run: %v", err)
	}
	if err := st.MarkLessonRead(ctx, uid, lesson.ID, day0.Format(time.RFC3339)); err != nil {
		t.Fatalf("mark lesson read: %v", err)
	}

	_, body := doRequest(t, ts, http.MethodGet, "/api/me/stats", token, nil)
	if int(body["streak_record"].(float64)) != 3 {
		t.Errorf("streak_record = %v, want 3 (spanning review_log + task_run_log + user_lesson_state)", body["streak_record"])
	}
	activity, _ := body["activity"].([]any)
	if len(activity) != 3 {
		t.Fatalf("expected 3 active days in the heatmap, got %d: %v", len(activity), activity)
	}
}
