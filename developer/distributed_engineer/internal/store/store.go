// Package store is the data-access layer: every read and write the app needs,
// expressed as methods on Store. Handlers depend on this, never on raw SQL.
package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"consensus/internal/models"
)

// Store wraps the connection pool.
type Store struct{ pool *pgxpool.Pool }

// New constructs a Store.
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// ----- steps -----

// Steps returns all steps with progress aggregates and derived deadline.
func (s *Store) Steps(ctx context.Context) ([]models.Step, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT st.id, st.phase, st.position,
		       count(t.id) AS total,
		       count(t.id) FILTER (WHERE tp.done) AS done,
		       max(t.deadline) AS deadline
		FROM steps st
		LEFT JOIN tasks t ON t.step_id = st.id
		LEFT JOIN task_progress tp ON tp.task_id = t.id
		GROUP BY st.id, st.phase, st.position
		ORDER BY st.position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Step
	for rows.Next() {
		var st models.Step
		var dl *time.Time
		if err := rows.Scan(&st.ID, &st.Phase, &st.Position, &st.TotalTasks, &st.DoneTasks, &dl); err != nil {
			return nil, err
		}
		if dl != nil {
			st.Deadline = *dl
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// ----- tasks -----

const taskCols = `
	t.id, t.step_id, t.phase, t.week, t.kind, t.title, t.concept, t.hrs, t.deadline,
	COALESCE(l.slug,''),
	t.position,
	COALESCE(tp.done,false), COALESCE(tp.status,'Not started'), COALESCE(tp.notes,''),
	(l.id IS NOT NULL) AS has_lesson`

const taskJoins = `
	FROM tasks t
	LEFT JOIN task_progress tp ON tp.task_id = t.id
	LEFT JOIN lessons l ON l.step_id = t.step_id AND l.position = t.position`

func scanTasks(rows pgx.Rows) ([]models.Task, error) {
	defer rows.Close()
	var out []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.StepID, &t.Phase, &t.Week, &t.Kind, &t.Title,
			&t.Concept, &t.Hrs, &t.Deadline, &t.LessonSlug, &t.Position,
			&t.Done, &t.Status, &t.Notes, &t.HasLesson); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// TasksByStep returns a step's tasks in order, with progress merged.
func (s *Store) TasksByStep(ctx context.Context, step int) ([]models.Task, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+taskCols+taskJoins+
		` WHERE t.step_id=$1 ORDER BY t.position`, step)
	if err != nil {
		return nil, err
	}
	return scanTasks(rows)
}

// AllTasks returns every task ordered by step then position.
func (s *Store) AllTasks(ctx context.Context) ([]models.Task, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+taskCols+taskJoins+
		` ORDER BY t.step_id, t.position`)
	if err != nil {
		return nil, err
	}
	return scanTasks(rows)
}

// ----- resources -----

// ResourcesByStep returns one step's resources.
func (s *Store) ResourcesByStep(ctx context.Context, step int) ([]models.Resource, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, step_id, name, kind, is_free, link, why, time_label, position
		FROM resources WHERE step_id=$1 ORDER BY position`, step)
	if err != nil {
		return nil, err
	}
	return scanResources(rows)
}

// AllResources returns every resource ordered by step then position.
func (s *Store) AllResources(ctx context.Context) ([]models.Resource, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, step_id, name, kind, is_free, link, why, time_label, position
		FROM resources ORDER BY step_id, position`)
	if err != nil {
		return nil, err
	}
	return scanResources(rows)
}

func scanResources(rows pgx.Rows) ([]models.Resource, error) {
	defer rows.Close()
	var out []models.Resource
	for rows.Next() {
		var r models.Resource
		if err := rows.Scan(&r.ID, &r.StepID, &r.Name, &r.Kind, &r.IsFree,
			&r.Link, &r.Why, &r.TimeLabel, &r.Position); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ----- quizzes -----

// QuizzesByStep returns one step's quizzes with saved progress.
func (s *Store) QuizzesByStep(ctx context.Context, step int) ([]models.Quiz, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT q.id, q.step_id, q.kind, q.question, q.options, q.answer, q.explanation, q.position,
		       COALESCE(qp.answer,''), COALESCE(qp.correct,false),
		       COALESCE(qp.self_done,false), COALESCE(qp.revealed,false),
		       (qp.quiz_id IS NOT NULL) AS answered
		FROM quizzes q
		LEFT JOIN quiz_progress qp ON qp.quiz_id = q.id
		WHERE q.step_id=$1 ORDER BY q.position`, step)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Quiz
	for rows.Next() {
		var q models.Quiz
		var answeredRow bool
		if err := rows.Scan(&q.ID, &q.StepID, &q.Kind, &q.Question, &q.Options,
			&q.Answer, &q.Explanation, &q.Position,
			&q.Progress.AnswerGiven, &q.Progress.Correct,
			&q.Progress.SelfDone, &q.Progress.Revealed, &answeredRow); err != nil {
			return nil, err
		}
		// "Answered" for theory means a graded answer exists.
		q.Progress.Answered = answeredRow && q.Progress.AnswerGiven != ""
		out = append(out, q)
	}
	return out, rows.Err()
}

// ----- lessons -----

// LessonBySlug fetches one rendered lesson.
func (s *Store) LessonBySlug(ctx context.Context, slug string) (*models.Lesson, error) {
	var l models.Lesson
	var stepID *int
	err := s.pool.QueryRow(ctx, `
		SELECT id, slug, COALESCE(step_id,0), title, summary, body_md, est_min, position
		FROM lessons WHERE slug=$1`, slug).
		Scan(&l.ID, &l.Slug, &stepID, &l.Title, &l.Summary, &l.BodyHTML, &l.EstMin, &l.Position)
	if err != nil {
		return nil, err
	}
	if stepID != nil {
		l.StepID = *stepID
	}
	return &l, nil
}

// LessonForTask returns the lesson attached to a task's (step,position) slot, if any.
func (s *Store) LessonForTask(ctx context.Context, taskID int64) (*models.Lesson, error) {
	var l models.Lesson
	err := s.pool.QueryRow(ctx, `
		SELECT l.id, l.slug, COALESCE(l.step_id,0), l.title, l.summary, l.body_md, l.est_min, l.position
		FROM tasks t JOIN lessons l ON l.step_id=t.step_id AND l.position=t.position
		WHERE t.id=$1`, taskID).
		Scan(&l.ID, &l.Slug, &l.StepID, &l.Title, &l.Summary, &l.BodyHTML, &l.EstMin, &l.Position)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// ----- progress mutations -----

// ToggleTask flips a task's done flag and records the study day.
func (s *Store) ToggleTask(ctx context.Context, taskID int64) (bool, error) {
	var done bool
	err := s.pool.QueryRow(ctx, `
		INSERT INTO task_progress (task_id, done, status, completed_at)
		VALUES ($1, true, 'Done', now())
		ON CONFLICT (task_id) DO UPDATE SET
			done = NOT task_progress.done,
			status = CASE WHEN NOT task_progress.done THEN 'Done' ELSE 'In progress' END,
			completed_at = CASE WHEN NOT task_progress.done THEN now() ELSE NULL END
		RETURNING done`, taskID).Scan(&done)
	if err != nil {
		return false, err
	}
	return done, s.MarkStudied(ctx)
}

// SetTaskStatus sets an explicit status (and syncs the done flag).
func (s *Store) SetTaskStatus(ctx context.Context, taskID int64, status string) error {
	done := status == "Done"
	_, err := s.pool.Exec(ctx, `
		INSERT INTO task_progress (task_id, status, done, completed_at)
		VALUES ($1,$2,$3, CASE WHEN $3 THEN now() ELSE NULL END)
		ON CONFLICT (task_id) DO UPDATE SET
			status=EXCLUDED.status, done=EXCLUDED.done,
			completed_at=CASE WHEN EXCLUDED.done THEN now() ELSE NULL END`,
		taskID, status, done)
	if err != nil {
		return err
	}
	return s.MarkStudied(ctx)
}

// SetTaskNotes saves freeform notes on a task.
func (s *Store) SetTaskNotes(ctx context.Context, taskID int64, notes string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO task_progress (task_id, notes) VALUES ($1,$2)
		ON CONFLICT (task_id) DO UPDATE SET notes=EXCLUDED.notes`, taskID, notes)
	return err
}

// GradeQuiz records a theory answer and whether it was correct.
func (s *Store) GradeQuiz(ctx context.Context, quizID int64, answer string) (correct bool, explanation, correctAns string, err error) {
	err = s.pool.QueryRow(ctx, `SELECT answer, explanation FROM quizzes WHERE id=$1`, quizID).
		Scan(&correctAns, &explanation)
	if err != nil {
		return
	}
	correct = answer == correctAns
	_, err = s.pool.Exec(ctx, `
		INSERT INTO quiz_progress (quiz_id, answer, correct, updated_at)
		VALUES ($1,$2,$3, now())
		ON CONFLICT (quiz_id) DO UPDATE SET answer=EXCLUDED.answer, correct=EXCLUDED.correct, updated_at=now()`,
		quizID, answer, correct)
	if err != nil {
		return
	}
	err = s.MarkStudied(ctx)
	return
}

// SetQuizSelfDone marks a code quiz as self-completed.
func (s *Store) SetQuizSelfDone(ctx context.Context, quizID int64, done bool) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO quiz_progress (quiz_id, self_done) VALUES ($1,$2)
		ON CONFLICT (quiz_id) DO UPDATE SET self_done=EXCLUDED.self_done, updated_at=now()`, quizID, done)
	if err != nil {
		return err
	}
	return s.MarkStudied(ctx)
}

// ToggleQuizReveal flips whether a code quiz's approach is revealed.
func (s *Store) ToggleQuizReveal(ctx context.Context, quizID int64) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO quiz_progress (quiz_id, revealed) VALUES ($1, true)
		ON CONFLICT (quiz_id) DO UPDATE SET revealed = NOT quiz_progress.revealed`, quizID)
	return err
}

// MarkStudied records that the learner studied today (idempotent per day).
func (s *Store) MarkStudied(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO study_log (day) VALUES (CURRENT_DATE)
		ON CONFLICT (day) DO NOTHING`)
	return err
}

// ResetProgress clears all of the learner's state (keeps content).
func (s *Store) ResetProgress(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	for _, q := range []string{
		`DELETE FROM submissions`,
		`DELETE FROM quiz_progress`,
		`DELETE FROM task_progress`,
		`DELETE FROM interview_progress`,
		`DELETE FROM study_log`,
	} {
		if _, err := tx.Exec(ctx, q); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
