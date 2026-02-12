package tracking

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(r fiber.Router, svc *Service, authMiddleware fiber.Handler) {
	r.Post("/sessions", authMiddleware, func(c *fiber.Ctx) error {
		var req Session
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if req.TripID == "" || req.UserID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "trip_id and user_id required")
		}
		session, err := svc.StartSession(c.Context(), req)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(session)
	})

	r.Post("/sessions/:id/points", authMiddleware, func(c *fiber.Ctx) error {
		var req TrackPoint
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		point, err := svc.AddPoint(c.Context(), c.Params("id"), req)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(point)
	})

	r.Get("/sessions/:id/summary", func(c *fiber.Ctx) error {
		summary, err := svc.Summary(c.Context(), c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(summary)
	})

	r.Get("/sessions/:id/points", func(c *fiber.Ctx) error {
		points, err := svc.Points(c.Context(), c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(points)
	})
}
