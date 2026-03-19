package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/shorten", shortenH.Handle)
	r.Get("/stats/{code}", statsH.Handle)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		scyllaStatus := "ok"
		if err := db.HealthCheck(session); err != nil {
			scyllaStatus = "error"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"scylla": scyllaStatus,
		})
	})
	r.Get("/{code}", redirectH.Handle)

	slog.Info("starting server", "port", cfg.ServerPort)
	if err := http.ListenAndServe(":"+cfg.ServerPort, r); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
