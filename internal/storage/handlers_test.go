package storage

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/pashagolub/pgxmock/v3"
)

func TestStorageUploadHandler(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectExec(`INSERT INTO storage_objects`).
		WithArgs(pgxmock.AnyArg(), "user-1", "https://storage.example/file", "photo").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	app := fiber.New()
	RegisterRoutes(app.Group("/storage"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	body, _ := json.Marshal(map[string]string{"user_id": "user-1", "file_name": "file", "kind": "photo"})
	req := httptest.NewRequest(http.MethodPost, "/storage/upload", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("upload status: %v", err)
	}
}

func TestStorageUploadDefaultFileName(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectExec(`INSERT INTO storage_objects`).
		WithArgs(pgxmock.AnyArg(), "user-1", "https://storage.example/upload", "photo").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	app := fiber.New()
	RegisterRoutes(app.Group("/storage"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	body, _ := json.Marshal(map[string]string{"user_id": "user-1", "kind": "photo"})
	req := httptest.NewRequest(http.MethodPost, "/storage/upload", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("upload status: %v", err)
	}
}

func TestStorageUploadError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectExec(`INSERT INTO storage_objects`).
		WithArgs(pgxmock.AnyArg(), "user-1", "https://storage.example/file", "photo").
		WillReturnError(errSave)

	app := fiber.New()
	RegisterRoutes(app.Group("/storage"), NewService(mock), func(c *fiber.Ctx) error { return c.Next() })

	body, _ := json.Marshal(map[string]string{"user_id": "user-1", "file_name": "file", "kind": "photo"})
	req := httptest.NewRequest(http.MethodPost, "/storage/upload", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected error status")
	}
}
