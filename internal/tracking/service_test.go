package tracking

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v3"
)

func TestStartSessionAddPointSummary(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	svc := NewService(mock, nil)

	mock.ExpectQuery(`INSERT INTO track_sessions`).
		WithArgs(pgxmock.AnyArg(), "trip-1", "user-1", pgxmock.AnyArg(), "active").
		WillReturnRows(pgxmock.NewRows([]string{"started_at", "status"}).AddRow(time.Now(), "active"))

	session, err := svc.StartSession(context.Background(), Session{TripID: "trip-1", UserID: "user-1"})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	mock.ExpectQuery(`SELECT ST_Y\(location::geometry\), ST_X\(location::geometry\), COALESCE\(elevation_m, 0\)`).
		WithArgs(session.ID).
		WillReturnRows(pgxmock.NewRows([]string{"lat", "lng", "elev"}).AddRow(0, 0, 0))

	mock.ExpectQuery(`INSERT INTO track_points`).
		WithArgs(session.ID, 106.8, -6.2, 10.0, pgxmock.AnyArg(), 1.2).
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at"}).AddRow(int64(1), time.Now()))

	point, err := svc.AddPoint(context.Background(), session.ID, TrackPoint{Lat: -6.2, Lng: 106.8, ElevationM: 10, SpeedMps: 1.2})
	if err != nil {
		t.Fatalf("add point: %v", err)
	}
	if point.ID == 0 {
		t.Fatalf("expected point id")
	}

	mock.ExpectQuery(`SELECT id, started_at, ended_at, COALESCE\(total_distance_m,0\), COALESCE\(total_elevation_gain_m,0\)`).
		WithArgs(session.ID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "started_at", "ended_at", "dist", "elev"}).AddRow(session.ID, time.Now().Add(-time.Minute), time.Time{}, 100.0, 10.0))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM track_points`).
		WithArgs(session.ID).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	summary, err := svc.Summary(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary.PointCount != 1 {
		t.Fatalf("unexpected summary")
	}

	mock.ExpectQuery(`SELECT id, session_id, ST_Y\(location::geometry\), ST_X\(location::geometry\), COALESCE\(elevation_m,0\), recorded_at, COALESCE\(speed_mps,0\), created_at`).
		WithArgs(session.ID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "session_id", "lat", "lng", "elevation_m", "recorded_at", "speed_mps", "created_at"}).
			AddRow(int64(1), session.ID, -6.2, 106.8, 10.0, time.Now(), 1.2, time.Now()))

	points, err := svc.Points(context.Background(), session.ID)
	if err != nil || len(points) != 1 {
		t.Fatalf("points: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestAddPointUpdatesTotals(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	svc := NewService(mock, nil)

	mock.ExpectQuery(`SELECT ST_Y\(location::geometry\), ST_X\(location::geometry\), COALESCE\(elevation_m, 0\)`).
		WithArgs("session-1").
		WillReturnRows(pgxmock.NewRows([]string{"lat", "lng", "elev"}).AddRow(-6.2, 106.8, 10.0))

	mock.ExpectQuery(`INSERT INTO track_points`).
		WithArgs("session-1", 106.9, -6.1, 20.0, pgxmock.AnyArg(), 1.2).
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at"}).AddRow(int64(2), time.Now()))

	mock.ExpectExec(`UPDATE track_sessions`).
		WithArgs("session-1", pgxmock.AnyArg(), 10.0).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	_, err = svc.AddPoint(context.Background(), "session-1", TrackPoint{Lat: -6.1, Lng: 106.9, ElevationM: 20, SpeedMps: 1.2})
	if err != nil {
		t.Fatalf("add point: %v", err)
	}
}

func TestStartSessionError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO track_sessions`).
		WithArgs(pgxmock.AnyArg(), "trip-1", "user-1", pgxmock.AnyArg(), "active").
		WillReturnError(errTrack)

	svc := NewService(mock, nil)
	_, err = svc.StartSession(context.Background(), Session{TripID: "trip-1", UserID: "user-1"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestAddPointInsertError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT ST_Y\(location::geometry\), ST_X\(location::geometry\), COALESCE\(elevation_m, 0\)`).
		WithArgs("session-2").
		WillReturnRows(pgxmock.NewRows([]string{"lat", "lng", "elev"}).AddRow(0, 0, 0))

	mock.ExpectQuery(`INSERT INTO track_points`).
		WithArgs("session-2", 106.8, -6.2, 0.0, pgxmock.AnyArg(), 0.0).
		WillReturnError(errTrack)

	svc := NewService(mock, nil)
	_, err = svc.AddPoint(context.Background(), "session-2", TrackPoint{Lat: -6.2, Lng: 106.8})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestSummaryQueryError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, started_at, ended_at, COALESCE\(total_distance_m,0\), COALESCE\(total_elevation_gain_m,0\)`).
		WithArgs("session-3").
		WillReturnError(errTrack)

	svc := NewService(mock, nil)
	_, err = svc.Summary(context.Background(), "session-3")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestPointsQueryError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, session_id, ST_Y\(location::geometry\), ST_X\(location::geometry\), COALESCE\(elevation_m,0\), recorded_at, COALESCE\(speed_mps,0\), created_at`).
		WithArgs("session-4").
		WillReturnError(errTrack)

	svc := NewService(mock, nil)
	_, err = svc.Points(context.Background(), "session-4")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestSummaryCountError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, started_at, ended_at, COALESCE\(total_distance_m,0\), COALESCE\(total_elevation_gain_m,0\)`).
		WithArgs("session-5").
		WillReturnRows(pgxmock.NewRows([]string{"id", "started_at", "ended_at", "dist", "elev"}).AddRow("session-5", time.Now(), time.Time{}, 0.0, 0.0))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM track_points`).
		WithArgs("session-5").
		WillReturnError(errTrack)

	svc := NewService(mock, nil)
	_, err = svc.Summary(context.Background(), "session-5")
	if err == nil {
		t.Fatalf("expected error")
	}
}

var errTrack = errors.New("track error")
