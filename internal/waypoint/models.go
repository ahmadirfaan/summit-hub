package waypoint

import "time"

type Waypoint struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Description string  `json:"description"`
	Type      string    `json:"type"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	ElevationM float64  `json:"elevation_m"`
	CreatedBy string    `json:"created_by"`
	IsVerified bool     `json:"is_verified"`
	CreatedAt time.Time `json:"created_at"`
}

type Review struct {
	ID         string    `json:"id"`
	WaypointID string    `json:"waypoint_id"`
	UserID     string    `json:"user_id"`
	Rating     int       `json:"rating"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
}

type Photo struct {
	ID         string    `json:"id"`
	WaypointID string    `json:"waypoint_id"`
	UserID     string    `json:"user_id"`
	PhotoURL   string    `json:"photo_url"`
	Caption    string    `json:"caption"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	TakenAt    time.Time `json:"taken_at"`
	CreatedAt  time.Time `json:"created_at"`
}
