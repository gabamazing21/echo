-- Interview-prep questions attached to lessons, plus the learner's progress.
-- Content is synced from content/interview/*.json on boot (idempotent UPSERT by slug),
-- so progress (FK'd to the stable id) survives content updates.

CREATE TABLE interview_questions (
    id                BIGSERIAL PRIMARY KEY,
    slug              TEXT NOT NULL UNIQUE,        -- stable natural key
    lesson_slug       TEXT NOT NULL,               -- the lesson this attaches to
    step_id           INTEGER NOT NULL,
    kind              TEXT NOT NULL,               -- coding | sql | conceptual | design | behavioral
    difficulty        TEXT NOT NULL DEFAULT '',    -- easy | medium | hard | ''
    prompt            TEXT NOT NULL,
    companies         TEXT NOT NULL DEFAULT '',    -- e.g. "Google, Meta, Amazon" (public tags)
    source_name       TEXT NOT NULL DEFAULT '',    -- e.g. "NeetCode 150", "LeetCode #146"
    source_url        TEXT NOT NULL DEFAULT '',    -- attempt + official solution live here
    model_answer_html TEXT NOT NULL DEFAULT '',    -- rendered markdown; revealed after attempt
    challenge_slug    TEXT NOT NULL DEFAULT '',    -- -> code_challenges.slug for in-app attempt
    position          INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_interview_lesson ON interview_questions(lesson_slug);
CREATE INDEX idx_interview_step ON interview_questions(step_id);

CREATE TABLE interview_progress (
    question_id BIGINT PRIMARY KEY REFERENCES interview_questions(id) ON DELETE CASCADE,
    attempted   BOOLEAN NOT NULL DEFAULT false,
    revealed    BOOLEAN NOT NULL DEFAULT false,
    outcome     TEXT NOT NULL DEFAULT 'unset',   -- unset | got_it | review
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
