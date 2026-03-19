package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/quangdangfit/url-shortener/internal/service"
)

type ShortenHandler struct {
	shortener *service.ShortenerService
	baseURL   string
}

func NewShortenHandler(shortener *service.ShortenerService, baseURL string) *ShortenHandler {
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

func (h *ShortenHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var req shortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url is required"})
		return
	}

	u, err := h.shortener.Shorten(req.URL, req.TTLDays)
	if err != nil {
		slog.Error("failed to shorten url", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	resp := shortenResponse{
		Code:     u.Code,
		ShortURL: h.baseURL + "/" + u.Code,
	}
	if u.ExpiresAt != nil {
		resp.ExpiresAt = u.ExpiresAt.Format("2006-01-02T15:04:05Z")
	}

	writeJSON(w, http.StatusCreated, resp)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
