package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

	body := `{"url":"https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Handle(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp shortenResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != "abc123" {
		t.Errorf("code = %q", resp.Code)
	}
	if resp.ShortURL != "http://localhost:8080/abc123" {
		t.Errorf("short_url = %q", resp.ShortURL)
	}
	if resp.ExpiresAt != "" {
		t.Errorf("expires_at = %q, want empty", resp.ExpiresAt)
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

	body := `{"url":"https://example.com","ttl_days":7}`
	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.Handle(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp shortenResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ExpiresAt == "" {
		t.Error("expected expires_at to be set")
	}
}

func TestShortenHandle_InvalidJSON(t *testing.T) {
	h := NewShortenHandler(&mockShortener{}, "http://localhost:8080")

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader("not json"))
	w := httptest.NewRecorder()

	h.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestShortenHandle_EmptyURL(t *testing.T) {
	h := NewShortenHandler(&mockShortener{}, "http://localhost:8080")

	body := `{"url":""}`
	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestShortenHandle_MissingURL(t *testing.T) {
	h := NewShortenHandler(&mockShortener{}, "http://localhost:8080")

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestShortenHandle_ServiceError(t *testing.T) {
	ms := &mockShortener{
		shortenFn: func(originalURL string, ttlDays *int) (*model.URL, error) {
			return nil, errors.New("internal error")
		},
	}
	h := NewShortenHandler(ms, "http://localhost:8080")

	body := `{"url":"https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.Handle(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q", ct)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["key"] != "value" {
		t.Errorf("body = %v", resp)
	}
}
