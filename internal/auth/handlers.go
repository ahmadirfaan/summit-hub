package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(r fiber.Router, svc *Service) {
	r.Post("/register", func(c *fiber.Ctx) error {
		var req RegisterRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
		}
		user, tokens, err := svc.Register(c.Context(), req)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"user": user, "tokens": tokens})
	})

	r.Post("/login", func(c *fiber.Ctx) error {
		var req LoginRequest
		if err := c.BodyParser(&req); err != nil || req.Email == "" || req.Password == "" {
			return fiber.NewError(fiber.StatusBadRequest, "email and password required")
		}
		_, resp, err := svc.Login(c.Context(), req)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		}
		return c.JSON(resp)
	})

	r.Post("/refresh", func(c *fiber.Ctx) error {
		var req RefreshRequest
		if err := c.BodyParser(&req); err != nil || req.RefreshToken == "" {
			return fiber.NewError(fiber.StatusBadRequest, "refresh_token required")
		}

		userID, err := svc.ValidateRefreshToken(c.Context(), req.RefreshToken)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		}

		resp, err := svc.GenerateTokens(c.Context(), userID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(resp)
	})

	r.Get("/jwt/verify", func(c *fiber.Ctx) error {
		token := parseBearer(c.Get("Authorization"))
		if token == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "missing bearer token")
		}

		userID, err := svc.ValidateAccessToken(token)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		}
		return c.JSON(fiber.Map{"user_id": userID})
	})
}

func parseBearer(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
