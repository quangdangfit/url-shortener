package usecase

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/quangdangfit/url-shortener/internal/domain"
	"github.com/quangdangfit/url-shortener/internal/port"
)

const (
	base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	codeLength  = 6
	maxRetries  = 10
)

type ShortenerUseCase struct {
	urlRepo port.URLRepository
}

func NewShortenerUseCase(urlRepo port.URLRepository) *ShortenerUseCase {
	return &ShortenerUseCase{urlRepo: urlRepo}
}

func (s *ShortenerUseCase) Shorten(originalURL string, ttlDays *int) (*domain.URL, error) {
	var code string
	var err error
	for i := 0; i < maxRetries; i++ {
		code = generateCode()
		exists, checkErr := s.urlRepo.Exists(code)
		if checkErr != nil {
			return nil, fmt.Errorf("check code collision: %w", checkErr)
		}
		if !exists {
			break
		}
		if i == maxRetries-1 {
			return nil, fmt.Errorf("failed to generate unique code after %d retries", maxRetries)
		}
	}

	now := time.Now().UTC()
	u := &domain.URL{
		Code:      code,
		Original:  originalURL,
		CreatedAt: now,
	}
	if ttlDays != nil && *ttlDays > 0 {
		exp := now.Add(time.Duration(*ttlDays) * 24 * time.Hour)
		u.ExpiresAt = &exp
	}

	err = s.urlRepo.Create(u)
	if err != nil {
		return nil, fmt.Errorf("create short url: %w", err)
	}
	return u, nil
}

func (s *ShortenerUseCase) Resolve(code string) (*domain.URL, error) {
	u, err := s.urlRepo.GetByCode(code)
	if err != nil {
		return nil, fmt.Errorf("resolve url: %w", err)
	}
	return u, nil
}

func generateCode() string {
	b := make([]byte, codeLength)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(base62Chars))))
		b[i] = base62Chars[n.Int64()]
	}
	return string(b)
}
