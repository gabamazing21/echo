// Command server is the Consensus learning-platform HTTP server.
package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	contentfs "consensus/content"
	"consensus/internal/checker"
	"consensus/internal/config"
	"consensus/internal/db"
	"consensus/internal/handlers"
	"consensus/internal/lessons"
	"consensus/internal/seed"
	"consensus/internal/store"
	"consensus/migrations"
	"consensus/web"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Root context cancelled on SIGINT/SIGTERM for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info("running migrations")
	if err := db.Migrate(cfg.DatabaseURL, migrations.FS); err != nil {
		return err
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	log.Info("connected to postgres")

	log.Info("seeding curriculum")
	if err := seed.Run(ctx, pool); err != nil {
		return err
	}

	if n, err := lessons.Sync(ctx, pool, contentfs.Lessons); err != nil {
		return err
	} else {
		log.Info("synced lessons", "count", n)
	}

	if n, err := seed.SyncChallenges(ctx, pool); err != nil {
		return err
	} else {
		log.Info("synced code challenges", "count", n)
	}

	if n, err := seed.SyncInterview(ctx, pool, contentfs.Interview); err != nil {
		return err
	} else {
		log.Info("synced interview questions", "count", n)
	}

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())

	// Optional HTTP Basic Auth over the whole app (except /healthz so the
	// platform health check still works). Enabled when APP_USER+APP_PASSWORD are
	// set — essential for a public deploy because the checker runs go test.
	if cfg.BasicAuthUser != "" && cfg.BasicAuthPassword != "" {
		e.Use(middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
			Skipper: func(c echo.Context) bool { return c.Path() == "/healthz" },
			Validator: func(u, p string, _ echo.Context) (bool, error) {
				return subtle.ConstantTimeCompare([]byte(u), []byte(cfg.BasicAuthUser)) == 1 &&
					subtle.ConstantTimeCompare([]byte(p), []byte(cfg.BasicAuthPassword)) == 1, nil
			},
		}))
		log.Info("basic auth enabled")
	}
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus: true, LogURI: true, LogMethod: true, LogLatency: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log.Info("req", "method", v.Method, "uri", v.URI, "status", v.Status, "dur", v.Latency.String())
			return nil
		},
	}))

	// Templates + static assets (both embedded in the binary).
	renderer, err := handlers.NewRenderer(web.Templates)
	if err != nil {
		return err
	}
	e.Renderer = renderer
	e.StaticFS("/static", echo.MustSubFS(web.Static, "static"))

	e.GET("/healthz", func(c echo.Context) error {
		if err := pool.Ping(c.Request().Context()); err != nil {
			return c.JSON(http.StatusServiceUnavailable, echo.Map{"status": "db down"})
		}
		return c.JSON(http.StatusOK, echo.Map{"status": "ok"})
	})

	// Application routes.
	runner := checker.New(cfg.CheckerTimeoutSec)
	reviewer := checker.NewReviewer(cfg.AnthropicAPIKey)
	h := handlers.New(store.New(pool), runner, reviewer)
	h.Register(e)

	// Start server in a goroutine so we can wait on the shutdown signal.
	go func() {
		addr := ":" + cfg.Port
		log.Info("listening", "addr", addr, "env", cfg.Env)
		if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return e.Shutdown(shutdownCtx)
}
