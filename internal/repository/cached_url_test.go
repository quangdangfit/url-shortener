package repository

import (
	"context"
	"testing"
	"time"

	"github.com/quangdangfit/url-shortener/internal/domain"
	"github.com/redis/go-redis/v9"

	"github.com/alicebob/miniredis/v2"
)

func setupMiniRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr, rdb
}

// --- mock URLRepo ---

type mockURLRepo struct {
	createFn    func(u *domain.URL) error
	getByCodeFn func(code string) (*domain.URL, error)
	existsFn    func(code string) (bool, error)
}

func (m *mockURLRepo) Create(u *domain.URL) error {
	if m.createFn != nil {
		return m.createFn(u)
	}
	return nil
}

func (m *mockURLRepo) GetByCode(code string) (*domain.URL, error) {
	if m.getByCodeFn != nil {
		return m.getByCodeFn(code)
	}
	return nil, nil
}

func (m *mockURLRepo) Exists(code string) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(code)
	}
	return false, nil
}

func TestCachedGetByCode_CacheMiss_ThenHit(t *testing.T) {
	_, rdb := setupMiniRedis(t)
	dbCalls := 0
	mock := &mockURLRepo{
		getByCodeFn: func(code string) (*domain.URL, error) {
			dbCalls++
			return &domain.URL{
				Code:      code,
				Original:  "https://example.com",
				CreatedAt: time.Now().UTC(),
			}, nil
		},
	}
	repo := NewCachedURLRepository(mock, rdb)

	u, err := repo.GetByCode("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if u.Code != "abc123" || u.Original != "https://example.com" {
		t.Errorf("unexpected url: %+v", u)
	}
	if dbCalls != 1 {
		t.Errorf("expected 1 DB call, got %d", dbCalls)
	}

	u2, err := repo.GetByCode("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if u2.Code != "abc123" || u2.Original != "https://example.com" {
		t.Errorf("unexpected cached url: %+v", u2)
	}
	if dbCalls != 1 {
		t.Errorf("expected still 1 DB call (cached), got %d", dbCalls)
	}
}

func TestCachedGetByCode_NotFound(t *testing.T) {
	_, rdb := setupMiniRedis(t)
	mock := &mockURLRepo{
		getByCodeFn: func(code string) (*domain.URL, error) {
			return nil, nil
		},
	}
	repo := NewCachedURLRepository(mock, rdb)

	u, err := repo.GetByCode("nope")
	if err != nil {
		t.Fatal(err)
	}
	if u != nil {
		t.Errorf("expected nil, got %+v", u)
	}

	exists, _ := rdb.HExists(context.Background(), cacheHashKey, "nope").Result()
	if exists {
		t.Error("should not cache not-found URLs")
	}
}

func TestCachedCreate_PopulatesCache(t *testing.T) {
	_, rdb := setupMiniRedis(t)
	dbCalls := 0
	mock := &mockURLRepo{
		createFn: func(u *domain.URL) error { return nil },
		getByCodeFn: func(code string) (*domain.URL, error) {
			dbCalls++
			return &domain.URL{Code: code, Original: "https://example.com", CreatedAt: time.Now().UTC()}, nil
		},
	}
	repo := NewCachedURLRepository(mock, rdb)

	u := &domain.URL{
		Code:      "new123",
		Original:  "https://example.com",
		CreatedAt: time.Now().UTC(),
	}
	if err := repo.Create(u); err != nil {
		t.Fatal(err)
	}

	result, err := repo.GetByCode("new123")
	if err != nil {
		t.Fatal(err)
	}
	if result.Original != "https://example.com" {
		t.Errorf("original = %q", result.Original)
	}
	if dbCalls != 0 {
		t.Errorf("expected 0 DB calls (cached via Create), got %d", dbCalls)
	}
}

func TestCachedExists_CacheHit(t *testing.T) {
	_, rdb := setupMiniRedis(t)
	dbCalls := 0
	mock := &mockURLRepo{
		createFn: func(u *domain.URL) error { return nil },
		existsFn: func(code string) (bool, error) {
			dbCalls++
			return true, nil
		},
	}
	repo := NewCachedURLRepository(mock, rdb)

	repo.Create(&domain.URL{Code: "exists1", Original: "https://example.com", CreatedAt: time.Now().UTC()})

	exists, err := repo.Exists("exists1")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected exists=true")
	}
	if dbCalls != 0 {
		t.Errorf("expected 0 DB calls, got %d", dbCalls)
	}
}

func TestCachedExists_CacheMiss(t *testing.T) {
	_, rdb := setupMiniRedis(t)
	mock := &mockURLRepo{
		existsFn: func(code string) (bool, error) {
			return true, nil
		},
	}
	repo := NewCachedURLRepository(mock, rdb)

	exists, err := repo.Exists("notcached")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("expected exists=true from DB fallback")
	}
}

func TestCachedGetByCode_FieldTTL(t *testing.T) {
	mr, rdb := setupMiniRedis(t)
	dbCalls := 0
	mock := &mockURLRepo{
		getByCodeFn: func(code string) (*domain.URL, error) {
			dbCalls++
			return &domain.URL{Code: code, Original: "https://example.com", CreatedAt: time.Now().UTC()}, nil
		},
	}
	repo := NewCachedURLRepository(mock, rdb)

	repo.GetByCode("ttl1")
	if dbCalls != 1 {
		t.Fatalf("expected 1 call, got %d", dbCalls)
	}

	repo.GetByCode("ttl1")
	if dbCalls != 1 {
		t.Fatalf("expected 1 call (cached), got %d", dbCalls)
	}

	mr.FastForward(6 * time.Minute)

	repo.GetByCode("ttl1")
	if dbCalls != 2 {
		t.Errorf("expected 2 calls after field TTL expiry, got %d", dbCalls)
	}
}

func TestCachedGetByCode_SingleHashKey(t *testing.T) {
	_, rdb := setupMiniRedis(t)
	mock := &mockURLRepo{
		createFn: func(u *domain.URL) error { return nil },
	}
	repo := NewCachedURLRepository(mock, rdb)

	repo.Create(&domain.URL{Code: "a1", Original: "https://a.com", CreatedAt: time.Now().UTC()})
	repo.Create(&domain.URL{Code: "b2", Original: "https://b.com", CreatedAt: time.Now().UTC()})
	repo.Create(&domain.URL{Code: "c3", Original: "https://c.com", CreatedAt: time.Now().UTC()})

	size, _ := rdb.HLen(context.Background(), cacheHashKey).Result()
	if size != 3 {
		t.Errorf("hash size = %d, want 3", size)
	}

	val, _ := rdb.HGet(context.Background(), cacheHashKey, "a1").Result()
	if val != "https://a.com" {
		t.Errorf("cached value = %q, want plain URL", val)
	}
}
