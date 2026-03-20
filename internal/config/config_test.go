package config

import (
	"os"
	"testing"
)

func TestGetEnv_WithValue(t *testing.T) {
	os.Setenv("TEST_KEY_CONFIG", "custom_value")
	defer os.Unsetenv("TEST_KEY_CONFIG")

	got := getEnv("TEST_KEY_CONFIG", "default")
	if got != "custom_value" {
		t.Errorf("getEnv() = %q, want %q", got, "custom_value")
	}
}

func TestGetEnv_Fallback(t *testing.T) {
	os.Unsetenv("TEST_KEY_MISSING")

	got := getEnv("TEST_KEY_MISSING", "fallback_val")
	if got != "fallback_val" {
		t.Errorf("getEnv() = %q, want %q", got, "fallback_val")
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Clear all relevant env vars
	keys := []string{"SCYLLA_HOSTS", "SCYLLA_KEYSPACE", "SCYLLA_CONSISTENCY", "SERVER_PORT", "BASE_URL"}
	for _, k := range keys {
		os.Unsetenv(k)
	}

	cfg := Load()

	if len(cfg.ScyllaHosts) != 1 || cfg.ScyllaHosts[0] != "localhost:9042" {
		t.Errorf("ScyllaHosts = %v, want [localhost:9042]", cfg.ScyllaHosts)
	}
	if cfg.ScyllaKeyspace != "urlshortener" {
		t.Errorf("ScyllaKeyspace = %q, want %q", cfg.ScyllaKeyspace, "urlshortener")
	}
	if cfg.ScyllaConsistency != "LOCAL_QUORUM" {
		t.Errorf("ScyllaConsistency = %q, want %q", cfg.ScyllaConsistency, "LOCAL_QUORUM")
	}
	if cfg.ServerPort != "8080" {
		t.Errorf("ServerPort = %q, want %q", cfg.ServerPort, "8080")
	}
	if cfg.BaseURL != "http://localhost:8080" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "http://localhost:8080")
	}
}

func TestLoad_CustomEnv(t *testing.T) {
	os.Setenv("SCYLLA_HOSTS", "host1:9042,host2:9042")
	os.Setenv("SCYLLA_KEYSPACE", "myks")
	os.Setenv("SCYLLA_CONSISTENCY", "ONE")
	os.Setenv("SERVER_PORT", "3000")
	os.Setenv("BASE_URL", "https://short.io")
	defer func() {
		os.Unsetenv("SCYLLA_HOSTS")
		os.Unsetenv("SCYLLA_KEYSPACE")
		os.Unsetenv("SCYLLA_CONSISTENCY")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("BASE_URL")
	}()

	cfg := Load()

	if len(cfg.ScyllaHosts) != 2 {
		t.Fatalf("ScyllaHosts length = %d, want 2", len(cfg.ScyllaHosts))
	}
	if cfg.ScyllaHosts[0] != "host1:9042" || cfg.ScyllaHosts[1] != "host2:9042" {
		t.Errorf("ScyllaHosts = %v", cfg.ScyllaHosts)
	}
	if cfg.ScyllaKeyspace != "myks" {
		t.Errorf("ScyllaKeyspace = %q", cfg.ScyllaKeyspace)
	}
	if cfg.ScyllaConsistency != "ONE" {
		t.Errorf("ScyllaConsistency = %q", cfg.ScyllaConsistency)
	}
	if cfg.ServerPort != "3000" {
		t.Errorf("ServerPort = %q", cfg.ServerPort)
	}
	if cfg.BaseURL != "https://short.io" {
		t.Errorf("BaseURL = %q", cfg.BaseURL)
	}
}
