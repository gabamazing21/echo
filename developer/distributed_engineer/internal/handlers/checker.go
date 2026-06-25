package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

func (h *Handler) checkerRedirect(c echo.Context) error {
	steps, err := h.store.ChallengeSteps(c.Request().Context())
	if err != nil {
		return err
	}
	first := 1
	if len(steps) > 0 {
		first = steps[0]
	}
	return c.Redirect(http.StatusFound, "/checker/"+strconv.Itoa(first))
}

func (h *Handler) checker(c echo.Context) error {
	ctx := c.Request().Context()
	step, _ := strconv.Atoi(c.Param("step"))
	if step == 0 {
		step = 1
	}
	challenges, err := h.store.ChallengesByStep(ctx, step)
	if err != nil {
		return err
	}
	steps, err := h.store.ChallengeSteps(ctx)
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "checker.html", echo.Map{
		"Active": "checker", "Step": step, "Challenges": challenges,
		"Steps": steps, "ClaudeEnabled": h.reviewer.Enabled(),
	})
}

// runChecker compiles the submission with the hidden tests and runs `go test`.
func (h *Handler) runChecker(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := idParam(c)
	if err != nil {
		return echo.ErrBadRequest
	}
	ch, err := h.store.Challenge(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "challenge not found")
	}
	code := c.FormValue("code")
	res, err := h.runner.Run(ctx, code, ch.TestCode)
	if err != nil {
		return err
	}
	if err := h.store.SaveSubmission(ctx, id, code, res.Passed, res.Output); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, echo.Map{
		"passed": res.Passed, "output": res.Output,
		"timedOut": res.TimedOut, "ms": res.Duration.Milliseconds(),
	})
}

// reviewChecker runs the submission, then asks Claude for a mentor-style review.
func (h *Handler) reviewChecker(c echo.Context) error {
	ctx := c.Request().Context()
	if !h.reviewer.Enabled() {
		return c.JSON(http.StatusOK, echo.Map{
			"review": "Claude review isn't configured. Set `ANTHROPIC_API_KEY` to enable mentor-style code reviews.",
		})
	}
	id, err := idParam(c)
	if err != nil {
		return echo.ErrBadRequest
	}
	ch, err := h.store.Challenge(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "challenge not found")
	}
	code := c.FormValue("code")
	res, err := h.runner.Run(ctx, code, ch.TestCode)
	if err != nil {
		return err
	}
	review, err := h.reviewer.Review(ctx, ch.Title, ch.Prompt, code, res.Output, res.Passed)
	if err != nil {
		return c.JSON(http.StatusOK, echo.Map{"review": "Review failed: " + err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"review": review, "passed": res.Passed})
}
