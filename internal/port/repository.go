package port

import "github.com/quangdangfit/url-shortener/internal/domain"

type URLRepository interface {
	Create(u *domain.URL) error
	GetByCode(code string) (*domain.URL, error)
	Exists(code string) (bool, error)
}

type ClickRepository interface {
	InsertClick(c *domain.Click) error
	IncrementCount(code, bucket string) error
	GetClickCounts(code string, buckets []string) ([]domain.ClickCount, error)
	GetTotalClicks(code string, buckets []string) (int64, error)
}
