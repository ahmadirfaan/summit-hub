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
)

func main() {
	cfg := config.Load()

	pg, err := db.ConnectPostgres(cfg)
	if err != nil {
		log.Printf("postgres connection failed: %v", err)
	}

	rdb := db.ConnectRedis(cfg)

	srv := server.NewServer(cfg, pg, rdb)

	go func() {
		if err := srv.App.Listen(cfg.ServerPort); err != nil {
			log.Printf("fiber stopped: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.App.ShutdownWithContext(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	if pg != nil {
		pg.Close()
	}
	if rdb != nil {
		_ = rdb.Close()
	}
}
