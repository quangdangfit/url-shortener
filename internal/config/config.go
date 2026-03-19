package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	ScyllaHosts       []string
	ScyllaKeyspace    string
	ScyllaConsistency string
	ServerPort        string
	BaseURL           string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		ScyllaHosts:       strings.Split(getEnv("SCYLLA_HOSTS", "localhost:9042"), ","),
		ScyllaKeyspace:    getEnv("SCYLLA_KEYSPACE", "urlshortener"),
		ScyllaConsistency: getEnv("SCYLLA_CONSISTENCY", "LOCAL_QUORUM"),
		ServerPort:        getEnv("SERVER_PORT", "8080"),
		BaseURL:           getEnv("BASE_URL", "http://localhost:8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
