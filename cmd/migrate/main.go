package main

import (
	"log/slog"
	"os"

	"github.com/quangdangfit/url-shortener/internal/config"
	"github.com/quangdangfit/url-shortener/internal/db"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := config.Load()

	if err := db.RunMigrations(cfg.ScyllaHosts); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	slog.Info("migrations completed successfully")
}
