package social

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/pashagolub/pgxmock/v3"
)

func TestSocialHandlers(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	createdAt := time.Now()
	mock.ExpectQuery(`INSERT INTO posts`).
		WithArgs(pgxmock.AnyArg(), "user-1", "hello", 106.8, -6.2, "public").
		WillReturnRows(pgxmock.NewRows([]string{"created_at"}).AddRow(createdAt))

	mock.ExpectQuery(`SELECT id, user_id, content, ST_Y\(location::geometry\), ST_X\(location::geometry\), visibility, created_at`).
		WithArgs("user-1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "content", "lat", "lng", "visibility", "created_at"}).
			AddRow("post-1", "user-1", "hello", -6.2, 106.8, "public", createdAt))

	mock.ExpectQuery(`SELECT id, post_id, photo_url, created_at`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "post_id", "photo_url", "created_at"}))

	app := fiber.New()
	RegisterRoutes(app.Group("/social"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	body, _ := json.Marshal(Post{UserID: "user-1", Content: "hello", Lat: -6.2, Lng: 106.8})
	req := httptest.NewRequest(http.MethodPost, "/social/posts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("create post status: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/social/feed?user_id=user-1", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("feed status: %v", err)
	}
}

func TestSocialHandlersBadRequest(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/social"), NewService(nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodPost, "/social/posts", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}

func TestSocialFeedMissingUser(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/social"), NewService(nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodGet, "/social/feed", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}

func TestSocialHandlersPhotoAndFollow(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	photoCreated := time.Now()
	mock.ExpectQuery(`INSERT INTO post_photos`).
		WithArgs(pgxmock.AnyArg(), "post-1", "https://photo").
		WillReturnRows(pgxmock.NewRows([]string{"created_at"}).AddRow(photoCreated))

	mock.ExpectExec(`INSERT INTO user_follows`).
		WithArgs("user-1", "user-2").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	app := fiber.New()
	RegisterRoutes(app.Group("/social"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	body, _ := json.Marshal(map[string]string{"photo_url": "https://photo"})
	req := httptest.NewRequest(http.MethodPost, "/social/posts/post-1/photos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("photo status: %v", err)
	}

	followBody, _ := json.Marshal(Follow{FollowerID: "user-1", FollowingID: "user-2"})
	req = httptest.NewRequest(http.MethodPost, "/social/follow", bytes.NewReader(followBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("follow status: %v", err)
	}
}

func TestSocialHandlersNearby(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	createdAt := time.Now()
	mock.ExpectQuery(`SELECT id, user_id, content, ST_Y\(location::geometry\), ST_X\(location::geometry\), visibility, created_at`).
		WithArgs(106.8, -6.2, 5000.0).
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "content", "lat", "lng", "visibility", "created_at"}).
			AddRow("post-1", "user-1", "hello", -6.2, 106.8, "public", createdAt))

	mock.ExpectQuery(`SELECT id, post_id, photo_url, created_at`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "post_id", "photo_url", "created_at"}))

	app := fiber.New()
	RegisterRoutes(app.Group("/social"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodGet, "/social/posts/nearby?lat=-6.2&lng=106.8", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("nearby status: %v", err)
	}
}

func TestSocialHandlersErrors(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO posts`).
		WithArgs(pgxmock.AnyArg(), "user-1", "hello", 0.0, 0.0, "public").
		WillReturnError(errSocial)

	mock.ExpectQuery(`INSERT INTO post_photos`).
		WithArgs(pgxmock.AnyArg(), "post-1", "url").
		WillReturnError(errSocial)

	mock.ExpectExec(`INSERT INTO user_follows`).
		WithArgs("user-1", "user-2").
		WillReturnError(errSocial)

	mock.ExpectQuery(`SELECT id, user_id, content, ST_Y\(location::geometry\), ST_X\(location::geometry\), visibility, created_at`).
		WithArgs("user-1").
		WillReturnError(errSocial)

	mock.ExpectQuery(`SELECT id, user_id, content, ST_Y\(location::geometry\), ST_X\(location::geometry\), visibility, created_at`).
		WithArgs(106.8, -6.2, 5000.0).
		WillReturnError(errSocial)

	app := fiber.New()
	RegisterRoutes(app.Group("/social"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	body, _ := json.Marshal(Post{UserID: "user-1", Content: "hello"})
	req := httptest.NewRequest(http.MethodPost, "/social/posts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected error")
	}

	photoBody, _ := json.Marshal(map[string]string{"photo_url": "url"})
	req = httptest.NewRequest(http.MethodPost, "/social/posts/post-1/photos", bytes.NewReader(photoBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected photo error")
	}

	followBody, _ := json.Marshal(Follow{FollowerID: "user-1", FollowingID: "user-2"})
	req = httptest.NewRequest(http.MethodPost, "/social/follow", bytes.NewReader(followBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected follow error")
	}

	req = httptest.NewRequest(http.MethodGet, "/social/feed?user_id=user-1", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected feed error")
	}

	req = httptest.NewRequest(http.MethodGet, "/social/posts/nearby?lat=-6.2&lng=106.8", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected nearby error")
	}
}

func TestSocialHandlersPhotoBadRequest(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/social"), NewService(nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodPost, "/social/posts/post-1/photos", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}

func TestSocialHandlersFollowBadRequest(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/social"), NewService(nil), func(c *fiber.Ctx) error { return c.Next() })

	req := httptest.NewRequest(http.MethodPost, "/social/follow", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}
