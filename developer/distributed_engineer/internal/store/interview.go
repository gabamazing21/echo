package store

import (
	"context"

	"consensus/internal/models"
)

// InterviewByLesson returns a lesson's interview questions in order, with the
// learner's progress and any resolved in-app challenge merged in.
func (s *Store) InterviewByLesson(ctx context.Context, lessonSlug string) ([]models.InterviewQuestion, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT q.id, q.slug, q.lesson_slug, q.step_id, q.kind, q.difficulty, q.prompt,
		       q.companies, q.source_name, q.source_url, q.model_answer_html, q.challenge_slug, q.position,
		       COALESCE(ip.revealed,false), COALESCE(ip.outcome,'unset'),
		       (c.id IS NOT NULL), COALESCE(c.step_id,0)
		FROM interview_questions q
		LEFT JOIN interview_progress ip ON ip.question_id = q.id
		LEFT JOIN code_challenges c ON c.slug = q.challenge_slug
		WHERE q.lesson_slug=$1
		ORDER BY q.position`, lessonSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.InterviewQuestion
	for rows.Next() {
		var q models.InterviewQuestion
		if err := rows.Scan(&q.ID, &q.Slug, &q.LessonSlug, &q.StepID, &q.Kind, &q.Difficulty,
			&q.Prompt, &q.Companies, &q.SourceName, &q.SourceURL, &q.ModelAnswerHTML,
			&q.ChallengeSlug, &q.Position, &q.Revealed, &q.Outcome,
			&q.HasChallenge, &q.ChallengeStep); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// ExtraLessonsByStep returns "bonus" interview-prep lessons for a step — those
// whose position is beyond the step's regular tasks (so they aren't linked to a
// roadmap task). Body is not loaded.
func (s *Store) ExtraLessonsByStep(ctx context.Context, step int) ([]models.Lesson, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT slug, title, summary, est_min, position
		FROM lessons
		WHERE step_id=$1
		  AND position > (SELECT COALESCE(MAX(position),0) FROM tasks WHERE step_id=$1)
		ORDER BY position`, step)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Lesson
	for rows.Next() {
		var l models.Lesson
		l.StepID = step
		if err := rows.Scan(&l.Slug, &l.Title, &l.Summary, &l.EstMin, &l.Position); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// RevealInterview marks a question's model answer as revealed (and attempted).
func (s *Store) RevealInterview(ctx context.Context, slug string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO interview_progress (question_id, revealed, attempted, updated_at)
		SELECT id, true, true, now() FROM interview_questions WHERE slug=$1
		ON CONFLICT (question_id) DO UPDATE SET revealed=true, attempted=true, updated_at=now()`, slug)
	if err != nil {
		return err
	}
	return s.MarkStudied(ctx)
}

// SetInterviewOutcome records the learner's self-assessment (got_it | review | unset).
func (s *Store) SetInterviewOutcome(ctx context.Context, slug, outcome string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO interview_progress (question_id, outcome, attempted, updated_at)
		SELECT id, $2, true, now() FROM interview_questions WHERE slug=$1
		ON CONFLICT (question_id) DO UPDATE SET outcome=EXCLUDED.outcome, attempted=true, updated_at=now()`,
		slug, outcome)
	if err != nil {
		return err
	}
	return s.MarkStudied(ctx)
}
