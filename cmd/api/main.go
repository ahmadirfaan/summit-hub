package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend-summithub/internal/config"
	"backend-summithub/internal/db"
	"backend-summithub/internal/server"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var mainDepsProvider = defaultDeps
var mainRunner = realMain

func main() {
	mainRunner(mainDepsProvider())
}

type mainDeps struct {
	loadConfig      func() config.Config
	connectPostgres func(config.Config) (*pgxpool.Pool, error)
	connectRedis    func(config.Config) *redis.Client
	notify          func(chan<- os.Signal, ...os.Signal)
	run             func(context.Context, config.Config, *pgxpool.Pool, *redis.Client, <-chan os.Signal, ListenFunc) error
}

func defaultDeps() mainDeps {
	return mainDeps{
		loadConfig:      config.Load,
		connectPostgres: db.ConnectPostgres,
		connectRedis:    db.ConnectRedis,
		notify:          signal.Notify,
		run:             Run,
	}
}

func realMain(deps mainDeps) {
	cfg := deps.loadConfig()

	pg, err := deps.connectPostgres(cfg)
	if err != nil {
		log.Printf("postgres connection failed: %v", err)
	}

	rdb := deps.connectRedis(cfg)

	signals := make(chan os.Signal, 1)
	deps.notify(signals, syscall.SIGINT, syscall.SIGTERM)

	if err := deps.run(context.Background(), cfg, pg, rdb, signals, nil); err != nil {
		log.Printf("server exited with error: %v", err)
	}
}

type ListenFunc func(app *fiber.App, addr string) error

var defaultListen ListenFunc = func(app *fiber.App, addr string) error {
	return app.Listen(addr)
}

// Run starts the HTTP server and waits for termination signals.
func Run(ctx context.Context, cfg config.Config, pg *pgxpool.Pool, rdb *redis.Client, signals <-chan os.Signal, listen ListenFunc) error {
	srv := server.NewServer(cfg, pg, rdb)

	if listen == nil {
		listen = defaultListen
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- listen(srv.App, cfg.ServerPort)
	}()

	select {
	case <-signals:
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil {
			return err
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.App.ShutdownWithContext(shutdownCtx); err != nil {
		return err
	}
	if pg != nil {
		pg.Close()
	}
	if rdb != nil {
		_ = rdb.Close()
	}
	return nil
}
