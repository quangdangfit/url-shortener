package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/quangdangfit/url-shortener/internal/model"
)

// --- mock click repo ---

type mockClickRepo struct {
	mu             sync.Mutex
	insertedClicks []*model.Click
	increments     []string // code:bucket
	insertErr      error
	incrementErr   error
	clickCountsFn  func(code string, buckets []string) ([]model.ClickCount, error)
	totalClicksFn  func(code string, buckets []string) (int64, error)
	done           chan struct{}
}

func (m *mockClickRepo) InsertClick(c *model.Click) error {
	m.mu.Lock()
	m.insertedClicks = append(m.insertedClicks, c)
	m.mu.Unlock()
	if m.done != nil {
		m.done <- struct{}{}
	}
	return m.insertErr
}

func (m *mockClickRepo) IncrementCount(code, bucket string) error {
	m.mu.Lock()
	m.increments = append(m.increments, code+":"+bucket)
	m.mu.Unlock()
	return m.incrementErr
}

func (m *mockClickRepo) GetClickCounts(code string, buckets []string) ([]model.ClickCount, error) {
	if m.clickCountsFn != nil {
		return m.clickCountsFn(code, buckets)
	}
	return nil, nil
}

func (m *mockClickRepo) GetTotalClicks(code string, buckets []string) (int64, error) {
	if m.totalClicksFn != nil {
		return m.totalClicksFn(code, buckets)
	}
	return 0, nil
}

// --- tests ---

func TestNewAnalyticsService(t *testing.T) {
	repo := &mockClickRepo{}
	svc := NewAnalyticsService(repo)
	if svc == nil {
		t.Fatal("NewAnalyticsService returned nil")
	}
}

func TestRecordClick_Success(t *testing.T) {
	done := make(chan struct{}, 2)
	repo := &mockClickRepo{done: done}
	svc := newAnalyticsService(repo, 100, 1)

	svc.RecordClick("abc123", "1.2.3.4", "Mozilla/5.0", "https://google.com")

	// Wait for worker to process
	<-done // InsertClick called

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if len(repo.insertedClicks) != 1 {
		t.Fatalf("insertedClicks = %d, want 1", len(repo.insertedClicks))
	}
	click := repo.insertedClicks[0]
	if click.Code != "abc123" {
		t.Errorf("code = %q", click.Code)
	}
	if click.Country != "1.2.3.4" {
		t.Errorf("country = %q", click.Country)
	}
	if click.Device != "desktop" {
		t.Errorf("device = %q", click.Device)
	}
	if click.Referer != "https://google.com" {
		t.Errorf("referer = %q", click.Referer)
	}
	if click.Bucket != time.Now().UTC().Format("2006-01-02") {
		t.Errorf("bucket = %q", click.Bucket)
	}
}

func TestRecordClick_ChannelFull(t *testing.T) {
	repo := &mockClickRepo{}
	// Channel size 1, no workers to drain it
	svc := newAnalyticsService(repo, 1, 0)

	// First call fills the channel
	svc.RecordClick("abc", "1.1.1.1", "ua", "ref")
	// Second call should hit the default branch (channel full) without blocking
	svc.RecordClick("abc", "1.1.1.1", "ua", "ref")
	// If we reach here without hanging, the test passes
}

func TestRecordClick_WorkerHandlesErrors(t *testing.T) {
	done := make(chan struct{}, 2)
	repo := &mockClickRepo{
		insertErr:    errors.New("insert fail"),
		incrementErr: errors.New("increment fail"),
		done:         done,
	}
	svc := newAnalyticsService(repo, 100, 1)

	svc.RecordClick("abc", "1.1.1.1", "ua", "ref")

	// Wait for worker to process (even with errors)
	<-done
	// Allow time for IncrementCount to also be called
	time.Sleep(10 * time.Millisecond)

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.insertedClicks) != 1 {
		t.Errorf("insertedClicks = %d, want 1", len(repo.insertedClicks))
	}
}

func TestGetStats_Success(t *testing.T) {
	repo := &mockClickRepo{
		totalClicksFn: func(code string, buckets []string) (int64, error) {
			return 42, nil
		},
		clickCountsFn: func(code string, buckets []string) ([]model.ClickCount, error) {
			return []model.ClickCount{
				{Code: "abc", Bucket: "2024-01-01", Total: 10},
				{Code: "abc", Bucket: "2024-01-02", Total: 32},
			}, nil
		},
	}
	svc := newAnalyticsService(repo, 100, 0)

	total, counts, err := svc.GetStats("abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 42 {
		t.Errorf("total = %d, want 42", total)
	}
	if len(counts) != 2 {
		t.Errorf("counts length = %d, want 2", len(counts))
	}
}

func TestGetStats_TotalClicksError(t *testing.T) {
	repo := &mockClickRepo{
		totalClicksFn: func(code string, buckets []string) (int64, error) {
			return 0, errors.New("db error")
		},
	}
	svc := newAnalyticsService(repo, 100, 0)

	_, _, err := svc.GetStats("abc")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetStats_ClickCountsError(t *testing.T) {
	repo := &mockClickRepo{
		totalClicksFn: func(code string, buckets []string) (int64, error) {
			return 10, nil
		},
		clickCountsFn: func(code string, buckets []string) ([]model.ClickCount, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newAnalyticsService(repo, 100, 0)

	_, _, err := svc.GetStats("abc")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDetectDevice(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		want      string
	}{
		{"mobile keyword", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0) Mobile Safari", "mobile"},
		{"android", "Mozilla/5.0 (Linux; Android 13) AppleWebKit/537.36", "mobile"},
		{"ipad", "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X)", "mobile"},
		{"opera mini", "Opera/9.80 (J2ME/MIDP; Opera Mini/7.0)", "mobile"},
		{"blackberry", "Mozilla/5.0 (BlackBerry; U)", "mobile"},
		{"webos", "Mozilla/5.0 (webOS/1.0)", "mobile"},
		{"opera mobi", "Opera/9.80 (Android; Opera Mobi)", "mobile"},
		{"ipod", "Mozilla/5.0 (iPod touch; CPU iPhone OS 16_0)", "mobile"},
		{"desktop chrome", "Mozilla/5.0 (Windows NT 10.0) Chrome/120.0.0.0", "desktop"},
		{"desktop firefox", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15) Firefox/120.0", "desktop"},
		{"empty", "", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectDevice(tt.userAgent)
			if got != tt.want {
				t.Errorf("detectDevice(%q) = %q, want %q", tt.userAgent, got, tt.want)
			}
		})
	}
}

func TestLast30DaysBuckets(t *testing.T) {
	buckets := last30DaysBuckets()
	if len(buckets) != 30 {
		t.Fatalf("len = %d, want 30", len(buckets))
	}
	today := time.Now().UTC().Format("2006-01-02")
	if buckets[0] != today {
		t.Errorf("first bucket = %q, want %q", buckets[0], today)
	}
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	if buckets[1] != yesterday {
		t.Errorf("second bucket = %q, want %q", buckets[1], yesterday)
	}
	day29 := time.Now().UTC().AddDate(0, 0, -29).Format("2006-01-02")
	if buckets[29] != day29 {
		t.Errorf("last bucket = %q, want %q", buckets[29], day29)
	}
}
