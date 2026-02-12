package trip

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/pashagolub/pgxmock/v3"
)

func TestTripHandlersCreateGetMember(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	createdAt := time.Now()
	mock.ExpectQuery(`INSERT INTO trips`).
		WithArgs(pgxmock.AnyArg(), "Trip A", "Mt", pgxmock.AnyArg(), pgxmock.AnyArg(), "desc", "user-1").
		WillReturnRows(pgxmock.NewRows([]string{"created_at"}).AddRow(createdAt))

	mock.ExpectQuery(`SELECT id, name, mountain_name, start_date, end_date, description, created_by, created_at`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "mountain_name", "start_date", "end_date", "description", "created_by", "created_at"}).
			AddRow("trip-1", "Trip A", "Mt", time.Now(), time.Now(), "desc", "user-1", createdAt))

	mock.ExpectQuery(`INSERT INTO trip_members`).
		WithArgs("trip-1", "user-2", "member").
		WillReturnRows(pgxmock.NewRows([]string{"joined_at"}).AddRow(time.Now()))

	app := fiber.New()
	RegisterRoutes(app.Group("/trips"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	body, _ := json.Marshal(Trip{Name: "Trip A", Mountain: "Mt", Description: "desc", CreatedBy: "user-1"})
	req := httptest.NewRequest(http.MethodPost, "/trips/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/trips/trip-1", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("get status: %v", err)
	}

	memberBody, _ := json.Marshal(map[string]string{"user_id": "user-2"})
	req = httptest.NewRequest(http.MethodPost, "/trips/trip-1/members", bytes.NewReader(memberBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("member status: %v", err)
	}
}

func TestTripHandlersBadRequest(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/trips"), NewService(nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodPost, "/trips/", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}

func TestTripHandlersUpdateDeleteMembersRoutes(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	app := fiber.New()
	RegisterRoutes(app.Group("/trips"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	start := time.Now()
	end := start.Add(2 * time.Hour)

	mock.ExpectQuery(`SELECT id, name, mountain_name, start_date, end_date, description, created_by, created_at`).
		WithArgs("trip-1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "mountain_name", "start_date", "end_date", "description", "created_by", "created_at"}).
			AddRow("trip-1", "Trip", "Mt", start, end, "desc", "user-1", time.Now()))

	mock.ExpectExec(`UPDATE trips`).
		WithArgs("trip-1", "Trip Updated", "Mt", pgxmock.AnyArg(), pgxmock.AnyArg(), "desc").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	updateBody, _ := json.Marshal(Trip{Name: "Trip Updated"})
	req := httptest.NewRequest(http.MethodPut, "/trips/trip-1", bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("update status: %v", err)
	}

	mock.ExpectExec(`DELETE FROM trips`).WithArgs("trip-1").WillReturnResult(pgxmock.NewResult("DELETE", 1))
	req = httptest.NewRequest(http.MethodDelete, "/trips/trip-1", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status: %v", err)
	}

	mock.ExpectQuery(`SELECT trip_id, user_id, role, joined_at`).
		WithArgs("trip-1").
		WillReturnRows(pgxmock.NewRows([]string{"trip_id", "user_id", "role", "joined_at"}).
			AddRow("trip-1", "user-1", "member", time.Now()))
	req = httptest.NewRequest(http.MethodGet, "/trips/trip-1/members", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("members status: %v", err)
	}

	mock.ExpectQuery(`INSERT INTO gpx_routes`).
		WithArgs(pgxmock.AnyArg(), "trip-1", "Route", "desc", 100.0, 10.0, "LINESTRING(0 0,1 1)", "user-1").
		WillReturnRows(pgxmock.NewRows([]string{"created_at"}).AddRow(time.Now()))

	routeBody, _ := json.Marshal(map[string]interface{}{
		"uploaded_by":            "user-1",
		"route":                  "LINESTRING(0 0,1 1)",
		"name":                   "Route",
		"description":            "desc",
		"total_distance_m":       100.0,
		"total_elevation_gain_m": 10.0,
	})
	req = httptest.NewRequest(http.MethodPost, "/trips/trip-1/routes", bytes.NewReader(routeBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("route create status: %v", err)
	}

	mock.ExpectQuery(`SELECT id, trip_id, name, description, total_distance_m, total_elevation_gain_m, ST_AsText\(route\), uploaded_by, created_at`).
		WithArgs("trip-1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "trip_id", "name", "description", "total_distance_m", "total_elevation_gain_m", "route", "uploaded_by", "created_at"}).
			AddRow("route-1", "trip-1", "Route", "desc", 100.0, 10.0, "LINESTRING(0 0,1 1)", "user-1", time.Now()))
	req = httptest.NewRequest(http.MethodGet, "/trips/trip-1/routes", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("routes status: %v", err)
	}
}

func TestTripHandlersGetNotFound(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, name, mountain_name, start_date, end_date, description, created_by, created_at`).
		WithArgs("missing").
		WillReturnError(errQuery)

	app := fiber.New()
	RegisterRoutes(app.Group("/trips"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodGet, "/trips/missing", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected not found")
	}
}

func TestTripHandlersCreateError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO trips`).
		WithArgs(pgxmock.AnyArg(), "Trip A", "Mt", pgxmock.AnyArg(), pgxmock.AnyArg(), "desc", "user-1").
		WillReturnError(errQuery)

	app := fiber.New()
	RegisterRoutes(app.Group("/trips"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	body, _ := json.Marshal(Trip{Name: "Trip A", Mountain: "Mt", Description: "desc", CreatedBy: "user-1"})
	req := httptest.NewRequest(http.MethodPost, "/trips/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected error")
	}
}

func TestTripHandlersMemberBadRequest(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/trips"), NewService(nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodPost, "/trips/trip-1/members", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}

func TestTripHandlersRouteBadRequest(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/trips"), NewService(nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodPost, "/trips/trip-1/routes", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}

func TestTripHandlersUpdateError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, name, mountain_name, start_date, end_date, description, created_by, created_at`).
		WithArgs("trip-err").
		WillReturnError(errQuery)

	app := fiber.New()
	RegisterRoutes(app.Group("/trips"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	body, _ := json.Marshal(Trip{Name: "Trip"})
	req := httptest.NewRequest(http.MethodPut, "/trips/trip-err", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected update error")
	}
}

func TestTripHandlersDeleteError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectExec(`DELETE FROM trips`).WithArgs("trip-err").WillReturnError(errQuery)

	app := fiber.New()
	RegisterRoutes(app.Group("/trips"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodDelete, "/trips/trip-err", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected delete error")
	}
}

func TestTripHandlersMembersError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT trip_id, user_id, role, joined_at`).
		WithArgs("trip-err").
		WillReturnError(errQuery)

	app := fiber.New()
	RegisterRoutes(app.Group("/trips"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodGet, "/trips/trip-err/members", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected members error")
	}
}

func TestTripHandlersRoutesError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, trip_id, name, description, total_distance_m, total_elevation_gain_m, ST_AsText\(route\), uploaded_by, created_at`).
		WithArgs("trip-err").
		WillReturnError(errQuery)

	app := fiber.New()
	RegisterRoutes(app.Group("/trips"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodGet, "/trips/trip-err/routes", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected routes error")
	}
}
