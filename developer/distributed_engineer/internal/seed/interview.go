package seed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"consensus/internal/lessons"
)

// interviewQ is one authored interview question (content/interview/<lesson-slug>.json).
type interviewQ struct {
	Slug          string `json:"slug"`
	Kind          string `json:"kind"`       // coding | sql | conceptual | design | behavioral
	Difficulty    string `json:"difficulty"` // easy | medium | hard | ""
	Prompt        string `json:"prompt"`
	Companies     string `json:"companies"`
	SourceName    string `json:"source_name"`
	SourceURL     string `json:"source_url"`
	ModelAnswer   string `json:"model_answer"` // markdown; rendered to HTML on sync
	ChallengeSlug string `json:"challenge_slug"`
}

// SyncInterview loads every content/interview/*.json file and UPSERTs its
// questions (by slug). The owning lesson's step_id is looked up from the lessons
// table (so lessons must be synced first). Returns the number of questions synced.
func SyncInterview(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS) (int, error) {
	entries, err := fs.ReadDir(fsys, "interview")
	if err != nil {
		return 0, fmt.Errorf("read interview dir: %w", err)
	}

	total := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		lessonSlug := strings.TrimSuffix(e.Name(), ".json")

		var stepID int
		err := pool.QueryRow(ctx, `SELECT step_id FROM lessons WHERE slug=$1`, lessonSlug).Scan(&stepID)
		if errors.Is(err, pgx.ErrNoRows) {
			// Question set for a lesson that doesn't exist yet — skip gracefully.
			continue
		}
		if err != nil {
			return total, fmt.Errorf("lookup step for %s: %w", lessonSlug, err)
		}

		raw, err := fs.ReadFile(fsys, "interview/"+e.Name())
		if err != nil {
			return total, err
		}
		var qs []interviewQ
		if err := json.Unmarshal(raw, &qs); err != nil {
			return total, fmt.Errorf("parse %s: %w", e.Name(), err)
		}

		for i, q := range qs {
			slug := q.Slug
			if slug == "" {
				slug = fmt.Sprintf("%s-q%d", lessonSlug, i+1)
			}
			answerHTML := ""
			if strings.TrimSpace(q.ModelAnswer) != "" {
				answerHTML, err = lessons.RenderHTML(q.ModelAnswer)
				if err != nil {
					return total, fmt.Errorf("render answer %s: %w", slug, err)
				}
			}
			_, err = pool.Exec(ctx, `
				INSERT INTO interview_questions
					(slug, lesson_slug, step_id, kind, difficulty, prompt, companies,
					 source_name, source_url, model_answer_html, challenge_slug, position)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
				ON CONFLICT (slug) DO UPDATE SET
					lesson_slug=EXCLUDED.lesson_slug, step_id=EXCLUDED.step_id, kind=EXCLUDED.kind,
					difficulty=EXCLUDED.difficulty, prompt=EXCLUDED.prompt, companies=EXCLUDED.companies,
					source_name=EXCLUDED.source_name, source_url=EXCLUDED.source_url,
					model_answer_html=EXCLUDED.model_answer_html, challenge_slug=EXCLUDED.challenge_slug,
					position=EXCLUDED.position`,
				slug, lessonSlug, stepID, q.Kind, q.Difficulty, q.Prompt, q.Companies,
				q.SourceName, q.SourceURL, answerHTML, q.ChallengeSlug, i+1)
			if err != nil {
				return total, fmt.Errorf("upsert %s: %w", slug, err)
			}
			total++
		}
	}
	return total, nil
}
