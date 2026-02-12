package server

import (
	"backend-summithub/internal/auth"
	"backend-summithub/internal/config"
	"backend-summithub/internal/social"
	"backend-summithub/internal/storage"
	"backend-summithub/internal/stream"
	"backend-summithub/internal/tracking"
	"backend-summithub/internal/trip"
	"backend-summithub/internal/waypoint"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	App    *fiber.App
	Cfg    config.Config
	DB     *pgxpool.Pool
	Redis  *redis.Client
	Stream *stream.Hub
}

func NewServer(cfg config.Config, db *pgxpool.Pool, redisClient *redis.Client) *Server {
	app := fiber.New()
	app.Use(recover.New())
	app.Use(logger.New())

	s := &Server{
		App:    app,
		Cfg:    cfg,
		DB:     db,
		Redis:  redisClient,
		Stream: stream.NewHub(redisClient),
	}

	registerRoutes(s)
	return s
}

func registerRoutes(s *Server) {
	s.App.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	jwtMiddleware := auth.JWTMiddleware(s.Cfg.JWTSecret)

	auth.RegisterRoutes(s.App.Group("/auth"), auth.NewService(s.Cfg.JWTSecret, s.DB))
	trip.RegisterRoutes(s.App.Group("/trips"), trip.NewService(s.DB), jwtMiddleware)
	tracking.RegisterRoutes(s.App.Group("/tracking"), tracking.NewService(s.DB, s.Stream), jwtMiddleware)
	waypoint.RegisterRoutes(s.App.Group("/waypoints"), waypoint.NewService(s.DB), jwtMiddleware)
	social.RegisterRoutes(s.App.Group("/social"), social.NewService(s.DB), jwtMiddleware)
	storage.RegisterRoutes(s.App.Group("/storage"), storage.NewService(s.DB), jwtMiddleware)
	stream.RegisterRoutes(s.App.Group("/stream"), s.Stream)
}
