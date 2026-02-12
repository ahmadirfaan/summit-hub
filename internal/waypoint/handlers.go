package waypoint

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(r fiber.Router, svc *Service, authMiddleware fiber.Handler) {
	r.Post("/", authMiddleware, func(c *fiber.Ctx) error {
		var req Waypoint
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if req.Name == "" || req.CreatedBy == "" {
			return fiber.NewError(fiber.StatusBadRequest, "name and created_by required")
		}
		wp, err := svc.CreateWaypoint(c.Context(), req)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(wp)
	})

	r.Get("/:id", func(c *fiber.Ctx) error {
		wp, err := svc.GetWaypoint(c.Context(), c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "waypoint not found")
		}
		return c.JSON(wp)
	})

	r.Put("/:id", authMiddleware, func(c *fiber.Ctx) error {
		var req Waypoint
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		wp, err := svc.UpdateWaypoint(c.Context(), c.Params("id"), req)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(wp)
	})

	r.Delete("/:id", authMiddleware, func(c *fiber.Ctx) error {
		if err := svc.DeleteWaypoint(c.Context(), c.Params("id")); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	r.Post("/:id/visit", authMiddleware, func(c *fiber.Ctx) error {
		var body struct {
			UserID string `json:"user_id"`
		}
		if err := c.BodyParser(&body); err != nil || body.UserID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "user_id required")
		}
		visited, err := svc.HasVisited(c.Context(), c.Params("id"), body.UserID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !visited {
			return fiber.NewError(fiber.StatusForbidden, "user has not visited waypoint")
		}
		return c.JSON(fiber.Map{"visited": visited})
	})

	r.Post("/:id/reviews", authMiddleware, func(c *fiber.Ctx) error {
		var body struct {
			UserID  string `json:"user_id"`
			Rating  int    `json:"rating"`
			Comment string `json:"comment"`
		}
		if err := c.BodyParser(&body); err != nil || body.UserID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "user_id required")
		}
		if body.Rating < 1 || body.Rating > 5 {
			return fiber.NewError(fiber.StatusBadRequest, "rating must be between 1 and 5")
		}
		review, err := svc.AddReview(c.Context(), c.Params("id"), body.UserID, body.Rating, body.Comment)
		if err != nil {
			return fiber.NewError(fiber.StatusForbidden, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(review)
	})

	r.Get("/:id/reviews", func(c *fiber.Ctx) error {
		reviews, err := svc.Reviews(c.Context(), c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(reviews)
	})

	r.Post("/:id/photos", authMiddleware, func(c *fiber.Ctx) error {
		var body struct {
			UserID   string  `json:"user_id"`
			PhotoURL string  `json:"photo_url"`
			Caption  string  `json:"caption"`
			Lat      float64 `json:"lat"`
			Lng      float64 `json:"lng"`
			TakenAt  string  `json:"taken_at"`
		}
		if err := c.BodyParser(&body); err != nil || body.UserID == "" || body.PhotoURL == "" {
			return fiber.NewError(fiber.StatusBadRequest, "user_id and photo_url required")
		}
		photo := Photo{
			WaypointID: c.Params("id"),
			UserID:     body.UserID,
			PhotoURL:   body.PhotoURL,
			Caption:    body.Caption,
			Lat:        body.Lat,
			Lng:        body.Lng,
			TakenAt:    time.Now(),
		}
		created, err := svc.AddPhoto(c.Context(), photo.WaypointID, photo.UserID, photo.PhotoURL, photo.Caption, photo.Lat, photo.Lng, photo.TakenAt)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(created)
	})

	r.Get("/:id/photos", func(c *fiber.Ctx) error {
		photos, err := svc.Photos(c.Context(), c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(photos)
	})

	r.Get("/search", func(c *fiber.Ctx) error {
		lat, _ := strconv.ParseFloat(c.Query("lat"), 64)
		lng, _ := strconv.ParseFloat(c.Query("lng"), 64)
		radius, _ := strconv.ParseFloat(c.Query("radius_km"), 64)
		if radius == 0 {
			radius = 5
		}
		results, err := svc.Search(c.Context(), lat, lng, radius)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(results)
	})
}
