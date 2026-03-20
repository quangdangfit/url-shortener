package handler

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/quangdangfit/url-shortener/internal/port"
)

type RedirectHandler struct {
	shortener port.Shortener
	analytics port.Analytics
}

func NewRedirectHandler(shortener port.Shortener, analytics port.Analytics) *RedirectHandler {
	return &RedirectHandler{shortener: shortener, analytics: analytics}
}

func (h *RedirectHandler) Handle(c *fiber.Ctx) error {
	code := c.Params("code")

	u, err := h.shortener.Resolve(code)
	if err != nil {
		slog.Error("failed to resolve url", "error", err, "code", code)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
	if u == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	if u.ExpiresAt != nil && u.ExpiresAt.Before(time.Now().UTC()) {
		return c.Status(fiber.StatusGone).JSON(fiber.Map{"error": "link has expired"})
	}

	// Async record click - does not block redirect
	ip := c.Get("X-Forwarded-For")
	if ip == "" {
		ip = c.IP()
	}
	h.analytics.RecordClick(code, ip, string(c.Request().Header.UserAgent()), c.Get("Referer"))

	return c.Redirect(u.Original, fiber.StatusMovedPermanently)
}
