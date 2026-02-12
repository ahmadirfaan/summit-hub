package waypoint

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v3"
)

func TestWaypointCRUD(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	createdAt := time.Now()
	mock.ExpectQuery(`INSERT INTO waypoints`).
		WithArgs(pgxmock.AnyArg(), "WP", "desc", "peak", 106.8, -6.2, 100.0, "user-1", false).
		WillReturnRows(pgxmock.NewRows([]string{"created_at"}).AddRow(createdAt))

	svc := NewService(mock)
	wp, err := svc.CreateWaypoint(context.Background(), Waypoint{
		Name:        "WP",
		Description: "desc",
		Type:        "peak",
		Lat:         -6.2,
		Lng:         106.8,
		ElevationM:  100,
		CreatedBy:   "user-1",
	})
	if err != nil {
		t.Fatalf("create waypoint: %v", err)
	}

	mock.ExpectQuery(`SELECT id, name, description, type, ST_Y\(location::geometry\), ST_X\(location::geometry\),`).
		WithArgs(wp.ID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "type", "lat", "lng", "elevation_m", "created_by", "is_verified", "created_at"}).
			AddRow(wp.ID, wp.Name, wp.Description, wp.Type, wp.Lat, wp.Lng, wp.ElevationM, wp.CreatedBy, wp.IsVerified, wp.CreatedAt))

	loaded, err := svc.GetWaypoint(context.Background(), wp.ID)
	if err != nil {
		t.Fatalf("get waypoint: %v", err)
	}
	if loaded.ID != wp.ID {
		t.Fatalf("unexpected waypoint")
	}

	mock.ExpectQuery(`SELECT id, name, description, type, ST_Y\(location::geometry\), ST_X\(location::geometry\),`).
		WithArgs(wp.ID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "type", "lat", "lng", "elevation_m", "created_by", "is_verified", "created_at"}).
			AddRow(wp.ID, wp.Name, wp.Description, wp.Type, wp.Lat, wp.Lng, wp.ElevationM, wp.CreatedBy, wp.IsVerified, wp.CreatedAt))

	mock.ExpectExec(`UPDATE waypoints`).
		WithArgs(wp.ID, "WP2", wp.Description, wp.Type, wp.Lng, wp.Lat, wp.ElevationM, true).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	updated, err := svc.UpdateWaypoint(context.Background(), wp.ID, Waypoint{Name: "WP2", IsVerified: true})
	if err != nil {
		t.Fatalf("update waypoint: %v", err)
	}
	if updated.Name != "WP2" {
		t.Fatalf("expected updated name")
	}

	mock.ExpectExec(`DELETE FROM waypoints`).WithArgs(wp.ID).WillReturnResult(pgxmock.NewResult("DELETE", 1))
	if err := svc.DeleteWaypoint(context.Background(), wp.ID); err != nil {
		t.Fatalf("delete waypoint: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestWaypointReviewsPhotosSearch(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	svc := NewService(mock)

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs("wp-1", "user-1").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`INSERT INTO waypoint_reviews`).
		WithArgs(pgxmock.AnyArg(), "wp-1", "user-1", 5, "nice").
		WillReturnRows(pgxmock.NewRows([]string{"created_at"}).AddRow(time.Now()))

	_, err = svc.AddReview(context.Background(), "wp-1", "user-1", 5, "nice")
	if err != nil {
		t.Fatalf("add review: %v", err)
	}

	mock.ExpectQuery(`SELECT id, waypoint_id, user_id, rating, comment, created_at`).
		WithArgs("wp-1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "waypoint_id", "user_id", "rating", "comment", "created_at"}).
			AddRow("rev-1", "wp-1", "user-1", 5, "nice", time.Now()))

	reviews, err := svc.Reviews(context.Background(), "wp-1")
	if err != nil || len(reviews) != 1 {
		t.Fatalf("reviews: %v", err)
	}

	mock.ExpectQuery(`INSERT INTO waypoint_photos`).
		WithArgs(pgxmock.AnyArg(), "wp-1", "user-1", "url", "cap", 106.8, -6.2, pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"created_at"}).AddRow(time.Now()))

	_, err = svc.AddPhoto(context.Background(), "wp-1", "user-1", "url", "cap", -6.2, 106.8, time.Now())
	if err != nil {
		t.Fatalf("add photo: %v", err)
	}

	mock.ExpectQuery(`SELECT id, waypoint_id, user_id, photo_url, caption, ST_Y\(location::geometry\), ST_X\(location::geometry\), taken_at, created_at`).
		WithArgs("wp-1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "waypoint_id", "user_id", "photo_url", "caption", "lat", "lng", "taken_at", "created_at"}).
			AddRow("photo-1", "wp-1", "user-1", "url", "cap", -6.2, 106.8, time.Now(), time.Now()))

	photos, err := svc.Photos(context.Background(), "wp-1")
	if err != nil || len(photos) != 1 {
		t.Fatalf("photos: %v", err)
	}

	mock.ExpectQuery(`SELECT id, name, description, type, ST_Y\(location::geometry\), ST_X\(location::geometry\),`).
		WithArgs(106.8, -6.2, 5000.0).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "type", "lat", "lng", "elevation_m", "created_by", "is_verified", "created_at"}).
			AddRow("wp-1", "WP", "desc", "peak", -6.2, 106.8, 100.0, "user-1", false, time.Now()))

	results, err := svc.Search(context.Background(), -6.2, 106.8, 5)
	if err != nil || len(results) != 1 {
		t.Fatalf("search: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestAddReviewNotVisited(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs("wp-2", "user-2").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	svc := NewService(mock)
	_, err = svc.AddReview(context.Background(), "wp-2", "user-2", 4, "ok")
	if err == nil {
		t.Fatalf("expected error for not visited")
	}
}

func TestAddReviewHasVisitedError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs("wp-err", "user-err").
		WillReturnError(errWaypoint)

	svc := NewService(mock)
	_, err = svc.AddReview(context.Background(), "wp-err", "user-err", 5, "great")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestAddReviewInsertError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs("wp-1", "user-1").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`INSERT INTO waypoint_reviews`).
		WithArgs(pgxmock.AnyArg(), "wp-1", "user-1", 5, "great").
		WillReturnError(errWaypoint)

	svc := NewService(mock)
	_, err = svc.AddReview(context.Background(), "wp-1", "user-1", 5, "great")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestHasVisitedError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).WithArgs("wp-err", "user-err").
		WillReturnError(errWaypoint)

	svc := NewService(mock)
	_, err = svc.HasVisited(context.Background(), "wp-err", "user-err")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestReviewsQueryError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, waypoint_id, user_id, rating, comment, created_at`).
		WithArgs("wp-err").
		WillReturnError(errWaypoint)

	svc := NewService(mock)
	_, err = svc.Reviews(context.Background(), "wp-err")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestPhotosQueryError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, waypoint_id, user_id, photo_url, caption, ST_Y\(location::geometry\), ST_X\(location::geometry\), taken_at, created_at`).
		WithArgs("wp-err").
		WillReturnError(errWaypoint)

	svc := NewService(mock)
	_, err = svc.Photos(context.Background(), "wp-err")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestPhotosScanError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, waypoint_id, user_id, photo_url, caption, ST_Y\(location::geometry\), ST_X\(location::geometry\), taken_at, created_at`).
		WithArgs("wp-err").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("photo-1"))

	svc := NewService(mock)
	_, err = svc.Photos(context.Background(), "wp-err")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestSearchQueryError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, name, description, type, ST_Y\(location::geometry\), ST_X\(location::geometry\),`).
		WithArgs(106.8, -6.2, 5000.0).
		WillReturnError(errWaypoint)

	svc := NewService(mock)
	_, err = svc.Search(context.Background(), -6.2, 106.8, 5)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestUpdateWaypointGetError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, name, description, type, ST_Y\(location::geometry\), ST_X\(location::geometry\),`).
		WithArgs("wp-err").
		WillReturnError(errWaypoint)

	svc := NewService(mock)
	_, err = svc.UpdateWaypoint(context.Background(), "wp-err", Waypoint{Name: "X"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDeleteWaypointError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectExec(`DELETE FROM waypoints`).WithArgs("wp-err").WillReturnError(errWaypoint)

	svc := NewService(mock)
	if err := svc.DeleteWaypoint(context.Background(), "wp-err"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCreateWaypointError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO waypoints`).
		WithArgs(pgxmock.AnyArg(), "WP", "desc", "peak", 106.8, -6.2, 0.0, "user-1", false).
		WillReturnError(errWaypoint)

	svc := NewService(mock)
	_, err = svc.CreateWaypoint(context.Background(), Waypoint{Name: "WP", Description: "desc", Type: "peak", Lat: -6.2, Lng: 106.8, CreatedBy: "user-1"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestUpdateWaypointExecError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, name, description, type, ST_Y\(location::geometry\), ST_X\(location::geometry\),`).
		WithArgs("wp-err").
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "type", "lat", "lng", "elevation_m", "created_by", "is_verified", "created_at"}).
			AddRow("wp-err", "WP", "desc", "peak", -6.2, 106.8, 100.0, "user-1", false, time.Now()))

	mock.ExpectExec(`UPDATE waypoints`).
		WithArgs("wp-err", "WP", "desc", "peak", 106.8, -6.2, 100.0, false).
		WillReturnError(errWaypoint)

	svc := NewService(mock)
	_, err = svc.UpdateWaypoint(context.Background(), "wp-err", Waypoint{})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestUpdateWaypointPatchFields(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	createdAt := time.Now()
	mock.ExpectQuery(`SELECT id, name, description, type, ST_Y\(location::geometry\), ST_X\(location::geometry\),`).
		WithArgs("wp-2").
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "description", "type", "lat", "lng", "elevation_m", "created_by", "is_verified", "created_at"}).
			AddRow("wp-2", "WP", "desc", "peak", -6.2, 106.8, 100.0, "user-1", false, createdAt))

	mock.ExpectExec(`UPDATE waypoints`).
		WithArgs("wp-2", "WP2", "desc2", "lake", 107.0, -6.0, 200.0, true).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	svc := NewService(mock)
	updated, err := svc.UpdateWaypoint(context.Background(), "wp-2", Waypoint{
		Name:        "WP2",
		Description: "desc2",
		Type:        "lake",
		Lat:         -6.0,
		Lng:         107.0,
		ElevationM:  200,
		IsVerified:  true,
	})
	if err != nil {
		t.Fatalf("update waypoint: %v", err)
	}
	if updated.Name != "WP2" || updated.Type != "lake" {
		t.Fatalf("expected updated fields")
	}
}

func TestAddPhotoInsertError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO waypoint_photos`).
		WithArgs(pgxmock.AnyArg(), "wp-1", "user-1", "url", "cap", 106.8, -6.2, pgxmock.AnyArg()).
		WillReturnError(errWaypoint)

	svc := NewService(mock)
	_, err = svc.AddPhoto(context.Background(), "wp-1", "user-1", "url", "cap", -6.2, 106.8, time.Now())
	if err == nil {
		t.Fatalf("expected error")
	}
}

var errWaypoint = errors.New("waypoint error")
