package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestJWTMiddleware(t *testing.T) {
	app := fiber.New()
	app.Get("/private", JWTMiddleware("secret"), func(c *fiber.Ctx) error {
		if c.Locals("user_id") == nil {
			return fiber.NewError(fiber.StatusUnauthorized)
		}
		return c.SendStatus(http.StatusOK)
	})

	svc := NewService("secret", nil)
	_ = svc

	// missing token
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized")
	}

	// valid token
	token, _ := svc.signToken("user-1", accessTokenTTL)
	req = httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ = app.Test(req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected ok")
	}
}
