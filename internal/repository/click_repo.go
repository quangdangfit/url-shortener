package repository

import (
	"fmt"

	"github.com/gocql/gocql"
	"github.com/quangdangfit/url-shortener/internal/model"
)

type ClickRepository struct {
	session *gocql.Session
}

func NewClickRepository(session *gocql.Session) *ClickRepository {
	return &ClickRepository{session: session}
}

func (r *ClickRepository) InsertClick(c *model.Click) error {
	query := `INSERT INTO clicks (code, bucket, clicked_at, click_id, country, device, referer)
	           VALUES (?, ?, ?, ?, ?, ?, ?)`
	if err := r.session.Query(query, c.Code, c.Bucket, c.ClickedAt, c.ClickID, c.Country, c.Device, c.Referer).Exec(); err != nil {
		return fmt.Errorf("insert click: %w", err)
	}
	return nil
}

func (r *ClickRepository) IncrementCount(code, bucket string) error {
	query := `UPDATE click_counts SET total = total + 1 WHERE code = ? AND bucket = ?`
	if err := r.session.Query(query, code, bucket).Exec(); err != nil {
		return fmt.Errorf("increment click count: %w", err)
	}
	return nil
}

func (r *ClickRepository) GetClickCounts(code string, buckets []string) ([]model.ClickCount, error) {
	query := `SELECT code, bucket, total FROM click_counts WHERE code = ? AND bucket IN ?`
	iter := r.session.Query(query, code, buckets).Iter()

	var counts []model.ClickCount
	var cc model.ClickCount
	for iter.Scan(&cc.Code, &cc.Bucket, &cc.Total) {
		counts = append(counts, cc)
	}
	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("get click counts: %w", err)
	}
	return counts, nil
}

func (r *ClickRepository) GetTotalClicks(code string, buckets []string) (int64, error) {
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
