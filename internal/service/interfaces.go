package service

import "github.com/quangdangfit/url-shortener/internal/model"

type Shortener interface {
	Shorten(originalURL string, ttlDays *int) (*model.URL, error)
	Resolve(code string) (*model.URL, error)
}

type Analytics interface {
	RecordClick(code, ip, userAgent, referer string)
	GetStats(code string) (int64, []model.ClickCount, error)
}
