package repository

import "github.com/quangdangfit/url-shortener/internal/model"

type URLRepo interface {
	Create(u *model.URL) error
	GetByCode(code string) (*model.URL, error)
	Exists(code string) (bool, error)
}

type ClickRepo interface {
	InsertClick(c *model.Click) error
	IncrementCount(code, bucket string) error
	GetClickCounts(code string, buckets []string) ([]model.ClickCount, error)
	GetTotalClicks(code string, buckets []string) (int64, error)
}
