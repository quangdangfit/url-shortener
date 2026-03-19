package benchmark

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/gocql/gocql"
)

func BenchmarkSingleWrite(b *testing.B) {
	session := benchSession(b)
	defer session.Close()

	code := "bench1"
	ensureURL(b, session, code)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		now := time.Now().UTC()
		bucket := now.Format("2006-01-02")
		session.Query(
			`INSERT INTO clicks (code, bucket, clicked_at, click_id, country, device, referer) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			code, bucket, now, gocql.TimeUUID(), "127.0.0.1", "desktop", "",
		).Exec()
	}
}

func BenchmarkConcurrentWrites(b *testing.B) {
	session := benchSession(b)
	defer session.Close()

	code := "bench2"
	ensureURL(b, session, code)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			now := time.Now().UTC()
			bucket := now.Format("2006-01-02")
			session.Query(
				`INSERT INTO clicks (code, bucket, clicked_at, click_id, country, device, referer) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				code, bucket, now, gocql.TimeUUID(), "127.0.0.1", "desktop", "",
			).Exec()
		}
	})
}

func BenchmarkBatchInsert(b *testing.B) {
	session := benchSession(b)
	defer session.Close()

	code := "bench3"
	ensureURL(b, session, code)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 1000; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				now := time.Now().UTC().Add(-time.Duration(rng.Intn(86400)) * time.Second)
				bucket := now.Format("2006-01-02")
				session.Query(
					`INSERT INTO clicks (code, bucket, clicked_at, click_id, country, device, referer) VALUES (?, ?, ?, ?, ?, ?, ?)`,
					code, bucket, now, gocql.TimeUUID(),
					fmt.Sprintf("10.0.0.%d", rng.Intn(255)), "mobile", "",
				).Exec()
			}()
		}
		wg.Wait()
	}
}

func benchSession(b *testing.B) *gocql.Session {
	b.Helper()
	cluster := gocql.NewCluster("localhost:9042")
	cluster.Keyspace = "urlshortener"
	cluster.Consistency = gocql.One
	cluster.NumConns = 4
	session, err := cluster.CreateSession()
	if err != nil {
		b.Fatalf("connect to scylla: %v", err)
	}
	return session
}

func ensureURL(b *testing.B, session *gocql.Session, code string) {
	b.Helper()
	session.Query(
		`INSERT INTO urls (code, original, created_at) VALUES (?, ?, ?)`,
		code, "https://example.com/bench", time.Now().UTC(),
	).Exec()
}
