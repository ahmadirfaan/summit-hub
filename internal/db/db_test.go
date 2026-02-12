package db

import (
	"context"
	"testing"

	"backend-summithub/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
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

func TestConnectPostgresSuccess(t *testing.T) {
	oldNew := newPoolFn
	oldPing := pingPoolFn
	defer func() {
		newPoolFn = oldNew
		pingPoolFn = oldPing
	}()

	newPoolFn = func(ctx context.Context, _ string) (*pgxpool.Pool, error) {
		return pgxpool.New(ctx, "postgres://user:pass@localhost:1/db")
	}
	pingPoolFn = func(_ context.Context, _ *pgxpool.Pool) error {
		return nil
	}

	cfg := config.Config{PostgresURL: "postgres://user:pass@localhost:1/db"}
	pool, err := ConnectPostgres(cfg)
	if err != nil {
		t.Fatalf("expected success")
	}
	if pool == nil {
		t.Fatalf("expected pool")
	}
	pool.Close()
}

func TestConnectRedisConfigured(t *testing.T) {
	cfg := config.Config{RedisAddr: "localhost:6379"}
	client := ConnectRedis(cfg)
	if client == nil {
		t.Fatalf("expected redis client")
	}
	_ = client.Close()
}
