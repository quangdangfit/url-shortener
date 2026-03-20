package service

import (
	"errors"
	"testing"

	"github.com/quangdangfit/url-shortener/internal/model"
)

// --- mock URL repo ---

type mockURLRepo struct {
	createFn    func(u *model.URL) error
	getByCodeFn func(code string) (*model.URL, error)
	existsFn    func(code string) (bool, error)
}

func (m *mockURLRepo) Create(u *model.URL) error                 { return m.createFn(u) }
func (m *mockURLRepo) GetByCode(code string) (*model.URL, error) { return m.getByCodeFn(code) }
func (m *mockURLRepo) Exists(code string) (bool, error)          { return m.existsFn(code) }

// --- tests ---

func TestNewShortenerService(t *testing.T) {
	repo := &mockURLRepo{}
	svc := NewShortenerService(repo)
	if svc == nil {
		t.Fatal("NewShortenerService returned nil")
	}
}

func TestShorten_Success(t *testing.T) {
	repo := &mockURLRepo{
		existsFn: func(code string) (bool, error) { return false, nil },
		createFn: func(u *model.URL) error { return nil },
	}
	svc := NewShortenerService(repo)

	u, err := svc.Shorten("https://example.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u == nil {
		t.Fatal("expected URL, got nil")
	}
	if len(u.Code) != codeLength {
		t.Errorf("code length = %d, want %d", len(u.Code), codeLength)
	}
	if u.Original != "https://example.com" {
		t.Errorf("original = %q", u.Original)
	}
	if u.ExpiresAt != nil {
		t.Error("expected nil ExpiresAt")
	}
}

func TestShorten_WithTTL(t *testing.T) {
	repo := &mockURLRepo{
		existsFn: func(code string) (bool, error) { return false, nil },
		createFn: func(u *model.URL) error { return nil },
	}
	svc := NewShortenerService(repo)

	ttl := 7
	u, err := svc.Shorten("https://example.com", &ttl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ExpiresAt == nil {
		t.Fatal("expected ExpiresAt to be set")
	}
}

func TestShorten_WithZeroTTL(t *testing.T) {
	repo := &mockURLRepo{
		existsFn: func(code string) (bool, error) { return false, nil },
		createFn: func(u *model.URL) error { return nil },
	}
	svc := NewShortenerService(repo)

	ttl := 0
	u, err := svc.Shorten("https://example.com", &ttl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ExpiresAt != nil {
		t.Error("expected nil ExpiresAt for zero TTL")
	}
}

func TestShorten_ExistsError(t *testing.T) {
	repo := &mockURLRepo{
		existsFn: func(code string) (bool, error) { return false, errors.New("db error") },
	}
	svc := NewShortenerService(repo)

	_, err := svc.Shorten("https://example.com", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestShorten_CreateError(t *testing.T) {
	repo := &mockURLRepo{
		existsFn: func(code string) (bool, error) { return false, nil },
		createFn: func(u *model.URL) error { return errors.New("insert error") },
	}
	svc := NewShortenerService(repo)

	_, err := svc.Shorten("https://example.com", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestShorten_CollisionRetry(t *testing.T) {
	callCount := 0
	repo := &mockURLRepo{
		existsFn: func(code string) (bool, error) {
			callCount++
			if callCount <= 3 {
				return true, nil // first 3 calls: collision
			}
			return false, nil
		},
		createFn: func(u *model.URL) error { return nil },
	}
	svc := NewShortenerService(repo)

	u, err := svc.Shorten("https://example.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u == nil {
		t.Fatal("expected URL")
	}
	if callCount != 4 {
		t.Errorf("exists called %d times, want 4", callCount)
	}
}

func TestShorten_MaxRetriesExceeded(t *testing.T) {
	repo := &mockURLRepo{
		existsFn: func(code string) (bool, error) { return true, nil }, // always collision
	}
	svc := NewShortenerService(repo)

	_, err := svc.Shorten("https://example.com", nil)
	if err == nil {
		t.Fatal("expected error for max retries")
	}
}

func TestResolve_Success(t *testing.T) {
	expected := &model.URL{Code: "abc123", Original: "https://example.com"}
	repo := &mockURLRepo{
		getByCodeFn: func(code string) (*model.URL, error) { return expected, nil },
	}
	svc := NewShortenerService(repo)

	u, err := svc.Resolve("abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u != expected {
		t.Error("unexpected URL returned")
	}
}

func TestResolve_NotFound(t *testing.T) {
	repo := &mockURLRepo{
		getByCodeFn: func(code string) (*model.URL, error) { return nil, nil },
	}
	svc := NewShortenerService(repo)

	u, err := svc.Resolve("abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u != nil {
		t.Error("expected nil URL")
	}
}

func TestResolve_Error(t *testing.T) {
	repo := &mockURLRepo{
		getByCodeFn: func(code string) (*model.URL, error) { return nil, errors.New("db error") },
	}
	svc := NewShortenerService(repo)

	_, err := svc.Resolve("abc123")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGenerateCode(t *testing.T) {
	code := generateCode()
	if len(code) != codeLength {
		t.Errorf("code length = %d, want %d", len(code), codeLength)
	}
	for _, c := range code {
		found := false
		for _, valid := range base62Chars {
			if c == valid {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("invalid character %c in code", c)
		}
	}

	// Verify codes are unique (probabilistic)
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		c := generateCode()
		if codes[c] {
			t.Errorf("duplicate code generated: %s", c)
		}
		codes[c] = true
	}
}
