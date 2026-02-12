package main

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"backend-summithub/internal/config"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func TestRunHandlesSignal(t *testing.T) {
	cfg := config.Config{ServerPort: ":0"}
	signals := make(chan os.Signal, 1)

	listenCalled := false
	listen := func(_ *fiber.App, _ string) error {
		listenCalled = true
		return nil
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		signals <- syscall.SIGINT
	}()

	if err := Run(context.Background(), cfg, nil, nil, signals, listen); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if !listenCalled {
		t.Fatalf("expected listen to be called")
	}
}

func TestRunContextCancel(t *testing.T) {
	cfg := config.Config{ServerPort: ":0"}
	signals := make(chan os.Signal, 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := Run(ctx, cfg, nil, nil, signals, func(_ *fiber.App, _ string) error { return nil }); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
}

func TestRunListenError(t *testing.T) {
	cfg := config.Config{ServerPort: ":0"}
	signals := make(chan os.Signal, 1)

	err := Run(context.Background(), cfg, nil, nil, signals, func(_ *fiber.App, _ string) error {
		return errListen
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRunDefaultListen(t *testing.T) {
	cfg := config.Config{ServerPort: ":0"}
	signals := make(chan os.Signal, 1)

	oldListen := defaultListen
	defaultListen = func(_ *fiber.App, _ string) error { return nil }
	defer func() { defaultListen = oldListen }()

	go func() {
		signals <- syscall.SIGINT
	}()

	if err := Run(context.Background(), cfg, nil, nil, signals, nil); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
}

func TestRunListenReturnsNil(t *testing.T) {
	cfg := config.Config{ServerPort: ":0"}
	signals := make(chan os.Signal, 1)

	if err := Run(context.Background(), cfg, nil, nil, signals, func(_ *fiber.App, _ string) error {
		return nil
	}); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
}

var errListen = context.Canceled

func TestRealMainHandlesErrors(t *testing.T) {
	calledNotify := false
	calledRun := false
	deps := mainDeps{
		loadConfig:      func() config.Config { return config.Config{ServerPort: ":0"} },
		connectPostgres: func(config.Config) (*pgxpool.Pool, error) { return nil, errListen },
		connectRedis:    func(config.Config) *redis.Client { return nil },
		notify: func(ch chan<- os.Signal, _ ...os.Signal) {
			calledNotify = true
			close(ch)
		},
		run: func(context.Context, config.Config, *pgxpool.Pool, *redis.Client, <-chan os.Signal, ListenFunc) error {
			calledRun = true
			return errListen
		},
	}

	realMain(deps)
	if !calledNotify {
		t.Fatalf("expected notify to be called")
	}
	if !calledRun {
		t.Fatalf("expected run to be called")
	}
}

func TestDefaultDeps(t *testing.T) {
	deps := defaultDeps()
	if deps.loadConfig == nil || deps.connectPostgres == nil || deps.connectRedis == nil || deps.notify == nil || deps.run == nil {
		t.Fatalf("expected default deps to be set")
	}
}

func TestMainUsesOverrides(t *testing.T) {
	oldProvider := mainDepsProvider
	oldRunner := mainRunner
	defer func() {
		mainDepsProvider = oldProvider
		mainRunner = oldRunner
	}()

	called := false
	mainDepsProvider = func() mainDeps { return mainDeps{} }
	mainRunner = func(mainDeps) { called = true }

	main()
	if !called {
		t.Fatalf("expected main runner to be called")
	}
}

func TestRunClosesResources(t *testing.T) {
	cfg := config.Config{ServerPort: ":0"}
	signals := make(chan os.Signal, 1)

	pool, err := pgxpool.New(context.Background(), "postgres://user:pass@localhost:1/db")
	if err != nil {
		t.Fatalf("pool create error: %v", err)
	}
	defer pool.Close()

	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})

	listen := func(_ *fiber.App, _ string) error {
		signals <- syscall.SIGINT
		return nil
	}

	if err := Run(context.Background(), cfg, pool, client, signals, listen); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
}

func TestRunShutdownError(t *testing.T) {
	cfg := config.Config{ServerPort: ":0"}
	signals := make(chan os.Signal, 1)

	oldShutdown := shutdownFn
	shutdownFn = func(_ *fiber.App, _ context.Context) error { return errListen }
	defer func() { shutdownFn = oldShutdown }()

	go func() {
		signals <- syscall.SIGINT
	}()

	if err := Run(context.Background(), cfg, nil, nil, signals, func(_ *fiber.App, _ string) error { return nil }); err == nil {
		t.Fatalf("expected shutdown error")
	}
}
