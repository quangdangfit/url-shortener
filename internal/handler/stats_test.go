package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/quangdangfit/url-shortener/internal/domain"
)

func TestStatsHandle_Success(t *testing.T) {
	ms := &mockShortener{
		resolveFn: func(code string) (*domain.URL, error) {
			return &domain.URL{
				Code:      code,
				Original:  "https://example.com",
				CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			}, nil
		},
	}
	ma := &mockAnalytics{
		getStatsFn: func(code string) (int64, []domain.ClickCount, error) {
			return 42, []domain.ClickCount{
				{Code: code, Bucket: "2024-01-03", Total: 20},
				{Code: code, Bucket: "2024-01-01", Total: 10},
				{Code: code, Bucket: "2024-01-02", Total: 12},
			}, nil
		},
	}
	h := NewStatsHandler(ms, ma)

	app := fiber.New()
	app.Get("/stats/:code", h.Handle)

	req := httptest.NewRequest(http.MethodGet, "/stats/abc123", nil)

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result statsResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Code != "abc123" {
		t.Errorf("code = %q", result.Code)
	}
	if result.OriginalURL != "https://example.com" {
		t.Errorf("original_url = %q", result.OriginalURL)
	}
	if result.TotalClicks != 42 {
		t.Errorf("total_clicks = %d", result.TotalClicks)
	}
	if len(result.ClicksByDay) != 3 {
		t.Fatalf("clicks_by_day length = %d, want 3", len(result.ClicksByDay))
	}
	if result.ClicksByDay[0].Date != "2024-01-01" {
		t.Errorf("first day = %q, want 2024-01-01", result.ClicksByDay[0].Date)
	}
	if result.ClicksByDay[1].Date != "2024-01-02" {
		t.Errorf("second day = %q, want 2024-01-02", result.ClicksByDay[1].Date)
	}
	if result.ClicksByDay[2].Date != "2024-01-03" {
		t.Errorf("third day = %q, want 2024-01-03", result.ClicksByDay[2].Date)
	}
	if result.CreatedAt != "2024-01-01T00:00:00Z" {
		t.Errorf("created_at = %q", result.CreatedAt)
	}
}

func TestStatsHandle_EmptyCounts(t *testing.T) {
	ms := &mockShortener{
		resolveFn: func(code string) (*domain.URL, error) {
			return &domain.URL{Code: code, Original: "https://example.com", CreatedAt: time.Now().UTC()}, nil
		},
	}
	ma := &mockAnalytics{
		getStatsFn: func(code string) (int64, []domain.ClickCount, error) {
			return 0, nil, nil
		},
	}
	h := NewStatsHandler(ms, ma)

	app := fiber.New()
	app.Get("/stats/:code", h.Handle)

	req := httptest.NewRequest(http.MethodGet, "/stats/abc", nil)

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d", resp.StatusCode)
	}

	var result statsResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if result.TotalClicks != 0 {
		t.Errorf("total_clicks = %d", result.TotalClicks)
	}
}

func TestStatsHandle_NotFound(t *testing.T) {
	ms := &mockShortener{
		resolveFn: func(code string) (*domain.URL, error) { return nil, nil },
	}
	h := NewStatsHandler(ms, &mockAnalytics{})

	app := fiber.New()
	app.Get("/stats/:code", h.Handle)

	req := httptest.NewRequest(http.MethodGet, "/stats/nope", nil)

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestStatsHandle_ResolveError(t *testing.T) {
	ms := &mockShortener{
		resolveFn: func(code string) (*domain.URL, error) { return nil, errors.New("db error") },
	}
	h := NewStatsHandler(ms, &mockAnalytics{})

	app := fiber.New()
	app.Get("/stats/:code", h.Handle)

	req := httptest.NewRequest(http.MethodGet, "/stats/abc", nil)

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestStatsHandle_GetStatsError(t *testing.T) {
	ms := &mockShortener{
		resolveFn: func(code string) (*domain.URL, error) {
			return &domain.URL{Code: code, Original: "https://example.com", CreatedAt: time.Now().UTC()}, nil
		},
	}
	ma := &mockAnalytics{
		getStatsFn: func(code string) (int64, []domain.ClickCount, error) {
			return 0, nil, errors.New("stats error")
		},
	}
	h := NewStatsHandler(ms, ma)

	app := fiber.New()
	app.Get("/stats/:code", h.Handle)

	req := httptest.NewRequest(http.MethodGet, "/stats/abc", nil)

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}
