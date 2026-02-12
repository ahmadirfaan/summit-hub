package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pashagolub/pgxmock/v3"
	"golang.org/x/crypto/bcrypt"
)

func TestRegisterAndLogin(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	createdAt := time.Now().Add(-time.Minute)
	updatedAt := time.Now().Add(-time.Minute)

	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs(pgxmock.AnyArg(), "user@example.com", "user", pgxmock.AnyArg(), "User One", "").
		WillReturnRows(pgxmock.NewRows([]string{"created_at", "updated_at"}).AddRow(createdAt, updatedAt))

	mock.ExpectExec(`INSERT INTO refresh_tokens`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	svc := NewService("test-secret", mock)
	user, tokens, err := svc.Register(context.Background(), RegisterRequest{
		Email:    "user@example.com",
		Username: "user",
		Password: "password123",
		FullName: "User One",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.ID == "" || tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("expected user and tokens")
	}

	passwordHash := user.PasswordHash

	mock.ExpectQuery(`SELECT id, email, username, password_hash, full_name, avatar_url, created_at, updated_at`).
		WithArgs("user@example.com").
		WillReturnRows(pgxmock.NewRows([]string{"id", "email", "username", "password_hash", "full_name", "avatar_url", "created_at", "updated_at"}).
			AddRow(user.ID, user.Email, user.Username, passwordHash, user.FullName, user.AvatarURL, createdAt, updatedAt))

	mock.ExpectExec(`INSERT INTO refresh_tokens`).
		WithArgs(pgxmock.AnyArg(), user.ID, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	_, loginTokens, err := svc.Login(context.Background(), LoginRequest{Email: "user@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if loginTokens.AccessToken == "" || loginTokens.RefreshToken == "" {
		t.Fatalf("expected login tokens")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestValidateRefreshToken(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectExec(`INSERT INTO refresh_tokens`).
		WithArgs(pgxmock.AnyArg(), "user-1", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	svc := NewService("test-secret", mock)
	tokens, err := svc.GenerateTokens(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("generate tokens: %v", err)
	}

	expiresAt := time.Now().Add(5 * time.Minute)
	mock.ExpectQuery(`SELECT user_id, expires_at`).
		WithArgs(tokens.RefreshToken).
		WillReturnRows(pgxmock.NewRows([]string{"user_id", "expires_at"}).AddRow("user-1", expiresAt))

	userID, err := svc.ValidateRefreshToken(context.Background(), tokens.RefreshToken)
	if err != nil {
		t.Fatalf("validate refresh: %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("unexpected user_id: %s", userID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRegisterMissingFields(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	svc := NewService("test-secret", mock)
	_, _, err = svc.Register(context.Background(), RegisterRequest{Email: "", Username: "u", Password: "p"})
	if err == nil {
		t.Fatalf("expected error for missing email")
	}
}

func TestLoginInvalidPassword(t *testing.T) {
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

	svc := NewService("test-secret", mock)
	_, _, err = svc.Login(context.Background(), LoginRequest{Email: "user@example.com", Password: "wrong"})
	if err == nil {
		t.Fatalf("expected invalid credentials")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGenerateTokensSaveRefreshError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectExec(`INSERT INTO refresh_tokens`).
		WithArgs(pgxmock.AnyArg(), "user-1", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgErr)

	svc := NewService("test-secret", mock)
	_, err = svc.GenerateTokens(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestGenerateTokensAccessSignError(t *testing.T) {
	oldSign := signTokenFn
	signTokenFn = func(_ *Service, _ string, _ time.Duration) (string, error) {
		return "", pgErr
	}
	defer func() { signTokenFn = oldSign }()

	svc := NewService("test-secret", nil)
	_, err := svc.GenerateTokens(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestGenerateTokensRefreshSignError(t *testing.T) {
	oldSign := signTokenFn
	call := 0
	signTokenFn = func(_ *Service, _ string, _ time.Duration) (string, error) {
		call++
		if call == 2 {
			return "", pgErr
		}
		return "token", nil
	}
	defer func() { signTokenFn = oldSign }()

	svc := NewService("test-secret", nil)
	_, err := svc.GenerateTokens(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRegisterHashError(t *testing.T) {
	oldHash := hashPasswordFn
	hashPasswordFn = func(_ []byte, _ int) ([]byte, error) {
		return nil, pgErr
	}
	defer func() { hashPasswordFn = oldHash }()

	svc := NewService("test-secret", nil)
	_, _, err := svc.Register(context.Background(), RegisterRequest{Email: "user@example.com", Username: "user", Password: "pass"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseTokenInvalid(t *testing.T) {
	oldParse := parseWithClaimsFn
	parseWithClaimsFn = func(_ string, _ jwt.Claims, _ jwt.Keyfunc, _ ...jwt.ParserOption) (*jwt.Token, error) {
		return &jwt.Token{Valid: false, Claims: &Claims{}}, nil
	}
	defer func() { parseWithClaimsFn = oldParse }()

	svc := NewService("test-secret", nil)
	_, err := svc.parseToken("token")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestValidateAccessTokenInvalid(t *testing.T) {
	svc := NewService("test-secret", nil)
	_, err := svc.ValidateAccessToken("invalid-token")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRegisterDBError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs(pgxmock.AnyArg(), "user@example.com", "user", pgxmock.AnyArg(), "", "").
		WillReturnError(pgErr)

	svc := NewService("test-secret", mock)
	_, _, err = svc.Register(context.Background(), RegisterRequest{Email: "user@example.com", Username: "user", Password: "pass"})
	if err == nil {
		t.Fatalf("expected db error")
	}
}

func TestLoginQueryError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, email, username, password_hash, full_name, avatar_url, created_at, updated_at`).
		WithArgs("user@example.com").
		WillReturnError(pgErr)

	svc := NewService("test-secret", mock)
	_, _, err = svc.Login(context.Background(), LoginRequest{Email: "user@example.com", Password: "pass"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRegisterGenerateTokensError(t *testing.T) {
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
		WillReturnError(pgErr)

	svc := NewService("test-secret", mock)
	_, _, err = svc.Register(context.Background(), RegisterRequest{Email: "user@example.com", Username: "user", Password: "pass"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoginGenerateTokensError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)

	mock.ExpectQuery(`SELECT id, email, username, password_hash, full_name, avatar_url, created_at, updated_at`).
		WithArgs("user@example.com").
		WillReturnRows(pgxmock.NewRows([]string{"id", "email", "username", "password_hash", "full_name", "avatar_url", "created_at", "updated_at"}).
			AddRow("user-1", "user@example.com", "user", string(hash), "", "", time.Now(), time.Now()))

	mock.ExpectExec(`INSERT INTO refresh_tokens`).
		WithArgs(pgxmock.AnyArg(), "user-1", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(pgErr)

	svc := NewService("test-secret", mock)
	_, _, err = svc.Login(context.Background(), LoginRequest{Email: "user@example.com", Password: "pass"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestValidateRefreshTokenExpired(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	mock.ExpectExec(`INSERT INTO refresh_tokens`).
		WithArgs(pgxmock.AnyArg(), "user-2", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	svc := NewService("test-secret", mock)
	tokens, err := svc.GenerateTokens(context.Background(), "user-2")
	if err != nil {
		t.Fatalf("generate tokens: %v", err)
	}

	mock.ExpectQuery(`SELECT user_id, expires_at`).
		WithArgs(tokens.RefreshToken).
		WillReturnRows(pgxmock.NewRows([]string{"user_id", "expires_at"}).AddRow("user-2", time.Now().Add(-time.Minute)))

	_, err = svc.ValidateRefreshToken(context.Background(), tokens.RefreshToken)
	if err == nil {
		t.Fatalf("expected expired token error")
	}
}

func TestValidateRefreshTokenLookupError(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("mock pool: %v", err)
	}
	defer mock.Close()

	svc := NewService("test-secret", mock)

	mock.ExpectExec(`INSERT INTO refresh_tokens`).
		WithArgs(pgxmock.AnyArg(), "user-3", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	tokens, err := svc.GenerateTokens(context.Background(), "user-3")
	if err != nil {
		t.Fatalf("generate tokens: %v", err)
	}

	mock.ExpectQuery(`SELECT user_id, expires_at`).
		WithArgs(tokens.RefreshToken).
		WillReturnError(pgErr)

	_, err = svc.ValidateRefreshToken(context.Background(), tokens.RefreshToken)
	if err == nil {
		t.Fatalf("expected error")
	}
}

var pgErr = errors.New("db error")
