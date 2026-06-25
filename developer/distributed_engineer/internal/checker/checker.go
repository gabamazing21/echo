// Package checker compiles a learner's Go submission together with a challenge's
// hidden test file and runs `go test`, returning pass/fail and the output —
// the ALX "checker" experience.
//
// Threat model: single-user, running the user's OWN code on their OWN machine,
// so we don't sandbox beyond a hard timeout + process kill. Before any public,
// multi-user deployment this MUST move into a container/gVisor sandbox.
package checker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Result is the outcome of running a submission.
type Result struct {
	Passed   bool          `json:"passed"`
	Output   string        `json:"output"`
	Duration time.Duration `json:"-"`
	TimedOut bool          `json:"timedOut"`
}

// Runner executes submissions. Timeout bounds a single run.
type Runner struct {
	Timeout time.Duration
}

// New builds a Runner with the given per-run timeout (seconds).
func New(timeoutSec int) *Runner {
	if timeoutSec <= 0 {
		timeoutSec = 20
	}
	return &Runner{Timeout: time.Duration(timeoutSec) * time.Second}
}

// Run writes userCode (solution.go) and testCode (solution_test.go) into a
// throwaway module and runs `go test`. Both files must be `package solution`.
func (r *Runner) Run(ctx context.Context, userCode, testCode string) (*Result, error) {
	dir, err := os.MkdirTemp("", "consensus-check-*")
	if err != nil {
		return nil, fmt.Errorf("mkdtemp: %w", err)
	}
	defer os.RemoveAll(dir)

	files := map[string]string{
		"go.mod":           "module solution\n\ngo 1.21\n",
		"solution.go":      userCode,
		"solution_test.go": testCode,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			return nil, fmt.Errorf("write %s: %w", name, err)
		}
	}

	runCtx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	start := time.Now()
	cmd := exec.CommandContext(runCtx, "go", "test", "-count=1", "-timeout", r.Timeout.String(), "./...")
	cmd.Dir = dir
	// Keep the module cache but isolate the build; disable network for safety.
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GONOSUMCHECK=1", "GOPROXY=off")

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	dur := time.Since(start)

	res := &Result{Output: cleanOutput(out.String(), dir), Duration: dur}
	if runCtx.Err() == context.DeadlineExceeded {
		res.TimedOut = true
		res.Passed = false
		if res.Output == "" {
			res.Output = fmt.Sprintf("⏱ timed out after %s — check for an infinite loop or deadlock.", r.Timeout)
		}
		return res, nil
	}
	// go test exits 0 on pass, non-zero on failure/compile error.
	res.Passed = err == nil
	return res, nil
}

// cleanOutput strips the temp-dir path from output so the learner sees tidy,
// relative file references (solution.go:12 instead of /var/folders/.../solution.go:12).
func cleanOutput(s, dir string) string {
	s = strings.ReplaceAll(s, dir+string(filepath.Separator), "")
	s = strings.ReplaceAll(s, dir, "")
	return strings.TrimSpace(s)
}
