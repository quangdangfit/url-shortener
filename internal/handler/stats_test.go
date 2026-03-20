package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quangdangfit/url-shortener/internal/model"
)

func TestStatsHandle_Success(t *testing.T) {
	ms := &mockShortener{
		resolveFn: func(code string) (*model.URL, error) {
			return &model.URL{
				Code:      code,
				Original:  "https://example.com",
				CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			}, nil
		},
	}
	ma := &mockAnalytics{
		getStatsFn: func(code string) (int64, []model.ClickCount, error) {
			return 42, []model.ClickCount{
				{Code: code, Bucket: "2024-01-03", Total: 20},
				{Code: code, Bucket: "2024-01-01", Total: 10},
				{Code: code, Bucket: "2024-01-02", Total: 12},
			}, nil
		},
	}
	h := NewStatsHandler(ms, ma)

	req := newChiRequest(http.MethodGet, "/stats/abc123", "abc123")
	w := httptest.NewRecorder()

	h.Handle(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp statsResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Code != "abc123" {
		t.Errorf("code = %q", resp.Code)
	}
	if resp.OriginalURL != "https://example.com" {
		t.Errorf("original_url = %q", resp.OriginalURL)
	}
	if resp.TotalClicks != 42 {
		t.Errorf("total_clicks = %d", resp.TotalClicks)
	}
	if len(resp.ClicksByDay) != 3 {
		t.Fatalf("clicks_by_day length = %d, want 3", len(resp.ClicksByDay))
	}
	// Verify sorted by date ascending
	if resp.ClicksByDay[0].Date != "2024-01-01" {
		t.Errorf("first day = %q, want 2024-01-01", resp.ClicksByDay[0].Date)
	}
	if resp.ClicksByDay[1].Date != "2024-01-02" {
		t.Errorf("second day = %q, want 2024-01-02", resp.ClicksByDay[1].Date)
	}
	if resp.ClicksByDay[2].Date != "2024-01-03" {
		t.Errorf("third day = %q, want 2024-01-03", resp.ClicksByDay[2].Date)
	}
	if resp.CreatedAt != "2024-01-01T00:00:00Z" {
		t.Errorf("created_at = %q", resp.CreatedAt)
	}
}

func TestStatsHandle_EmptyCounts(t *testing.T) {
	ms := &mockShortener{
		resolveFn: func(code string) (*model.URL, error) {
			return &model.URL{Code: code, Original: "https://example.com", CreatedAt: time.Now().UTC()}, nil
		},
	}
	ma := &mockAnalytics{
		getStatsFn: func(code string) (int64, []model.ClickCount, error) {
			return 0, nil, nil
		},
	}
	h := NewStatsHandler(ms, ma)

	req := newChiRequest(http.MethodGet, "/stats/abc", "abc")
	w := httptest.NewRecorder()

	h.Handle(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d", w.Code)
	}

	var resp statsResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.TotalClicks != 0 {
		t.Errorf("total_clicks = %d", resp.TotalClicks)
	}
}

func TestStatsHandle_NotFound(t *testing.T) {
	ms := &mockShortener{
		resolveFn: func(code string) (*model.URL, error) { return nil, nil },
	}
	h := NewStatsHandler(ms, &mockAnalytics{})

	req := newChiRequest(http.MethodGet, "/stats/nope", "nope")
	w := httptest.NewRecorder()

	h.Handle(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestStatsHandle_ResolveError(t *testing.T) {
	ms := &mockShortener{
		resolveFn: func(code string) (*model.URL, error) { return nil, errors.New("db error") },
	}
	h := NewStatsHandler(ms, &mockAnalytics{})

	req := newChiRequest(http.MethodGet, "/stats/abc", "abc")
	w := httptest.NewRecorder()

	h.Handle(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestStatsHandle_GetStatsError(t *testing.T) {
	ms := &mockShortener{
		resolveFn: func(code string) (*model.URL, error) {
			return &model.URL{Code: code, Original: "https://example.com", CreatedAt: time.Now().UTC()}, nil
		},
	}
	ma := &mockAnalytics{
		getStatsFn: func(code string) (int64, []model.ClickCount, error) {
			return 0, nil, errors.New("stats error")
		},
	}
	h := NewStatsHandler(ms, ma)

	req := newChiRequest(http.MethodGet, "/stats/abc", "abc")
	w := httptest.NewRecorder()

	h.Handle(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
