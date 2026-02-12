package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/pashagolub/pgxmock/v3"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthHandlersRegisterLoginVerify(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	createdAt := time.Now()
	updatedAt := time.Now()

	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs(pgxmock.AnyArg(), "user@example.com", "user", pgxmock.AnyArg(), "", "").
		WillReturnRows(pgxmock.NewRows([]string{"created_at", "updated_at"}).AddRow(createdAt, updatedAt))
	mock.ExpectExec(`INSERT INTO refresh_tokens`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	svc := NewService("test-secret", mock)
	app := fiber.New()
	RegisterRoutes(app.Group("/auth"), svc)

	registerBody, _ := json.Marshal(RegisterRequest{Email: "user@example.com", Username: "user", Password: "pass"})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status: %v", err)
	}

	passwordBytes, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	passwordHash := string(passwordBytes)
	mock.ExpectQuery(`SELECT id, email, username, password_hash, full_name, avatar_url, created_at, updated_at`).
		WithArgs("user@example.com").
		WillReturnRows(pgxmock.NewRows([]string{"id", "email", "username", "password_hash", "full_name", "avatar_url", "created_at", "updated_at"}).
			AddRow("user-1", "user@example.com", "user", passwordHash, "", "", createdAt, updatedAt))
	mock.ExpectExec(`INSERT INTO refresh_tokens`).
		WithArgs(pgxmock.AnyArg(), "user-1", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	loginBody, _ := json.Marshal(LoginRequest{Email: "user@example.com", Password: "pass"})
	req = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("login status: %v", err)
	}

	mock.ExpectExec(`INSERT INTO refresh_tokens`).
		WithArgs(pgxmock.AnyArg(), "user-1", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	tokens, _ := svc.GenerateTokens(context.Background(), "user-1")

	req = httptest.NewRequest(http.MethodGet, "/auth/jwt/verify", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("verify status: %v", err)
	}
}

func TestAuthRefreshInvalidToken(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/auth"), NewService("test-secret", nil))

	body := []byte(`{"refresh_token":"bad"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized")
	}
}

func TestAuthRefreshSuccess(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	svc := NewService("secret", mock)

	mock.ExpectExec(`INSERT INTO refresh_tokens`).
		WithArgs(pgxmock.AnyArg(), "user-1", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	refresh, err := svc.GenerateTokens(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("generate tokens: %v", err)
	}

	mock.ExpectQuery(`SELECT user_id, expires_at`).
		WithArgs(refresh.RefreshToken).
		WillReturnRows(pgxmock.NewRows([]string{"user_id", "expires_at"}).AddRow("user-1", time.Now().Add(5*time.Minute)))

	mock.ExpectExec(`INSERT INTO refresh_tokens`).
		WithArgs(pgxmock.AnyArg(), "user-1", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	app := fiber.New()
	RegisterRoutes(app.Group("/auth"), svc)

	body, _ := json.Marshal(map[string]string{"refresh_token": refresh.RefreshToken})
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("refresh status: %v", err)
	}
}

func TestAuthRegisterBadPayload(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/auth"), NewService("secret", nil))

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}

func TestAuthLoginBadRequest(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/auth"), NewService("secret", nil))

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte(`{"email":""}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}

func TestAuthRefreshBadRequest(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/auth"), NewService("secret", nil))

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad request")
	}
}

func TestAuthVerifyMissingBearer(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/auth"), NewService("secret", nil))

	req := httptest.NewRequest(http.MethodGet, "/auth/jwt/verify", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized")
	}
}

func TestAuthVerifyInvalidToken(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app.Group("/auth"), NewService("secret", nil))

	req := httptest.NewRequest(http.MethodGet, "/auth/jwt/verify", nil)
	req.Header.Set("Authorization", "Bearer bad")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized")
	}
}

func TestParseBearer(t *testing.T) {
	if parseBearer("bad") != "" {
		t.Fatalf("expected empty token")
	}
	if parseBearer("Bearer token") != "token" {
		t.Fatalf("expected token")
	}
}

func TestAuthRegisterServiceError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs(pgxmock.AnyArg(), "user@example.com", "user", pgxmock.AnyArg(), "", "").
		WillReturnError(pgErr)

	app := fiber.New()
	RegisterRoutes(app.Group("/auth"), NewService("secret", mock))

	body, _ := json.Marshal(RegisterRequest{Email: "user@example.com", Username: "user", Password: "pass"})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected register error")
	}
}

func TestAuthLoginUnauthorized(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	mock.ExpectQuery(`SELECT id, email, username, password_hash, full_name, avatar_url, created_at, updated_at`).
		WithArgs("user@example.com").
		WillReturnRows(pgxmock.NewRows([]string{"id", "email", "username", "password_hash", "full_name", "avatar_url", "created_at", "updated_at"}).
			AddRow("user-1", "user@example.com", "user", string(hash), "", "", time.Now(), time.Now()))

	app := fiber.New()
	RegisterRoutes(app.Group("/auth"), NewService("secret", mock))

	body, _ := json.Marshal(LoginRequest{Email: "user@example.com", Password: "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized")
	}
}

func TestAuthRefreshGenerateTokensError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	svc := NewService("secret", mock)
	refresh, err := svc.signToken("user-1", refreshTokenTTL)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	mock.ExpectQuery(`SELECT user_id, expires_at`).
		WithArgs(refresh).
		WillReturnRows(pgxmock.NewRows([]string{"user_id", "expires_at"}).AddRow("user-1", time.Now().Add(time.Minute)))

	mock.ExpectExec(`INSERT INTO refresh_tokens`).
		WithArgs(pgxmock.AnyArg(), "user-1", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgErr)

	app := fiber.New()
	RegisterRoutes(app.Group("/auth"), svc)

	body, _ := json.Marshal(map[string]string{"refresh_token": refresh})
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected refresh error")
	}
}
