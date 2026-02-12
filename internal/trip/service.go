package trip

import (
	"context"
	"time"

	"backend-summithub/internal/db"

	"github.com/google/uuid"
)

type Service struct {
	db db.Querier
}

func NewService(db db.Querier) *Service {
	return &Service{db: db}
}

func (s *Service) CreateTrip(ctx context.Context, input Trip) (Trip, error) {
	input.ID = uuid.NewString()
	row := s.db.QueryRow(ctx, `
		INSERT INTO trips (id, name, mountain_name, start_date, end_date, description, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING created_at
	`, input.ID, input.Name, input.Mountain, timePtr(input.StartDate), timePtr(input.EndDate), input.Description, input.CreatedBy)
	if err := row.Scan(&input.CreatedAt); err != nil {
		return Trip{}, err
	}
	return input, nil
}

func (s *Service) UpdateTrip(ctx context.Context, id string, patch Trip) (Trip, error) {
	trip, err := s.GetTrip(ctx, id)
	if err != nil {
		return Trip{}, err
	}
	if patch.Name != "" {
		trip.Name = patch.Name
	}
	if patch.Mountain != "" {
		trip.Mountain = patch.Mountain
	}
	if !patch.StartDate.IsZero() {
		trip.StartDate = patch.StartDate
	}
	if !patch.EndDate.IsZero() {
		trip.EndDate = patch.EndDate
	}
	if patch.Description != "" {
		trip.Description = patch.Description
	}

	_, err = s.db.Exec(ctx, `
		UPDATE trips
		SET name=$2, mountain_name=$3, start_date=$4, end_date=$5, description=$6
		WHERE id=$1
	`, trip.ID, trip.Name, trip.Mountain, timePtr(trip.StartDate), timePtr(trip.EndDate), trip.Description)
	if err != nil {
		return Trip{}, err
	}
	return trip, nil
}

func (s *Service) GetTrip(ctx context.Context, id string) (Trip, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id, name, mountain_name, start_date, end_date, description, created_by, created_at
		FROM trips WHERE id=$1
	`, id)
	var trip Trip
	if err := row.Scan(&trip.ID, &trip.Name, &trip.Mountain, &trip.StartDate, &trip.EndDate, &trip.Description, &trip.CreatedBy, &trip.CreatedAt); err != nil {
		return Trip{}, err
	}
	return trip, nil
}

func (s *Service) DeleteTrip(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM trips WHERE id=$1`, id)
	return err
}

func (s *Service) AddMember(ctx context.Context, tripID, userID, role string) (TripMember, error) {
	if role == "" {
		role = "member"
	}
	row := s.db.QueryRow(ctx, `
		INSERT INTO trip_members (trip_id, user_id, role)
		VALUES ($1,$2,$3)
		ON CONFLICT (trip_id, user_id) DO UPDATE SET role=EXCLUDED.role
		RETURNING joined_at
	`, tripID, userID, role)
	member := TripMember{TripID: tripID, UserID: userID, Role: role}
	if err := row.Scan(&member.JoinedAt); err != nil {
		return TripMember{}, err
	}
	return member, nil
}

func (s *Service) Members(ctx context.Context, tripID string) ([]TripMember, error) {
	rows, err := s.db.Query(ctx, `
		SELECT trip_id, user_id, role, joined_at
		FROM trip_members WHERE trip_id=$1
		ORDER BY joined_at
	`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []TripMember
	for rows.Next() {
		var m TripMember
		if err := rows.Scan(&m.TripID, &m.UserID, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

func (s *Service) AddRoute(ctx context.Context, route GPXRoute) (GPXRoute, error) {
	if route.ID == "" {
		route.ID = uuid.NewString()
	}
	row := s.db.QueryRow(ctx, `
		INSERT INTO gpx_routes (id, trip_id, name, description, total_distance_m, total_elevation_gain_m, route, uploaded_by)
		VALUES ($1,$2,$3,$4,$5,$6, ST_GeogFromText($7), $8)
		RETURNING created_at
	`, route.ID, route.TripID, route.Name, route.Description, route.TotalDistanceM, route.TotalElevationGainM, route.RouteWKT, route.UploadedBy)
	if err := row.Scan(&route.CreatedAt); err != nil {
		return GPXRoute{}, err
	}
	return route, nil
}

func (s *Service) Routes(ctx context.Context, tripID string) ([]GPXRoute, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, trip_id, name, description, total_distance_m, total_elevation_gain_m, ST_AsText(route), uploaded_by, created_at
		FROM gpx_routes WHERE trip_id=$1
		ORDER BY created_at DESC
	`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []GPXRoute
	for rows.Next() {
		var r GPXRoute
		if err := rows.Scan(&r.ID, &r.TripID, &r.Name, &r.Description, &r.TotalDistanceM, &r.TotalElevationGainM, &r.RouteWKT, &r.UploadedBy, &r.CreatedAt); err != nil {
			return nil, err
		}
		routes = append(routes, r)
	}
	return routes, nil
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
