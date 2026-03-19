package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
)

func NewSession(hosts []string, keyspace, consistency string) (*gocql.Session, error) {
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = keyspace
	cluster.NumConns = 4
	cluster.Timeout = 10 * time.Second
	cluster.ConnectTimeout = 10 * time.Second

	switch strings.ToUpper(consistency) {
	case "LOCAL_QUORUM":
		cluster.Consistency = gocql.LocalQuorum
	case "QUORUM":
		cluster.Consistency = gocql.Quorum
	case "ONE":
		cluster.Consistency = gocql.One
	default:
		cluster.Consistency = gocql.LocalQuorum
	}

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("create scylla session: %w", err)
	}
	return session, nil
}

func HealthCheck(session *gocql.Session) error {
	iter := session.Query("SELECT now() FROM system.local").Iter()
	var t time.Time
	if iter.Scan(&t) {
		return iter.Close()
	}
	if err := iter.Close(); err != nil {
		return fmt.Errorf("scylla health check: %w", err)
	}
	return nil
}
