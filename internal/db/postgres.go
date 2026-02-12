package db

import (
	"context"
	"time"

	"backend-summithub/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectPostgres(cfg config.Config) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := newPoolFn(ctx, cfg.PostgresURL)
	if err != nil {
		return nil, err
	}
	if err := pingPoolFn(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

var newPoolFn = pgxpool.New

var pingPoolFn = func(ctx context.Context, pool *pgxpool.Pool) error {
	return pool.Ping(ctx)
}
