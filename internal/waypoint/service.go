package waypoint

import (
	"context"
	"errors"
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

func (s *Service) CreateWaypoint(ctx context.Context, input Waypoint) (Waypoint, error) {
	input.ID = uuid.NewString()
	row := s.db.QueryRow(ctx, `
		INSERT INTO waypoints (id, name, description, type, location, elevation_m, created_by, is_verified)
		VALUES ($1,$2,$3,$4, ST_SetSRID(ST_MakePoint($5,$6), 4326)::geography, $7, $8, $9)
		RETURNING created_at
	`, input.ID, input.Name, input.Description, input.Type, input.Lng, input.Lat, input.ElevationM, input.CreatedBy, input.IsVerified)
	if err := row.Scan(&input.CreatedAt); err != nil {
		return Waypoint{}, err
	}
	return input, nil
}

func (s *Service) UpdateWaypoint(ctx context.Context, id string, patch Waypoint) (Waypoint, error) {
	wp, err := s.GetWaypoint(ctx, id)
	if err != nil {
		return Waypoint{}, err
	}
	if patch.Name != "" {
		wp.Name = patch.Name
	}
	if patch.Description != "" {
		wp.Description = patch.Description
	}
	if patch.Type != "" {
		wp.Type = patch.Type
	}
	if patch.Lat != 0 {
		wp.Lat = patch.Lat
	}
	if patch.Lng != 0 {
		wp.Lng = patch.Lng
	}
	if patch.ElevationM != 0 {
		wp.ElevationM = patch.ElevationM
	}
	if patch.IsVerified {
		wp.IsVerified = true
	}

	_, err = s.db.Exec(ctx, `
		UPDATE waypoints
		SET name=$2, description=$3, type=$4,
		    location=ST_SetSRID(ST_MakePoint($5,$6), 4326)::geography,
		    elevation_m=$7, is_verified=$8
		WHERE id=$1
	`, wp.ID, wp.Name, wp.Description, wp.Type, wp.Lng, wp.Lat, wp.ElevationM, wp.IsVerified)
	if err != nil {
		return Waypoint{}, err
	}
	return wp, nil
}

func (s *Service) GetWaypoint(ctx context.Context, id string) (Waypoint, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id, name, description, type, ST_Y(location::geometry), ST_X(location::geometry),
		       COALESCE(elevation_m,0), created_by, is_verified, created_at
		FROM waypoints WHERE id=$1
	`, id)
	var wp Waypoint
	if err := row.Scan(&wp.ID, &wp.Name, &wp.Description, &wp.Type, &wp.Lat, &wp.Lng, &wp.ElevationM, &wp.CreatedBy, &wp.IsVerified, &wp.CreatedAt); err != nil {
		return Waypoint{}, err
	}
	return wp, nil
}

func (s *Service) DeleteWaypoint(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM waypoints WHERE id=$1`, id)
	return err
}

func (s *Service) HasVisited(ctx context.Context, waypointID, userID string) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM waypoints w
			JOIN track_points tp ON ST_DWithin(tp.location, w.location, 50)
			JOIN track_sessions ts ON ts.id = tp.session_id
			WHERE w.id = $1 AND ts.user_id = $2
		)
	`, waypointID, userID).Scan(&ok)
	return ok, err
}

func (s *Service) AddReview(ctx context.Context, waypointID, userID string, rating int, comment string) (Review, error) {
	visited, err := s.HasVisited(ctx, waypointID, userID)
	if err != nil {
		return Review{}, err
	}
	if !visited {
		return Review{}, errors.New("user has not visited waypoint")
	}

	review := Review{
		ID:         uuid.NewString(),
		WaypointID: waypointID,
		UserID:     userID,
		Rating:     rating,
		Comment:    comment,
	}
	row := s.db.QueryRow(ctx, `
		INSERT INTO waypoint_reviews (id, waypoint_id, user_id, rating, comment)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (waypoint_id, user_id) DO UPDATE
		SET rating=EXCLUDED.rating, comment=EXCLUDED.comment
		RETURNING created_at
	`, review.ID, review.WaypointID, review.UserID, review.Rating, review.Comment)
	if err := row.Scan(&review.CreatedAt); err != nil {
		return Review{}, err
	}
	return review, nil
}

func (s *Service) Reviews(ctx context.Context, waypointID string) ([]Review, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, waypoint_id, user_id, rating, comment, created_at
		FROM waypoint_reviews WHERE waypoint_id=$1
		ORDER BY created_at DESC
	`, waypointID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var r Review
		if err := rows.Scan(&r.ID, &r.WaypointID, &r.UserID, &r.Rating, &r.Comment, &r.CreatedAt); err != nil {
			return nil, err
		}
		reviews = append(reviews, r)
	}
	return reviews, nil
}

func (s *Service) AddPhoto(ctx context.Context, waypointID, userID, url, caption string, lat, lng float64, takenAt time.Time) (Photo, error) {
	photo := Photo{
		ID:         uuid.NewString(),
		WaypointID: waypointID,
		UserID:     userID,
		PhotoURL:   url,
		Caption:    caption,
		Lat:        lat,
		Lng:        lng,
		TakenAt:    takenAt,
	}
	row := s.db.QueryRow(ctx, `
		INSERT INTO waypoint_photos (id, waypoint_id, user_id, photo_url, caption, location, taken_at)
		VALUES ($1,$2,$3,$4,$5, ST_SetSRID(ST_MakePoint($6,$7), 4326)::geography, $8)
		RETURNING created_at
	`, photo.ID, photo.WaypointID, photo.UserID, photo.PhotoURL, photo.Caption, photo.Lng, photo.Lat, photo.TakenAt)
	if err := row.Scan(&photo.CreatedAt); err != nil {
		return Photo{}, err
	}
	return photo, nil
}

func (s *Service) Photos(ctx context.Context, waypointID string) ([]Photo, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, waypoint_id, user_id, photo_url, caption, ST_Y(location::geometry), ST_X(location::geometry), taken_at, created_at
		FROM waypoint_photos WHERE waypoint_id=$1
		ORDER BY created_at DESC
	`, waypointID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []Photo
	for rows.Next() {
		var p Photo
		if err := rows.Scan(&p.ID, &p.WaypointID, &p.UserID, &p.PhotoURL, &p.Caption, &p.Lat, &p.Lng, &p.TakenAt, &p.CreatedAt); err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}
	return photos, nil
}

func (s *Service) Search(ctx context.Context, lat, lng, radiusKm float64) ([]Waypoint, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, description, type, ST_Y(location::geometry), ST_X(location::geometry),
		       COALESCE(elevation_m,0), created_by, is_verified, created_at
		FROM waypoints
		WHERE ST_DWithin(location, ST_SetSRID(ST_MakePoint($1,$2), 4326)::geography, $3)
		ORDER BY created_at DESC
	`, lng, lat, radiusKm*1000)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Waypoint
	for rows.Next() {
		var wp Waypoint
		if err := rows.Scan(&wp.ID, &wp.Name, &wp.Description, &wp.Type, &wp.Lat, &wp.Lng, &wp.ElevationM, &wp.CreatedBy, &wp.IsVerified, &wp.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, wp)
	}
	return results, nil
}
