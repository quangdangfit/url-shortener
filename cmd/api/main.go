package main

import (
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/quangdangfit/url-shortener/internal/config"
	"github.com/quangdangfit/url-shortener/internal/db"
	"github.com/quangdangfit/url-shortener/internal/handler"
	"github.com/quangdangfit/url-shortener/internal/repository"
	"github.com/quangdangfit/url-shortener/internal/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := config.Load()

	session, err := db.NewSession(cfg.ScyllaHosts, cfg.ScyllaKeyspace, cfg.ScyllaConsistency)
	if err != nil {
		slog.Error("failed to connect to scylla", "error", err)
		os.Exit(1)
	}
	defer session.Close()
	slog.Info("connected to scylla")

	urlRepo := repository.NewURLRepository(session)
	clickRepo := repository.NewClickRepository(session)

	shortenerSvc := service.NewShortenerService(urlRepo)
	analyticsSvc := service.NewAnalyticsService(clickRepo)

	shortenH := handler.NewShortenHandler(shortenerSvc, cfg.BaseURL)
	redirectH := handler.NewRedirectHandler(shortenerSvc, analyticsSvc)
	statsH := handler.NewStatsHandler(shortenerSvc, analyticsSvc)

	app := fiber.New()
	app.Use(logger.New())
	app.Use(recover.New())

	app.Post("/shorten", shortenH.Handle)
	app.Get("/stats/:code", statsH.Handle)
	app.Get("/health", func(c *fiber.Ctx) error {
		scyllaStatus := "ok"
		if err := db.HealthCheck(session); err != nil {
			scyllaStatus = "error"
		}
		return c.JSON(fiber.Map{
			"status": "ok",
			"scylla": scyllaStatus,
		})
	})
	app.Get("/:code", redirectH.Handle)

	slog.Info("starting server", "port", cfg.ServerPort)
	if err := app.Listen(":" + cfg.ServerPort); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
