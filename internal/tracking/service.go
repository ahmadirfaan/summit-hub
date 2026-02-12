package tracking

import (
	"context"
	"encoding/json"
	"time"

	"backend-summithub/internal/shared/geo"
	"backend-summithub/internal/stream"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db  *pgxpool.Pool
	hub *stream.Hub
}

func NewService(db *pgxpool.Pool, hub *stream.Hub) *Service {
	return &Service{db: db, hub: hub}
}

func (s *Service) StartSession(ctx context.Context, input Session) (Session, error) {
	input.ID = uuid.NewString()
	if input.StartedAt.IsZero() {
		input.StartedAt = time.Now()
	}
	if input.Status == "" {
		input.Status = "active"
	}

	row := s.db.QueryRow(ctx, `
		INSERT INTO track_sessions (id, trip_id, user_id, started_at, status)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING started_at, status
	`, input.ID, input.TripID, input.UserID, input.StartedAt, input.Status)
	if err := row.Scan(&input.StartedAt, &input.Status); err != nil {
		return Session{}, err
	}
	return input, nil
}

func (s *Service) AddPoint(ctx context.Context, sessionID string, input TrackPoint) (TrackPoint, error) {
	if input.RecordedAt.IsZero() {
		input.RecordedAt = time.Now()
	}

	var lastLat, lastLng, lastElevation float64
	_ = s.db.QueryRow(ctx, `
		SELECT ST_Y(location::geometry), ST_X(location::geometry), COALESCE(elevation_m, 0)
		FROM track_points
		WHERE session_id=$1
		ORDER BY recorded_at DESC
		LIMIT 1
	`, sessionID).Scan(&lastLat, &lastLng, &lastElevation)

	row := s.db.QueryRow(ctx, `
		INSERT INTO track_points (session_id, location, elevation_m, recorded_at, speed_mps)
		VALUES ($1, ST_SetSRID(ST_MakePoint($2,$3), 4326)::geography, $4, $5, $6)
		RETURNING id, created_at
	`, sessionID, input.Lng, input.Lat, input.ElevationM, input.RecordedAt, input.SpeedMps)
	if err := row.Scan(&input.ID, &input.CreatedAt); err != nil {
		return TrackPoint{}, err
	}
	input.SessionID = sessionID

	if lastLat != 0 || lastLng != 0 {
		deltaM := geo.HaversineKm(lastLat, lastLng, input.Lat, input.Lng) * 1000
		deltaElev := 0.0
		if input.ElevationM > lastElevation {
			deltaElev = input.ElevationM - lastElevation
		}
		_, _ = s.db.Exec(ctx, `
			UPDATE track_sessions
			SET total_distance_m = COALESCE(total_distance_m,0) + $2,
			    total_elevation_gain_m = COALESCE(total_elevation_gain_m,0) + $3
			WHERE id=$1
		`, sessionID, deltaM, deltaElev)
	}

	if s.hub != nil {
		payload, _ := json.Marshal(input)
		s.hub.Broadcast(sessionID, payload)
	}

	return input, nil
}

func (s *Service) Summary(ctx context.Context, sessionID string) (Summary, error) {
	var session Session
	row := s.db.QueryRow(ctx, `
		SELECT id, started_at, ended_at, COALESCE(total_distance_m,0), COALESCE(total_elevation_gain_m,0)
		FROM track_sessions WHERE id=$1
	`, sessionID)
	if err := row.Scan(&session.ID, &session.StartedAt, &session.EndedAt, &session.TotalDistanceM, &session.TotalElevationGainM); err != nil {
		return Summary{}, err
	}

	var pointCount int
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM track_points WHERE session_id=$1`, sessionID).Scan(&pointCount); err != nil {
		return Summary{}, err
	}

	duration := time.Since(session.StartedAt)
	if !session.EndedAt.IsZero() {
		duration = session.EndedAt.Sub(session.StartedAt)
	}
	avgSpeed := 0.0
	if duration.Seconds() > 0 {
		avgSpeed = session.TotalDistanceM / duration.Seconds()
	}

	return Summary{
		SessionID:     session.ID,
		PointCount:    pointCount,
		DistanceM:     session.TotalDistanceM,
		DurationSec:   int64(duration.Seconds()),
		AverageSpeedM: avgSpeed,
	}, nil
}

func (s *Service) Points(ctx context.Context, sessionID string) ([]TrackPoint, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, session_id, ST_Y(location::geometry), ST_X(location::geometry), COALESCE(elevation_m,0), recorded_at, COALESCE(speed_mps,0), created_at
		FROM track_points WHERE session_id=$1
		ORDER BY recorded_at
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []TrackPoint
	for rows.Next() {
		var p TrackPoint
		if err := rows.Scan(&p.ID, &p.SessionID, &p.Lat, &p.Lng, &p.ElevationM, &p.RecordedAt, &p.SpeedMps, &p.CreatedAt); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, nil
}
