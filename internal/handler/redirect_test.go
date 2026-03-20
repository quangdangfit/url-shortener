package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/quangdangfit/url-shortener/internal/model"
)

// --- mock analytics ---

type mockAnalytics struct {
	recordClickFn func(code, ip, userAgent, referer string)
	getStatsFn    func(code string) (int64, []model.ClickCount, error)
}

func (m *mockAnalytics) RecordClick(code, ip, userAgent, referer string) {
	if m.recordClickFn != nil {
		m.recordClickFn(code, ip, userAgent, referer)
	}
}

func (m *mockAnalytics) GetStats(code string) (int64, []model.ClickCount, error) {
	if m.getStatsFn != nil {
		return m.getStatsFn(code)
	}
	return 0, nil, nil
}

func newChiRequest(method, path, code string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("code", code)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// --- tests ---

func TestRedirectHandle_Success(t *testing.T) {
	var recordedCode, recordedIP string
	ms := &mockShortener{
		resolveFn: func(code string) (*model.URL, error) {
			return &model.URL{Code: code, Original: "https://example.com"}, nil
		},
	}
	ma := &mockAnalytics{
		recordClickFn: func(code, ip, userAgent, referer string) {
			recordedCode = code
			recordedIP = ip
		},
	}
	h := NewRedirectHandler(ms, ma)

	req := newChiRequest(http.MethodGet, "/abc123", "abc123")
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	w := httptest.NewRecorder()

	h.Handle(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMovedPermanently)
	}
	if loc := w.Header().Get("Location"); loc != "https://example.com" {
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
		resolveFn: func(code string) (*model.URL, error) {
			return &model.URL{Code: code, Original: "https://example.com"}, nil
		},
	}
	ma := &mockAnalytics{
		recordClickFn: func(code, ip, userAgent, referer string) {
			recordedIP = ip
		},
	}
	h := NewRedirectHandler(ms, ma)

	req := newChiRequest(http.MethodGet, "/abc123", "abc123")
	// No X-Forwarded-For, should fall back to RemoteAddr
	w := httptest.NewRecorder()

	h.Handle(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("status = %d", w.Code)
	}
	if recordedIP == "" {
		t.Error("expected IP from RemoteAddr")
	}
}

func TestRedirectHandle_NotFound(t *testing.T) {
	ms := &mockShortener{
		resolveFn: func(code string) (*model.URL, error) { return nil, nil },
	}
	h := NewRedirectHandler(ms, &mockAnalytics{})

	req := newChiRequest(http.MethodGet, "/nope", "nope")
	w := httptest.NewRecorder()

	h.Handle(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRedirectHandle_ResolveError(t *testing.T) {
	ms := &mockShortener{
		resolveFn: func(code string) (*model.URL, error) { return nil, errors.New("db error") },
	}
	h := NewRedirectHandler(ms, &mockAnalytics{})

	req := newChiRequest(http.MethodGet, "/abc", "abc")
	w := httptest.NewRecorder()

	h.Handle(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestRedirectHandle_Expired(t *testing.T) {
	past := time.Now().UTC().Add(-1 * time.Hour)
	ms := &mockShortener{
		resolveFn: func(code string) (*model.URL, error) {
			return &model.URL{Code: code, Original: "https://example.com", ExpiresAt: &past}, nil
		},
	}
	h := NewRedirectHandler(ms, &mockAnalytics{})

	req := newChiRequest(http.MethodGet, "/abc", "abc")
	w := httptest.NewRecorder()

	h.Handle(w, req)

	if w.Code != http.StatusGone {
		t.Errorf("status = %d, want %d", w.Code, http.StatusGone)
	}
}

func TestRedirectHandle_NotExpired(t *testing.T) {
	future := time.Now().UTC().Add(24 * time.Hour)
	ms := &mockShortener{
		resolveFn: func(code string) (*model.URL, error) {
			return &model.URL{Code: code, Original: "https://example.com", ExpiresAt: &future}, nil
		},
	}
	h := NewRedirectHandler(ms, &mockAnalytics{})

	req := newChiRequest(http.MethodGet, "/abc", "abc")
	w := httptest.NewRecorder()

	h.Handle(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMovedPermanently)
	}
}
