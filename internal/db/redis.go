package db

import (
	"backend-summithub/internal/config"
	"github.com/redis/go-redis/v9"
)

func ConnectRedis(cfg config.Config) *redis.Client {
	if cfg.RedisAddr == "" {
		return nil
	}

	return redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	})
}
