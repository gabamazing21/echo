// Package handlers wires HTTP routes to the store and renders pages.
package handlers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"consensus/internal/checker"
	"consensus/internal/store"
)

// Handler holds dependencies shared by all routes.
type Handler struct {
	store    *store.Store
	runner   *checker.Runner
	reviewer *checker.Reviewer
}

// New builds a Handler.
func New(s *store.Store, runner *checker.Runner, reviewer *checker.Reviewer) *Handler {
	return &Handler{store: s, runner: runner, reviewer: reviewer}
}

// Register attaches all routes to the Echo instance.
func (h *Handler) Register(e *echo.Echo) {
	e.GET("/", func(c echo.Context) error { return c.Redirect(http.StatusFound, "/today") })
	e.GET("/today", h.today)
	e.GET("/roadmap", h.roadmap)
	e.GET("/resources", h.resources)
	e.GET("/quiz", h.quizRedirect)
	e.GET("/quiz/:step", h.quiz)
	e.GET("/lesson/:slug", h.lesson)
	e.GET("/checker", h.checkerRedirect)
	e.GET("/checker/:step", h.checker)

	api := e.Group("/api")
	api.POST("/checker/:id/run", h.runChecker)
	api.POST("/checker/:id/review", h.reviewChecker)
	api.POST("/interview/:slug/reveal", h.revealInterview)
	api.POST("/interview/:slug/outcome", h.interviewOutcome)
	api.POST("/task/:id/toggle", h.toggleTask)
	api.POST("/task/:id/status", h.setTaskStatus)
	api.POST("/task/:id/notes", h.setTaskNotes)
	api.POST("/quiz/:id/grade", h.gradeQuiz)
	api.POST("/quiz/:id/self", h.quizSelf)
	api.POST("/quiz/:id/reveal", h.quizReveal)
	api.POST("/reset", h.reset)
}

// ----- pages -----

func (h *Handler) today(c echo.Context) error {
	ctx := c.Request().Context()
	plan, err := h.store.Today(ctx)
	if err != nil {
		return err
	}
	dash, err := h.store.Dashboard(ctx)
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "today.html", echo.Map{
		"Active": "today", "Plan": plan, "Dash": dash,
	})
}

func (h *Handler) roadmap(c echo.Context) error {
	ctx := c.Request().Context()
	steps, err := h.store.Steps(ctx)
	if err != nil {
		return err
	}
	type stepView struct {
		Step      any
		Tasks     any
		Resources any
		Extra     any
	}
	var views []stepView
	for _, st := range steps {
		tasks, err := h.store.TasksByStep(ctx, st.ID)
		if err != nil {
			return err
		}
		res, err := h.store.ResourcesByStep(ctx, st.ID)
		if err != nil {
			return err
		}
		extra, err := h.store.ExtraLessonsByStep(ctx, st.ID)
		if err != nil {
			return err
		}
		views = append(views, stepView{Step: st, Tasks: tasks, Resources: res, Extra: extra})
	}
	return c.Render(http.StatusOK, "roadmap.html", echo.Map{
		"Active": "roadmap", "Steps": views,
	})
}

func (h *Handler) resources(c echo.Context) error {
	ctx := c.Request().Context()
	res, err := h.store.AllResources(ctx)
	if err != nil {
		return err
	}
	steps, err := h.store.Steps(ctx)
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "resources.html", echo.Map{
		"Active": "resources", "Resources": res, "Steps": steps,
	})
}

func (h *Handler) quizRedirect(c echo.Context) error {
	return c.Redirect(http.StatusFound, "/quiz/1")
}

func (h *Handler) quiz(c echo.Context) error {
	ctx := c.Request().Context()
	step, _ := strconv.Atoi(c.Param("step"))
	if step == 0 {
		step = 1
	}
	quizzes, err := h.store.QuizzesByStep(ctx, step)
	if err != nil {
		return err
	}
	steps, err := h.store.Steps(ctx)
	if err != nil {
		return err
	}
	// theory score
	var theoryTotal, theoryGot, codeCount int
	for _, q := range quizzes {
		switch q.Kind {
		case "Theory":
			theoryTotal++
			if q.Progress.Correct {
				theoryGot++
			}
		case "Code":
			codeCount++
		}
	}
	return c.Render(http.StatusOK, "quiz.html", echo.Map{
		"Active": "quiz", "Step": step, "Quizzes": quizzes, "Steps": steps,
		"TheoryTotal": theoryTotal, "TheoryGot": theoryGot, "CodeCount": codeCount,
	})
}

func (h *Handler) lesson(c echo.Context) error {
	ctx := c.Request().Context()
	l, err := h.store.LessonBySlug(ctx, c.Param("slug"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "lesson not found")
	}
	steps, err := h.store.Steps(ctx)
	if err != nil {
		return err
	}
	interview, err := h.store.InterviewByLesson(ctx, l.Slug)
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "lesson.html", echo.Map{
		"Active": "roadmap", "Lesson": l, "Steps": steps, "Interview": interview,
	})
}

// ----- JSON API (progress mutations) -----

func idParam(c echo.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}

func (h *Handler) toggleTask(c echo.Context) error {
	id, err := idParam(c)
	if err != nil {
		return echo.ErrBadRequest
	}
	done, err := h.store.ToggleTask(c.Request().Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, echo.Map{"done": done})
}

func (h *Handler) setTaskStatus(c echo.Context) error {
	id, err := idParam(c)
	if err != nil {
		return echo.ErrBadRequest
	}
	status := c.FormValue("status")
	if err := h.store.SetTaskStatus(c.Request().Context(), id, status); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, echo.Map{"ok": true})
}

func (h *Handler) setTaskNotes(c echo.Context) error {
	id, err := idParam(c)
	if err != nil {
		return echo.ErrBadRequest
	}
	if err := h.store.SetTaskNotes(c.Request().Context(), id, c.FormValue("notes")); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, echo.Map{"ok": true})
}

func (h *Handler) gradeQuiz(c echo.Context) error {
	id, err := idParam(c)
	if err != nil {
		return echo.ErrBadRequest
	}
	correct, exp, ans, err := h.store.GradeQuiz(c.Request().Context(), id, c.FormValue("answer"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, echo.Map{"correct": correct, "explanation": exp, "answer": ans})
}

func (h *Handler) quizSelf(c echo.Context) error {
	id, err := idParam(c)
	if err != nil {
		return echo.ErrBadRequest
	}
	done := c.FormValue("done") == "true"
	if err := h.store.SetQuizSelfDone(c.Request().Context(), id, done); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, echo.Map{"ok": true})
}

func (h *Handler) quizReveal(c echo.Context) error {
	id, err := idParam(c)
	if err != nil {
		return echo.ErrBadRequest
	}
	if err := h.store.ToggleQuizReveal(c.Request().Context(), id); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, echo.Map{"ok": true})
}

func (h *Handler) reset(c echo.Context) error {
	if err := h.store.ResetProgress(c.Request().Context()); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, echo.Map{"ok": true})
}
