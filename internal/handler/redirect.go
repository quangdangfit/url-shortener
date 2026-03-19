package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/quangdangfit/url-shortener/internal/service"
)

type RedirectHandler struct {
	shortener *service.ShortenerService
	analytics *service.AnalyticsService
}

func NewRedirectHandler(shortener *service.ShortenerService, analytics *service.AnalyticsService) *RedirectHandler {
	return &RedirectHandler{shortener: shortener, analytics: analytics}
}

func (h *RedirectHandler) Handle(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	u, err := h.shortener.Resolve(code)
	if err != nil {
		slog.Error("failed to resolve url", "error", err, "code", code)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	if u == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if u.ExpiresAt != nil && u.ExpiresAt.Before(time.Now().UTC()) {
		writeJSON(w, http.StatusGone, map[string]string{"error": "link has expired"})
		return
	}

	// Async record click - does not block redirect
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}
	h.analytics.RecordClick(code, ip, r.UserAgent(), r.Referer())

	http.Redirect(w, r, u.Original, http.StatusMovedPermanently)
}
