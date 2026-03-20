package handler

import (
	"log/slog"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/quangdangfit/url-shortener/internal/service"
)

type StatsHandler struct {
	shortener service.Shortener
	analytics service.Analytics
}

func NewStatsHandler(shortener service.Shortener, analytics service.Analytics) *StatsHandler {
	return &StatsHandler{shortener: shortener, analytics: analytics}
}

type statsResponse struct {
	Code        string     `json:"code"`
	OriginalURL string     `json:"original_url"`
	TotalClicks int64      `json:"total_clicks"`
	ClicksByDay []dayCount `json:"clicks_by_day"`
	CreatedAt   string     `json:"created_at"`
}

type dayCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

func (h *StatsHandler) Handle(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	u, err := h.shortener.Resolve(code)
	if err != nil {
		slog.Error("failed to resolve url for stats", "error", err, "code", code)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	if u == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	total, counts, err := h.analytics.GetStats(code)
	if err != nil {
		slog.Error("failed to get stats", "error", err, "code", code)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	clicksByDay := make([]dayCount, 0, len(counts))
	for _, c := range counts {
		clicksByDay = append(clicksByDay, dayCount{Date: c.Bucket, Count: c.Total})
	}
	sort.Slice(clicksByDay, func(i, j int) bool {
		return clicksByDay[i].Date < clicksByDay[j].Date
	})

	writeJSON(w, http.StatusOK, statsResponse{
		Code:        u.Code,
		OriginalURL: u.Original,
		TotalClicks: total,
		ClicksByDay: clicksByDay,
		CreatedAt:   u.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}
