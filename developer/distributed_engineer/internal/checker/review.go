package checker

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Reviewer asks Claude to review a learner's submission like a mentor.
type Reviewer struct{ apiKey string }

// NewReviewer builds a Reviewer. An empty apiKey disables the feature.
func NewReviewer(apiKey string) *Reviewer { return &Reviewer{apiKey: apiKey} }

// Enabled reports whether Claude review is configured.
func (r *Reviewer) Enabled() bool { return r.apiKey != "" }

const reviewSystem = `You are a senior Go engineer mentoring someone learning to become a distributed systems engineer. Review their solution to a coding challenge.

Be warm but rigorous, like a great ALX peer reviewer. Cover, briefly:
1. Correctness — does it actually solve the problem? Any bugs or missed edge cases?
2. Idiomatic Go — naming, error handling, slices/maps usage, would a Go reviewer flag anything?
3. One concrete improvement they should make next.

Keep it under ~250 words. Use plain language. Do not rewrite the whole solution for them — teach, don't solve. Format with short markdown sections.`

// Review returns Claude's review of the submission as markdown.
func (r *Reviewer) Review(ctx context.Context, title, prompt, code, testOutput string, passed bool) (string, error) {
	if r.apiKey == "" {
		return "", fmt.Errorf("claude review not configured: set ANTHROPIC_API_KEY")
	}
	client := anthropic.NewClient(option.WithAPIKey(r.apiKey))

	status := "FAILED the hidden tests"
	if passed {
		status = "PASSED the hidden tests"
	}
	user := fmt.Sprintf(
		"Challenge: %s\n\nProblem:\n%s\n\nThe learner's submission %s.\n\nTest output:\n```\n%s\n```\n\nTheir code:\n```go\n%s\n```",
		title, prompt, status, truncate(testOutput, 4000), code)

	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeOpus4_8,
		MaxTokens: 8000,
		System:    []anthropic.TextBlockParam{{Text: reviewSystem}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(user)),
		},
	})
	if err != nil {
		return "", err
	}

	var b strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			b.WriteString(t.Text)
		}
	}
	return strings.TrimSpace(b.String()), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n…(truncated)"
}
