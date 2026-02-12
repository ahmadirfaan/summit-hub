package trip

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(r fiber.Router, svc *Service, authMiddleware fiber.Handler) {
	r.Post("/", authMiddleware, func(c *fiber.Ctx) error {
		var req Trip
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if req.Name == "" || req.CreatedBy == "" {
			return fiber.NewError(fiber.StatusBadRequest, "name and created_by required")
		}
		trip, err := svc.CreateTrip(c.Context(), req)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(trip)
	})

	r.Get("/:id", func(c *fiber.Ctx) error {
		trip, err := svc.GetTrip(c.Context(), c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "trip not found")
		}
		return c.JSON(trip)
	})

	r.Put("/:id", authMiddleware, func(c *fiber.Ctx) error {
		var req Trip
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		trip, err := svc.UpdateTrip(c.Context(), c.Params("id"), req)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(trip)
	})

	r.Delete("/:id", authMiddleware, func(c *fiber.Ctx) error {
		if err := svc.DeleteTrip(c.Context(), c.Params("id")); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	r.Post("/:id/members", authMiddleware, func(c *fiber.Ctx) error {
		var body struct {
			UserID string `json:"user_id"`
			Role   string `json:"role"`
		}
		if err := c.BodyParser(&body); err != nil || body.UserID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "user_id required")
		}
		member, err := svc.AddMember(c.Context(), c.Params("id"), body.UserID, body.Role)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(member)
	})

	r.Get("/:id/members", func(c *fiber.Ctx) error {
		members, err := svc.Members(c.Context(), c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(members)
	})

	r.Post("/:id/routes", authMiddleware, func(c *fiber.Ctx) error {
		var body struct {
			UploadedBy          string  `json:"uploaded_by"`
			RouteWKT            string  `json:"route"`
			Name                string  `json:"name"`
			Description         string  `json:"description"`
			TotalDistanceM      float64 `json:"total_distance_m"`
			TotalElevationGainM float64 `json:"total_elevation_gain_m"`
		}
		if err := c.BodyParser(&body); err != nil || body.UploadedBy == "" || body.RouteWKT == "" {
			return fiber.NewError(fiber.StatusBadRequest, "uploaded_by and route required")
		}
		route, err := svc.AddRoute(c.Context(), GPXRoute{
			TripID:              c.Params("id"),
			UploadedBy:          body.UploadedBy,
			RouteWKT:            body.RouteWKT,
			Name:                body.Name,
			Description:         body.Description,
			TotalDistanceM:      body.TotalDistanceM,
			TotalElevationGainM: body.TotalElevationGainM,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(route)
	})

	r.Get("/:id/routes", func(c *fiber.Ctx) error {
		routes, err := svc.Routes(c.Context(), c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(routes)
	})
}
