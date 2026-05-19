package handler

import (
	"bytes"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/quangdangfit/url-shortener/internal/port"
)

type FrontendHandler struct {
	shortener port.Shortener
	analytics port.Analytics
	baseURL   string
}

func NewFrontendHandler(shortener port.Shortener, analytics port.Analytics, baseURL string) *FrontendHandler {
	return &FrontendHandler{shortener: shortener, analytics: analytics, baseURL: baseURL}
}

// ShortenRequest for form submission
type ShortenRequest struct {
	URL     string `form:"url" json:"url"`
	TTLDays *int   `form:"ttl_days" json:"ttl_days,omitempty"`
}

// ShortenResponse for HTMX
type ShortenResponse struct {
	Code        string `json:"code"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

// StatsData for stats page
type StatsData struct {
	Code        string
	ShortURL    string
	OriginalURL string
	TotalClicks int64
	Dates       []string
	Counts      []int64
	CreatedAt   string
}

// ServeIndex serves the main page
func (h *FrontendHandler) ServeIndex(c *fiber.Ctx) error {
	return c.SendFile("web/index.html")
}

// HandleShorten handles the shorten form submission for HTMX
func (h *FrontendHandler) HandleShorten(c *fiber.Ctx) error {
	var req ShortenRequest
	
	// Try to parse as form data first, then as JSON
	contentType := c.Get("Content-Type")
	if contentType == "application/json" {
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}
	} else {
		// Form data - parse manually
		url := c.FormValue("url")
		ttlDaysStr := c.FormValue("ttl_days")
		
		if url == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "url is required"})
		}
		
		req.URL = url
		if ttlDaysStr != "" {
			var ttlDays int
			if _, err := fmt.Sscanf(ttlDaysStr, "%d", &ttlDays); err == nil {
				req.TTLDays = &ttlDays
			}
		}
	}

	if req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "url is required"})
	}

	u, err := h.shortener.Shorten(req.URL, req.TTLDays)
	if err != nil {
		slog.Error("failed to shorten url", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}

	resp := ShortenResponse{
		Code:        u.Code,
		ShortURL:    h.baseURL + "/" + u.Code,
		OriginalURL: u.Original,
	}
	if u.ExpiresAt != nil {
		resp.ExpiresAt = u.ExpiresAt.Format("2006-01-02")
	}

	// Return HTML fragment for HTMX
	tmpl := template.Must(template.ParseFiles("web/result.html"))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, resp); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "template error"})
	}
	c.Set("Content-Type", "text/html")
	return c.Send(buf.Bytes())
}

// ServeStats serves the stats page
func (h *FrontendHandler) ServeStats(c *fiber.Ctx) error {
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

	// Extract dates and counts for chart
	dates := make([]string, 0, len(counts))
	clickCounts := make([]int64, 0, len(counts))
	for _, cc := range counts {
		dates = append(dates, cc.Bucket)
		clickCounts = append(clickCounts, cc.Total)
	}

	data := StatsData{
		Code:        u.Code,
		ShortURL:    h.baseURL + "/" + u.Code,
		OriginalURL: u.Original,
		TotalClicks: total,
		Dates:       dates,
		Counts:      clickCounts,
		CreatedAt:   u.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	tmpl := template.Must(template.ParseFiles("web/stats.html"))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "template error"})
	}
	c.Set("Content-Type", "text/html")
	return c.Send(buf.Bytes())
}

// SetupStaticFiles sets up static file serving
func SetupStaticFiles(app *fiber.App) {
	// Serve static files from web/static
	app.Use("/static", filesystem.New(filesystem.Config{
		Root:   http.Dir("./web/static"),
		Browse: true,
	}))
}