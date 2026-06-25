// Package seed loads the curriculum (steps, tasks, resources, quizzes) from an
// embedded JSON snapshot — ported verbatim from the original Consensus tracker —
// and UPSERTs it into Postgres on boot. Idempotent: re-running updates content
// in place via the (step_id, position) natural keys, so your progress survives.
package seed

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed data/seed.json
var seedJSON []byte

// snapshot mirrors the shape of the original app's SEED object.
type snapshot struct {
	Tasks []struct {
		Step     int    `json:"step"`
		Phase    string `json:"phase"`
		Week     string `json:"week"`
		Type     string `json:"type"`
		Task     string `json:"task"`
		Concept  string `json:"concept"`
		Hrs      int    `json:"hrs"`
		Deadline string `json:"deadline"`
	} `json:"tasks"`
	Resources []struct {
		Step int    `json:"step"`
		Name string `json:"name"`
		Type string `json:"type"`
		Free string `json:"free"`
		Link string `json:"link"`
		Why  string `json:"why"`
		Time string `json:"time"`
	} `json:"resources"`
	Quizzes []struct {
		Step int    `json:"step"`
		Type string `json:"type"`
		Q    string `json:"q"`
		Opts string `json:"opts"`
		Ans  string `json:"ans"`
		Exp  string `json:"exp"`
	} `json:"quizzes"`
	Phases map[string]string `json:"phases"`
}

// Run loads the embedded curriculum into the database within one transaction.
func Run(ctx context.Context, pool *pgxpool.Pool) error {
	var s snapshot
	if err := json.Unmarshal(seedJSON, &s); err != nil {
		return fmt.Errorf("parse seed.json: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit

	if err := seedSteps(ctx, tx, s.Phases); err != nil {
		return fmt.Errorf("seed steps: %w", err)
	}
	if err := seedTasks(ctx, tx, s); err != nil {
		return fmt.Errorf("seed tasks: %w", err)
	}
	if err := seedResources(ctx, tx, s); err != nil {
		return fmt.Errorf("seed resources: %w", err)
	}
	if err := seedQuizzes(ctx, tx, s); err != nil {
		return fmt.Errorf("seed quizzes: %w", err)
	}
	return tx.Commit(ctx)
}

func seedSteps(ctx context.Context, tx pgx.Tx, phases map[string]string) error {
	for numStr, phase := range phases {
		var num int
		if _, err := fmt.Sscanf(numStr, "%d", &num); err != nil {
			return err
		}
		_, err := tx.Exec(ctx,
			`INSERT INTO steps (id, phase, position) VALUES ($1, $2, $1)
			 ON CONFLICT (id) DO UPDATE SET phase = EXCLUDED.phase`,
			num, phase)
		if err != nil {
			return err
		}
	}
	return nil
}

func seedTasks(ctx context.Context, tx pgx.Tx, s snapshot) error {
	// position is per-step, assigned in source order.
	pos := map[int]int{}
	for _, t := range s.Tasks {
		pos[t.Step]++
		_, err := tx.Exec(ctx, `
			INSERT INTO tasks (step_id, phase, week, kind, title, concept, hrs, deadline, position)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			ON CONFLICT (step_id, position) DO UPDATE SET
				phase=EXCLUDED.phase, week=EXCLUDED.week, kind=EXCLUDED.kind,
				title=EXCLUDED.title, concept=EXCLUDED.concept, hrs=EXCLUDED.hrs,
				deadline=EXCLUDED.deadline`,
			t.Step, t.Phase, t.Week, t.Type, t.Task, t.Concept, t.Hrs, t.Deadline, pos[t.Step])
		if err != nil {
			return err
		}
	}
	return nil
}

func seedResources(ctx context.Context, tx pgx.Tx, s snapshot) error {
	pos := map[int]int{}
	for _, r := range s.Resources {
		pos[r.Step]++
		_, err := tx.Exec(ctx, `
			INSERT INTO resources (step_id, name, kind, is_free, link, why, time_label, position)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (step_id, position) DO UPDATE SET
				name=EXCLUDED.name, kind=EXCLUDED.kind, is_free=EXCLUDED.is_free,
				link=EXCLUDED.link, why=EXCLUDED.why, time_label=EXCLUDED.time_label`,
			r.Step, r.Name, r.Type, r.Free == "Free", r.Link, r.Why, r.Time, pos[r.Step])
		if err != nil {
			return err
		}
	}
	return nil
}

func seedQuizzes(ctx context.Context, tx pgx.Tx, s snapshot) error {
	pos := map[int]int{}
	for _, q := range s.Quizzes {
		pos[q.Step]++
		_, err := tx.Exec(ctx, `
			INSERT INTO quizzes (step_id, kind, question, options, answer, explanation, position)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT (step_id, position) DO UPDATE SET
				kind=EXCLUDED.kind, question=EXCLUDED.question, options=EXCLUDED.options,
				answer=EXCLUDED.answer, explanation=EXCLUDED.explanation`,
			q.Step, q.Type, q.Q, q.Opts, q.Ans, q.Exp, pos[q.Step])
		if err != nil {
			return err
		}
	}
	return nil
}
