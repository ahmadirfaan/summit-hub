package storage

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) SaveObject(ctx context.Context, userID, url, kind string) (string, error) {
	id := uuid.NewString()
	_, err := s.db.Exec(ctx, `
		INSERT INTO storage_objects (id, user_id, url, kind)
		VALUES ($1,$2,$3,$4)
	`, id, userID, url, kind)
	if err != nil {
		return "", err
	}
	return id, nil
}

func RegisterRoutes(r fiber.Router, svc *Service, authMiddleware fiber.Handler) {
	r.Post("/upload", authMiddleware, func(c *fiber.Ctx) error {
		var body struct {
			UserID   string `json:"user_id"`
			FileName string `json:"file_name"`
			Kind     string `json:"kind"`
		}
		_ = c.BodyParser(&body)
		if body.FileName == "" {
			body.FileName = "upload"
		}
		url := "https://storage.example/" + body.FileName
		id, err := svc.SaveObject(c.Context(), body.UserID, url, body.Kind)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{
			"id":         id,
			"url":        url,
			"expires_at": time.Now().Add(15 * time.Minute),
		})
	})
}
