package port

import "github.com/quangdangfit/url-shortener/internal/domain"

type Shortener interface {
	Shorten(originalURL string, ttlDays *int) (*domain.URL, error)
	Resolve(code string) (*domain.URL, error)
}

type Analytics interface {
	RecordClick(code, ip, userAgent, referer string)
	GetStats(code string) (int64, []domain.ClickCount, error)
}
