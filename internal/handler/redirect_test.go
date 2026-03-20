package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/quangdangfit/url-shortener/internal/domain"
)

// --- mock analytics ---

type mockAnalytics struct {
	recordClickFn func(code, ip, userAgent, referer string)
	getStatsFn    func(code string) (int64, []domain.ClickCount, error)
}

func (m *mockAnalytics) RecordClick(code, ip, userAgent, referer string) {
	if m.recordClickFn != nil {
		m.recordClickFn(code, ip, userAgent, referer)
	}
}

func (m *mockAnalytics) GetStats(code string) (int64, []domain.ClickCount, error) {
	if m.getStatsFn != nil {
		return m.getStatsFn(code)
	}
	return 0, nil, nil
}

// --- tests ---

func TestRedirectHandle_Success(t *testing.T) {
	var recordedCode, recordedIP string
	ms := &mockShortener{
		resolveFn: func(code string) (*domain.URL, error) {
			return &domain.URL{Code: code, Original: "https://example.com"}, nil
		},
	}
	ma := &mockAnalytics{
		recordClickFn: func(code, ip, userAgent, referer string) {
			recordedCode = code
			recordedIP = ip
		},
	}
	h := NewRedirectHandler(ms, ma)

	app := fiber.New()
	app.Get("/:code", h.Handle)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMovedPermanently {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusMovedPermanently)
	}
	if loc := resp.Header.Get("Location"); loc != "https://example.com" {
		t.Errorf("location = %q", loc)
	}
	if recordedCode != "abc123" {
		t.Errorf("recorded code = %q", recordedCode)
	}
	if recordedIP != "10.0.0.1" {
		t.Errorf("recorded IP = %q", recordedIP)
	}
}

func TestRedirectHandle_FallbackIP(t *testing.T) {
	var recordedIP string
	ms := &mockShortener{
		resolveFn: func(code string) (*domain.URL, error) {
			return &domain.URL{Code: code, Original: "https://example.com"}, nil
		},
	}
	ma := &mockAnalytics{
		recordClickFn: func(code, ip, userAgent, referer string) {
			recordedIP = ip
		},
	}
	h := NewRedirectHandler(ms, ma)

	app := fiber.New()
	app.Get("/:code", h.Handle)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMovedPermanently {
		t.Errorf("status = %d", resp.StatusCode)
	}
	if recordedIP == "" {
		t.Error("expected IP from RemoteAddr")
	}
}

func TestRedirectHandle_NotFound(t *testing.T) {
	ms := &mockShortener{
		resolveFn: func(code string) (*domain.URL, error) { return nil, nil },
	}
	h := NewRedirectHandler(ms, &mockAnalytics{})

	app := fiber.New()
	app.Get("/:code", h.Handle)

	req := httptest.NewRequest(http.MethodGet, "/nope", nil)

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestRedirectHandle_ResolveError(t *testing.T) {
	ms := &mockShortener{
		resolveFn: func(code string) (*domain.URL, error) { return nil, errors.New("db error") },
	}
	h := NewRedirectHandler(ms, &mockAnalytics{})

	app := fiber.New()
	app.Get("/:code", h.Handle)

	req := httptest.NewRequest(http.MethodGet, "/abc", nil)

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestRedirectHandle_Expired(t *testing.T) {
	past := time.Now().UTC().Add(-1 * time.Hour)
	ms := &mockShortener{
		resolveFn: func(code string) (*domain.URL, error) {
			return &domain.URL{Code: code, Original: "https://example.com", ExpiresAt: &past}, nil
		},
	}
	h := NewRedirectHandler(ms, &mockAnalytics{})

	app := fiber.New()
	app.Get("/:code", h.Handle)

	req := httptest.NewRequest(http.MethodGet, "/abc", nil)

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusGone {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusGone)
	}
}

func TestRedirectHandle_NotExpired(t *testing.T) {
	future := time.Now().UTC().Add(24 * time.Hour)
	ms := &mockShortener{
		resolveFn: func(code string) (*domain.URL, error) {
			return &domain.URL{Code: code, Original: "https://example.com", ExpiresAt: &future}, nil
		},
	}
	h := NewRedirectHandler(ms, &mockAnalytics{})

	app := fiber.New()
	app.Get("/:code", h.Handle)

	req := httptest.NewRequest(http.MethodGet, "/abc", nil)

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMovedPermanently {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusMovedPermanently)
	}
}
