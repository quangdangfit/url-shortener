package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/quangdangfit/url-shortener/internal/model"
)

// --- mock shortener ---

type mockShortener struct {
	shortenFn func(originalURL string, ttlDays *int) (*model.URL, error)
	resolveFn func(code string) (*model.URL, error)
}

func (m *mockShortener) Shorten(originalURL string, ttlDays *int) (*model.URL, error) {
	return m.shortenFn(originalURL, ttlDays)
}

func (m *mockShortener) Resolve(code string) (*model.URL, error) {
	return m.resolveFn(code)
}

// --- tests ---

func TestShortenHandle_Success(t *testing.T) {
	ms := &mockShortener{
		shortenFn: func(originalURL string, ttlDays *int) (*model.URL, error) {
			return &model.URL{
				Code:      "abc123",
				Original:  originalURL,
				CreatedAt: time.Now().UTC(),
			}, nil
		},
	}
	h := NewShortenHandler(ms, "http://localhost:8080")

	app := fiber.New()
	app.Post("/shorten", h.Handle)

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(`{"url":"https://example.com"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var result shortenResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Code != "abc123" {
		t.Errorf("code = %q", result.Code)
	}
	if result.ShortURL != "http://localhost:8080/abc123" {
		t.Errorf("short_url = %q", result.ShortURL)
	}
	if result.ExpiresAt != "" {
		t.Errorf("expires_at = %q, want empty", result.ExpiresAt)
	}
}

func TestShortenHandle_WithTTL(t *testing.T) {
	exp := time.Now().UTC().Add(7 * 24 * time.Hour)
	ms := &mockShortener{
		shortenFn: func(originalURL string, ttlDays *int) (*model.URL, error) {
			return &model.URL{
				Code:      "abc123",
				Original:  originalURL,
				CreatedAt: time.Now().UTC(),
				ExpiresAt: &exp,
			}, nil
		},
	}
	h := NewShortenHandler(ms, "http://localhost:8080")

	app := fiber.New()
	app.Post("/shorten", h.Handle)

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(`{"url":"https://example.com","ttl_days":7}`))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var result shortenResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if result.ExpiresAt == "" {
		t.Error("expected expires_at to be set")
	}
}

func TestShortenHandle_InvalidJSON(t *testing.T) {
	h := NewShortenHandler(&mockShortener{}, "http://localhost:8080")

	app := fiber.New()
	app.Post("/shorten", h.Handle)

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestShortenHandle_EmptyURL(t *testing.T) {
	h := NewShortenHandler(&mockShortener{}, "http://localhost:8080")

	app := fiber.New()
	app.Post("/shorten", h.Handle)

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(`{"url":""}`))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestShortenHandle_MissingURL(t *testing.T) {
	h := NewShortenHandler(&mockShortener{}, "http://localhost:8080")

	app := fiber.New()
	app.Post("/shorten", h.Handle)

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestShortenHandle_ServiceError(t *testing.T) {
	ms := &mockShortener{
		shortenFn: func(originalURL string, ttlDays *int) (*model.URL, error) {
			return nil, errors.New("internal error")
		},
	}
	h := NewShortenHandler(ms, "http://localhost:8080")

	app := fiber.New()
	app.Post("/shorten", h.Handle)

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(`{"url":"https://example.com"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestWriteJSON(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"key": "value"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	var result map[string]string
	json.Unmarshal(body, &result)
	if result["key"] != "value" {
		t.Errorf("body = %v", result)
	}
}
