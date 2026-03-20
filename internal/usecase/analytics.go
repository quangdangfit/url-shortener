package usecase

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/quangdangfit/url-shortener/internal/domain"
	"github.com/quangdangfit/url-shortener/internal/port"
)

type AnalyticsUseCase struct {
	clickRepo port.ClickRepository
	clickCh   chan *domain.Click
}

func NewAnalyticsUseCase(clickRepo port.ClickRepository) *AnalyticsUseCase {
	return newAnalyticsUseCase(clickRepo, 10000, 4)
}

func newAnalyticsUseCase(clickRepo port.ClickRepository, chanSize, numWorkers int) *AnalyticsUseCase {
	s := &AnalyticsUseCase{
		clickRepo: clickRepo,
		clickCh:   make(chan *domain.Click, chanSize),
	}
	s.startWorkers(numWorkers)
	return s
}

func (s *AnalyticsUseCase) startWorkers(n int) {
	for i := 0; i < n; i++ {
		go func() {
			for click := range s.clickCh {
				if err := s.clickRepo.InsertClick(click); err != nil {
					slog.Error("failed to insert click", "error", err, "code", click.Code)
				}
				if err := s.clickRepo.IncrementCount(click.Code, click.Bucket); err != nil {
					slog.Error("failed to increment click count", "error", err, "code", click.Code)
				}
			}
		}()
	}
}

func (s *AnalyticsUseCase) RecordClick(code, ip, userAgent, referer string) {
	now := time.Now().UTC()
	click := &domain.Click{
		Code:      code,
		Bucket:    now.Format("2006-01-02"),
		ClickedAt: now,
		ClickID:   uuid.New().String(),
		Country:   ip,
		Device:    detectDevice(userAgent),
		Referer:   referer,
	}

	select {
	case s.clickCh <- click:
	default:
		slog.Warn("click channel full, dropping event", "code", code)
	}
}

func (s *AnalyticsUseCase) GetStats(code string) (int64, []domain.ClickCount, error) {
	buckets := last30DaysBuckets()

	total, err := s.clickRepo.GetTotalClicks(code, buckets)
	if err != nil {
		return 0, nil, fmt.Errorf("get total clicks: %w", err)
	}

	counts, err := s.clickRepo.GetClickCounts(code, buckets)
	if err != nil {
		return 0, nil, fmt.Errorf("get click counts by day: %w", err)
	}

	return total, counts, nil
}

func last30DaysBuckets() []string {
	buckets := make([]string, 30)
	now := time.Now().UTC()
	for i := 0; i < 30; i++ {
		buckets[i] = now.AddDate(0, 0, -i).Format("2006-01-02")
	}
	return buckets
}

func detectDevice(userAgent string) string {
	ua := strings.ToLower(userAgent)
	mobileKeywords := []string{"mobile", "android", "iphone", "ipad", "ipod", "webos", "blackberry", "opera mini", "opera mobi"}
	for _, kw := range mobileKeywords {
		if strings.Contains(ua, kw) {
			return "mobile"
		}
	}
	if userAgent == "" {
		return "unknown"
	}
	return "desktop"
}
