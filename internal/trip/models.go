package trip

import "time"

type Trip struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Mountain  string    `json:"mountain_name"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Description string  `json:"description"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type TripMember struct {
	TripID    string    `json:"trip_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	JoinedAt  time.Time `json:"joined_at"`
}

type GPXRoute struct {
	ID         string    `json:"id"`
	TripID     string    `json:"trip_id"`
	Name       string    `json:"name"`
	Description string   `json:"description"`
	TotalDistanceM float64 `json:"total_distance_m"`
	TotalElevationGainM float64 `json:"total_elevation_gain_m"`
	RouteWKT   string    `json:"route"`
	UploadedBy string    `json:"uploaded_by"`
	CreatedAt  time.Time `json:"created_at"`
}
