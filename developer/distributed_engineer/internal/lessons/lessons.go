// Package lessons renders authored Markdown lessons to safe HTML and syncs them
// from the embedded content filesystem into the database on boot.
//
// A lesson is a Markdown file with a YAML frontmatter header, e.g.:
//
//	---
//	slug: go-toolchain-first-program
//	step: 1
//	title: The Go toolchain & your first program
//	summary: Install Go, learn go run/build/mod, ship hello world.
//	est_min: 45
//	position: 1
//	---
//	# Lesson body in Markdown...
package lessons

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	meta "github.com/yuin/goldmark-meta"
)

// md is a configured Markdown renderer: GFM tables/strikethrough/autolinks,
// frontmatter parsing, and (intentionally) no raw HTML passthrough.
var md = goldmark.New(
	goldmark.WithExtensions(highlighting.GFM, meta.Meta),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(html.WithHardWraps()),
)

// Rendered is a lesson converted to HTML plus its parsed metadata.
type Rendered struct {
	Slug     string
	StepID   int
	Title    string
	Summary  string
	EstMin   int
	Position int
	HTML     string
}

// RenderHTML converts a plain Markdown string (no frontmatter) to safe HTML.
// Used for interview-question model answers.
func RenderHTML(markdown string) (string, error) {
	var buf bytes.Buffer
	if err := md.Convert([]byte(markdown), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Render parses one lesson file's bytes into HTML + metadata.
func Render(raw []byte) (*Rendered, error) {
	var buf bytes.Buffer
	ctx := parser.NewContext()
	if err := md.Convert(raw, &buf, parser.WithContext(ctx)); err != nil {
		return nil, err
	}
	m := meta.Get(ctx)

	r := &Rendered{
		Slug:    asString(m["slug"]),
		Title:   asString(m["title"]),
		Summary: asString(m["summary"]),
		HTML:    buf.String(),
	}
	r.StepID = asInt(m["step"])
	r.EstMin = asInt(m["est_min"])
	r.Position = asInt(m["position"])

	if r.Slug == "" {
		return nil, fmt.Errorf("lesson missing 'slug' in frontmatter")
	}
	if r.Title == "" {
		return nil, fmt.Errorf("lesson %q missing 'title'", r.Slug)
	}
	return r, nil
}

// Sync renders every lessons/*.md in fsys and UPSERTs it into the lessons table.
// Returns the number of lessons synced.
func Sync(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS) (int, error) {
	entries, err := fs.ReadDir(fsys, "lessons")
	if err != nil {
		return 0, fmt.Errorf("read lessons dir: %w", err)
	}

	n := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		raw, err := fs.ReadFile(fsys, "lessons/"+e.Name())
		if err != nil {
			return n, err
		}
		r, err := Render(raw)
		if err != nil {
			return n, fmt.Errorf("%s: %w", e.Name(), err)
		}

		var stepID any = r.StepID
		if r.StepID == 0 {
			stepID = nil // NULL — unattached lesson
		}
		_, err = pool.Exec(ctx, `
			INSERT INTO lessons (slug, step_id, title, summary, body_md, est_min, position, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7, now())
			ON CONFLICT (slug) DO UPDATE SET
				step_id=EXCLUDED.step_id, title=EXCLUDED.title, summary=EXCLUDED.summary,
				body_md=EXCLUDED.body_md, est_min=EXCLUDED.est_min, position=EXCLUDED.position,
				updated_at=now()`,
			r.Slug, stepID, r.Title, r.Summary, r.HTML, r.EstMin, r.Position)
		if err != nil {
			return n, fmt.Errorf("upsert %s: %w", r.Slug, err)
		}
		n++
	}
	return n, nil
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func asInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}
