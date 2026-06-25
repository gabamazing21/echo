-- Consensus schema, v1. Single-user learning platform.
-- Content tables (steps, tasks, resources, quizzes, lessons, code_challenges)
-- are seeded/synced from source-of-truth data. Progress tables hold YOUR state.

-- ----- content: the curriculum spine -----

CREATE TABLE steps (
    id       INTEGER PRIMARY KEY,          -- 1..10, the step number
    phase    TEXT    NOT NULL,             -- e.g. "Go fundamentals"
    position INTEGER NOT NULL
);

CREATE TABLE tasks (
    id       BIGSERIAL PRIMARY KEY,
    step_id  INTEGER NOT NULL REFERENCES steps(id) ON DELETE CASCADE,
    phase    TEXT    NOT NULL,
    week     TEXT    NOT NULL,             -- "W1".."W4"
    kind     TEXT    NOT NULL,             -- Topic | Code project | Quiz | Practice | Task
    title    TEXT    NOT NULL,
    concept  TEXT    NOT NULL,
    hrs      INTEGER NOT NULL,
    deadline DATE    NOT NULL,
    -- optional link to a long-form lesson that teaches this task's concept
    lesson_slug TEXT,
    position INTEGER NOT NULL
);
CREATE INDEX idx_tasks_step ON tasks(step_id);
CREATE INDEX idx_tasks_deadline ON tasks(deadline);

CREATE TABLE resources (
    id         BIGSERIAL PRIMARY KEY,
    step_id    INTEGER NOT NULL REFERENCES steps(id) ON DELETE CASCADE,
    name       TEXT    NOT NULL,
    kind       TEXT    NOT NULL,
    is_free    BOOLEAN NOT NULL,
    link       TEXT    NOT NULL,
    why        TEXT    NOT NULL,
    time_label TEXT    NOT NULL,
    position   INTEGER NOT NULL
);
CREATE INDEX idx_resources_step ON resources(step_id);

CREATE TABLE quizzes (
    id          BIGSERIAL PRIMARY KEY,
    step_id     INTEGER NOT NULL REFERENCES steps(id) ON DELETE CASCADE,
    kind        TEXT    NOT NULL,          -- Theory | Code
    question    TEXT    NOT NULL,
    options     TEXT    NOT NULL DEFAULT '',
    answer      TEXT    NOT NULL DEFAULT '',
    explanation TEXT    NOT NULL DEFAULT '',
    position    INTEGER NOT NULL
);
CREATE INDEX idx_quizzes_step ON quizzes(step_id);

-- Long-form authored lessons, synced from content/lessons/*.md on boot.
CREATE TABLE lessons (
    id         BIGSERIAL PRIMARY KEY,
    slug       TEXT NOT NULL UNIQUE,
    step_id    INTEGER REFERENCES steps(id) ON DELETE SET NULL,
    title      TEXT NOT NULL,
    summary    TEXT NOT NULL DEFAULT '',
    body_md    TEXT NOT NULL,
    est_min    INTEGER NOT NULL DEFAULT 0,
    position   INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_lessons_step ON lessons(step_id);

-- Auto-gradable code challenges (the "checker"). Hidden test_code is never
-- shown to the learner; it's compiled alongside their submission.
CREATE TABLE code_challenges (
    id           BIGSERIAL PRIMARY KEY,
    step_id      INTEGER NOT NULL REFERENCES steps(id) ON DELETE CASCADE,
    slug         TEXT NOT NULL UNIQUE,
    title        TEXT NOT NULL,
    prompt       TEXT NOT NULL,
    starter_code TEXT NOT NULL DEFAULT '',
    test_code    TEXT NOT NULL,            -- hidden _test.go contents
    position     INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_challenges_step ON code_challenges(step_id);

-- Scraper-cached free reference content (with attribution), surfaced beside lessons.
CREATE TABLE cached_sources (
    id          BIGSERIAL PRIMARY KEY,
    url         TEXT NOT NULL UNIQUE,
    title       TEXT NOT NULL DEFAULT '',
    kind        TEXT NOT NULL DEFAULT '',
    excerpt     TEXT NOT NULL DEFAULT '',  -- sanitized excerpt, not full re-host
    attribution TEXT NOT NULL DEFAULT '',
    step_id     INTEGER REFERENCES steps(id) ON DELETE SET NULL,
    fetched_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ----- progress: YOUR state (single user) -----

CREATE TABLE task_progress (
    task_id      BIGINT PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
    done         BOOLEAN NOT NULL DEFAULT false,
    status       TEXT NOT NULL DEFAULT 'Not started',
    notes        TEXT NOT NULL DEFAULT '',
    completed_at TIMESTAMPTZ
);

CREATE TABLE quiz_progress (
    quiz_id   BIGINT PRIMARY KEY REFERENCES quizzes(id) ON DELETE CASCADE,
    answer    TEXT NOT NULL DEFAULT '',
    correct   BOOLEAN NOT NULL DEFAULT false,
    self_done BOOLEAN NOT NULL DEFAULT false,
    revealed  BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE submissions (
    id           BIGSERIAL PRIMARY KEY,
    challenge_id BIGINT NOT NULL REFERENCES code_challenges(id) ON DELETE CASCADE,
    code         TEXT NOT NULL,
    passed       BOOLEAN NOT NULL,
    output       TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_submissions_challenge ON submissions(challenge_id);

-- One row per day you studied — powers the streak.
CREATE TABLE study_log (
    day         DATE PRIMARY KEY,
    minutes     INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
