package repository

import (
	"context"
	"log/slog"
	"time"

	"github.com/quangdangfit/url-shortener/internal/domain"
	"github.com/quangdangfit/url-shortener/internal/port"
	"github.com/redis/go-redis/v9"
)

const (
	cacheHashKey = "urls"
	cacheTTL     = 5 * time.Minute
)

type CachedURLRepository struct {
	repo  port.URLRepository
	redis *redis.Client
}

func NewCachedURLRepository(repo port.URLRepository, redis *redis.Client) *CachedURLRepository {
	return &CachedURLRepository{repo: repo, redis: redis}
}

func (r *CachedURLRepository) setField(ctx context.Context, code, original string) {
	if err := r.redis.HSet(ctx, cacheHashKey, code, original).Err(); err != nil {
		slog.Warn("failed to cache url", "error", err, "code", code)
		return
	}
	r.redis.HExpire(ctx, cacheHashKey, cacheTTL, code)
}

func (r *CachedURLRepository) Create(u *domain.URL) error {
	if err := r.repo.Create(u); err != nil {
		return err
	}

	r.setField(context.Background(), u.Code, u.Original)
	return nil
}

func (r *CachedURLRepository) GetByCode(code string) (*domain.URL, error) {
	ctx := context.Background()

	if val, err := r.redis.HGet(ctx, cacheHashKey, code).Result(); err == nil {
		return &domain.URL{Code: code, Original: val}, nil
	}

	u, err := r.repo.GetByCode(code)
	if err != nil || u == nil {
		return u, err
	}

	r.setField(ctx, code, u.Original)
	return u, nil
}

func (r *CachedURLRepository) Exists(code string) (bool, error) {
	ctx := context.Background()

	exists, err := r.redis.HExists(ctx, cacheHashKey, code).Result()
	if err == nil && exists {
		return true, nil
	}

	return r.repo.Exists(code)
}
