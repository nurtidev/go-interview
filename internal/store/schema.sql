-- Schema for the go-interview backend. All timestamps are RFC3339 UTC strings.
-- CREATE TABLE IF NOT EXISTS is sufficient for the MVP migration story.

-- name/interview_date power the profile screen (GET/PATCH /api/me); both are
-- nullable and unset until the user fills them in. NOTE: CREATE TABLE IF NOT
-- EXISTS never alters an already-existing table, so pre-existing local
-- databases won't pick these columns up from here alone; store.Open
-- additionally runs an idempotent ALTER TABLE ADD COLUMN migration for
-- exactly this reason (see migrateUserProfileColumns).
CREATE TABLE IF NOT EXISTS users (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    email          TEXT UNIQUE NOT NULL,
    password_hash  TEXT NOT NULL,
    created_at     TEXT NOT NULL,
    name           TEXT,          -- nullable
    interview_date TEXT           -- nullable; YYYY-MM-DD
);

CREATE TABLE IF NOT EXISTS questions (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    slug          TEXT UNIQUE NOT NULL,
    section       TEXT NOT NULL,
    title         TEXT NOT NULL,
    difficulty    TEXT NOT NULL,
    tags          TEXT NOT NULL,          -- json array
    question_md   TEXT NOT NULL,
    answer_levels TEXT NOT NULL,          -- json array of {level, text_md}
    follow_ups    TEXT NOT NULL,          -- json array
    position      INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_questions_section ON questions(section, position);

CREATE TABLE IF NOT EXISTS user_question_state (
    user_id       INTEGER NOT NULL,
    question_id   INTEGER NOT NULL,
    ease          REAL NOT NULL DEFAULT 2.5,
    interval_days REAL NOT NULL DEFAULT 0,
    repetitions   INTEGER NOT NULL DEFAULT 0,
    due_at        TEXT NOT NULL,
    status        TEXT NOT NULL,          -- learning|review
    updated_at    TEXT NOT NULL,
    PRIMARY KEY (user_id, question_id)
);

CREATE INDEX IF NOT EXISTS idx_uqs_due ON user_question_state(user_id, due_at);

CREATE TABLE IF NOT EXISTS review_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL,
    question_id INTEGER NOT NULL,
    grade       TEXT NOT NULL,
    reviewed_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_review_log_user ON review_log(user_id, reviewed_at);

-- Livecoding: interactive coding/SQL tasks with an in-browser runner.

CREATE TABLE IF NOT EXISTS coding_tasks (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    slug           TEXT UNIQUE NOT NULL,
    kind           TEXT NOT NULL,          -- go|sql
    title          TEXT NOT NULL,
    difficulty     TEXT NOT NULL,
    tags           TEXT NOT NULL,          -- json array
    statement_md   TEXT NOT NULL,
    starter_code   TEXT NOT NULL,
    hints          TEXT NOT NULL,          -- json array
    solution_md    TEXT NOT NULL,
    time_limit_sec INTEGER NOT NULL DEFAULT 0,
    race           INTEGER NOT NULL DEFAULT 0,
    test_code      TEXT NOT NULL,
    schema_sql     TEXT NOT NULL,
    seed_sql       TEXT NOT NULL,
    expected       TEXT NOT NULL,          -- json object {columns, rows, order_matters}
    position       INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_coding_tasks_kind ON coding_tasks(kind, position);

-- due_at/gave_up/solve_count power the "25/5" framework's give-up and
-- re-solve mechanics. NOTE: CREATE TABLE IF NOT EXISTS never alters an
-- already-existing table, so pre-existing local databases won't pick these
-- columns up from here alone; store.Open additionally runs idempotent
-- ALTER TABLE ADD COLUMN migrations for exactly this reason.
CREATE TABLE IF NOT EXISTS user_task_state (
    user_id     INTEGER NOT NULL,
    task_id     INTEGER NOT NULL,
    status      TEXT NOT NULL,          -- attempted|solved
    last_code   TEXT NOT NULL,
    solved_at   TEXT,                   -- RFC3339, NULL until solved
    updated_at  TEXT NOT NULL,
    due_at      TEXT,                   -- RFC3339, NULL until first solve; when to re-solve
    gave_up     INTEGER NOT NULL DEFAULT 0,  -- 0|1, set by POST .../giveup, cleared on next solve
    solve_count INTEGER NOT NULL DEFAULT 0,  -- times solved; drives the due_at spacing (7/21/60 days)
    PRIMARY KEY (user_id, task_id)
);

CREATE TABLE IF NOT EXISTS task_run_log (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    task_id INTEGER NOT NULL,
    passed  INTEGER NOT NULL,          -- 0|1
    ran_at  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_task_run_log_user ON task_run_log(user_id, ran_at);

-- "Учебник": lessons with inline diagrams, each linking back to questions and
-- coding tasks for reinforcement.

CREATE TABLE IF NOT EXISTS lessons (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    slug              TEXT UNIQUE NOT NULL,
    topic             TEXT NOT NULL,          -- go-internals|concurrency
    title             TEXT NOT NULL,
    minutes           INTEGER NOT NULL DEFAULT 0,
    tags              TEXT NOT NULL,          -- json array
    body_md           TEXT NOT NULL,
    related_questions TEXT NOT NULL,          -- json array of question slugs
    related_tasks     TEXT NOT NULL,          -- json array of coding_task slugs
    position          INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_lessons_position ON lessons(position);

CREATE TABLE IF NOT EXISTS user_lesson_state (
    user_id   INTEGER NOT NULL,
    lesson_id INTEGER NOT NULL,
    read_at   TEXT NOT NULL,
    PRIMARY KEY (user_id, lesson_id)
);
