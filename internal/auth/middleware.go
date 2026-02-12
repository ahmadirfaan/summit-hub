package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// JWTMiddleware validates bearer tokens and stores user_id in locals.
func JWTMiddleware(secret string) fiber.Handler {
	secretBytes := []byte(secret)
	return func(c *fiber.Ctx) error {
		token := bearerFromHeader(c.Get("Authorization"))
		if token == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "missing bearer token")
		}

		parsed, err := parseMiddlewareClaimsFn(token, &Claims{}, func(_ *jwt.Token) (interface{}, error) {
			return secretBytes, nil
		})
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		}

		claims, ok := parsed.Claims.(*Claims)
		if !ok || !parsed.Valid {
			return fiber.NewError(fiber.StatusUnauthorized, "token invalid")
		}

		c.Locals("user_id", claims.UserID)
		return c.Next()
	}
}

var parseMiddlewareClaimsFn = jwt.ParseWithClaims

func bearerFromHeader(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
