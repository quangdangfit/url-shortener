package repository

import (
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/quangdangfit/url-shortener/internal/domain"
)

type ScyllaURLRepository struct {
	session *gocql.Session
}

func NewScyllaURLRepository(session *gocql.Session) *ScyllaURLRepository {
	return &ScyllaURLRepository{session: session}
}

func (r *ScyllaURLRepository) Create(u *domain.URL) error {
	query := `INSERT INTO urls (code, original, created_at, expires_at) VALUES (?, ?, ?, ?)`
	if err := r.session.Query(query, u.Code, u.Original, u.CreatedAt, u.ExpiresAt).Exec(); err != nil {
		return fmt.Errorf("insert url: %w", err)
	}
	return nil
}

func (r *ScyllaURLRepository) GetByCode(code string) (*domain.URL, error) {
	query := `SELECT code, original, created_at, expires_at FROM urls WHERE code = ?`
	var u domain.URL
	var expiresAt time.Time
	err := r.session.Query(query, code).Scan(&u.Code, &u.Original, &u.CreatedAt, &expiresAt)
	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get url by code: %w", err)
	}
	if !expiresAt.IsZero() {
		u.ExpiresAt = &expiresAt
	}
	return &u, nil
}

func (r *ScyllaURLRepository) Exists(code string) (bool, error) {
	query := `SELECT code FROM urls WHERE code = ?`
	var c string
	err := r.session.Query(query, code).Scan(&c)
	if err != nil {
		if err == gocql.ErrNotFound {
			return false, nil
		}
		return false, fmt.Errorf("check url exists: %w", err)
	}
	return true, nil
}
