package tracking

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

func TestTrackingHandlers(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO track_sessions`).
		WithArgs(pgxmock.AnyArg(), "trip-1", "user-1", pgxmock.AnyArg(), "active").
		WillReturnRows(pgxmock.NewRows([]string{"started_at", "status"}).AddRow(time.Now(), "active"))

	mock.ExpectQuery(`SELECT ST_Y\(location::geometry\), ST_X\(location::geometry\), COALESCE\(elevation_m, 0\)`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"lat", "lng", "elev"}).AddRow(0, 0, 0))

	mock.ExpectQuery(`INSERT INTO track_points`).
		WithArgs(pgxmock.AnyArg(), 106.8, -6.2, 0.0, pgxmock.AnyArg(), 0.0).
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at"}).AddRow(int64(1), time.Now()))

	app := fiber.New()
	RegisterRoutes(app.Group("/tracking"), NewService(mock, nil), func(c *fiber.Ctx) error { return c.Next() })

	body, _ := json.Marshal(Session{TripID: "trip-1", UserID: "user-1"})
	req := httptest.NewRequest(http.MethodPost, "/tracking/sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("start session status: %v", err)
	}

	pointBody, _ := json.Marshal(TrackPoint{Lat: -6.2, Lng: 106.8})
	req = httptest.NewRequest(http.MethodPost, "/tracking/sessions/session-1/points", bytes.NewReader(pointBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("add point status: %v", err)
	}
}

func TestTrackingHandlersBadRequest(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/tracking"), NewService(nil, nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodPost, "/tracking/sessions", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}

func TestTrackingHandlersSessionParseError(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/tracking"), NewService(nil, nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodPost, "/tracking/sessions", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}

func TestTrackingHandlersSummaryPoints(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, started_at, ended_at, COALESCE\(total_distance_m,0\), COALESCE\(total_elevation_gain_m,0\)`).
		WithArgs("session-1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "started_at", "ended_at", "dist", "elev"}).AddRow("session-1", time.Now(), time.Time{}, 100.0, 10.0))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM track_points`).
		WithArgs("session-1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))

	mock.ExpectQuery(`SELECT id, session_id, ST_Y\(location::geometry\), ST_X\(location::geometry\), COALESCE\(elevation_m,0\), recorded_at, COALESCE\(speed_mps,0\), created_at`).
		WithArgs("session-1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "session_id", "lat", "lng", "elevation_m", "recorded_at", "speed_mps", "created_at"}).
			AddRow(int64(1), "session-1", -6.2, 106.8, 10.0, time.Now(), 1.2, time.Now()))

	app := fiber.New()
	RegisterRoutes(app.Group("/tracking"), NewService(mock, nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodGet, "/tracking/sessions/session-1/summary", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("summary status: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/tracking/sessions/session-1/points", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("points status: %v", err)
	}
}

func TestTrackingHandlersPointBadRequest(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/tracking"), NewService(nil, nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodPost, "/tracking/sessions/session-1/points", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}

func TestTrackingHandlersSummaryError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, started_at, ended_at, COALESCE\(total_distance_m,0\), COALESCE\(total_elevation_gain_m,0\)`).
		WithArgs("session-err").
		WillReturnError(errTrack)

	app := fiber.New()
	RegisterRoutes(app.Group("/tracking"), NewService(mock, nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodGet, "/tracking/sessions/session-err/summary", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected error")
	}
}

func TestTrackingHandlersPointsError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, session_id, ST_Y\(location::geometry\), ST_X\(location::geometry\), COALESCE\(elevation_m,0\), recorded_at, COALESCE\(speed_mps,0\), created_at`).
		WithArgs("session-err").
		WillReturnError(errTrack)

	app := fiber.New()
	RegisterRoutes(app.Group("/tracking"), NewService(mock, nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodGet, "/tracking/sessions/session-err/points", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected error")
	}
}

func TestTrackingHandlersStartSessionError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO track_sessions`).
		WithArgs(pgxmock.AnyArg(), "trip-1", "user-1", pgxmock.AnyArg(), "active").
		WillReturnError(errTrack)

	app := fiber.New()
	RegisterRoutes(app.Group("/tracking"), NewService(mock, nil), func(c *fiber.Ctx) error { return c.Next() })

	body, _ := json.Marshal(Session{TripID: "trip-1", UserID: "user-1"})
	req := httptest.NewRequest(http.MethodPost, "/tracking/sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected error")
	}
}

func TestTrackingHandlersAddPointError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT ST_Y\(location::geometry\), ST_X\(location::geometry\), COALESCE\(elevation_m, 0\)`).
		WithArgs("session-err").
		WillReturnRows(pgxmock.NewRows([]string{"lat", "lng", "elev"}).AddRow(0, 0, 0))

	mock.ExpectQuery(`INSERT INTO track_points`).
		WithArgs("session-err", 106.8, -6.2, 0.0, pgxmock.AnyArg(), 0.0).
		WillReturnError(errTrack)

	app := fiber.New()
	RegisterRoutes(app.Group("/tracking"), NewService(mock, nil), func(c *fiber.Ctx) error { return c.Next() })

	pointBody, _ := json.Marshal(TrackPoint{Lat: -6.2, Lng: 106.8})
	req := httptest.NewRequest(http.MethodPost, "/tracking/sessions/session-err/points", bytes.NewReader(pointBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected error")
	}
}
