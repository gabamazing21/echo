package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// revealInterview marks a question's model answer as revealed (after the learner
// has attempted it).
func (h *Handler) revealInterview(c echo.Context) error {
	slug := c.Param("slug")
	if slug == "" {
		return echo.ErrBadRequest
	}
	if err := h.store.RevealInterview(c.Request().Context(), slug); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, echo.Map{"ok": true})
}

// interviewOutcome records the learner's self-assessment of a question.
func (h *Handler) interviewOutcome(c echo.Context) error {
	slug := c.Param("slug")
	outcome := c.FormValue("outcome")
	switch outcome {
	case "got_it", "review", "unset":
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid outcome")
	}
	if err := h.store.SetInterviewOutcome(c.Request().Context(), slug, outcome); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, echo.Map{"ok": true})
}
