package social

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v3"
)

func TestCreatePostAndPhotos(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	createdAt := time.Now()
	mock.ExpectQuery(`INSERT INTO posts`).
		WithArgs(pgxmock.AnyArg(), "user-1", "hello", 106.8, -6.2, "public").
		WillReturnRows(pgxmock.NewRows([]string{"created_at"}).AddRow(createdAt))

	svc := NewService(mock)
	post, err := svc.CreatePost(context.Background(), Post{UserID: "user-1", Content: "hello", Lat: -6.2, Lng: 106.8})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if post.CreatedAt.IsZero() {
		t.Fatalf("expected created_at")
	}

	photoCreated := time.Now()
	mock.ExpectQuery(`INSERT INTO post_photos`).
		WithArgs(pgxmock.AnyArg(), post.ID, "https://photo").
		WillReturnRows(pgxmock.NewRows([]string{"created_at"}).AddRow(photoCreated))

	photo, err := svc.AddPhoto(context.Background(), post.ID, "https://photo")
	if err != nil {
		t.Fatalf("add photo: %v", err)
	}
	if photo.ID == "" {
		t.Fatalf("expected photo id")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestFeedAndNearby(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	createdAt := time.Now()
	mock.ExpectQuery(`SELECT id, user_id, content, ST_Y\(location::geometry\), ST_X\(location::geometry\), visibility, created_at`).
		WithArgs("user-1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "content", "lat", "lng", "visibility", "created_at"}).
			AddRow("post-1", "user-1", "content", -6.2, 106.8, "public", createdAt))

	mock.ExpectQuery(`SELECT id, post_id, photo_url, created_at`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "post_id", "photo_url", "created_at"}).
			AddRow("photo-1", "post-1", "https://photo", createdAt))

	svc := NewService(mock)
	feed, err := svc.Feed(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("feed: %v", err)
	}
	if len(feed) != 1 || len(feed[0].Photos) != 1 {
		t.Fatalf("unexpected feed result")
	}

	mock.ExpectQuery(`SELECT id, user_id, content, ST_Y\(location::geometry\), ST_X\(location::geometry\), visibility, created_at`).
		WithArgs(106.8, -6.2, 1000.0).
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "content", "lat", "lng", "visibility", "created_at"}).
			AddRow("post-2", "user-2", "near", -6.2, 106.8, "public", createdAt))

	mock.ExpectQuery(`SELECT id, post_id, photo_url, created_at`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "post_id", "photo_url", "created_at"}))

	nearby, err := svc.Nearby(context.Background(), -6.2, 106.8, 1)
	if err != nil {
		t.Fatalf("nearby: %v", err)
	}
	if len(nearby) != 1 {
		t.Fatalf("unexpected nearby result")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestFollow(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectExec(`INSERT INTO user_follows`).
		WithArgs("user-1", "user-2").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	svc := NewService(mock)
	if err := svc.Follow(context.Background(), "user-1", "user-2"); err != nil {
		t.Fatalf("follow: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestFeedEmpty(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, user_id, content, ST_Y\(location::geometry\), ST_X\(location::geometry\), visibility, created_at`).
		WithArgs("user-1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "content", "lat", "lng", "visibility", "created_at"}))

	svc := NewService(mock)
	feed, err := svc.Feed(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("feed: %v", err)
	}
	if len(feed) != 0 {
		t.Fatalf("expected empty feed")
	}
}

func TestCreatePostError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO posts`).
		WithArgs(pgxmock.AnyArg(), "user-1", "hello", 106.8, -6.2, "public").
		WillReturnError(errSocial)

	svc := NewService(mock)
	_, err = svc.CreatePost(context.Background(), Post{UserID: "user-1", Content: "hello", Lat: -6.2, Lng: 106.8})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestAddPhotoError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO post_photos`).
		WithArgs(pgxmock.AnyArg(), "post-1", "url").
		WillReturnError(errSocial)

	svc := NewService(mock)
	_, err = svc.AddPhoto(context.Background(), "post-1", "url")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestFollowError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectExec(`INSERT INTO user_follows`).
		WithArgs("user-1", "user-2").
		WillReturnError(errSocial)

	svc := NewService(mock)
	if err := svc.Follow(context.Background(), "user-1", "user-2"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestFeedQueryError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, user_id, content, ST_Y\(location::geometry\), ST_X\(location::geometry\), visibility, created_at`).
		WithArgs("user-1").
		WillReturnError(errSocial)

	svc := NewService(mock)
	_, err = svc.Feed(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNearbyQueryError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, user_id, content, ST_Y\(location::geometry\), ST_X\(location::geometry\), visibility, created_at`).
		WithArgs(106.8, -6.2, 1000.0).
		WillReturnError(errSocial)

	svc := NewService(mock)
	_, err = svc.Nearby(context.Background(), -6.2, 106.8, 1)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestFeedPhotosQueryError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	createdAt := time.Now()
	mock.ExpectQuery(`SELECT id, user_id, content, ST_Y\(location::geometry\), ST_X\(location::geometry\), visibility, created_at`).
		WithArgs("user-1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "content", "lat", "lng", "visibility", "created_at"}).
			AddRow("post-1", "user-1", "content", -6.2, 106.8, "public", createdAt))

	mock.ExpectQuery(`SELECT id, post_id, photo_url, created_at`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(errSocial)

	svc := NewService(mock)
	_, err = svc.Feed(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestSortPosts(t *testing.T) {
	newer := time.Now()
	older := newer.Add(-time.Hour)
	posts := []Post{
		{ID: "old", CreatedAt: older},
		{ID: "new", CreatedAt: newer},
	}
	sorted := sortPosts(posts)
	if sorted[0].ID != "new" {
		t.Fatalf("expected newest post first")
	}
}

func TestFeedScanError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, user_id, content, ST_Y\(location::geometry\), ST_X\(location::geometry\), visibility, created_at`).
		WithArgs("user-1").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("post-1"))

	svc := NewService(mock)
	_, err = svc.Feed(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNearbyPhotosQueryError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	createdAt := time.Now()
	mock.ExpectQuery(`SELECT id, user_id, content, ST_Y\(location::geometry\), ST_X\(location::geometry\), visibility, created_at`).
		WithArgs(106.8, -6.2, 1000.0).
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "content", "lat", "lng", "visibility", "created_at"}).
			AddRow("post-1", "user-1", "content", -6.2, 106.8, "public", createdAt))

	mock.ExpectQuery(`SELECT id, post_id, photo_url, created_at`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(errSocial)

	svc := NewService(mock)
	_, err = svc.Nearby(context.Background(), -6.2, 106.8, 1)
	if err == nil {
		t.Fatalf("expected error")
	}
}

var errSocial = errors.New("social error")
