package db

import (
	"fmt"

	"github.com/gocql/gocql"
)

func RunMigrations(hosts []string) error {
	cluster := gocql.NewCluster(hosts...)
	cluster.Consistency = gocql.All

	session, err := cluster.CreateSession()
	if err != nil {
		return fmt.Errorf("connect for migrations: %w", err)
	}
	defer session.Close()

	statements := []string{
		`CREATE KEYSPACE IF NOT EXISTS urlshortener
		 WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}`,

		`CREATE TABLE IF NOT EXISTS urlshortener.urls (
			code        TEXT PRIMARY KEY,
			original    TEXT,
			created_at  TIMESTAMP,
			expires_at  TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS urlshortener.clicks (
			code        TEXT,
			bucket      TEXT,
			clicked_at  TIMESTAMP,
			click_id    UUID,
			country     TEXT,
			device      TEXT,
			referer     TEXT,
			PRIMARY KEY ((code, bucket), clicked_at, click_id)
		) WITH CLUSTERING ORDER BY (clicked_at DESC)
		  AND default_time_to_live = 7776000`,

		`CREATE TABLE IF NOT EXISTS urlshortener.click_counts (
			code        TEXT,
			bucket      TEXT,
			total       COUNTER,
			PRIMARY KEY (code, bucket)
		)`,
	}

	for _, stmt := range statements {
		if err := session.Query(stmt).Exec(); err != nil {
			return fmt.Errorf("migration failed: %w\nstatement: %s", err, stmt)
		}
	}

	return nil
}
