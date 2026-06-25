package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"consensus/internal/models"
)

// ChallengeSteps returns the step numbers that have at least one challenge.
func (s *Store) ChallengeSteps(ctx context.Context) ([]int, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT step_id FROM code_challenges ORDER BY step_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var n int
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// ChallengesByStep returns a step's challenges with the learner's last attempt
// merged in. TestCode is intentionally left empty here (server-only).
func (s *Store) ChallengesByStep(ctx context.Context, step int) ([]models.Challenge, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.step_id, c.slug, c.title, c.prompt, c.starter_code, c.position,
		       last.code, last.passed
		FROM code_challenges c
		LEFT JOIN LATERAL (
			SELECT code, passed FROM submissions
			WHERE challenge_id = c.id ORDER BY created_at DESC LIMIT 1
		) last ON true
		WHERE c.step_id=$1 ORDER BY c.position`, step)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Challenge
	for rows.Next() {
		var c models.Challenge
		var lastCode *string
		var lastPassed *bool
		if err := rows.Scan(&c.ID, &c.StepID, &c.Slug, &c.Title, &c.Prompt,
			&c.StarterCode, &c.Position, &lastCode, &lastPassed); err != nil {
			return nil, err
		}
		if lastCode != nil {
			c.HasLast = true
			c.LastCode = *lastCode
			if lastPassed != nil {
				c.LastPassed = *lastPassed
			}
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Challenge fetches one challenge including the hidden TestCode.
func (s *Store) Challenge(ctx context.Context, id int64) (*models.Challenge, error) {
	var c models.Challenge
	err := s.pool.QueryRow(ctx, `
		SELECT id, step_id, slug, title, prompt, starter_code, test_code, position
		FROM code_challenges WHERE id=$1`, id).
		Scan(&c.ID, &c.StepID, &c.Slug, &c.Title, &c.Prompt, &c.StarterCode, &c.TestCode, &c.Position)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// SaveSubmission records a run and marks the day as studied.
func (s *Store) SaveSubmission(ctx context.Context, challengeID int64, code string, passed bool, output string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO submissions (challenge_id, code, passed, output)
		VALUES ($1,$2,$3,$4)`, challengeID, code, passed, output)
	if err != nil {
		return err
	}
	return s.MarkStudied(ctx)
}
