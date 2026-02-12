package tracking

import "time"

type Session struct {
	ID        string    `json:"id"`
	TripID    string    `json:"trip_id"`
	UserID    string    `json:"user_id"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
	TotalDistanceM      float64 `json:"total_distance_m"`
	TotalElevationGainM float64 `json:"total_elevation_gain_m"`
	Status              string  `json:"status"`
}

type TrackPoint struct {
	ID         int64     `json:"id"`
	SessionID  string    `json:"session_id"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	ElevationM float64   `json:"elevation_m"`
	RecordedAt time.Time `json:"recorded_at"`
	SpeedMps   float64   `json:"speed_mps"`
	CreatedAt  time.Time `json:"created_at"`
}

type Summary struct {
	SessionID     string  `json:"session_id"`
	PointCount    int     `json:"point_count"`
	DistanceM     float64 `json:"distance_m"`
	DurationSec   int64   `json:"duration_sec"`
	AverageSpeedM float64 `json:"average_speed_mps"`
}
