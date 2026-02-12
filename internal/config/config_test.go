package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg := Load()
	if cfg.ServerPort == "" {
		t.Fatalf("expected default server port")
	}
	if cfg.PostgresURL == "" {
		t.Fatalf("expected default postgres url")
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("SERVER_PORT", ":9000")
	t.Setenv("POSTGRES_URL", "postgres://example")
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("JWT_SECRET", "secret")

	cfg := Load()
	if cfg.ServerPort != ":9000" {
		t.Fatalf("expected override port")
	}
	if cfg.PostgresURL != "postgres://example" {
		t.Fatalf("expected override postgres")
	}
	if cfg.RedisAddr != "redis:6379" {
		t.Fatalf("expected override redis")
	}
	if cfg.JWTSecret != "secret" {
		t.Fatalf("expected override secret")
	}
}
