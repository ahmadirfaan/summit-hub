package db

import (
	"testing"

	"backend-summithub/internal/config"
)

func TestConnectRedisEmpty(t *testing.T) {
	cfg := config.Config{RedisAddr: ""}
	client := ConnectRedis(cfg)
	if client != nil {
		t.Fatalf("expected nil redis client when addr empty")
	}
}

func TestConnectPostgresInvalidURL(t *testing.T) {
	cfg := config.Config{PostgresURL: "invalid-url"}
	pool, err := ConnectPostgres(cfg)
	if err == nil {
		t.Fatalf("expected error for invalid url")
	}
	if pool != nil {
		pool.Close()
	}
}

func TestConnectPostgresPingError(t *testing.T) {
	cfg := config.Config{PostgresURL: "postgres://user:pass@localhost:1/db"}
	pool, err := ConnectPostgres(cfg)
	if err == nil {
		t.Fatalf("expected ping error")
	}
	if pool != nil {
		pool.Close()
	}
}

func TestConnectRedisConfigured(t *testing.T) {
	cfg := config.Config{RedisAddr: "localhost:6379"}
	client := ConnectRedis(cfg)
	if client == nil {
		t.Fatalf("expected redis client")
	}
	_ = client.Close()
}
