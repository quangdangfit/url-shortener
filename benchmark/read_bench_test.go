package benchmark

import (
	"testing"
	"time"

	"github.com/gocql/gocql"
)

func BenchmarkRedirect(b *testing.B) {
	session := benchSession(b)
	defer session.Close()

	code := "benchR1"
	ensureURL(b, session, code)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var original string
		session.Query(`SELECT original FROM urls WHERE code = ?`, code).Scan(&original)
	}
}

func BenchmarkStats(b *testing.B) {
	session := benchSession(b)
	defer session.Close()

	code := "benchR2"
	ensureURL(b, session, code)

	// Seed some counter data
	now := time.Now().UTC()
	for d := 0; d < 30; d++ {
		bucket := now.AddDate(0, 0, -d).Format("2006-01-02")
		session.Query(`UPDATE click_counts SET total = total + 1 WHERE code = ? AND bucket = ?`, code, bucket).Exec()
	}

	buckets := make([]string, 30)
	for d := 0; d < 30; d++ {
		buckets[d] = now.AddDate(0, 0, -d).Format("2006-01-02")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iter := session.Query(`SELECT bucket, total FROM click_counts WHERE code = ? AND bucket IN ?`, code, buckets).Iter()
		var bucket string
		var total int64
		for iter.Scan(&bucket, &total) {
		}
		iter.Close()
	}
}

func BenchmarkStatsRange(b *testing.B) {
	session := benchSession(b)
	defer session.Close()

	code := "benchR3"
	ensureURL(b, session, code)

	now := time.Now().UTC()
	for d := 0; d < 30; d++ {
		bucket := now.AddDate(0, 0, -d).Format("2006-01-02")
		for c := 0; c < 10; c++ {
			session.Query(
				`INSERT INTO clicks (code, bucket, clicked_at, click_id, country, device, referer) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				code, bucket, now.Add(-time.Duration(d*24+c)*time.Hour), gocql.TimeUUID(), "127.0.0.1", "desktop", "",
			).Exec()
		}
	}

	buckets := make([]string, 30)
	for d := 0; d < 30; d++ {
		buckets[d] = now.AddDate(0, 0, -d).Format("2006-01-02")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iter := session.Query(`SELECT code, bucket, total FROM click_counts WHERE code = ? AND bucket IN ?`, code, buckets).Iter()
		var c string
		var bk string
		var total int64
		for iter.Scan(&c, &bk, &total) {
		}
		iter.Close()
	}
}
