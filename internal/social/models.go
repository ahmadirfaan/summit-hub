package social

import "time"

type Post struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	Visibility string   `json:"visibility"`
	Photos    []PostPhoto `json:"photos,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Follow struct {
	FollowerID string `json:"follower_id"`
	FollowingID string `json:"following_id"`
}

type PostPhoto struct {
	ID     string `json:"id"`
	PostID string `json:"post_id"`
	URL    string `json:"photo_url"`
	CreatedAt time.Time `json:"created_at"`
}
