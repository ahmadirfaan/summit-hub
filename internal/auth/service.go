package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

type Service struct {
	secret []byte
	db     *pgxpool.Pool
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func NewService(secret string, db *pgxpool.Pool) *Service {
	return &Service{
		secret: []byte(secret),
		db:     db,
	}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (User, TokenResponse, error) {
	if req.Email == "" || req.Username == "" || req.Password == "" {
		return User{}, TokenResponse{}, errors.New("email, username, password required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, TokenResponse{}, err
	}

	user := User{
		ID:           uuid.NewString(),
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: string(hash),
		FullName:     req.FullName,
		AvatarURL:    req.AvatarURL,
	}

	row := s.db.QueryRow(ctx, `
		INSERT INTO users (id, email, username, password_hash, full_name, avatar_url)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING created_at, updated_at
	`, user.ID, user.Email, user.Username, user.PasswordHash, user.FullName, user.AvatarURL)
	if err := row.Scan(&user.CreatedAt, &user.UpdatedAt); err != nil {
		return User{}, TokenResponse{}, err
	}

	tokens, err := s.GenerateTokens(ctx, user.ID)
	if err != nil {
		return User{}, TokenResponse{}, err
	}
	return user, tokens, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (User, TokenResponse, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id, email, username, password_hash, full_name, avatar_url, created_at, updated_at
		FROM users WHERE email = $1
	`, req.Email)

	var user User
	if err := row.Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.FullName, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return User{}, TokenResponse{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return User{}, TokenResponse{}, errors.New("invalid credentials")
	}

	tokens, err := s.GenerateTokens(ctx, user.ID)
	if err != nil {
		return User{}, TokenResponse{}, err
	}
	return user, tokens, nil
}

func (s *Service) GenerateTokens(ctx context.Context, userID string) (TokenResponse, error) {
	access, err := s.signToken(userID, accessTokenTTL)
	if err != nil {
		return TokenResponse{}, err
	}

	refresh, err := s.signToken(userID, refreshTokenTTL)
	if err != nil {
		return TokenResponse{}, err
	}

	if err := s.saveRefreshToken(ctx, refresh, userID, refreshTokenTTL); err != nil {
		return TokenResponse{}, err
	}

	return TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    int64(accessTokenTTL.Seconds()),
	}, nil
}

func (s *Service) ValidateRefreshToken(ctx context.Context, token string) (string, error) {
	claims, err := s.parseToken(token)
	if err != nil {
		return "", err
	}

	userID, expiresAt, err := s.lookupRefreshToken(ctx, token)
	if err != nil || userID != claims.UserID || time.Now().After(expiresAt) {
		return "", errors.New("refresh token invalid")
	}
	return claims.UserID, nil
}

func (s *Service) ValidateAccessToken(token string) (string, error) {
	claims, err := s.parseToken(token)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

func (s *Service) signToken(userID string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *Service) parseToken(token string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(_ *jwt.Token) (interface{}, error) {
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("token invalid")
	}
	return claims, nil
}

func (s *Service) saveRefreshToken(ctx context.Context, token, userID string, ttl time.Duration) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at)
		VALUES ($1,$2,$3,$4)
	`, uuid.NewString(), userID, token, time.Now().Add(ttl))
	return err
}

func (s *Service) lookupRefreshToken(ctx context.Context, token string) (string, time.Time, error) {
	row := s.db.QueryRow(ctx, `
		SELECT user_id, expires_at
		FROM refresh_tokens
		WHERE token = $1 AND revoked_at IS NULL
	`, token)
	var userID string
	var expiresAt time.Time
	if err := row.Scan(&userID, &expiresAt); err != nil {
		return "", time.Time{}, err
	}
	return userID, expiresAt, nil
}
