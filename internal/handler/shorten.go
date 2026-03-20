package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/quangdangfit/url-shortener/internal/port"
)

type ShortenHandler struct {
	shortener port.Shortener
	baseURL   string
}

func NewShortenHandler(shortener port.Shortener, baseURL string) *ShortenHandler {
	return &ShortenHandler{shortener: shortener, baseURL: baseURL}
}

type shortenRequest struct {
	URL     string `json:"url"`
	TTLDays *int   `json:"ttl_days,omitempty"`
}

type shortenResponse struct {
	Code      string `json:"code"`
	ShortURL  string `json:"short_url"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

func (h *ShortenHandler) Handle(c *fiber.Ctx) error {
	var req shortenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "url is required"})
	}

	u, err := h.shortener.Shorten(req.URL, req.TTLDays)
	if err != nil {
		slog.Error("failed to shorten url", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}

	resp := shortenResponse{
		Code:     u.Code,
		ShortURL: h.baseURL + "/" + u.Code,
	}
	if u.ExpiresAt != nil {
		resp.ExpiresAt = u.ExpiresAt.Format("2006-01-02T15:04:05Z")
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}
