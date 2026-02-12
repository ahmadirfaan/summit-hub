package trip

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v3"
)

func TestCreateAndGetTrip(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	createdAt := time.Now()

	mock.ExpectQuery(`INSERT INTO trips`).
		WithArgs(pgxmock.AnyArg(), "Trip A", "Mountain", pgxmock.AnyArg(), pgxmock.AnyArg(), "desc", "user-1").
		WillReturnRows(pgxmock.NewRows([]string{"created_at"}).AddRow(createdAt))

	svc := NewService(mock)
	trip, err := svc.CreateTrip(context.Background(), Trip{
		Name:        "Trip A",
		Mountain:    "Mountain",
		StartDate:   time.Now(),
		EndDate:     time.Now().Add(24 * time.Hour),
		Description: "desc",
		CreatedBy:   "user-1",
	})
	if err != nil {
		t.Fatalf("create trip: %v", err)
	}

	mock.ExpectQuery(`SELECT id, name, mountain_name, start_date, end_date, description, created_by, created_at`).
		WithArgs(trip.ID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "mountain_name", "start_date", "end_date", "description", "created_by", "created_at"}).
			AddRow(trip.ID, trip.Name, trip.Mountain, trip.StartDate, trip.EndDate, trip.Description, trip.CreatedBy, trip.CreatedAt))

	loaded, err := svc.GetTrip(context.Background(), trip.ID)
	if err != nil {
		t.Fatalf("get trip: %v", err)
	}
	if loaded.ID != trip.ID || loaded.Name != trip.Name {
		t.Fatalf("unexpected trip loaded")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpdateDeleteMembersRoutes(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	svc := NewService(mock)

	mock.ExpectQuery(`SELECT id, name, mountain_name, start_date, end_date, description, created_by, created_at`).
		WithArgs("trip-1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "mountain_name", "start_date", "end_date", "description", "created_by", "created_at"}).
			AddRow("trip-1", "Trip", "Mt", time.Now(), time.Now(), "desc", "user-1", time.Now()))

	mock.ExpectExec(`UPDATE trips`).
		WithArgs("trip-1", "Trip2", "Mt", pgxmock.AnyArg(), pgxmock.AnyArg(), "desc").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	updated, err := svc.UpdateTrip(context.Background(), "trip-1", Trip{Name: "Trip2"})
	if err != nil {
		t.Fatalf("update trip: %v", err)
	}
	if updated.Name != "Trip2" {
		t.Fatalf("unexpected update")
	}

	mock.ExpectExec(`DELETE FROM trips`).WithArgs("trip-1").WillReturnResult(pgxmock.NewResult("DELETE", 1))
	if err := svc.DeleteTrip(context.Background(), "trip-1"); err != nil {
		t.Fatalf("delete trip: %v", err)
	}

	mock.ExpectQuery(`INSERT INTO trip_members`).
		WithArgs("trip-1", "user-2", "member").
		WillReturnRows(pgxmock.NewRows([]string{"joined_at"}).AddRow(time.Now()))
	member, err := svc.AddMember(context.Background(), "trip-1", "user-2", "")
	if err != nil || member.UserID != "user-2" {
		t.Fatalf("add member: %v", err)
	}

	mock.ExpectQuery(`SELECT trip_id, user_id, role, joined_at`).
		WithArgs("trip-1").
		WillReturnRows(pgxmock.NewRows([]string{"trip_id", "user_id", "role", "joined_at"}).
			AddRow("trip-1", "user-2", "member", time.Now()))
	members, err := svc.Members(context.Background(), "trip-1")
	if err != nil || len(members) != 1 {
		t.Fatalf("members: %v", err)
	}

	mock.ExpectQuery(`INSERT INTO gpx_routes`).
		WithArgs(pgxmock.AnyArg(), "trip-1", "Route", "desc", 100.0, 10.0, "LINESTRING(0 0,1 1)", "user-1").
		WillReturnRows(pgxmock.NewRows([]string{"created_at"}).AddRow(time.Now()))

	_, err = svc.AddRoute(context.Background(), GPXRoute{
		TripID:              "trip-1",
		Name:                "Route",
		Description:         "desc",
		TotalDistanceM:      100,
		TotalElevationGainM: 10,
		RouteWKT:            "LINESTRING(0 0,1 1)",
		UploadedBy:          "user-1",
	})
	if err != nil {
		t.Fatalf("add route: %v", err)
	}

	mock.ExpectQuery(`SELECT id, trip_id, name, description, total_distance_m, total_elevation_gain_m, ST_AsText\(route\), uploaded_by, created_at`).
		WithArgs("trip-1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "trip_id", "name", "description", "total_distance_m", "total_elevation_gain_m", "route", "uploaded_by", "created_at"}).
			AddRow("route-1", "trip-1", "Route", "desc", 100.0, 10.0, "LINESTRING(0 0,1 1)", "user-1", time.Now()))

	routes, err := svc.Routes(context.Background(), "trip-1")
	if err != nil || len(routes) != 1 {
		t.Fatalf("routes: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpdateTripGetError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, name, mountain_name, start_date, end_date, description, created_by, created_at`).
		WithArgs("trip-404").
		WillReturnError(errQuery)

	svc := NewService(mock)
	_, err = svc.UpdateTrip(context.Background(), "trip-404", Trip{Name: "X"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestUpdateTripExecError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	start := time.Now()
	end := start.Add(2 * time.Hour)

	mock.ExpectQuery(`SELECT id, name, mountain_name, start_date, end_date, description, created_by, created_at`).
		WithArgs("trip-err").
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "mountain_name", "start_date", "end_date", "description", "created_by", "created_at"}).
			AddRow("trip-err", "Trip", "Mt", start, end, "desc", "user-1", time.Now()))

	mock.ExpectExec(`UPDATE trips`).
		WithArgs("trip-err", "Trip", "Mt", pgxmock.AnyArg(), pgxmock.AnyArg(), "desc").
		WillReturnError(errQuery)

	svc := NewService(mock)
	_, err = svc.UpdateTrip(context.Background(), "trip-err", Trip{})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestUpdateTripPatchFields(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	start := time.Now()
	end := start.Add(2 * time.Hour)
	newStart := start.Add(24 * time.Hour)
	newEnd := end.Add(24 * time.Hour)

	mock.ExpectQuery(`SELECT id, name, mountain_name, start_date, end_date, description, created_by, created_at`).
		WithArgs("trip-2").
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "mountain_name", "start_date", "end_date", "description", "created_by", "created_at"}).
			AddRow("trip-2", "Trip", "Mt", start, end, "desc", "user-1", time.Now()))

	mock.ExpectExec(`UPDATE trips`).
		WithArgs("trip-2", "Trip2", "Mt2", pgxmock.AnyArg(), pgxmock.AnyArg(), "desc2").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	svc := NewService(mock)
	updated, err := svc.UpdateTrip(context.Background(), "trip-2", Trip{
		Name:        "Trip2",
		Mountain:    "Mt2",
		StartDate:   newStart,
		EndDate:     newEnd,
		Description: "desc2",
	})
	if err != nil {
		t.Fatalf("update trip: %v", err)
	}
	if updated.Name != "Trip2" || updated.Mountain != "Mt2" {
		t.Fatalf("expected updated fields")
	}
}

func TestDeleteTripError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectExec(`DELETE FROM trips`).WithArgs("trip-1").WillReturnError(errQuery)

	svc := NewService(mock)
	if err := svc.DeleteTrip(context.Background(), "trip-1"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMembersQueryError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT trip_id, user_id, role, joined_at`).
		WithArgs("trip-err").
		WillReturnError(errQuery)

	svc := NewService(mock)
	_, err = svc.Members(context.Background(), "trip-err")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRoutesQueryError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, trip_id, name, description, total_distance_m, total_elevation_gain_m, ST_AsText\(route\), uploaded_by, created_at`).
		WithArgs("trip-err").
		WillReturnError(errQuery)

	svc := NewService(mock)
	_, err = svc.Routes(context.Background(), "trip-err")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestCreateTripError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO trips`).
		WithArgs(pgxmock.AnyArg(), "Trip", "", pgxmock.AnyArg(), pgxmock.AnyArg(), "", "user-1").
		WillReturnError(errQuery)

	svc := NewService(mock)
	_, err = svc.CreateTrip(context.Background(), Trip{Name: "Trip", CreatedBy: "user-1"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestAddRouteError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO gpx_routes`).
		WithArgs(pgxmock.AnyArg(), "trip-1", "", "", 0.0, 0.0, "LINESTRING(0 0,1 1)", "user-1").
		WillReturnError(errQuery)

	svc := NewService(mock)
	_, err = svc.AddRoute(context.Background(), GPXRoute{TripID: "trip-1", RouteWKT: "LINESTRING(0 0,1 1)", UploadedBy: "user-1"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestAddMemberError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO trip_members`).
		WithArgs("trip-1", "user-2", "member").
		WillReturnError(errQuery)

	svc := NewService(mock)
	_, err = svc.AddMember(context.Background(), "trip-1", "user-2", "")
	if err == nil {
		t.Fatalf("expected error")
	}
}

var errQuery = errors.New("query error")
