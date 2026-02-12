package social

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(r fiber.Router, svc *Service, authMiddleware fiber.Handler) {
	r.Post("/posts", authMiddleware, func(c *fiber.Ctx) error {
		var req Post
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if req.UserID == "" || req.Content == "" {
			return fiber.NewError(fiber.StatusBadRequest, "user_id and content required")
		}
		post, err := svc.CreatePost(c.Context(), req)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(post)
	})

	r.Post("/posts/:id/photos", authMiddleware, func(c *fiber.Ctx) error {
		var body struct {
			PhotoURL string `json:"photo_url"`
		}
		if err := c.BodyParser(&body); err != nil || body.PhotoURL == "" {
			return fiber.NewError(fiber.StatusBadRequest, "photo_url required")
		}
		photo, err := svc.AddPhoto(c.Context(), c.Params("id"), body.PhotoURL)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(photo)
	})

	r.Post("/follow", authMiddleware, func(c *fiber.Ctx) error {
		var req Follow
		if err := c.BodyParser(&req); err != nil || req.FollowerID == "" || req.FollowingID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "follower_id and following_id required")
		}
		if err := svc.Follow(c.Context(), req.FollowerID, req.FollowingID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.SendStatus(fiber.StatusCreated)
	})

	r.Get("/feed", authMiddleware, func(c *fiber.Ctx) error {
		userID := c.Query("user_id")
		if userID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "user_id required")
		}
		feed, err := svc.Feed(c.Context(), userID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(feed)
	})

	r.Get("/posts/nearby", func(c *fiber.Ctx) error {
		lat, _ := strconv.ParseFloat(c.Query("lat"), 64)
		lng, _ := strconv.ParseFloat(c.Query("lng"), 64)
		radius, _ := strconv.ParseFloat(c.Query("radius_km"), 64)
		if radius == 0 {
			radius = 5
		}
		posts, err := svc.Nearby(c.Context(), lat, lng, radius)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(posts)
	})
}
