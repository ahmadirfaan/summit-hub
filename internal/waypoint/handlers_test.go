package waypoint

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

func TestWaypointHandlers(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	createdAt := time.Now()
	mock.ExpectQuery(`INSERT INTO waypoints`).
		WithArgs(pgxmock.AnyArg(), "WP", "desc", "peak", 106.8, -6.2, 100.0, "user-1", false).
		WillReturnRows(pgxmock.NewRows([]string{"created_at"}).AddRow(createdAt))

	mock.ExpectQuery(`SELECT id, name, description, type, ST_Y\(location::geometry\), ST_X\(location::geometry\),`).
		WithArgs("wp-1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "type", "lat", "lng", "elevation_m", "created_by", "is_verified", "created_at"}).
			AddRow("wp-1", "WP", "desc", "peak", -6.2, 106.8, 100.0, "user-1", false, createdAt))

	app := fiber.New()
	RegisterRoutes(app.Group("/waypoints"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	body, _ := json.Marshal(Waypoint{Name: "WP", Description: "desc", Type: "peak", Lat: -6.2, Lng: 106.8, ElevationM: 100, CreatedBy: "user-1"})
	req := httptest.NewRequest(http.MethodPost, "/waypoints/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("create waypoint status: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/waypoints/wp-1", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("get waypoint status: %v", err)
	}
}

func TestWaypointHandlersBadRequest(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/waypoints"), NewService(nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodPost, "/waypoints/", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}

func TestWaypointHandlersCreateParseError(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/waypoints"), NewService(nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodPost, "/waypoints/", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}

func TestWaypointHandlersUpdateParseError(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/waypoints"), NewService(nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodPut, "/waypoints/wp-1", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}

func TestWaypointHandlersUpdateDelete(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	createdAt := time.Now()
	mock.ExpectQuery(`SELECT id, name, description, type, ST_Y\(location::geometry\), ST_X\(location::geometry\),`).
		WithArgs("wp-1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "type", "lat", "lng", "elevation_m", "created_by", "is_verified", "created_at"}).
			AddRow("wp-1", "WP", "desc", "peak", -6.2, 106.8, 100.0, "user-1", false, createdAt))

	mock.ExpectExec(`UPDATE waypoints`).
		WithArgs("wp-1", "WP2", "desc", "peak", 106.8, -6.2, 100.0, false).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectExec(`DELETE FROM waypoints`).
		WithArgs("wp-1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	app := fiber.New()
	RegisterRoutes(app.Group("/waypoints"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	body, _ := json.Marshal(Waypoint{Name: "WP2"})
	req := httptest.NewRequest(http.MethodPut, "/waypoints/wp-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("update status: %v", err)
	}

	req = httptest.NewRequest(http.MethodDelete, "/waypoints/wp-1", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status: %v", err)
	}
}

func TestWaypointHandlersDeleteError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectExec(`DELETE FROM waypoints`).
		WithArgs("wp-err").
		WillReturnError(errWaypoint)

	app := fiber.New()
	RegisterRoutes(app.Group("/waypoints"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodDelete, "/waypoints/wp-err", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected delete error")
	}
}

func TestWaypointHandlersVisitReview(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs("wp-1", "user-1").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs("wp-1", "user-1").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`INSERT INTO waypoint_reviews`).
		WithArgs(pgxmock.AnyArg(), "wp-1", "user-1", 5, "great").
		WillReturnRows(pgxmock.NewRows([]string{"created_at"}).AddRow(time.Now()))

	mock.ExpectQuery(`SELECT id, waypoint_id, user_id, rating, comment, created_at`).
		WithArgs("wp-1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "waypoint_id", "user_id", "rating", "comment", "created_at"}).
			AddRow("rev-1", "wp-1", "user-1", 5, "great", time.Now()))

	app := fiber.New()
	RegisterRoutes(app.Group("/waypoints"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	visitBody, _ := json.Marshal(map[string]string{"user_id": "user-1"})
	req := httptest.NewRequest(http.MethodPost, "/waypoints/wp-1/visit", bytes.NewReader(visitBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("visit status: %v", err)
	}

	reviewBody, _ := json.Marshal(map[string]interface{}{"user_id": "user-1", "rating": 5, "comment": "great"})
	req = httptest.NewRequest(http.MethodPost, "/waypoints/wp-1/reviews", bytes.NewReader(reviewBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("review status: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/waypoints/wp-1/reviews", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("reviews status: %v", err)
	}
}

func TestWaypointHandlersPhotosSearch(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO waypoint_photos`).
		WithArgs(pgxmock.AnyArg(), "wp-1", "user-1", "url", "cap", 106.8, -6.2, pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"created_at"}).AddRow(time.Now()))

	mock.ExpectQuery(`SELECT id, waypoint_id, user_id, photo_url, caption, ST_Y\(location::geometry\), ST_X\(location::geometry\), taken_at, created_at`).
		WithArgs("wp-1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "waypoint_id", "user_id", "photo_url", "caption", "lat", "lng", "taken_at", "created_at"}).
			AddRow("photo-1", "wp-1", "user-1", "url", "cap", -6.2, 106.8, time.Now(), time.Now()))

	mock.ExpectQuery(`SELECT id, name, description, type, ST_Y\(location::geometry\), ST_X\(location::geometry\),`).
		WithArgs(106.8, -6.2, 5000.0).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "type", "lat", "lng", "elevation_m", "created_by", "is_verified", "created_at"}).
			AddRow("wp-1", "WP", "desc", "peak", -6.2, 106.8, 100.0, "user-1", false, time.Now()))

	app := fiber.New()
	RegisterRoutes(app.Group("/waypoints"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	photoBody, _ := json.Marshal(map[string]interface{}{"user_id": "user-1", "photo_url": "url", "caption": "cap", "lat": -6.2, "lng": 106.8})
	req := httptest.NewRequest(http.MethodPost, "/waypoints/wp-1/photos", bytes.NewReader(photoBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("photo create status: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/waypoints/wp-1/photos", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("photos status: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/waypoints/search?lat=-6.2&lng=106.8", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("search status: %v", err)
	}
}

func TestWaypointHandlersErrors(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO waypoints`).
		WithArgs(pgxmock.AnyArg(), "WP", "", "", 0.0, 0.0, 0.0, "user-1", false).
		WillReturnError(errWaypoint)

	mock.ExpectQuery(`SELECT id, name, description, type, ST_Y\(location::geometry\), ST_X\(location::geometry\),`).
		WithArgs("missing").
		WillReturnError(errWaypoint)

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs("wp-1", "user-1").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	mock.ExpectQuery(`SELECT id, name, description, type, ST_Y\(location::geometry\), ST_X\(location::geometry\),`).
		WithArgs("wp-err").
		WillReturnError(errWaypoint)

	mock.ExpectQuery(`SELECT id, waypoint_id, user_id, rating, comment, created_at`).
		WithArgs("wp-err").
		WillReturnError(errWaypoint)

	mock.ExpectQuery(`INSERT INTO waypoint_photos`).
		WithArgs(pgxmock.AnyArg(), "wp-1", "user-1", "url", "", 0.0, 0.0, pgxmock.AnyArg()).
		WillReturnError(errWaypoint)

	mock.ExpectQuery(`SELECT id, waypoint_id, user_id, photo_url, caption, ST_Y\(location::geometry\), ST_X\(location::geometry\), taken_at, created_at`).
		WithArgs("wp-err").
		WillReturnError(errWaypoint)

	mock.ExpectQuery(`SELECT id, name, description, type, ST_Y\(location::geometry\), ST_X\(location::geometry\),`).
		WithArgs(0.0, 0.0, 5000.0).
		WillReturnError(errWaypoint)

	app := fiber.New()
	RegisterRoutes(app.Group("/waypoints"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	body, _ := json.Marshal(Waypoint{Name: "WP", CreatedBy: "user-1"})
	req := httptest.NewRequest(http.MethodPost, "/waypoints/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected create error")
	}

	req = httptest.NewRequest(http.MethodGet, "/waypoints/missing", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected not found")
	}

	visitBody, _ := json.Marshal(map[string]string{"user_id": "user-1"})
	req = httptest.NewRequest(http.MethodPost, "/waypoints/wp-1/visit", bytes.NewReader(visitBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected forbidden")
	}

	updateBody, _ := json.Marshal(Waypoint{Name: "X"})
	req = httptest.NewRequest(http.MethodPut, "/waypoints/wp-err", bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected update error")
	}

	req = httptest.NewRequest(http.MethodGet, "/waypoints/wp-err/reviews", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected reviews error")
	}

	photoBody, _ := json.Marshal(map[string]string{"user_id": "user-1", "photo_url": "url"})
	req = httptest.NewRequest(http.MethodPost, "/waypoints/wp-1/photos", bytes.NewReader(photoBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected photo error")
	}

	req = httptest.NewRequest(http.MethodGet, "/waypoints/wp-err/photos", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected photos error")
	}

	req = httptest.NewRequest(http.MethodGet, "/waypoints/search", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected search error")
	}
}

func TestWaypointHandlersVisitError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs("wp-err", "user-1").
		WillReturnError(errWaypoint)

	app := fiber.New()
	RegisterRoutes(app.Group("/waypoints"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	visitBody, _ := json.Marshal(map[string]string{"user_id": "user-1"})
	req := httptest.NewRequest(http.MethodPost, "/waypoints/wp-err/visit", bytes.NewReader(visitBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected visit error")
	}
}

func TestWaypointHandlersReviewServiceError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs("wp-1", "user-1").
		WillReturnError(errWaypoint)

	app := fiber.New()
	RegisterRoutes(app.Group("/waypoints"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	reviewBody, _ := json.Marshal(map[string]interface{}{"user_id": "user-1", "rating": 5, "comment": "great"})
	req := httptest.NewRequest(http.MethodPost, "/waypoints/wp-1/reviews", bytes.NewReader(reviewBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected review error")
	}
}

func TestWaypointHandlersSearchDefaultRadius(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, name, description, type, ST_Y\(location::geometry\), ST_X\(location::geometry\),`).
		WithArgs(106.8, -6.2, 5000.0).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "type", "lat", "lng", "elevation_m", "created_by", "is_verified", "created_at"}).
			AddRow("wp-1", "WP", "desc", "peak", -6.2, 106.8, 100.0, "user-1", false, time.Now()))

	app := fiber.New()
	RegisterRoutes(app.Group("/waypoints"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodGet, "/waypoints/search?lat=-6.2&lng=106.8&radius_km=", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("expected search ok")
	}
}

func TestWaypointHandlersReviewBadRequest(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/waypoints"), NewService(nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodPost, "/waypoints/wp-1/reviews", bytes.NewReader([]byte(`{"user_id":"u","rating":6}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}

func TestWaypointHandlersPhotosBadRequest(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/waypoints"), NewService(nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodPost, "/waypoints/wp-1/photos", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}

func TestWaypointHandlersVisitBadRequest(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/waypoints"), NewService(nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodPost, "/waypoints/wp-1/visit", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}

func TestWaypointHandlersReviewMissingUser(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/waypoints"), NewService(nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodPost, "/waypoints/wp-1/reviews", bytes.NewReader([]byte(`{"rating":5}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}
