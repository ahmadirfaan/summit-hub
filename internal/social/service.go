package social

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) CreatePost(ctx context.Context, input Post) (Post, error) {
	input.ID = uuid.NewString()
	if input.Visibility == "" {
		input.Visibility = "public"
	}
	row := s.db.QueryRow(ctx, `
		INSERT INTO posts (id, user_id, content, location, visibility)
		VALUES ($1,$2,$3, ST_SetSRID(ST_MakePoint($4,$5), 4326)::geography, $6)
		RETURNING created_at
	`, input.ID, input.UserID, input.Content, input.Lng, input.Lat, input.Visibility)
	if err := row.Scan(&input.CreatedAt); err != nil {
		return Post{}, err
	}
	return input, nil
}

func (s *Service) AddPhoto(ctx context.Context, postID, url string) (PostPhoto, error) {
	photo := PostPhoto{
		ID:     uuid.NewString(),
		PostID: postID,
		URL:    url,
	}
	row := s.db.QueryRow(ctx, `
		INSERT INTO post_photos (id, post_id, photo_url)
		VALUES ($1,$2,$3)
		RETURNING created_at
	`, photo.ID, photo.PostID, photo.URL)
	if err := row.Scan(&photo.CreatedAt); err != nil {
		return PostPhoto{}, err
	}
	return photo, nil
}

func (s *Service) Follow(ctx context.Context, followerID, followingID string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO user_follows (follower_id, following_id)
		VALUES ($1,$2)
		ON CONFLICT DO NOTHING
	`, followerID, followingID)
	return err
}

func (s *Service) Feed(ctx context.Context, userID string) ([]Post, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, content, ST_Y(location::geometry), ST_X(location::geometry), visibility, created_at
		FROM posts
		WHERE user_id=$1
		   OR user_id IN (SELECT following_id FROM user_follows WHERE follower_id=$1)
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	var ids []string
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.UserID, &p.Content, &p.Lat, &p.Lng, &p.Visibility, &p.CreatedAt); err != nil {
			return nil, err
		}
		ids = append(ids, p.ID)
		posts = append(posts, p)
	}

	photos, err := s.loadPhotos(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range posts {
		posts[i].Photos = photos[posts[i].ID]
	}
	return posts, nil
}

func (s *Service) Nearby(ctx context.Context, lat, lng, radiusKm float64) ([]Post, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, content, ST_Y(location::geometry), ST_X(location::geometry), visibility, created_at
		FROM posts
		WHERE ST_DWithin(location, ST_SetSRID(ST_MakePoint($1,$2), 4326)::geography, $3)
		ORDER BY created_at DESC
	`, lng, lat, radiusKm*1000)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	var ids []string
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.UserID, &p.Content, &p.Lat, &p.Lng, &p.Visibility, &p.CreatedAt); err != nil {
			return nil, err
		}
		ids = append(ids, p.ID)
		posts = append(posts, p)
	}
	photos, err := s.loadPhotos(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range posts {
		posts[i].Photos = photos[posts[i].ID]
	}
	return sortPosts(posts), nil
}

func (s *Service) loadPhotos(ctx context.Context, postIDs []string) (map[string][]PostPhoto, error) {
	if len(postIDs) == 0 {
		return map[string][]PostPhoto{}, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, post_id, photo_url, created_at
		FROM post_photos WHERE post_id = ANY($1)
		ORDER BY created_at
	`, postIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	photos := map[string][]PostPhoto{}
	for rows.Next() {
		var p PostPhoto
		if err := rows.Scan(&p.ID, &p.PostID, &p.URL, &p.CreatedAt); err != nil {
			return nil, err
		}
		photos[p.PostID] = append(photos[p.PostID], p)
	}
	return photos, nil
}

func sortPosts(posts []Post) []Post {
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].CreatedAt.After(posts[j].CreatedAt)
	})
	return posts
}
