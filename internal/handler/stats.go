package handler

import (
	"log/slog"
	"sort"

	"github.com/gofiber/fiber/v2"
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

func (h *StatsHandler) Handle(c *fiber.Ctx) error {
	code := c.Params("code")

	u, err := h.shortener.Resolve(code)
	if err != nil {
		slog.Error("failed to resolve url for stats", "error", err, "code", code)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
	if u == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}

	total, counts, err := h.analytics.GetStats(code)
	if err != nil {
		slog.Error("failed to get stats", "error", err, "code", code)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}

	clicksByDay := make([]dayCount, 0, len(counts))
	for _, cc := range counts {
		clicksByDay = append(clicksByDay, dayCount{Date: cc.Bucket, Count: cc.Total})
	}
	sort.Slice(clicksByDay, func(i, j int) bool {
		return clicksByDay[i].Date < clicksByDay[j].Date
	})

	return c.JSON(statsResponse{
		Code:        u.Code,
		OriginalURL: u.Original,
		TotalClicks: total,
		ClicksByDay: clicksByDay,
		CreatedAt:   u.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}
