package repository

import (
	"fmt"

	"github.com/gocql/gocql"
	"github.com/quangdangfit/url-shortener/internal/domain"
)

type ScyllaClickRepository struct {
	session *gocql.Session
}

func NewScyllaClickRepository(session *gocql.Session) *ScyllaClickRepository {
	return &ScyllaClickRepository{session: session}
}

func (r *ScyllaClickRepository) InsertClick(c *domain.Click) error {
	clickID, err := gocql.ParseUUID(c.ClickID)
	if err != nil {
		return fmt.Errorf("parse click id: %w", err)
	}

	query := `INSERT INTO clicks (code, bucket, clicked_at, click_id, country, device, referer)
	           VALUES (?, ?, ?, ?, ?, ?, ?)`
	if err := r.session.Query(query, c.Code, c.Bucket, c.ClickedAt, clickID, c.Country, c.Device, c.Referer).Exec(); err != nil {
		return fmt.Errorf("insert click: %w", err)
	}
	return nil
}

func (r *ScyllaClickRepository) IncrementCount(code, bucket string) error {
	query := `UPDATE click_counts SET total = total + 1 WHERE code = ? AND bucket = ?`
	if err := r.session.Query(query, code, bucket).Exec(); err != nil {
		return fmt.Errorf("increment click count: %w", err)
	}
	return nil
}

func (r *ScyllaClickRepository) GetClickCounts(code string, buckets []string) ([]domain.ClickCount, error) {
	query := `SELECT code, bucket, total FROM click_counts WHERE code = ? AND bucket IN ?`
	iter := r.session.Query(query, code, buckets).Iter()

	var counts []domain.ClickCount
	var cc domain.ClickCount
	for iter.Scan(&cc.Code, &cc.Bucket, &cc.Total) {
		counts = append(counts, cc)
	}
	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("get click counts: %w", err)
	}
	return counts, nil
}

func (r *ScyllaClickRepository) GetTotalClicks(code string, buckets []string) (int64, error) {
	counts, err := r.GetClickCounts(code, buckets)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, c := range counts {
		total += c.Total
	}
	return total, nil
}
