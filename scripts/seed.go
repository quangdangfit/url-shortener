package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/gocql/gocql"
)

const (
	base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numCodes    = 1000
)

func main() {
	cluster := gocql.NewCluster("localhost:9042")
	cluster.Keyspace = "urlshortener"
	cluster.Consistency = gocql.One
	cluster.Timeout = 10 * time.Second

	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("connect to scylla: %v", err)
	}
	defer session.Close()
	fmt.Println("Connected to ScyllaDB")

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	now := time.Now().UTC()
	totalInserts := 0

	// Realistic distribution: most codes get few clicks, some get many
	clickDistribution := func() int {
		r := rng.Float64()
		switch {
		case r < 0.7:
			return 10 + rng.Intn(90) // 70% get 10-100 clicks
		case r < 0.9:
			return 100 + rng.Intn(900) // 20% get 100-1000 clicks
		case r < 0.98:
			return 1000 + rng.Intn(9000) // 8% get 1000-10000 clicks
		default:
			return 10000 + rng.Intn(90000) // 2% get 10k-100k clicks
		}
	}

	for i := 0; i < numCodes; i++ {
		code := randomCode(rng)
		createdAt := now.AddDate(0, 0, -rng.Intn(90))
		hash := sha256.Sum256([]byte(code))

		if err := session.Query(
			`INSERT INTO urls (code, original, created_at) VALUES (?, ?, ?)`,
			code, fmt.Sprintf("https://example.com/page/%s", hex.EncodeToString(hash[:])), createdAt,
		).Exec(); err != nil {
			log.Printf("insert url %s: %v", code, err)
			continue
		}

		numClicks := clickDistribution()
		devices := []string{"mobile", "desktop", "unknown"}

		for j := 0; j < numClicks; j++ {
			clickedAt := now.Add(-time.Duration(rng.Intn(90*24)) * time.Hour)
			bucket := clickedAt.Format("2006-01-02")
			device := devices[rng.Intn(len(devices))]

			if err := session.Query(
				`INSERT INTO clicks (code, bucket, clicked_at, click_id, country, device, referer) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				code, bucket, clickedAt, gocql.TimeUUID(),
				fmt.Sprintf("192.168.1.%d", rng.Intn(255)),
				device, "https://google.com",
			).Exec(); err != nil {
				log.Printf("insert click: %v", err)
				continue
			}

			if err := session.Query(
				`UPDATE click_counts SET total = total + 1 WHERE code = ? AND bucket = ?`,
				code, bucket,
			).Exec(); err != nil {
				log.Printf("increment count: %v", err)
			}

			totalInserts++
			if totalInserts%10000 == 0 {
				fmt.Printf("Progress: %d inserts completed\n", totalInserts)
			}
		}

		if (i+1)%100 == 0 {
			fmt.Printf("Seeded %d/%d codes\n", i+1, numCodes)
		}
	}

	fmt.Printf("Done! Total click inserts: %d\n", totalInserts)
}

func randomCode(rng *rand.Rand) string {
	b := make([]byte, 6)
	for i := range b {
		b[i] = base62Chars[rng.Intn(len(base62Chars))]
	}
	return string(b)
}
